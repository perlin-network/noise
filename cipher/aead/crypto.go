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

	reader := hkdf.New(fn, ephemeralBuf, nil, context)

	sharedKey := make([]byte, sharedKeyLength)
	if _, err := reader.Read(sharedKey); err != nil {
		return nil, nil, errors.Wrap(err, "failed to derive key via HKDF")
	}

	block, err := aes.NewCipher(sharedKey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to init AES-256")
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to init GCM (Galois Counter Mode) for AES-256 cipher")
	}

	return gcm, sharedKey, nil
}
