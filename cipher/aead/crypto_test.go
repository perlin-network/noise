package aead

import (
	"crypto/sha256"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/group/edwards25519"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
)

type quickPoint struct {
	kyber.Point
}

func (p *quickPoint) Generate(rand *rand.Rand, size int) reflect.Value {
	curve := edwards25519.NewBlakeSHA256Ed25519()
	scalar := curve.Scalar().Pick(curve.RandomStream())
	return reflect.ValueOf(&quickPoint{Point: curve.Point().Mul(scalar, curve.Point().Base())})
}

func TestDeriveSharedKey(t *testing.T) {
	check := func(ephemeralSharedKey *quickPoint, context []byte) bool {
		_, _, err := deriveCipherSuite(sha256.New, ephemeralSharedKey, context)

		if err != nil {
			return false
		}

		return true
	}

	if err := quick.Check(check, nil); err != nil {
		t.Error(err)
	}
}
