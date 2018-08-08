package peer

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/bits"

	"github.com/perlin-network/noise/crypto/blake2b"
	"github.com/perlin-network/noise/internal/protobuf"
)

// ID is an identity of nodes, using its public key hash and network address.
type ID protobuf.ID

// CreateID is a factory function creating ID.
func CreateID(address string, publicKey []byte) ID {
	return ID{Address: address, PublicKey: publicKey, PublicKeyHash: blake2b.New().HashBytes(publicKey)}
}

// String returns the identity address and public key.
func (id ID) String() string {
	return fmt.Sprintf("ID{Address: %v, PublicKeyHash: %v}", id.Address, id.PublicKeyHash)
}

// Equals determines if two peer IDs are equal to each other based on the contents of their public keys.
func (id ID) Equals(other ID) bool {
	return bytes.Equal(id.PublicKeyHash, other.PublicKeyHash)
}

// Less determines if this peer ID's public key is less than other ID's public key.
func (id ID) Less(other interface{}) bool {
	if other, is := other.(ID); is {
		return bytes.Compare(id.PublicKeyHash, other.PublicKeyHash) == -1
	}
	return false
}

// PublicKeyHex generates a hex-encoded string of public key hash of this given peer ID.
func (id ID) PublicKeyHex() string {
	return hex.EncodeToString(id.PublicKeyHash)
}

// Xor performs XOR (^) over another peer ID's public key hash.
func (id ID) Xor(other ID) ID {
	result := make([]byte, len(id.PublicKeyHash))

	for i := 0; i < len(id.PublicKeyHash) && i < len(other.PublicKeyHash); i++ {
		result[i] = id.PublicKeyHash[i] ^ other.PublicKeyHash[i]
	}
	return ID{Address: id.Address, PublicKeyHash: result}
}

// PrefixLen returns the number of prefixed zeros in a peer ID.
func (id ID) PrefixLen() int {
	for i, b := range id.PublicKeyHash {
		if b != 0 {
			return i*8 + bits.LeadingZeros8(uint8(b))
		}
	}
	return len(id.PublicKeyHash)*8 - 1
}
