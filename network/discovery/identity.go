package discovery

import (
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/crypto/blake2b"
	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/peer"
)

const (
	// number of preceding bits of 0 in the H(H(key_public)) for NodeID generation
	c1 = 16
	// number of preceding bits of 0 in the H(NodeID xor X) for checking if dynamic cryptopuzzle is solved
	c2 = 16
)

// GenerateKeyPairAndID generates an S/Kademlia keypair and node ID
func GenerateKeyPairAndID(address string) (*crypto.KeyPair, peer.ID) {
	for {
		kp := ed25519.RandomKeyPair()
		if isValidKeyPair(kp.PublicKey, c1) {
			id := peer.CreateID(address, kp.PublicKey)
			return kp, id
		}
	}
}

// isValidKeyPair checks if the S/Kademlia static cryptopuzzle generates a valid node ID
func isValidKeyPair(publicKey []byte, c1 int) bool {
	b := blake2b.New()
	nodeID := b.HashBytes(publicKey)
	P := b.HashBytes(nodeID)
	return peer.PrefixLen(P) >= c1
}
