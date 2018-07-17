//go:generate mockgen -destination=mocks/mock_signature_policy.go -package=mocks github.com/perlin-network/noise/crypto SignaturePolicy
//go:generate mockgen -destination=mocks/mock_hash_policy.go -package=mocks github.com/perlin-network/noise/crypto HashPolicy

package crypto

import (
	"math/big"
)

// SignaturePolicy defines the creation and validation of a cryptographic signature.
type SignaturePolicy interface {
	GenerateKeys() ([]byte, []byte, error)
	PrivateKeySize() int
	PrivateToPublic(privateKey []byte) ([]byte, error)
	PublicKeySize() int
	Sign(privateKey []byte, message []byte) []byte
	RandomKeyPair() *KeyPair
	Verify(publicKey []byte, message []byte, signature []byte) bool
}

// HashPolicy defines how to create a cryptographic hash.
type HashPolicy interface {
	HashBytes(b []byte) []byte
}

// Hash returns a hash of a big integer given a hash policy.
func Hash(hp HashPolicy, s *big.Int) *big.Int {
	return s.SetBytes(hp.HashBytes(s.Bytes()))
}
