//go:generate mockgen -destination=mocks/mock_signature_policy.go -package=mocks github.com/perlin-network/noise/crypto SignaturePolicy
//go:generate mockgen -destination=mocks/mock_hash_policy.go -package=mocks github.com/perlin-network/noise/crypto HashPolicy

package crypto

import (
	"math/big"
)

type SignaturePolicy interface {
	GenerateKeys() ([]byte, []byte, error)
	PrivateKeySize() int
	PrivateToPublic(privateKey []byte) ([]byte, error)
	PublicKeySize() int
	Sign(privateKey []byte, message []byte) []byte
	Verify(publicKey []byte, message []byte, signature []byte) bool
}

type HashPolicy interface {
	HashBytes(b []byte) []byte
}

func Hash(hp HashPolicy, s *big.Int) *big.Int {
	return s.SetBytes(hp.HashBytes(s.Bytes()))
}
