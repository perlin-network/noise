package crypto

import (
	"math/big"
)

func Hash(hp HashPolicy, s *big.Int) *big.Int {
	return s.SetBytes(hp.HashBytes(s.Bytes()))
}
