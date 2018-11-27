package protocol

import (
	"bytes"
	"context"
	"encoding/hex"
	"github.com/monnand/dhkx"
	"github.com/perlin-network/noise/log"
	"github.com/pkg/errors"
	"sync"
	"sync/atomic"
)

type Node struct {
	controller               *Controller
	connAdapter              ConnectionAdapter
	idAdapter                IdentityAdapter
	peers                    sync.Map // string -> *PendingPeer | *EstablishedPeer
	services                 []ServiceInterface
	dhGroup                  *dhkx.DHGroup
	dhKeypair                *dhkx.DHKey
	customHandshakeProcessor HandshakeProcessor

	// uint64 -> *RequestState
	Requests     sync.Map
	RequestNonce uint64
}

// RequestState represents a state of a request.
type RequestState struct {
	data        chan *MessageBody
	closeSignal chan struct{}
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
		controller:   c,
		connAdapter:  ca,
		idAdapter:    id,
		services:     []ServiceInterface{},
		dhGroup:      g,
		dhKeypair:    privKey,
		RequestNonce: 0,
	}
}

func (n *Node) SetCustomHandshakeProcessor(p HandshakeProcessor) {
	n.customHandshakeProcessor = p
}

func (n *Node) AddService(s ServiceInterface) {
	n.services = append(n.services, s)
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

		for _, svc := range n.services {
			svc.PeerDisconnect(id)
		}
	}
}

func (n *Node) dispatchIncomingMessageAsync(peer *EstablishedPeer, raw []byte) {
	if peer.kxState != KeyExchange_Done {
		err := peer.continueKeyExchange(n.controller, n.idAdapter, n.customHandshakeProcessor, raw)
		if err != nil {
			n.removePeer(peer.RemoteEndpoint())
			log.Error().Msgf("cannot continue key exchange: %v", err)
		}
		return
	}

	go func() {
		if err := n.dispatchIncomingMessage(peer, raw); err != nil {
			log.Warn().Msgf("%+v", err)
		}
	}()
}

func (n *Node) dispatchIncomingMessage(peer *EstablishedPeer, raw []byte) error {
	_body, err := peer.UnwrapMessage(n.controller, raw)
	if err != nil {
		return errors.Wrap(err, "cannot unwrap message")
	}

	body, err := DeserializeMessageBody(bytes.NewReader(_body))
	if err != nil {
		return errors.Wrap(err, "cannot deserialize message body")
	}

	if rq, ok := n.Requests.Load(body.RequestNonce); ok {
		rq := rq.(*RequestState)
		rq.data <- body
		return nil
	}

	msg := &Message{
		Sender:    peer.adapter.RemoteEndpoint(),
		Recipient: n.idAdapter.MyIdentity(),
		Body:      body,
		Metadata:  peer.adapter.Metadata(),
	}
	for _, svc := range n.services {
		replyBody, err := svc.Receive(msg)
		if err != nil {
			return errors.Wrapf(err, "Error processing request for service=%d", body.Service)
		}
		if replyBody != nil {
			replyBody.RequestNonce = body.RequestNonce
			if err := n.Send(&Message{
				Sender:    n.idAdapter.MyIdentity(),
				Recipient: peer.adapter.RemoteEndpoint(),
				Body:      replyBody,
			}); err != nil {
				return errors.Wrapf(err, "Error replying to request for service=%d", body.Service)
			}
		}
	}
	return nil
}

func (n *Node) Start() {
	go func() {
		// call startup on all the nodes first
		for _, svc := range n.services {
			svc.Startup(n)
		}

		for adapter := range n.connAdapter.EstablishPassively(n.controller, n.idAdapter.MyIdentity()) {
			adapter := adapter // the outer adapter is shared?
			peer, err := EstablishPeerWithMessageAdapter(n.controller, n.dhGroup, n.dhKeypair, n.idAdapter, adapter, true)
			if err != nil {
				log.Error().Err(err).Msg("cannot establish peer")
				continue
			}
			for _, svc := range n.services {
				svc.PeerConnect(adapter.RemoteEndpoint())
			}

			n.peers.Store(string(adapter.RemoteEndpoint()), peer)
			adapter.StartRecvMessage(n.controller, func(message []byte) {
				if message == nil {
					//fmt.Printf("message is nil, removing peer: %b\n", adapter.RemoteEndpoint())
					n.removePeer(adapter.RemoteEndpoint())
				} else {
					n.dispatchIncomingMessageAsync(peer, message)
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
				log.Error().Msgf("unable to establish connection actively: %+v", err)
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
							n.removePeer(remote)
						} else {
							n.dispatchIncomingMessageAsync(established, message)
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
	if string(message.Recipient) == string(n.idAdapter.MyIdentity()) {
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
	for _, peerPublicKey := range n.connAdapter.GetPeerIDs() {
		if string(peerPublicKey) == string(n.idAdapter.MyIdentity()) {
			// don't sent to yourself
			continue
		}

		// copy the struct
		msg := *msgTemplate
		msg.Recipient = peerPublicKey
		if err := n.Send(&msg); err != nil {
			log.Warn().Msgf("Unable to broadcast to %v: %v", hex.EncodeToString(peerPublicKey), err)
		}
	}

	return nil
}

// Request sends a message and waits for the reply before returning or times out
func (n *Node) Request(ctx context.Context, target []byte, body *MessageBody) (*MessageBody, error) {
	if body == nil {
		return nil, errors.New("message body was empty")
	}
	if body.Service == 0 {
		return nil, errors.New("missing service in message body")
	}
	if string(target) == string(n.idAdapter.MyIdentity()) {
		return nil, errors.New("making request to itself")
	}
	body.RequestNonce = atomic.AddUint64(&n.RequestNonce, 1)
	msg := &Message{
		Sender:    n.idAdapter.MyIdentity(),
		Recipient: target,
		Body:      body,
	}

	// start tracking the request
	channel := make(chan *MessageBody, 1)
	closeSignal := make(chan struct{})

	n.Requests.Store(body.RequestNonce, &RequestState{
		data:        channel,
		closeSignal: closeSignal,
	})

	// send the message
	if err := n.Send(msg); err != nil {
		return nil, err
	}

	// stop tracking the request
	defer close(closeSignal)
	defer n.Requests.Delete(body.RequestNonce)

	select {
	case res := <-channel:
		return res, nil
	case <-ctx.Done():
		return nil, errors.Wrap(ctx.Err(), "Did not receive response")
	}
}

func (n *Node) GetIdentityAdapter() IdentityAdapter {
	return n.idAdapter
}

func (n *Node) GetConnectionAdapter() ConnectionAdapter {
	return n.connAdapter
}
