package crypto

import (
	"crypto/rand"
	"golang.org/x/crypto/ed25519"
)

type Provider interface {
	GenerateKeyPair() (KeyPair, error)
	PrivateKeySize() int
	PublicKeySize() int
	PrivateToPublic(privateKey []byte) ([]byte, error)
	Sign(privateKey []byte, message []byte) []byte
	Verify(publicKey []byte, message []byte, signature []byte) bool
}

type Ed25519 struct {}

func NewEd25519() *Ed25519 {
	return &Ed25519{}
}

func (p *Ed25519) GenerateKeyPair() (KeyPair, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return KeyPair{}, err
	}
	return KeyPair {
		PublicKey: publicKey,
		PrivateKey: privateKey,
	}, nil
}

func (p *Ed25519) PrivateKeySize() int {
	return ed25519.PrivateKeySize
}

func (p *Ed25519) PublicKeySize() int {
	return ed25519.PublicKeySize
}

func (p *Ed25519) PrivateToPublic(privateKey []byte) ([]byte, error) {
	return ([]byte)(ed25519.PrivateKey(privateKey).Public().(ed25519.PublicKey)), nil
}

func (p *Ed25519) Sign(privateKey []byte, message []byte) []byte {
	return ed25519.Sign(ed25519.PrivateKey(privateKey), message)
}

func (p *Ed25519) Verify(publicKey []byte, message []byte, signature []byte) bool {
	return ed25519.Verify(publicKey, message, signature)
}

