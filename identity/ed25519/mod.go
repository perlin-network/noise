package ed25519

import (
	"encoding/hex"
	"fmt"
	"github.com/perlin-network/noise/identity"
	"github.com/perlin-network/noise/internal/edwards25519"
	"github.com/pkg/errors"
)

var (
	_ identity.Keypair = (*Keypair)(nil)
)

type Keypair struct {
	privateKey edwards25519.PrivateKey
	publicKey  edwards25519.PublicKey
}

func LoadKeys(privateKeyBuf []byte) *Keypair {
	if len(privateKeyBuf) != edwards25519.PrivateKeySize {
		panic(errors.Errorf("edwards25519: private key is not %d bytes", edwards25519.PrivateKeySize))
	}

	privateKey := edwards25519.PrivateKey(privateKeyBuf)

	return &Keypair{
		privateKey: privateKey,
		publicKey:  privateKey.Public().(edwards25519.PublicKey),
	}
}

func RandomKeys() *Keypair {
	publicKey, privateKey, err := edwards25519.GenerateKey(nil)
	if err != nil {
		panic(errors.Wrap(err, "edwards25519: failed to generate random keypair"))
	}

	return &Keypair{
		privateKey: privateKey,
		publicKey:  publicKey,
	}
}

func (p *Keypair) ID() []byte {
	return p.publicKey
}

func (p *Keypair) PublicKey() []byte {
	return p.publicKey
}

func (p *Keypair) Sign(buf []byte) ([]byte, error) {
	if len(p.privateKey) != edwards25519.PrivateKeySize {
		return nil, errors.Errorf("edwards25519: private key expected to be %d bytes, but is %d bytes", edwards25519.PrivateKeySize, len(p.privateKey))
	}

	return edwards25519.Sign(p.privateKey, buf), nil
}

func (p *Keypair) Verify(publicKeyBuf []byte, buf []byte, signature []byte) error {
	if len(publicKeyBuf) != edwards25519.PublicKeySize {
		return errors.Errorf("edwards25519: public key expected to be %d bytes, but is %d bytes", edwards25519.PublicKeySize, len(publicKeyBuf))
	}

	if edwards25519.Verify(publicKeyBuf, buf, signature) {
		return nil
	} else {
		return errors.New("unable to verify signature")
	}
}

func (p *Keypair) String() string {
	return fmt.Sprintf("Ed25519(public: %s, private: %s)", hex.EncodeToString(p.PublicKey()), hex.EncodeToString(p.PrivateKey()))
}

func (p *Keypair) PrivateKey() []byte {
	return p.privateKey
}
