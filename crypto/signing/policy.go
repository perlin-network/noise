//go:generate mockgen -destination=../mocks/mock_signature_policy.go -package=mocks github.com/perlin-network/noise/crypto/signing SignaturePolicy

package signing

type SignaturePolicy interface {
	GenerateKeys() ([]byte, []byte, error)
	PrivateKeySize() int
	PrivateToPublic(privateKey []byte) ([]byte, error)
	PublicKeySize() int
	Sign(privateKey []byte, message []byte) []byte
	Verify(publicKey []byte, message []byte, signature []byte) bool
}
