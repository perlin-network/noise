package crypto

type HashPolicy interface {
	HashBytes(b []byte) []byte
}

type SignaturePolicy interface {
	PrivateKeySize() int
	PublicKeySize() int
	PrivateToPublic(privateKey []byte) ([]byte, error)
	Sign(privateKey []byte, message []byte) []byte
	Verify(publicKey []byte, message []byte, signature []byte) bool
}
