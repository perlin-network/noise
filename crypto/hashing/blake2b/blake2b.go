package blake2b

import (
	blake2blib "golang.org/x/crypto/blake2b"
)

type Blake2b struct{}

func New() *Blake2b {
	return &Blake2b{}
}

func (p *Blake2b) HashBytes(bytes []byte) []byte {
	result := blake2blib.Sum256(bytes)
	return result[:]
}
