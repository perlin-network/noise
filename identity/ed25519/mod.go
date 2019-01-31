package ed25519

import (
	"fmt"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/identity"
	"github.com/pkg/errors"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/group/edwards25519"
	"go.dedis.ch/kyber/v3/sign/schnorr"
)

var (
	_     identity.Manager     = (*manager)(nil)
	suite crypto.EllipticSuite = edwards25519.NewBlakeSHA256Ed25519()
)

type manager struct {
	privateKey kyber.Scalar
	publicKey  kyber.Point

	publicKeyBuf []byte
}

func New(privateKeyBuf []byte) *manager {
	privateKey := suite.Scalar().SetBytes(privateKeyBuf)
	publicKey := suite.Point().Mul(privateKey, suite.Point().Base())

	publicKeyBuf, err := publicKey.MarshalBinary()

	if err != nil {
		panic(errors.Wrap(err, "failed to marshal public key"))
	}

	return &manager{privateKey: privateKey, publicKey: publicKey, publicKeyBuf: publicKeyBuf}
}

func Random() *manager {
	privateKey := suite.Scalar().Pick(suite.RandomStream())
	publicKey := suite.Point().Mul(privateKey, suite.Point().Base())

	publicKeyBuf, err := publicKey.MarshalBinary()

	if err != nil {
		panic(errors.Wrap(err, "failed to marshal public key"))
	}

	return &manager{privateKey: privateKey, publicKey: publicKey, publicKeyBuf: publicKeyBuf}
}

func (p *manager) PublicID() []byte {
	return p.publicKeyBuf
}

func (p *manager) Sign(buf []byte) ([]byte, error) {
	return schnorr.Sign(suite, p.privateKey, buf)
}

func (p *manager) Verify(publicKeyBuf []byte, buf []byte, signature []byte) error {
	point := suite.Point()

	if err := point.UnmarshalBinary(publicKeyBuf); err != nil {
		return errors.Wrap(err, "an invalid public key was provided")
	}

	return schnorr.Verify(suite, point, buf, signature)
}

func (p *manager) String() string {
	return fmt.Sprintf("ed25519-manager{publicKey: %s, privateKey: %s}", p.publicKey.String(), p.privateKey.String())
}
