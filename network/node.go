package network

import (
	"bufio"
	"bytes"
	"github.com/perlin-network/noise/log"
	"github.com/pkg/errors"
	"sync"
)

type Node struct {
	controller  *Controller
	connAdapter ConnectionAdapter
	idAdapter   IdentityAdapter
	peers       sync.Map // string -> *PendingPeer | MessageAdapter
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
	}
}

func (n *Node) Start() {
	go func() {
		for adapter := range n.connAdapter.EstablishPassively(n.controller) {
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
			msgAdapter, err = n.connAdapter.EstablishActively(n.controller, remote)
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

func (n *Node) Recv(remote []byte) (*MessageBody, error) {
	msgAdapter, err := n.getMessageAdapter(remote)
	if err != nil {
		return nil, err
	}

	raw, err := msgAdapter.RecvMessage(n.controller)
	if err != nil {
		n.peers.Delete(remote)
		return nil, err
	}

	msg, err := DeserializeMessage(bufio.NewReader(bytes.NewReader(raw)))
	if err != nil {
		return nil, err
	}

	bodySerialized := msg.Body.Serialize() // FIXME: body is serialized unnecessarily

	if n.idAdapter.Verify(remote, bodySerialized, msg.Signature) {
		return msg.Body, nil
	} else {
		return nil, errors.New("cannot validate remote identity")
	}
}
