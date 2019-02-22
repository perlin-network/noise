package eddsa

import (
	"bytes"
	"github.com/perlin-network/noise/internal/edwards25519"
	"github.com/stretchr/testify/assert"
	"testing"
	"testing/quick"
)

func TestBadKey(t *testing.T) {
	badKey := []byte("this is a bad key")
	message := []byte("this is a message")

	scheme := New()
	_, err := scheme.Sign(badKey, message)
	assert.Error(t, err)

	_, err = Sign(badKey, message)
	assert.Error(t, err)

	err = scheme.Verify(badKey, message, []byte("random signature"))
	assert.Error(t, err)

	err = Verify(badKey, message, []byte("random signature"))
	assert.Error(t, err)
}

func TestSignAndVerify(t *testing.T) {
	scheme := New()

	quick.Check(func(message []byte, bad []byte) bool {
		publicKey, privateKey, err := edwards25519.GenerateKey(nil)
		if err != nil {
			return false
		}

		sign1, err := scheme.Sign(privateKey, message)
		if err != nil {
			return false
		}

		sign2, err := Sign(privateKey, message)
		if err != nil {
			return false
		}

		if !bytes.Equal(sign1, sign2) {
			return false
		}

		if scheme.Verify(publicKey, message, sign2) != nil {
			return false
		}

		if Verify(publicKey, message, sign1) != nil {
			return false
		}

		// Check invalid signature
		if scheme.Verify(publicKey, message, append(sign2, bad...)) != nil {
			return false
		}

		if Verify(publicKey, message, append(sign1, bad...)) != nil {
			return false
		}

		return true
	}, nil)
}
