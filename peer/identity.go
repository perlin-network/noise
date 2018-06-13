package peer

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/perlin-network/noise/protobuf"
	"golang.org/x/crypto/ed25519"
)

const IdSize = ed25519.PublicKeySize

type ID protobuf.ID

func CreateID(address string, publicKey []byte) ID {
	return ID{PublicKey: publicKey[:IdSize], Address: address}
}

func (id ID) String() string {
	return fmt.Sprintf("ID{PublicKey: %v, Address: %v}", id.PublicKey, id.Address)
}

func (id ID) Equals(other ID) bool {
	return bytes.Equal(id.PublicKey[:IdSize], other.PublicKey[:IdSize])
}

func (id ID) Less(other interface{}) bool {
	if other, is := other.(ID); is {
		return bytes.Compare(id.PublicKey[:IdSize], other.PublicKey[:IdSize]) == -1
	}
	return false
}

func (id ID) Hex() string {
	return hex.EncodeToString(id.PublicKey[:])
}

func (id ID) Xor(other ID) ID {
	var result [IdSize]byte
	for i := 0; i < IdSize; i++ {
		result[i] = id.PublicKey[i] ^ other.PublicKey[i]
	}
	return ID{Address: id.Address, PublicKey: result[:]}
}

// Returns the number of prefixed zeros in a peer ID.
func (id ID) PrefixLen() int {
	for x := 0; x < IdSize; x++ {
		for y := 0; y < 8; y++ {
			if (id.PublicKey[x]>>uint8(7-y))&0x1 != 0 {
				return x*8 + y
			}
		}
	}
	return IdSize*8 - 1
}

func (id ID) PublicKeyHex() string {
	return hex.EncodeToString(id.PublicKey)
}
