package peer

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/bits"

	"github.com/perlin-network/noise/crypto/blake2b"
	"github.com/perlin-network/noise/internal/protobuf"
)

const (
	keyNonce = "nonce"
)

// ID is an identity of nodes, using its public key hash and network address.
type ID protobuf.ID

// CreateID is a factory function creating ID.
func CreateID(address string, publicKey []byte, opts ...IDOption) ID {
	id := ID{Address: address, PublicKey: publicKey, Id: blake2b.New().HashBytes(publicKey)}

	for _, opt := range opts {
		id = opt(&id)
	}

	return id
}

// String returns the identity address and public key.
func (id ID) String() string {
	return fmt.Sprintf("ID{Address: %v, Id: %v}", id.Address, id.Id)
}

// Equals determines if two peer IDs are equal to each other based on the node IDs.
func (id ID) Equals(other ID) bool {
	return bytes.Equal(id.Id, other.GetID())
}

// Less determines if this peer ID's node ID is less than other ID's node ID.
func (id ID) Less(other interface{}) bool {
	if other, is := other.(ID); is {
		return bytes.Compare(id.Id, other.GetID()) == -1
	}
	return false
}

// GetAddress returns the ID's address.
func (id ID) GetAddress() string {
	return id.Address
}

// GetID returns the ID's node ID.
func (id ID) GetID() []byte {
	return id.Id
}

// GetPublicKey returns the ID's public key.
func (id ID) GetPublicKey() []byte {
	return id.PublicKey
}

// PublicKeyHex generates a hex-encoded string of public key hash of this given peer ID.
func (id ID) PublicKeyHex() string {
	return hex.EncodeToString(id.PublicKey)
}

// Value returns the metadata value associated with the key if it exists.
func (id ID) Value(key string) interface{} {
	if val, ok := id.Metadata[key]; ok {
		return val
	}
	return nil
}

// withValue returns a copy of parent in which the value associated with key is val.
func (id ID) WithValue(key string, val []byte) ID {
	if &key == nil || key == "" {
		panic("nil key")
	}
	copy := id
	if copy.Metadata == nil {
		copy.Metadata = make(map[string][]byte, 0)
	}
	copy.Metadata[key] = val
	return copy
}

// Xor performs XOR (^) over another peer ID's public key.
func (id ID) Xor(other ID) ID {
	result := Xor(id.PublicKey, other.GetPublicKey())

	return ID{Address: id.Address, PublicKey: result}
}

// XorID performs XOR (^) over another peer ID's public key hash.
func (id ID) XorID(other ID) ID {
	result := Xor(id.Id, other.GetID())

	return ID{Address: id.Address, Id: result}
}

// PrefixLen returns the number of prefixed zeros in a peer ID.
func (id ID) PrefixLen() int {
	return PrefixLen(id.Id)
}

// PrefixLen returns the number of prefixed zeroes in a byte slice.
func PrefixLen(bytes []byte) int {
	for i, b := range bytes {
		if b != 0 {
			return i*8 + bits.LeadingZeros8(uint8(b))
		}
	}
	return len(bytes)*8 - 1
}

// Xor performs an xor operation on two byte slices.
func Xor(a, b []byte) []byte {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}

	dst := make([]byte, n)
	for i := 0; i < n; i++ {
		dst[i] = a[i] ^ b[i]
	}
	return dst
}

// IDOption are configurable options when creating the identity.
type IDOption func(*ID) ID

// WithNonce sets the ID's nonce metadata.
func WithNonce(nonce []byte) IDOption {
	return func(id *ID) ID {
		return id.WithValue(keyNonce, nonce)
	}
}

// GetNonce returns the ID's nonce if it exists.
func GetNonce(id ID) []byte {
	nonce, ok := id.Value(keyNonce).([]byte)
	if !ok {
		return nil
	}
	return nonce
}
