package ed25519

import (
	"crypto/rand"

	"github.com/perlin-network/noise/crypto"

	ed25519lib "golang.org/x/crypto/ed25519"
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
	publicKey, privateKey, err := ed25519lib.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, publicKey, nil
}

// PrivateKeySize returns the private key length.
func (p *Ed25519) PrivateKeySize() int {
	return ed25519lib.PrivateKeySize
}

// PrivateToPublic returns the public key given the private key.
func (p *Ed25519) PrivateToPublic(privateKey []byte) ([]byte, error) {
	return ([]byte)(ed25519lib.PrivateKey(privateKey).Public().(ed25519lib.PublicKey)), nil
}

// PublicKeySize returns the public key length.
func (p *Ed25519) PublicKeySize() int {
	return ed25519lib.PublicKeySize
}

// Sign returns an ed25519-signed message given an private key and message.
func (p *Ed25519) Sign(privateKey []byte, message []byte) []byte {
	if len(privateKey) != ed25519lib.PrivateKeySize {
		return make([]byte, 0)
	}
	return ed25519lib.Sign(ed25519lib.PrivateKey(privateKey), message)
}

// Verify returns true if the  signature was signed using the given public key and message.
func (p *Ed25519) Verify(publicKey []byte, message []byte, signature []byte) bool {
	if len(publicKey) != ed25519lib.PublicKeySize {
		return false
	}
	return ed25519lib.Verify(publicKey, message, signature)
}

// RandomKeyPair generates a randomly seeded ed25519 key pair.
func RandomKeyPair() *crypto.KeyPair {
	publicKey, privateKey, err := ed25519lib.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	return &crypto.KeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}
}
