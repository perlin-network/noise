package peer

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/perlin-network/noise/protobuf"
	"golang.org/x/crypto/ed25519"
)

const IDSize = ed25519.PublicKeySize

type ID protobuf.ID

func CreateID(address string, publicKey []byte) ID {
	return ID{PublicKey: publicKey[:IDSize], Address: address}
}

func (id ID) String() string {
	return fmt.Sprintf("ID{PublicKey: %v, Address: %v}", id.PublicKey, id.Address)
}

// Determines if two peer IDs are equal to each other based on the contents of their public keys.
func (id ID) Equals(other ID) bool {
	return bytes.Equal(id.PublicKey[:IDSize], other.PublicKey[:IDSize])
}

// Interprets public keys of two peer IDs as big integers, and determines which one is smaller than the other.
func (id ID) Less(other interface{}) bool {
	if other, is := other.(ID); is {
		return bytes.Compare(id.PublicKey[:IDSize], other.PublicKey[:IDSize]) == -1
	}
	return false
}

// Generates hex-encoded string of public key of this given peer ID.
func (id ID) PublicKeyHex() string {
	return hex.EncodeToString(id.PublicKey)
}

// Performs XOR (^) over another peer ID's public key.
func (id ID) Xor(other ID) ID {
	var result [IDSize]byte
	for i := 0; i < IDSize; i++ {
		result[i] = id.PublicKey[i] ^ other.PublicKey[i]
	}
	return ID{Address: id.Address, PublicKey: result[:]}
}

// Returns the number of prefixed zeros in a peer ID.
func (id ID) PrefixLen() int {
	for x := 0; x < IDSize; x++ {
		for y := 0; y < 8; y++ {
			if (id.PublicKey[x]>>uint8(7-y))&0x1 != 0 {
				return x*8 + y
			}
		}
	}
	return IDSize*8 - 1
}
