package peer

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/bits"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/crypto/blake2b"
	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/internal/protobuf"
)

const (
	// number of preceding bits of 0 in the H(H(key_public)) for NodeID generation
	c1 = 16
	// number of preceding bits of 0 in the H(NodeID xor X) for checking if dynamic cryptopuzzle is solved
	c2 = 16
)

// ID is an identity of nodes, using its public key hash and network address.
type ID protobuf.ID

// CreateID is a factory function creating ID.
func CreateID(address string, publicKey []byte) ID {
	return ID{Address: address, PublicKey: publicKey, Id: blake2b.New().HashBytes(publicKey)}
}

// String returns the identity address and public key.
func (id ID) String() string {
	return fmt.Sprintf("ID{Address: %v, Id: %v}", id.Address, id.Id)
}

// Equals determines if two peer IDs are equal to each other based on the contents of their public keys.
func (id ID) Equals(other ID) bool {
	return bytes.Equal(id.Id, other.Id)
}

// Less determines if this peer ID's public key is less than other ID's public key.
func (id ID) Less(other interface{}) bool {
	if other, is := other.(ID); is {
		return bytes.Compare(id.Id, other.Id) == -1
	}
	return false
}

// PublicKeyHex generates a hex-encoded string of public key hash of this given peer ID.
func (id ID) PublicKeyHex() string {
	return hex.EncodeToString(id.PublicKey)
}

// Xor performs XOR (^) over another peer ID's public key.
func (id ID) Xor(other ID) ID {
	result := make([]byte, len(id.PublicKey))

	for i := 0; i < len(id.PublicKey) && i < len(other.PublicKey); i++ {
		result[i] = id.PublicKey[i] ^ other.PublicKey[i]
	}
	return ID{Address: id.Address, PublicKey: result}
}

// XorID performs XOR (^) over another peer ID's public key hash.
func (id ID) XorID(other ID) ID {
	result := make([]byte, len(id.Id))

	for i := 0; i < len(id.Id) && i < len(other.Id); i++ {
		result[i] = id.Id[i] ^ other.Id[i]
	}
	return ID{Address: id.Address, Id: result}
}

// PrefixLen returns the number of prefixed zeros in a peer ID.
func (id ID) PrefixLen() int {
	return prefixLen(id.Id)
}

// prefixLen returns the number of prefixed zeroes in a byte slice
func prefixLen(bytes []byte) int {
	for i, b := range bytes {
		if b != 0 {
			return i*8 + bits.LeadingZeros8(uint8(b))
		}
	}
	return len(bytes)*8 - 1
}

// GenerateKeyPairAndID generates an S/Kademlia keypair and node ID
func GenerateKeyPairAndID(address string) (*crypto.KeyPair, ID) {
	for {
		kp := ed25519.RandomKeyPair()
		if isValidKeyPair(kp.PublicKey, c1) {
			id := CreateID(address, kp.PublicKey)
			return kp, id
		}
	}
}

// isValidKeyPair checks if the S/Kademlia static cryptopuzzle generates a valid node ID
func isValidKeyPair(publicKey []byte, c1 int) bool {
	b := blake2b.New()
	nodeID := b.HashBytes(publicKey)
	P := b.HashBytes(nodeID)
	return prefixLen(P) >= c1
}
