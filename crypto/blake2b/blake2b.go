package blake2b

import (
	"github.com/perlin-network/noise/crypto"

	blake2blib "golang.org/x/crypto/blake2b"
)

type Blake2b struct{}

var (
	_ crypto.HashPolicy = (*Blake2b)(nil)
)

func New() *Blake2b {
	return &Blake2b{}
}

func (p *Blake2b) HashBytes(bytes []byte) []byte {
	result := blake2blib.Sum256(bytes)
	return result[:]
}
