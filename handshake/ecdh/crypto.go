package ecdh

import (
	"github.com/perlin-network/noise/crypto"
	"go.dedis.ch/kyber/v3"
)

func computeSharedKey(suite crypto.EllipticSuite, nodePrivateKey kyber.Scalar, remotePublicKey kyber.Point) kyber.Point {
	return suite.Point().Mul(nodePrivateKey, remotePublicKey)
}
