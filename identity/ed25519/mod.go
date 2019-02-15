package ed25519

import (
	"encoding/hex"
	"fmt"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/crypto/blake2b"
	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/identity"
	"github.com/pkg/errors"
)

var (
	_ identity.Manager = (*Manager)(nil)
)

type Manager struct {
	keypair *crypto.KeyPair
	signer  crypto.SignaturePolicy
	hasher  crypto.HashPolicy
}

func New(privateKeyBuf []byte) *Manager {
	b := blake2b.New()
	sp := ed25519.New()
	kp, err := crypto.FromPrivateKey(sp, hex.EncodeToString(privateKeyBuf))
	if err != nil {
		panic(errors.Wrap(err, "failed to marshal public key"))
	}
	return &Manager{
		keypair: kp,
		signer:  sp,
		hasher:  b,
	}
}

func Random() *Manager {
	privateKey := ed25519.RandomKeyPair().PrivateKey
	return New(privateKey)
}

func (p *Manager) PublicID() []byte {
	return p.keypair.PublicKey
}

func (p *Manager) Sign(buf []byte) ([]byte, error) {
	return p.keypair.Sign(p.signer, p.hasher, buf)
}

func (p *Manager) Verify(publicKeyBuf []byte, buf []byte, signature []byte) error {
	if crypto.Verify(p.signer, p.hasher, publicKeyBuf, buf, signature) {
		return nil
	}
	return errors.New("unable to verify signature")
}

func (p *Manager) String() string {
	return fmt.Sprintf("Ed25519(public: %s, private: %s)", hex.EncodeToString(p.PublicID()), hex.EncodeToString(p.PrivateKey()))
}

func (p *Manager) PrivateKey() []byte {
	return p.keypair.PrivateKey
}
