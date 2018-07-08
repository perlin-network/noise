package none

import (
	"crypto/rand"
	"github.com/perlin-network/noise/crypto"
)

func RandomKeyPair() *crypto.KeyPair {
	kp := &crypto.KeyPair {
		PrivateKey: []byte{},
		PublicKey: make([]byte, 32),
	}

	_, err := rand.Read(kp.PublicKey)
	if err != nil {
		panic(err)
	}

	return kp
}

type None struct{}

func (p *None) HashBytes(b []byte) []byte {
	return nil
}

func (p *None) PrivateKeySize() int {
	return 0
}

func (p *None) PublicKeySize() int {
	return 32
}

func (p *None) PrivateToPublic(privateKey []byte) ([]byte, error) {
	panic("PrivateToPublic not supported on None")
}

func (p *None) Sign(privateKey []byte, message []byte) []byte {
	return []byte{42} // An empty byte slice becomes nil when unmarshaled by the receiver
}

func (p *None) Verify(publicKey []byte, message []byte, signature []byte) bool {
	return true
}
