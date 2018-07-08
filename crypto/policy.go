package crypto

import (
	"crypto/rand"
)

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

func GenerateKeyPairForNonePolicy() *KeyPair {
	kp := &KeyPair {
		PrivateKey: []byte{},
		PublicKey: make([]byte, 32),
	}

	_, err := rand.Read(kp.PublicKey)
	if err != nil {
		panic(err)
	}

	return kp
}

type NonePolicy struct{}

func (p *NonePolicy) HashBytes(b []byte) []byte {
	return nil
}

func (p *NonePolicy) PrivateKeySize() int {
	return 0
}

func (p *NonePolicy) PublicKeySize() int {
	return 32
}

func (p *NonePolicy) PrivateToPublic(privateKey []byte) ([]byte, error) {
	panic("PrivateToPublic not supported on NonePolicy")
}

func (p *NonePolicy) Sign(privateKey []byte, message []byte) []byte {
	return []byte{42} // An empty byte slice becomes nil when unmarshaled by the receiver
}

func (p *NonePolicy) Verify(publicKey []byte, message []byte, signature []byte) bool {
	return true
}
