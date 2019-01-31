package protocol

import (
	"github.com/perlin-network/noise"
)

const (
	KeyNetworkPolicy = "networkPolicy"
)

type NetworkPolicy interface {
	EnforceNetworkPolicy(node *noise.Node)

	OnSessionEstablished(node *noise.Node, peer *noise.Peer) error
}

func EnforceNetworkPolicy(node *noise.Node, policy NetworkPolicy) NetworkPolicy {
	MustIdentityPolicy(node)

	node.Set(KeyNetworkPolicy, policy)
	policy.EnforceNetworkPolicy(node)

	// If a handshake policy exists, we register our peer to the overlay network
	// when an authenticated session has been established.
	if HasHandshakePolicy(node) {
		OnEachSessionEstablished(node, policy.OnSessionEstablished)
	} else {
		node.OnPeerInit(policy.OnSessionEstablished)
	}

	return policy
}

func HasNetworkPolicy(node *noise.Node) bool {
	return node.Has(KeyNetworkPolicy)
}

func LoadNetworkPolicy(node *noise.Node) NetworkPolicy {
	manager := node.Get(KeyNetworkPolicy)

	if manager == nil {
		return nil
	}

	if manager, ok := manager.(NetworkPolicy); ok {
		return manager
	}

	return nil
}

func MustNetworkPolicy(node *noise.Node) NetworkPolicy {
	manager := LoadNetworkPolicy(node)

	if manager == nil {
		panic("noise: node must have a network policy enforced")
	}

	return manager
}
