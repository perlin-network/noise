package aead

import (
	"crypto/aes"
	"crypto/cipher"
	"github.com/perlin-network/noise/internal/edwards25519"
	"github.com/pkg/errors"
	"golang.org/x/crypto/hkdf"
	"hash"
)

const sharedKeyLength = 32

// deriveCipherSuite derives an AEAD via. AES-256 GCM (Galois Counter Mode) cipher suite given an ephemeral shared key
// typically produced from a handshake/key exchange protocol.
func deriveCipherSuite(fn func() hash.Hash, ephemeralSharedKey []byte, context []byte) (cipher.AEAD, []byte, error) {
	deriver := hkdf.New(fn, ephemeralSharedKey, nil, context)

	sharedKey := make([]byte, sharedKeyLength)
	if _, err := deriver.Read(sharedKey); err != nil {
		return nil, nil, errors.Wrap(err, "failed to derive key via HKDF")
	}

	block, _ := aes.NewCipher(sharedKey)
	gcm, _ := cipher.NewGCM(block)

	return gcm, sharedKey, nil
}

func isEd25519GroupElement(buf []byte) bool {
	if len(buf) != edwards25519.PublicKeySize {
		return false
	}

	var buff [32]byte
	copy(buff[:], buf)

	var A edwards25519.ExtendedGroupElement
	return A.FromBytes(&buff)
}
