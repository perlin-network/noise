package protocol

import (
	"github.com/perlin-network/noise"
)

const (
	KeyCipherPolicy = "cipherPolicy"
)

type CipherPolicy interface {
	EnforceCipherPolicy(node *noise.Node)
	Encrypt(peer *noise.Peer, buf []byte) ([]byte, error)
	Decrypt(peer *noise.Peer, buf []byte) ([]byte, error)
}

func EnforceCipherPolicy(node *noise.Node, policy CipherPolicy) CipherPolicy {
	node.Set(KeyCipherPolicy, policy)
	policy.EnforceCipherPolicy(node)

	OnEachSessionEstablished(node, func(node *noise.Node, peer *noise.Peer) error {
		peer.BeforeMessageSent(func(node *noise.Node, peer *noise.Peer, buf []byte) (bytes []byte, e error) {
			buf, e = policy.Encrypt(peer, buf)

			// If errors occur encrypting a message for a peer, disconnect the peer.
			if e != nil {
				peer.Disconnect()
			}

			return buf, e
		})

		peer.BeforeMessageReceived(func(node *noise.Node, peer *noise.Peer, buf []byte) (bytes []byte, e error) {
			buf, e = policy.Decrypt(peer, buf)

			// If errors occur decrypting a message from a peer, disconnect the peer.
			if e != nil {
				peer.Disconnect()
			}

			return buf, e
		})

		return nil
	})

	return policy
}

func HasCipherPolicy(node *noise.Node) bool {
	return node.Has(KeyCipherPolicy)
}

func LoadCipherPolicy(node *noise.Node) CipherPolicy {
	manager := node.Get(KeyCipherPolicy)

	if manager == nil {
		return nil
	}

	if manager, ok := manager.(CipherPolicy); ok {
		return manager
	}

	return nil
}

func MustCipherPolicy(node *noise.Node) CipherPolicy {
	manager := LoadCipherPolicy(node)

	if manager == nil {
		panic("noise: node must have a cipher policy enforced")
	}

	return manager
}
