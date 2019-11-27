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

package skademlia

import (
	"crypto/rand"
	"github.com/perlin-network/noise/edwards25519"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
)

// generateKeys attempts to randomly generate a suitable Ed25519 keypair which satisfies the
// condition that blake2b(blake2b(publicKey)) has at least c1 prefixed zero bits.
func generateKeys(c1 int) (publicKey edwards25519.PublicKey, privateKey edwards25519.PrivateKey, id [blake2b.Size256]byte, checksum [blake2b.Size256]byte, err error) { // nolint:lll
	for {
		publicKey, privateKey, err = edwards25519.GenerateKey(nil)

		if err != nil {
			err = errors.Wrap(err, "failed to generate random keys")
			return
		}

		id = blake2b.Sum256(publicKey[:])
		checksum = blake2b.Sum256(id[:])

		if staticPuzzle := prefixLen(checksum[:]); staticPuzzle >= c1 {
			return
		}
	}
}

// generateNonce attempts to randomly generate a suitable nonce which satisfies the condition
// that xor(checksum, nonce) has at least c2 prefixed zero bits.
func generateNonce(checksum [blake2b.Size256]byte, c2 int) ([blake2b.Size256]byte, error) {
	var nonce [blake2b.Size256]byte

	for {
		n, err := rand.Read(nonce[:])

		if err != nil {
			return nonce, err
		}

		if n != blake2b.Size256 {
			return nonce, errors.Errorf("failed to generate %d bytes", blake2b.Size256)
		}

		if dynamicPuzzle := prefixLen(xor(checksum[:], nonce[:])); dynamicPuzzle >= c2 {
			return nonce, nil
		}
	}
}

// verifyPuzzle checks whether or not given the checksum of an id and a corresponding nonce, that
// they suffice both S/Kademlia's static and dynamic puzzle given protocol parameters c1 and c2.
func verifyPuzzle(checksum, nonce [blake2b.Size256]byte, c1, c2 int) error {
	if staticPuzzle := prefixLen(checksum[:]); staticPuzzle < c1 {
		return errors.Errorf(
			"failed to pass static puzzle as prefix length of checksum is %d, yet c1 is %d", staticPuzzle, c1,
		)
	}

	if dynamicPuzzle := prefixLen(xor(checksum[:], nonce[:])); dynamicPuzzle < c2 {
		return errors.Errorf(
			"failed to pass dynamic puzzle as prefix length of xor(checksum, nonce) is %d, yet c2 is %d",
			dynamicPuzzle, c2,
		)
	}

	return nil
}
