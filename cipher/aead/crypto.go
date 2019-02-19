package aead

import (
	"crypto/aes"
	"crypto/cipher"
	"github.com/pkg/errors"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
	"hash"
)

const sharedKeyLength = 32

// deriveCipherSuite derives an AEAD cipher suite given an ephemeral shared key
// typically produced from a handshake/key exchange protocol.
func (b *block) deriveCipherSuite(fn func() hash.Hash, ephemeralSharedKey []byte, context []byte) (cipher.AEAD, []byte, error) {
	deriver := hkdf.New(fn, ephemeralSharedKey, nil, context)

	sharedKey := make([]byte, sharedKeyLength)
	if _, err := deriver.Read(sharedKey); err != nil {
		return nil, nil, errors.Wrap(err, "failed to derive key via HKDF")
	}

	suite, err := b.suiteFn(sharedKey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to derive AEAD suite")
	}

	return suite, sharedKey, nil
}

// AEAD via. AES-256 GCM (Galois Counter Mode).
func AES256_GCM(sharedKey []byte) (cipher.AEAD, error) {
	block, _ := aes.NewCipher(sharedKey)
	suite, _ := cipher.NewGCM(block)

	return suite, nil
}

// AEAD via. ChaCha20 Poly1305. Expects a 256-bit shared key.
func ChaCha20_Poly1305(sharedKey []byte) (cipher.AEAD, error) {
	return chacha20poly1305.New(sharedKey)
}

// AEAD via. XChaCha20 Poly1305. Expected a 256-bit shared key.
func XChaCha20_Poly1305(sharedKey []byte) (cipher.AEAD, error) {
	return chacha20poly1305.NewX(sharedKey)
}
