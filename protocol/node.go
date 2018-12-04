package protocol

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"github.com/monnand/dhkx"
	"github.com/perlin-network/noise/log"
	"github.com/pkg/errors"
	"sync"
	"sync/atomic"
	"time"
)

var (
	_ SendAdapter = (*Node)(nil)
)

// Node is a struct that wraps all the send/receive handlers
type Node struct {
	controller  *Controller
	connAdapter ConnectionAdapter
	idAdapter   IdentityAdapter

	services                 []ServiceInterface
	dhGroup                  *dhkx.DHGroup
	dhKeypair                *dhkx.DHKey
	customHandshakeProcessor HandshakeProcessor

	// string -> *PendingPeer | *EstablishedPeer
	peers sync.Map

	// uint64 -> *RequestState
	Requests     sync.Map
	RequestNonce uint64
}

// RequestState represents a state of a request.
type RequestState struct {
	data        chan *MessageBody
	closeSignal chan struct{}
	requestTime time.Time
}

// NewNode constructs a new instance of Node
func NewNode(c *Controller, id IdentityAdapter) *Node {
	dhGroup, err := dhkx.GetGroup(0)
	if err != nil {
		panic(err)
	}

	dhKeypair, err := dhGroup.GeneratePrivateKey(nil)
	if err != nil {
		panic(err)
	}

	return &Node{
		controller:   c,
		idAdapter:    id,
		services:     []ServiceInterface{},
		dhGroup:      dhGroup,
		dhKeypair:    dhKeypair,
		RequestNonce: 0,
	}
}

func (n *Node) SetConnectionAdapter(ca ConnectionAdapter) {
	n.connAdapter = ca
}

func (n *Node) SetCustomHandshakeProcessor(p HandshakeProcessor) {
	n.customHandshakeProcessor = p
}

func (n *Node) AddService(s ServiceInterface) {
	n.services = append(n.services, s)
}

func (n *Node) GetIdentityAdapter() IdentityAdapter {
	return n.idAdapter
}

func (n *Node) GetConnectionAdapter() ConnectionAdapter {
	return n.connAdapter
}

func (n *Node) ManuallyRemovePeer(remote []byte) {
	n.removePeer(remote)
}

