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
	"bytes"
	"github.com/perlin-network/noise/edwards25519"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/blake2b"
	"testing"
	"testing/quick"
)

func TestPreventLoadingIllegalKeys(t *testing.T) {
	f := func(address string, privateKey edwards25519.PrivateKey, nonce [blake2b.Size256]byte, c1, c2 byte) bool {
		c1 /= 32
		c2 /= 32

		publicKey := privateKey.Public()

		id := blake2b.Sum256(publicKey[:])
		checksum := blake2b.Sum256(id[:])

		err := verifyPuzzle(checksum, nonce, int(c1), int(c2))
		keys, err2 := LoadKeys(privateKey, int(c1), int(c2))

		if keys == nil && !assert.Error(t, err2) {
			return false
		}

		if keys != nil && !assert.NoError(t, err2) {
			return false
		}

		if err != nil && err2 != nil {
			return assert.Equal(t, errors.Cause(err).Error(), errors.Cause(err2).Error())
		}

		return true
	}

	assert.NoError(t, quick.Check(f, &quick.Config{MaxCount: 1000}))
}

func TestCreateThenLoadKeys(t *testing.T) {
	f := func(address string, c1, c2 byte) bool {
		c1 /= 32
		c2 /= 32

		keys, err := NewKeys(int(c1), int(c2))

		if !assert.NotNil(t, keys) || !assert.NoError(t, err) {
			return false
		}

		if keys == nil || err != nil {
			return false
		}

		if !assert.NotZero(t, keys.publicKey) {
			return false
		}

		if !assert.NotZero(t, keys.privateKey) {
			return false
		}

		if !assert.NotZero(t, keys.checksum) {
			return false
		}

		if !assert.NotZero(t, keys.nonce) {
			return false
		}

		if !assert.Equal(t, keys.c1, int(c1)) || !assert.Equal(t, keys.c2, int(c2)) {
			return false
		}

		reloaded, err := LoadKeys(keys.privateKey, int(c1), int(c2))

		if !assert.NotNil(t, reloaded) || !assert.NoError(t, err) {
			return false
		}

		if reloaded == nil || err != nil {
			return false
		}

		if !assert.Equal(t, keys.publicKey, reloaded.publicKey) {
			return false
		}

		if !assert.Equal(t, keys.privateKey, reloaded.privateKey) {
			return false
		}

		if !assert.Equal(t, keys.checksum, reloaded.checksum) {
			return false
		}

		if !assert.Equal(t, keys.c1, reloaded.c1) || !assert.Equal(t, keys.c2, reloaded.c2) {
			return false
		}

		return true
	}

	assert.NoError(t, quick.Check(f, &quick.Config{MaxCount: 100}))
}

func TestMarshalUnmarshalID(t *testing.T) {
	var zero ID

	f := func(address string, pub edwards25519.PublicKey, nonce [blake2b.Size256]byte, buf []byte) bool {
		m := NewID(address, pub, nonce)
		m2, err := UnmarshalID(bytes.NewReader(m.Marshal()))

		if m3, err := UnmarshalID(bytes.NewReader(buf)); (m3 == zero && !assert.Error(t, err)) || (m3 != zero && !assert.NoError(t, err)) { // nolint:lll
			return false
		}

		return assert.NoError(t, err) && assert.EqualValues(t, *m, m2)
	}

	assert.NoError(t, quick.Check(f, &quick.Config{MaxCount: 1000}))
}
