package crypto

import (
	"math/big"

	"golang.org/x/crypto/blake2b"
)

func Hash(s *big.Int) *big.Int {
	return s.SetBytes(HashBytes(s.Bytes()))
}

func HashBytes(bytes []byte) []byte {
	result := blake2b.Sum256(bytes)
	return result[:]
}
