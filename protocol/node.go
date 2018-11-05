package protocol

import (
	"bufio"
	"bytes"
	"github.com/perlin-network/noise/log"
	"github.com/pkg/errors"
	"sync"
)

type Service func(message *MessageBody)

type Node struct {
	controller  *Controller
	connAdapter ConnectionAdapter
	idAdapter   IdentityAdapter
	peers       sync.Map // string -> *PendingPeer | MessageAdapter
	services    map[uint16]Service
}

type PendingPeer struct {
	Done    chan struct{}
	Adapter MessageAdapter
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

func (n *Node) dispatchIncomingMessage(raw []byte) {
	msg, err := DeserializeMessage(bufio.NewReader(bytes.NewReader(raw)))
	if err != nil {
		log.Error().Err(err).Msg("unable to deserialize message")
		return
	}

	bodySerialized := msg.Body.Serialize() // FIXME: body is serialized unnecessarily

	if n.idAdapter.Verify(msg.Body.Sender, bodySerialized, msg.Signature) {
		if svc, ok := n.services[msg.Body.Service]; ok {
			svc(msg.Body)
		} else {
			log.Debug().Msgf("message to unknown service %d dropped", msg.Body.Service)
		}
	} else {
		log.Error().Err(err).Msg("unable to verify message")
	}
}

func (n *Node) Start() {
	go func() {
		for adapter := range n.connAdapter.EstablishPassively(n.controller, n.dispatchIncomingMessage) {
			n.peers.Store(adapter.RemoteEndpoint(), adapter)
		}
	}()
}

func (n *Node) getMessageAdapter(remote []byte) (MessageAdapter, error) {
	var msgAdapter MessageAdapter

	peer, loaded := n.peers.LoadOrStore(remote, &PendingPeer{Done: make(chan struct{})})
	switch peer := peer.(type) {
	case *PendingPeer:
		if loaded {
			select {
			case <-peer.Done:
				msgAdapter = peer.Adapter
				if msgAdapter == nil {
					return nil, errors.New("cannot establish connection")
				}
			case <-n.controller.Cancellation:
				return nil, errors.New("cancelled")
			}
		} else {
			var err error
			msgAdapter, err = n.connAdapter.EstablishActively(n.controller, remote, n.dispatchIncomingMessage)
			if err != nil {
				log.Error().Err(err).Msg("unable to establish connection actively")
				msgAdapter = nil
			}

			if msgAdapter != nil {
				n.peers.Store(remote, msgAdapter)
			} else {
				n.peers.Delete(remote)
			}

			peer.Adapter = msgAdapter
			close(peer.Done)

			if msgAdapter == nil {
				return nil, errors.New("cannot establish connection")
			}
		}
	case MessageAdapter:
		msgAdapter = peer
	default:
		panic("unexpected peer type")
	}

	return msgAdapter, nil
}

func (n *Node) Send(body *MessageBody) error {
	msg := &Message{
		Signature: n.idAdapter.Sign(body.Serialize()), // FIXME: body is serialized twice
		Body:      body,
	}
	serialized := msg.Serialize()

	msgAdapter, err := n.getMessageAdapter(body.Recipient)
	if err != nil {
		return err
	}

	err = msgAdapter.SendMessage(n.controller, serialized)
	if err != nil {
		n.peers.Delete(body.Recipient)
		return err
	}

	return nil
}
