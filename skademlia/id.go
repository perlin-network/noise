package skademlia

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/payload"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"math/bits"
)

var (
	_ protocol.ID = (*ID)(nil)
)

type ID struct {
	address   string
	publicKey []byte

	hash []byte
}

func (a ID) Equals(other protocol.ID) bool {
	if other, ok := other.(ID); ok {
		return bytes.Equal(a.hash, other.hash)
	}

	return false
}

func (a ID) PublicID() []byte {
	return a.publicKey
}

func (a ID) Hash() []byte {
	return a.hash
}

func NewID(address string, publicKey []byte) ID {
	hash := blake2b.Sum256(publicKey)
	return ID{
		address:   address,
		publicKey: publicKey,
		hash:      hash[:],
	}
}

func (a ID) String() string {
	return fmt.Sprintf("%s(%s)(%s)", a.address, hex.EncodeToString(a.publicKey)[:16], hex.EncodeToString(a.hash)[:16])
}

func (a ID) Read(reader payload.Reader) (msg noise.Message, err error) {
	a.address, err = reader.ReadString()
	if err != nil {
		return nil, errors.Wrap(err, "kademlia: failed to deserialize ID address")
	}

	a.publicKey, err = reader.ReadBytes()
	if err != nil {
		return nil, errors.Wrap(err, "kademlia: failed to deserialize ID public key")
	}

	hash := blake2b.Sum256(a.publicKey)
	a.hash = hash[:]

	return a, nil
}

func (a ID) Write() []byte {
	return payload.NewWriter(nil).
		WriteString(a.address).
		WriteBytes(a.publicKey).Bytes()
}

func prefixLen(buf []byte) int {
	for i, b := range buf {
		if b != 0 {
			return i*8 + bits.LeadingZeros8(uint8(b))
		}
	}

	return len(buf)*8 - 1
}

func xor(a, b []byte) []byte {
	if len(a) != len(b) {
		panic("kademlia: len(a) and len(b) must be equal for xor(a, b)")
	}

	c := make([]byte, len(a))

	for i := 0; i < len(a); i++ {
		c[i] = a[i] ^ b[i]
	}

	return c
}

func prefixDiff(a, b []byte, n int) int {
	bytes, total := xor(a, b), 0

	for i, b := range bytes {
		if n <= 8*i {
			break
		} else if n > 8*i && n < 8*(i+1) {
			shift := 8 - uint(n%8)
			b = b >> shift
		}
		total += bits.OnesCount8(uint8(b))
	}
	return total
}
