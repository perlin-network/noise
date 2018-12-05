package base

import (
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/crypto/blake2b"
	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/protocol"
)

var _ protocol.IdentityAdapter = (*IdentityAdapter)(nil)

type IdentityAdapter struct {
	keypair *crypto.KeyPair
}

func NewIdentityAdapter() *IdentityAdapter {
	kp := ed25519.RandomKeyPair()
	return NewIdentityAdapterFromKeypair(kp)
}

func NewIdentityAdapterFromKeypair(kp *crypto.KeyPair) *IdentityAdapter {
	return &IdentityAdapter{
		keypair: kp,
	}
}

func (a *IdentityAdapter) MyIdentity() []byte {
	return a.keypair.PublicKey
}

func (a *IdentityAdapter) Sign(input []byte) []byte {
	ret, err := a.keypair.Sign(ed25519.New(), blake2b.New(), input)
	if err != nil {
		panic(err)
	}
	return ret
}

func (a *IdentityAdapter) Verify(id, data, signature []byte) bool {
	return crypto.Verify(ed25519.New(), blake2b.New(), id, data, signature)
}

func (a *IdentityAdapter) SignatureSize() int {
	return ed25519.SignatureSize
}

func (a *IdentityAdapter) GetKeyPair() *crypto.KeyPair {
	return a.keypair
}
