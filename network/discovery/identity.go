package discovery

import (
	"crypto/rand"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/crypto/blake2b"
	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/peer"

	"github.com/pkg/errors"
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
		if checkHashedBytesPrefixLen(kp.PublicKey, c1) {
			id := peer.CreateID(address, kp.PublicKey)

			x := generateDynamicPuzzleX(id.Id, c2)
			id.X = x

			return kp, id
		}
	}
}

// checkHashedBytesPrefixLen checks if the hashed bytes has prefix length of c
func checkHashedBytesPrefixLen(a []byte, c int) bool {
	b := blake2b.New()
	nodeID := b.HashBytes(a)
	P := b.HashBytes(nodeID)
	return peer.PrefixLen(P) >= c
}

// randomBytes generates a random byte slice with specified length
func randomBytes(len int) ([]byte, error) {
	randBytes := make([]byte, len)
	n, err := rand.Read(randBytes)
	if err != nil {
		return nil, err
	}
	if n != len {
		return nil, errors.Errorf("failed to generate %d bytes", len)
	}
	return randBytes, nil
}

// generateDynamicPuzzleX returns random bytes X which satisfies that the hash of the nodeID xored with X
// has at least a prefix length of c
func generateDynamicPuzzleX(nodeID []byte, c int) []byte {
	len := len(nodeID)
	for {
		x, err := randomBytes(len)
		if err != nil {
			continue
		}
		if checkDynamicPuzzle(nodeID, x, c) {
			return x
		}
	}
}

// checkDynamicPuzzle checks whether the nodeID and bytes x solves the S/Kademlia dynamic puzzle for c prefix length
func checkDynamicPuzzle(nodeID, x []byte, c int) bool {
	xored := peer.Xor(nodeID, x)
	return checkHashedBytesPrefixLen(xored, c)
}
