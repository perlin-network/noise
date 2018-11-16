package protocol

import (
	"bytes"
	_ "fmt"
	"github.com/monnand/dhkx"
	"github.com/perlin-network/noise/log"
	"github.com/pkg/errors"
	"sync"
)

type Service func(message *Message)

type Node struct {
	controller               *Controller
	connAdapter              ConnectionAdapter
	idAdapter                IdentityAdapter
	peers                    sync.Map // string -> *PendingPeer | *EstablishedPeer
	services                 map[uint16]Service
	dhGroup                  *dhkx.DHGroup
	dhKeypair                *dhkx.DHKey
	customHandshakeProcessor HandshakeProcessor
}

func NewNode(c *Controller, ca ConnectionAdapter, id IdentityAdapter) *Node {
	g, err := dhkx.GetGroup(0)
	if err != nil {
		panic(err)
	}

	privKey, err := g.GeneratePrivateKey(nil)
	if err != nil {
		panic(err)
	}

	return &Node{
		controller:  c,
		connAdapter: ca,
		idAdapter:   id,
		services:    make(map[uint16]Service),
		dhGroup:     g,
		dhKeypair:   privKey,
	}
}

func (n *Node) SetCustomHandshakeProcessor(p HandshakeProcessor) {
	n.customHandshakeProcessor = p
}

func (n *Node) AddService(id uint16, s Service) {
	n.services[id] = s
}

func (n *Node) removePeer(id []byte) {
	//fmt.Printf("removing peer: %b\n", id)
	peer, ok := n.peers.Load(string(id))
	if ok {
		if peer, ok := peer.(*EstablishedPeer); ok {
			peer.Close()
		}
		n.peers.Delete(string(id))
		//fmt.Printf("deleting peer: %b\n", id)
	}
}

func (n *Node) dispatchIncomingMessage(peer *EstablishedPeer, raw []byte) {
	if peer.kxState != KeyExchange_Done {
		err := peer.continueKeyExchange(n.controller, n.idAdapter, n.customHandshakeProcessor, raw)
		if err != nil {
			log.Error().Err(err).Msg("cannot continue key exchange")
			n.removePeer(peer.RemoteEndpoint())
		}
		return
	}

	_body, err := peer.UnwrapMessage(n.controller, raw)
	if err != nil {
		log.Error().Err(err).Msg("cannot unwrap message")
	}

	body, err := DeserializeMessageBody(bytes.NewReader(_body))
	if err != nil {
		log.Error().Err(err).Msg("cannot deserialize message body")
	}

	if svc, ok := n.services[body.Service]; ok {
		svc(&Message{
			Sender:    peer.adapter.RemoteEndpoint(),
			Recipient: n.idAdapter.MyIdentity(),
			Body:      body,
		})
	} else {
		log.Debug().Msgf("message to unknown service %d dropped", body.Service)
	}
}

func (n *Node) Start() {
	go func() {
		for adapter := range n.connAdapter.EstablishPassively(n.controller, n.idAdapter.MyIdentity()) {
			adapter := adapter // the outer adapter is shared?
			peer, err := EstablishPeerWithMessageAdapter(n.controller, n.dhGroup, n.dhKeypair, n.idAdapter, adapter, true)
			if err != nil {
				log.Error().Err(err).Msg("cannot establish peer")
				continue
			}

			n.peers.Store(string(adapter.RemoteEndpoint()), peer)
			adapter.StartRecvMessage(n.controller, func(message []byte) {
				if message == nil {
					//fmt.Printf("message is nil, removing peer: %b\n", adapter.RemoteEndpoint())
					//n.removePeer(adapter.RemoteEndpoint())
				} else {
					n.dispatchIncomingMessage(peer, message)
				}
			})
		}
	}()
}

func (n *Node) getPeer(remote []byte) (*EstablishedPeer, error) {
	var established *EstablishedPeer

	peer, loaded := n.peers.LoadOrStore(string(remote), &PendingPeer{Done: make(chan struct{})})
	switch peer := peer.(type) {
	case *PendingPeer:
		if loaded {
			select {
			case <-peer.Done:
				established = peer.Established
				if established == nil {
					return nil, errors.New("cannot establish connection")
				}
			case <-n.controller.Cancellation:
				return nil, errors.New("cancelled")
			}
		} else {
			msgAdapter, err := n.connAdapter.EstablishActively(n.controller, n.idAdapter.MyIdentity(), remote)
			if err != nil {
				log.Error().Err(err).Msg("unable to establish connection actively")
				msgAdapter = nil
			}

			if msgAdapter != nil {
				established, err = EstablishPeerWithMessageAdapter(n.controller, n.dhGroup, n.dhKeypair, n.idAdapter, msgAdapter, false)
				if err != nil {
					established = nil
					msgAdapter = nil
					n.removePeer(remote)
					log.Error().Err(err).Msg("cannot establish peer")
				} else {
					n.peers.Store(string(remote), established)
					msgAdapter.StartRecvMessage(n.controller, func(message []byte) {
						if message == nil {
							//fmt.Printf("getPeer removing since nil\n")
							//n.removePeer(remote)
						} else {
							n.dispatchIncomingMessage(established, message)
						}
					})
				}
			} else {
				n.removePeer(remote)
			}

			close(peer.Done)

			if msgAdapter == nil {
				return nil, errors.New("cannot establish connection")
			}
		}
	case *EstablishedPeer:
		established = peer
	default:
		panic("unexpected peer type")
	}

	<-established.kxDone
	if established.kxState == KeyExchange_Failed {
		return nil, errors.New("key exchange failed")
	} else if established.kxState == KeyExchange_Done {
		return established, nil
	} else {
		panic("invalid kxState")
	}
}

func (n *Node) ManuallyRemovePeer(remote []byte) {
	n.removePeer(remote)
}

func (n *Node) Send(message *Message) error {
	if !bytes.Equal(message.Sender, n.idAdapter.MyIdentity()) {
		return errors.New("sender mismatch")
	}

	peer, err := n.getPeer(message.Recipient)
	if err != nil {
		return err
	}

	err = peer.SendMessage(n.controller, message.Body.Serialize())
	if err != nil {
		n.removePeer(message.Recipient)
		return err
	}

	return nil
}

// Broadcast sends a message body to all it's peers
func (n *Node) Broadcast(body *MessageBody) error {
	msgTemplate := &Message{
		Sender: n.idAdapter.MyIdentity(),
		Body:   body,
	}
	connections := n.connAdapter.GetConnectionIDs()
	for _, peerID := range connections {
		if bytes.Equal(n.idAdapter.MyIdentity(), peerID) {
			// skip sending to itself
			continue
		}
		_, ok := n.peers.Load(string(peerID))
		if ok {
			// HACK: due to coupling, skip if not found in routing table
			continue
		}
		// copy the struct
		msg := *msgTemplate
		msg.Recipient = peerID
		n.Send(&msg)
	}
	return nil
}

func (n *Node) GetIdentityAdapter() IdentityAdapter {
	return n.idAdapter
}
