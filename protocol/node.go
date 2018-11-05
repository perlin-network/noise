package protocol

import (
	"bytes"
	"github.com/perlin-network/noise/log"
	"github.com/pkg/errors"
	"sync"
)

type Service func(message *Message)

type Node struct {
	controller  *Controller
	connAdapter ConnectionAdapter
	idAdapter   IdentityAdapter
	peers       sync.Map // string -> *PendingPeer | *EstablishedPeer
	services    map[uint16]Service
}

func NewNode(c *Controller, ca ConnectionAdapter, id IdentityAdapter) *Node {
	return &Node{
		controller:  c,
		connAdapter: ca,
		idAdapter:   id,
		services:    make(map[uint16]Service),
	}
}

func (n *Node) AddService(id uint16, s Service) {
	n.services[id] = s
}

func (n *Node) dispatchIncomingMessage(peer *EstablishedPeer, raw []byte) {
	if peer.kxState != KeyExchange_Done {
		err := peer.continueKeyExchange(n.controller, n.idAdapter, raw)
		if err != nil {
			log.Error().Err(err).Msg("cannot continue key exchange")
			n.peers.Delete(string(peer.RemoteEndpoint()))
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
			peer, err := EstablishPeerWithMessageAdapter(n.controller, n.idAdapter, adapter, true)
			if err != nil {
				log.Error().Err(err).Msg("cannot establish peer")
				continue
			}

			n.peers.Store(string(adapter.RemoteEndpoint()), peer)
			adapter.StartRecvMessage(n.controller, func(message []byte) {
				n.dispatchIncomingMessage(peer, message)
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
				established, err = EstablishPeerWithMessageAdapter(n.controller, n.idAdapter, msgAdapter, false)
				if err != nil {
					established = nil
					msgAdapter = nil
					n.peers.Delete(string(remote))
					log.Error().Err(err).Msg("cannot establish peer")
				} else {
					n.peers.Store(string(remote), established)
					msgAdapter.StartRecvMessage(n.controller, func(message []byte) {
						n.dispatchIncomingMessage(established, message)
					})
				}
			} else {
				n.peers.Delete(string(remote))
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
		n.peers.Delete(string(message.Recipient))
		return err
	}

	return nil
}
