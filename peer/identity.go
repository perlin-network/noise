package peer

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/bits"

	"github.com/perlin-network/noise/protobuf"
)

// ID is an identity of nodes, using its public key and network address.
type ID protobuf.ID

// CreateID is a factory function creating ID.
func CreateID(address string, publicKey []byte) ID {
	return ID{Address: address, PublicKey: publicKey}
}

// String returns the identity address and public key.
func (id ID) String() string {
	return fmt.Sprintf("ID{Address: %v, PublicKey: %v}", id.Address, id.PublicKey)
}

// Equals determines if two peer IDs are equal to each other based on the contents of their public keys.
func (id ID) Equals(other ID) bool {
	return bytes.Equal(id.PublicKey, other.PublicKey)
}

// Less determines if this peer ID's public key is less than other ID's public key.
func (id ID) Less(other interface{}) bool {
	if other, is := other.(ID); is {
		return bytes.Compare(id.PublicKey, other.PublicKey) == -1
	}
	return false
}

// PublicKeyHex generates a hex-encoded string of public key of this given peer ID.
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

// PrefixLen returns the number of prefixed zeros in a peer ID.
func (id ID) PrefixLen() int {
	for i, b := range id.PublicKey {
		if b != 0 {
			return i*8 + bits.LeadingZeros8(uint8(b))
		}
	}
	return len(id.PublicKey) * 8
}
