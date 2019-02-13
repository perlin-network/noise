package ed25519

import (
	"encoding/hex"
	"fmt"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/identity"
	"github.com/pkg/errors"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/group/edwards25519"
	"go.dedis.ch/kyber/v3/sign/schnorr"
)

var (
	_     identity.Manager     = (*Manager)(nil)
	suite crypto.EllipticSuite = edwards25519.NewBlakeSHA256Ed25519()
)

type Manager struct {
	privateKey kyber.Scalar
	publicKey  kyber.Point

	publicKeyBuf []byte
}

func New(privateKeyBuf []byte) *Manager {
	privateKey := suite.Scalar().SetBytes(privateKeyBuf)
	publicKey := suite.Point().Mul(privateKey, suite.Point().Base())

	publicKeyBuf, err := publicKey.MarshalBinary()

	if err != nil {
		panic(errors.Wrap(err, "failed to marshal public key"))
	}

	return &Manager{
		privateKey:   privateKey,
		publicKey:    publicKey,
		publicKeyBuf: publicKeyBuf,
	}
}

func Random() *Manager {
	privateKey := suite.Scalar().Pick(suite.RandomStream())

	privateKeyBytes, err := hex.DecodeString(privateKey.String())
	if err != nil {
		panic(errors.Wrap(err, "failed to marshal private key"))
	}

	return New(privateKeyBytes)
}

func (p *Manager) PublicID() []byte {
	return p.publicKeyBuf
}

func (p *Manager) Sign(buf []byte) ([]byte, error) {
	return schnorr.Sign(suite, p.privateKey, buf)
}

func (p *Manager) Verify(publicKeyBuf []byte, buf []byte, signature []byte) error {
	point := suite.Point()

	if err := point.UnmarshalBinary(publicKeyBuf); err != nil {
		return errors.Wrap(err, "an invalid public key was provided")
	}

	return schnorr.Verify(suite, point, buf, signature)
}

func (p *Manager) String() string {
	return fmt.Sprintf("ed25519-manager{publicKey: %s, privateKey: %s}", p.publicKey.String(), p.privateKey.String())
}

func (p *Manager) PrivateKey() string {
	return p.privateKey.String()
}
