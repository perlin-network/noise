package crypto

import "go.dedis.ch/kyber/v3"

type EllipticSuite interface {
	kyber.Group
	kyber.HashFactory
	kyber.XOFFactory
	kyber.Random
}
