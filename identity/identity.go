package identity

import (
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/crypto/blake2b"
	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/protocol"
)

var _ protocol.IdentityAdapter = (*DefaultIdentityAdapter)(nil)

type DefaultIdentityAdapter struct {
	keypair *crypto.KeyPair
}

func NewDefaultIdentityAdapter() *DefaultIdentityAdapter {
	kp := ed25519.RandomKeyPair()
	return NewDefaultIdentityAdapterFromKeypair(kp)
}

func NewDefaultIdentityAdapterFromKeypair(kp *crypto.KeyPair) *DefaultIdentityAdapter {
	return &DefaultIdentityAdapter{
		keypair: kp,
	}
}

func (a *DefaultIdentityAdapter) MyIdentity() []byte {
	return a.keypair.PublicKey
}

func (a *DefaultIdentityAdapter) Sign(input []byte) []byte {
	ret, err := a.keypair.Sign(ed25519.New(), blake2b.New(), input)
	if err != nil {
		panic(err)
	}
	return ret
}

func (a *DefaultIdentityAdapter) Verify(id, data, signature []byte) bool {
	return crypto.Verify(ed25519.New(), blake2b.New(), id, data, signature)
}

func (a *DefaultIdentityAdapter) SignatureSize() int {
	return ed25519.SignatureSize
}

func (a *DefaultIdentityAdapter) GetKeyPair() *crypto.KeyPair {
	return a.keypair
}
