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

package handshake

import (
	"crypto/sha512"
	"github.com/perlin-network/noise/edwards25519"
)

func computeSharedKey(nodePrivateKey edwards25519.PrivateKey, remotePublicKey edwards25519.PublicKey) []byte {
	var nodeSecretKeyBuf, sharedKeyBuf [32]byte
	copy(nodeSecretKeyBuf[:], deriveSecretKey(nodePrivateKey))

	var sharedKeyElement, publicKeyElement edwards25519.ExtendedGroupElement
	publicKeyElement.FromBytes((*[32]byte)(&remotePublicKey))

	edwards25519.GeScalarMult(&sharedKeyElement, &nodeSecretKeyBuf, &publicKeyElement)

	sharedKeyElement.ToBytes(&sharedKeyBuf)

	return sharedKeyBuf[:]
}

func deriveSecretKey(privateKey edwards25519.PrivateKey) []byte {
	digest := sha512.Sum512(privateKey[:32])
	digest[0] &= 248
	digest[31] &= 127
	digest[31] |= 64

	return digest[:32]
}
