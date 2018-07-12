//go:generate mockgen -destination=../mocks/mock_hash_policy.go -package=mocks github.com/perlin-network/noise/crypto/hashing HashPolicy

package hashing

import (
	"math/big"
)

type HashPolicy interface {
	HashBytes(b []byte) []byte
}

func Hash(hp HashPolicy, s *big.Int) *big.Int {
	return s.SetBytes(hp.HashBytes(s.Bytes()))
}