func (n *Node) removePeer(id []byte) {
	peer, ok := n.peers.Load(string(id))
	if ok {
		if peer, ok := peer.(*EstablishedPeer); ok {
			peer.Close()
		}
		n.peers.Delete(string(id))

		for _, svc := range n.services {
			svc.PeerDisconnect(id)
		}
	}
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
					return nil, errors.New("cannot establish connection, established is nil")
				}
			case <-n.controller.Cancellation:
				return nil, errors.New("cancelled")
			}
		} else {
			msgAdapter, err := n.connAdapter.Dial(n.controller, n.idAdapter.MyIdentity(), remote)
			if err != nil {
				log.Error().
					Err(err).
					Msgf("unable to establish connection actively")
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
					msgAdapter.OnRecvMessage(n.controller, func(ctx context.Context, message []byte) {
						if message == nil {
							n.removePeer(remote)
						} else {
							n.dispatchIncomingMessage(ctx, established, message)
						}
					})
				}
			} else {
				n.removePeer(remote)
			}

			close(peer.Done)

			if msgAdapter == nil {
				return nil, errors.New("cannot establish connection, msgAdapter is nil")
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

func (n *Node) dispatchIncomingMessage(ctx context.Context, peer *EstablishedPeer, raw []byte) {
	if peer.kxState != KeyExchange_Done {
		if err := peer.continueKeyExchange(n.controller, n.idAdapter, n.customHandshakeProcessor, raw); err != nil {
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

	go func() {
		if err := n.processMessageBody(ctx, peer, body); err != nil {
			log.Warn().Msgf("%+v", err)
		}
	}()
}

func (n *Node) processMessageBody(ctx context.Context, peer *EstablishedPeer, body *MessageBody) error {

	// see if there is a matching request/response waiting for this nonce
	if rq, ok := n.Requests.Load(makeRequestReplyKey(peer.adapter.RemoteEndpoint(), body.RequestNonce)); ok {
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

	// forward the message to the services
	for _, svc := range n.services {
		replyBody, err := svc.Receive(ctx, msg)
		if err != nil {
			return errors.Wrapf(err, "Error processing request for service=%d", body.Service)
		}
		if replyBody != nil {
			// if there is a reply body, send it back to the sender
			replyBody.RequestNonce = body.RequestNonce
			if err := n.Send(context.Background(), &Message{
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

// Start causes the node to start listening for connections
func (n *Node) Start() {
	if n.connAdapter == nil {
		log.Fatal().Msg("connection adapter not setup")
	}
	go func() {
		// call startup on all the nodes first
		for _, svc := range n.services {
			svc.Startup(n)
		}

		for msgAdapter := range n.connAdapter.Accept(n.controller, n.idAdapter.MyIdentity()) {
			msgAdapter := msgAdapter // the outer adapter is shared?
			peer, err := EstablishPeerWithMessageAdapter(n.controller, n.dhGroup, n.dhKeypair, n.idAdapter, msgAdapter, true)
			if err != nil {
				log.Error().Err(err).Msg("cannot establish peer")
				continue
			}
			for _, svc := range n.services {
				svc.PeerConnect(msgAdapter.RemoteEndpoint())
			}

			n.peers.Store(string(msgAdapter.RemoteEndpoint()), peer)
			msgAdapter.OnRecvMessage(n.controller, func(ctx context.Context, message []byte) {
				if message == nil {
					//fmt.Printf("message is nil, removing peer: %b\n", adapter.RemoteEndpoint())
					n.removePeer(msgAdapter.RemoteEndpoint())
				} else {
					n.dispatchIncomingMessage(ctx, peer, message)
				}
			})
		}
	}()
}

// Stop terminates all connections for the node
func (n *Node) Stop() {
	n.peers.Range(func(remote interface{}, established interface{}) bool {
		id := remote.(string)
		if peer, ok := established.(*EstablishedPeer); ok {
			peer.Close()
		}
		n.peers.Delete(id)

		for _, svc := range n.services {
			svc.PeerDisconnect([]byte(id))
		}
		return true
	})
}

// Send will send a message to the recipient in the message field
func (n *Node) Send(ctx context.Context, message *Message) error {
	if bytes.Equal(message.Recipient, n.idAdapter.MyIdentity()) {
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
func (n *Node) Broadcast(ctx context.Context, body *MessageBody) error {
	msgTemplate := &Message{
		Sender: n.idAdapter.MyIdentity(),
		Body:   body,
	}
	for _, peerPublicKey := range n.connAdapter.GetPeerIDs() {
		if bytes.Equal(peerPublicKey, n.idAdapter.MyIdentity()) {
			// don't sent to yourself
			continue
		}

		// copy the struct
		msg := *msgTemplate
		msg.Recipient = peerPublicKey
		if err := n.Send(ctx, &msg); err != nil {
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

	n.Requests.Store(makeRequestReplyKey(msg.Recipient, body.RequestNonce), &RequestState{
		data:        channel,
		closeSignal: closeSignal,
		requestTime: time.Now(),
	})

	// send the message
	if err := n.Send(ctx, msg); err != nil {
		return nil, err
	}

	// stop tracking the request
	defer close(closeSignal)
	defer n.Requests.Delete(makeRequestReplyKey(msg.Recipient, body.RequestNonce))

	select {
	case res := <-channel:
		return res, nil
	case <-ctx.Done():
		return nil, errors.Wrap(ctx.Err(), "Did not receive response")
	}
}

// makeRequestReplyKey generates a key to map a request reply
func makeRequestReplyKey(receiver []byte, nonce uint64) string {
	return fmt.Sprintf("%s-%d", hex.EncodeToString(receiver), nonce)
}
