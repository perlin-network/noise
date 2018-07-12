package hashing

import (
	blake2blib "golang.org/x/crypto/blake2b"
)

type Blake2b struct{}

var (
	_ HashPolicy = (*Blake2b)(nil)
)

func NewBlake2b() *Blake2b {
	return &Blake2b{}
}

func (p *Blake2b) HashBytes(bytes []byte) []byte {
	result := blake2blib.Sum256(bytes)
	return result[:]
}
