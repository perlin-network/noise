// Copyright (c) 2019 Perlin
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package cipher

import (
	"crypto/aes"
	"crypto/cipher"
	"hash"

	"github.com/pkg/errors"
	"golang.org/x/crypto/hkdf"
)

type suiteFn func([]byte) (cipher.AEAD, error)
type hashFn func() hash.Hash

const sharedKeyLength = 32

func DeriveAEAD(
	suiteFn suiteFn, hashFn hashFn, ephemeralSharedKey []byte, context []byte,
) (cipher.AEAD, []byte, error) {
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
	// TODO: document this somewhere, and/or use an alternative implementation of AES
	// if !cpu.Initialized || (cpu.Initialized && !cpu.ARM64.HasAES && !cpu.X86.HasAES && !cpu.S390X.HasAESGCM) {
	// 	panic("UNSUPPORTED: CPU does not support AES-NI instructions.")
	// }
	return func(sharedKey []byte) (cipher.AEAD, error) {
		block, _ := aes.NewCipher(sharedKey)
		suite, _ := cipher.NewGCM(block)

		return suite, nil
	}
}
