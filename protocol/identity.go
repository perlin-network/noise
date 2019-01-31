package protocol

import (
	"fmt"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/callbacks"
	"github.com/pkg/errors"
	"sync"
)

const (
	KeyAuthCallbacks = "auth.callbacks"
	KeyAuthSignal    = "auth.signal"
	KeyAuthOnce      = "auth.once"

	KeyIdentityPolicy = "identityPolicy"
	KeySharedKey      = "sharedKey"
	KeyPeerID         = "peer."
	KeyID             = "id"
)

type ID interface {
	fmt.Stringer
	noise.Message

	Equals(other ID) bool

	PublicID() []byte
	Hash() []byte
}

type IdentityPolicy interface {
	EnforceIdentityPolicy(node *noise.Node)

	OnSessionEstablished(node *noise.Node, peer *noise.Peer) error
}

func EnforceIdentityPolicy(node *noise.Node, policy IdentityPolicy) IdentityPolicy {
	node.Set(KeyIdentityPolicy, policy)

	node.OnPeerInit(func(node *noise.Node, peer *noise.Peer) error {
		peer.Set(KeyAuthCallbacks, defaultAuthCallbackManager())
		peer.Set(KeyAuthSignal, make(chan struct{}))
		peer.Set(KeyAuthOnce, new(sync.Once))

		return nil
	})

	policy.EnforceIdentityPolicy(node)

	// If a handshake policy exists, we register our peer to the overlay network
	// when an authenticated session has been established.
	if HasHandshakePolicy(node) {
		OnEachSessionEstablished(node, policy.OnSessionEstablished)
	} else {
		node.OnPeerInit(policy.OnSessionEstablished)
	}

	return policy
}

func HasIdentityPolicy(node *noise.Node) bool {
	return node.Has(KeyIdentityPolicy)
}

func LoadIdentityPolicy(node *noise.Node) IdentityPolicy {
	manager := node.Get(KeyIdentityPolicy)

	if manager == nil {
		return nil
	}

	if manager, ok := manager.(IdentityPolicy); ok {
		return manager
	}

	return nil
}

func MustIdentityPolicy(node *noise.Node) IdentityPolicy {
	if !HasIdentityPolicy(node) {
		panic("noise: node must have an identity policy enforced")
	}

	return LoadIdentityPolicy(node)
}

func HasSharedKey(peer *noise.Peer) bool {
	return peer.Has(KeySharedKey)
}

func LoadSharedKey(peer *noise.Peer) []byte {
	sharedKey := peer.Get(KeySharedKey)

	if sharedKey == nil {
		return nil
	}

	if sharedKey, ok := sharedKey.([]byte); ok {
		return sharedKey
	}

	return nil
}

func MustSharedKey(peer *noise.Peer) []byte {
	if !HasSharedKey(peer) {
		panic("noise: shared key must be established via protocol for peer")
	}

	return LoadSharedKey(peer)
}

func SetSharedKey(peer *noise.Peer, sharedKey []byte) {
	peer.Set(KeySharedKey, sharedKey)
}

func DeleteSharedKey(peer *noise.Peer) {
	peer.Delete(KeySharedKey)
}

func SetNodeID(node *noise.Node, id ID) {
	node.Set(KeyID, id)
}

func DeleteNodeID(node *noise.Node) {
	node.Delete(KeyID)
}

func SetPeerID(peer *noise.Peer, id ID) {
	peer.Node().Set(KeyPeerID+string(id.Hash()), peer)
	peer.Set(KeyID, id)
}

func DeletePeerID(peer *noise.Peer) {
	peer.Node().Delete(KeyPeerID + string(PeerID(peer).Hash()))
	peer.Delete(KeyID)
}

func NodeID(node *noise.Node) ID {
	t := node.Get(KeyID)

	if t == nil {
		return nil
	}

	if t, ok := t.(ID); ok {
		return t
	}

	return nil
}

func PeerID(peer *noise.Peer) ID {
	t := peer.Get(KeyID)

	if t == nil {
		return nil
	}

	if t, ok := t.(ID); ok {
		return t
	}

	return nil
}

func Peer(node *noise.Node, id ID) *noise.Peer {
	t := node.Get(KeyPeerID + string(id.Hash()))

	if t == nil {
		return nil
	}

	if t, ok := t.(*noise.Peer); ok {
		return t
	}

	return nil
}

func AuthenticatePeer(peer *noise.Peer, id ID) {
	once := peer.Get(KeyAuthOnce).(*sync.Once)

	once.Do(func() {
		SetPeerID(peer, id)

		manager := peer.Get(KeyAuthCallbacks)
		manager.(*callbacks.SequentialCallbackManager).RunCallbacks(peer.Node(), peer, id)

		close(peer.Get(KeyAuthSignal).(chan struct{}))
	})
}

func OnPeerAuthenticated(peer *noise.Peer, c func(node *noise.Node, peer *noise.Peer, id ID) error) {
	manager := peer.Get(KeyAuthCallbacks)

	manager.(*callbacks.SequentialCallbackManager).RegisterCallback(func(params ...interface{}) error {
		if len(params) != 3 {
			panic(errors.Errorf("protocol: OnPeerAuthenticated received unexpected args %v", params))
		}

		node, ok := params[0].(*noise.Node)
		if !ok {
			return nil
		}

		peer, ok := params[1].(*noise.Peer)
		if !ok {
			return nil
		}

		id, ok := params[2].(ID)
		if !ok {
			return nil
		}

		return c(node, peer, id)
	})
}

func OnEachPeerAuthenticated(n *noise.Node, c func(node *noise.Node, peer *noise.Peer, id ID) error) {
	n.OnPeerInit(func(node *noise.Node, peer *noise.Peer) error {
		OnPeerAuthenticated(peer, c)
		return nil
	})
}

func defaultAuthCallbackManager() *callbacks.SequentialCallbackManager {
	manager := callbacks.NewSequentialCallbackManager()

	// Deregister peer ID when a peer disconnects.
	manager.RegisterCallback(func(params ...interface{}) error {
		if len(params) != 3 {
			panic(errors.Errorf("protocol: OnPeerAuthenticated received unexpected args %v", params))
		}

		peer, ok := params[1].(*noise.Peer)
		if !ok {
			return nil
		}

		peer.OnDisconnect(func(node *noise.Node, peer *noise.Peer) error {
			DeletePeerID(peer)
			return callbacks.DeregisterCallback
		})

		return nil
	})

	return manager
}

func BlockUntilAuthenticated(peer *noise.Peer) {
	<-peer.Get(KeyAuthSignal).(chan struct{})
}
