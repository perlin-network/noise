package noise

import (
	"crypto/ed25519"
	"crypto/sha512"
	"fmt"
	"github.com/oasislabs/ed25519/extra/x25519"
	"golang.org/x/crypto/curve25519"
	"math/big"
)

var curve25519P, _ = new(big.Int).SetString("57896044618658097711785492504343953926634992332820282019728792003956564819949", 10)

func ed25519PublicKeyToCurve25519(pk PublicKey) []byte {
	// ed25519.PublicKey is a little endian representation of the y-coordinate,
	// with the most significant bit set based on the sign of the x-coordinate.
	bigEndianY := make([]byte, ed25519.PublicKeySize)
	for i, b := range pk {
		bigEndianY[ed25519.PublicKeySize-i-1] = b
	}
	bigEndianY[0] &= 0b0111_1111

	// The Montgomery u-coordinate is derived through the bilinear map
	//
	//     u = (1 + y) / (1 - y)
	//
	// See https://blog.filippo.io/using-ed25519-keys-for-encryption.
	y := new(big.Int).SetBytes(bigEndianY)
	denom := big.NewInt(1)
	denom.ModInverse(denom.Sub(denom, y), curve25519P) // 1 / (1 - y)
	u := y.Mul(y.Add(y, big.NewInt(1)), denom)
	u.Mod(u, curve25519P)

	out := make([]byte, curve25519.PointSize)
	uBytes := u.Bytes()
	for i, b := range uBytes {
		out[len(uBytes)-i-1] = b
	}

	return out
}

func ed25519PrivateKeyToCurve25519(pk PrivateKey) []byte {
	h := sha512.New()
	h.Write(pk[:curve25519.ScalarSize])
	out := h.Sum(nil)
	return out[:curve25519.ScalarSize]
}

// ECDH transform all Ed25519 points to Curve25519 points and performs a Diffie-Hellman handshake
// to derive a shared key. It throws an error should the Ed25519 points be invalid.
func ECDH(ourPrivateKey PrivateKey, peerPublicKey PublicKey) ([]byte, error) {
	shared, err := x25519.X25519(ed25519PrivateKeyToCurve25519(ourPrivateKey), ed25519PublicKeyToCurve25519(peerPublicKey))
	if err != nil {
		return nil, fmt.Errorf("could not derive a shared key: %w", err)
	}

	return shared, nil
}
