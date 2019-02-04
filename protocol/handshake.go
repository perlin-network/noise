package protocol

import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/callbacks"
	"github.com/perlin-network/noise/log"
	"github.com/pkg/errors"
	"sync"
)

const (
	KeyEstablishSessionCallbacks = "session.establish.callbacks"
	KeyEstablishSessionSignal    = "session.establish.signal"
	KeyEstablishSessionOnce      = "session.establish.once"

	KeyHandshakePolicy = "handshakePolicy"
)

type HandshakePolicy interface {
	EnforceHandshakePolicy(node *noise.Node)
	DoHandshake(peer *noise.Peer, opcode noise.Opcode, message noise.Message) error
	Opcodes() []noise.Opcode
}

func EnforceHandshakePolicy(node *noise.Node, policy HandshakePolicy) HandshakePolicy {
	node.Set(KeyHandshakePolicy, policy)

	node.OnPeerInit(func(node *noise.Node, peer *noise.Peer) error {
		peer.Set(KeyEstablishSessionCallbacks, callbacks.NewSequentialCallbackManager())
		peer.Set(KeyEstablishSessionSignal, make(chan struct{}))
		peer.Set(KeyEstablishSessionOnce, new(sync.Once))

		return nil
	})

	for _, opcode := range policy.Opcodes() {
		node.OnMessageReceived(opcode, func(node *noise.Node, opcode noise.Opcode, peer *noise.Peer, message noise.Message) error {
			err := policy.DoHandshake(peer, opcode, message)

			// Handshake successful; initialize session.
			if errors.Cause(err) == callbacks.DeregisterCallback {
				EstablishSession(peer)
			}

			return err
		})
	}

	policy.EnforceHandshakePolicy(node)

	return policy
}

func HasHandshakePolicy(node *noise.Node) bool {
	return node.Has(KeyHandshakePolicy)
}

func LoadHandshakePolicy(node *noise.Node) HandshakePolicy {
	manager := node.Get(KeyHandshakePolicy)

	if manager == nil {
		return nil
	}

	if manager, ok := manager.(HandshakePolicy); ok {
		return manager
	}

	return nil
}

func MustHandshakePolicy(node *noise.Node) HandshakePolicy {
	if !HasHandshakePolicy(node) {
		panic("noise: node must have a handshake policy enforced")
	}

	return LoadHandshakePolicy(node)
}

func EstablishSession(peer *noise.Peer) {
	once := peer.Get(KeyEstablishSessionOnce).(*sync.Once)

	once.Do(func() {
		manager := peer.Get(KeyEstablishSessionCallbacks)
		if errs := manager.(*callbacks.SequentialCallbackManager).RunCallbacks(peer.Node()); len(errs) > 0 {
			log.Error().Errs("errors", errs).Msg("Got errors running SequentialCallback callbacks.")
		}

		close(peer.Get(KeyEstablishSessionSignal).(chan struct{}))
	})
}

// OnSessionEstablished registers a callback for whenever a peer has successfully established a session.
//
// A session is considered to be established based on the protocol our node follows, and need not be
// established if the protocol does not require so.
func OnSessionEstablished(peer *noise.Peer, c func(node *noise.Node, peer *noise.Peer) error) {
	manager := peer.Get(KeyEstablishSessionCallbacks)

	manager.(*callbacks.SequentialCallbackManager).RegisterCallback(func(params ...interface{}) error {
		node, ok := params[0].(*noise.Node)
		if !ok {
			return nil
		}

		return c(node, peer)
	})
}

// OnEachSessionEstablished registers a callback for whenever a peer has successfully established a session.
//
// A session is considered to be established based on the protocol our node follows, and need not be
// established if the protocol does not require so.
func OnEachSessionEstablished(n *noise.Node, c func(node *noise.Node, peer *noise.Peer) error) {
	n.OnPeerInit(func(node *noise.Node, peer *noise.Peer) error {
		OnSessionEstablished(peer, c)
		return nil
	})
}

func BlockUntilSessionEstablished(peer *noise.Peer) {
	<-peer.Get(KeyEstablishSessionSignal).(chan struct{})
}
