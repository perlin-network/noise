package ed25519

import (
	"crypto/rand"

	"github.com/perlin-network/noise/crypto"
)

// Ed25519 represents the ed25519 cryptographic signature scheme.
type Ed25519 struct {
}

var (
	_ crypto.SignaturePolicy = (*Ed25519)(nil)
)

// New returns an Ed25519 structure.
func New() *Ed25519 {
	return &Ed25519{}
}

// GenerateKeys generates a private and public key using the ed25519 signature scheme.
func (p *Ed25519) GenerateKeys() ([]byte, []byte, error) {
	publicKey, privateKey, err := GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, publicKey, nil
}

// PrivateKeySize returns the private key length.
func (p *Ed25519) PrivateKeySize() int {
	return PrivateKeySize
}

// PrivateToPublic returns the public key given the private key.
func (p *Ed25519) PrivateToPublic(privateKey []byte) ([]byte, error) {
	return []byte(PrivateKey(privateKey).Public().(PublicKey)), nil
}

// PublicKeySize returns the public key length.
func (p *Ed25519) PublicKeySize() int {
	return PublicKeySize
}

// RandomKeyPair generates a randomly seeded ed25519 key pair.
func (p *Ed25519) RandomKeyPair() *crypto.KeyPair {
	return RandomKeyPair()
}

// Sign returns an ed25519-signed message given an private key and message.
func (p *Ed25519) Sign(privateKey []byte, message []byte) []byte {
	if len(privateKey) != PrivateKeySize {
		return make([]byte, 0)
	}
	return Sign(PrivateKey(privateKey), message)
}

// Verify returns true if the  signature was signed using the given public key and message.
func (p *Ed25519) Verify(publicKey []byte, message []byte, signature []byte) bool {
	if len(publicKey) != PublicKeySize {
		return false
	}
	return Verify(publicKey, message, signature)
}

// RandomKeyPair generates a randomly seeded ed25519 key pair.
func RandomKeyPair() *crypto.KeyPair {
	publicKey, privateKey, err := GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	return &crypto.KeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}
}
