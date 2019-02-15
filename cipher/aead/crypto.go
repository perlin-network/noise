package aead

import (
	"crypto/aes"
	"crypto/cipher"
	"github.com/pkg/errors"
	"go.dedis.ch/kyber/v3"
	"golang.org/x/crypto/hkdf"
	"hash"
)

const sharedKeyLength = 32

// deriveCipherSuite derives an AEAD via. AES-256 GCM (Galois Counter Mode) cipher suite given an ephemeral shared key
// typically produced from a handshake/key exchange protocol.
func deriveCipherSuite(fn func() hash.Hash, ephemeralSharedKey kyber.Point, context []byte) (cipher.AEAD, []byte, error) {
	ephemeralBuf, err := ephemeralSharedKey.MarshalBinary()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to marshal ephemeral shared key for AEAD")
	}

	deriver := hkdf.New(fn, ephemeralBuf, nil, context)

	sharedKey := make([]byte, sharedKeyLength)
	if _, err := deriver.Read(sharedKey); err != nil {
		return nil, nil, errors.Wrap(err, "failed to derive key via HKDF")
	}

	block, _ := aes.NewCipher(sharedKey)
	gcm, _ := cipher.NewGCM(block)

	return gcm, sharedKey, nil
}
