package ed25519

import (
	"crypto/rand"

	"github.com/perlin-network/noise/crypto"

	ed25519lib "golang.org/x/crypto/ed25519"
)

type Ed25519 struct {
}

var (
	_ crypto.SignaturePolicy = (*Ed25519)(nil)
)

func New() *Ed25519 {
	return &Ed25519{}
}

func (p *Ed25519) GenerateKeys() ([]byte, []byte, error) {
	publicKey, privateKey, err := ed25519lib.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, publicKey, nil
}

func (p *Ed25519) PrivateKeySize() int {
	return ed25519lib.PrivateKeySize
}

func (p *Ed25519) PrivateToPublic(privateKey []byte) ([]byte, error) {
	return ([]byte)(ed25519lib.PrivateKey(privateKey).Public().(ed25519lib.PublicKey)), nil
}

func (p *Ed25519) PublicKeySize() int {
	return ed25519lib.PublicKeySize
}

func (p *Ed25519) Sign(privateKey []byte, message []byte) []byte {
	if len(privateKey) != ed25519lib.PrivateKeySize {
		return make([]byte, 0)
	}
	return ed25519lib.Sign(ed25519lib.PrivateKey(privateKey), message)
}

func (p *Ed25519) Verify(publicKey []byte, message []byte, signature []byte) bool {
	if len(publicKey) != ed25519lib.PublicKeySize {
		return false
	}
	return ed25519lib.Verify(publicKey, message, signature)
}

func RandomKeyPair() *crypto.KeyPair {
	p := New()
	publicKey, privateKey, err := p.GenerateKeys()
	if err != nil {
		panic(err)
	}
	return &crypto.KeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}
}
