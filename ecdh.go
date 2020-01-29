package noise

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"github.com/agl/ed25519/extra25519"
	"github.com/oasislabs/ed25519/extra/x25519"
)

// ECDH transform all Ed25519 points to Curve25519 points and performs a Diffie-Hellman handshake
// to derive a shared key. It throws an error should the Ed25519 points be invalid.
func ECDH(ourPrivateKey PrivateKey, peerPublicKey PublicKey) ([]byte, error) {
	var (
		curve25519Sec [x25519.ScalarSize]byte
		curve25519Pub [x25519.PointSize]byte
	)

	extra25519.PrivateKeyToCurve25519(&curve25519Sec, (*[ed25519.PrivateKeySize]byte)(&ourPrivateKey))

	if !extra25519.PublicKeyToCurve25519(&curve25519Pub, (*[ed25519.PublicKeySize]byte)(&peerPublicKey)) {
		return nil, errors.New("got an invalid ed25519 public key")
	}

	shared, err := x25519.X25519(curve25519Sec[:], curve25519Pub[:])
	if err != nil {
		return nil, fmt.Errorf("could not derive a shared key: %w", err)
	}

	return shared, nil
}
