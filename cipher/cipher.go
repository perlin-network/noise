package cipher

import (
	"crypto/aes"
	"crypto/cipher"
	"github.com/pkg/errors"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/sys/cpu"
	"hash"
)

type suiteFn func([]byte) (cipher.AEAD, error)
type hashFn func() hash.Hash

const sharedKeyLength = 32

func DeriveAEAD(suiteFn suiteFn, hashFn hashFn, ephemeralSharedKey []byte, context []byte) (cipher.AEAD, []byte, error) {
	deriver := hkdf.New(hashFn, ephemeralSharedKey, nil, context)

	sharedKey := make([]byte, sharedKeyLength)
	if _, err := deriver.Read(sharedKey); err != nil {
		return nil, nil, errors.Wrap(err, "failed to derive key via hkdf")
	}

	suite, err := suiteFn(sharedKey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to derive aead suite")
	}

	return suite, sharedKey, nil
}

// AEAD via. AES-256 GCM (Galois Counter Mode).
func Aes256GCM() func(sharedKey []byte) (cipher.AEAD, error) {
	if !cpu.Initialized || (cpu.Initialized && !cpu.ARM64.HasAES && !cpu.X86.HasAES && !cpu.S390X.HasAESGCM) {
		panic("UNSUPPORTED: CPU does not support AES-NI instructions.")
	}

	return func(sharedKey []byte) (cipher.AEAD, error) {
		block, _ := aes.NewCipher(sharedKey)
		suite, _ := cipher.NewGCM(block)

		return suite, nil
	}
}
