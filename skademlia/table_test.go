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
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/blake2b"
	"sync"
	"testing"
	"testing/quick"
)

func TestUpdateAndDeleteFromTable(t *testing.T) {
	f := func(self edwards25519.PublicKey, targets []edwards25519.PublicKey) bool {
		table := NewTable(NewID("127.0.0.1", self, [blake2b.Size256]byte{}))

		for _, target := range targets {
			target := NewID("127.0.0.1", target, [blake2b.Size256]byte{})

			bucket := table.buckets[getBucketID(table.self.checksum, target.checksum)]

			// Test updating table when selective buckets are full.

			if bucket.Len() == table.getBucketSize() && !assert.Error(t, table.Update(target)) {
				return false
			}

			if bucket.Len() == table.getBucketSize() && !assert.Nil(t, table.Find(bucket, target)) {
				return false
			}

			// Test deleting entries when selective buckets are full.

			if bucket.Len() == table.getBucketSize() {
				front := bucket.Front()

				if !assert.False(t, table.Delete(bucket, nil)) {
					return false
				}

				if !assert.True(t, table.Delete(bucket, front.Value.(*ID))) {
					return false
				}

				if !assert.NoError(t, table.Update(front.Value.(*ID))) {
					return false
				}
			}

			// Test updating table when selective buckets are not full.

			if bucket.Len() < table.getBucketSize() && !assert.NoError(t, table.Update(target)) {
				return false
			}

			if bucket.Len() < table.getBucketSize() && !assert.NotNil(t, table.Find(bucket, target)) {
				return false
			}
		}

		// Attempt to push our own ID back into the table.
		assert.NoError(t, table.Update(table.self))

		// Have updates on nil IDs do nothing.
		assert.NoError(t, table.Update(nil))

		return true
	}

	assert.NoError(t, quick.Check(f, &quick.Config{MaxCount: 1000}))
}

func TestFindClosestPeers(t *testing.T) {
	t.Parallel()

	var nodes []*ID

	nodes = append(nodes,
		&ID{address: "0000"},
		&ID{address: "0001"},
		&ID{address: "0002"},
		&ID{address: "0003"},
		&ID{address: "0004"},
		&ID{address: "0005"},
	)

	var publicKey edwards25519.PublicKey

	copy(publicKey[:], []byte("12345678901234567890123456789010"))
	nodes[0].checksum = publicKey

	copy(publicKey[:], []byte("12345678901234567890123456789011"))
	nodes[1].checksum = publicKey

	copy(publicKey[:], []byte("12345678901234567890123456789012"))
	nodes[2].checksum = publicKey

	copy(publicKey[:], []byte("12345678901234567890123456789013"))
	nodes[3].checksum = publicKey

	copy(publicKey[:], []byte("12345678901234567890123456789014"))
	nodes[4].checksum = publicKey

	copy(publicKey[:], []byte("00000000000000000000000000000000"))
	nodes[5].checksum = publicKey

	table := NewTable(nodes[0])

	for i := 1; i <= 5; i++ {
		assert.NoError(t, table.Update(nodes[i]))
	}

	var testee []*ID
	for _, peer := range table.FindClosest(nodes[5], 3) {
		testee = append(testee, peer)
	}
	assert.Equalf(t, 3, len(testee), "expected 3 peers got %+v", testee)

	answerKeys := []int{2, 3, 4}
	for i, key := range answerKeys {
		_answer := nodes[key]
		assert.EqualValues(t, _answer, testee[i])
	}

	testee = []*ID{}
	for _, peer := range table.FindClosest(nodes[4], 2) {
		testee = append(testee, peer)
	}
	assert.Equalf(t, 2, len(testee), "expected 2 peers got %v", testee)
	answerKeys = []int{2, 3}
	for i, key := range answerKeys {
		_answer := nodes[key]
		assert.EqualValues(t, _answer, testee[i])
	}
}

func TestUpdateSamePublicKey(t *testing.T) {
	getPK := func() edwards25519.PublicKey {
		var pk edwards25519.PublicKey
		if _, err := rand.Read(pk[:]); err != nil {
			t.Fatal(err)
		}
		return pk
	}

	rootID := NewID("127.0.0.1", getPK(), [blake2b.Size256]byte{})
	table := NewTable(rootID)

	updated := NewID("127.0.0.2", getPK(), [blake2b.Size256]byte{})
	if !assert.NoError(t, table.Update(updated)) {
		return
	}

	// we create new id with same public key but different address
	addressToChange := "127.0.0.3"
	updatedCopy := NewID(addressToChange, updated.publicKey, updated.nonce)

	wg := sync.WaitGroup{}
	// ensure that updating table (and id) safe to be done concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		if !assert.NoError(t, table.Update(updatedCopy)) {
			return
		}
	}()

	found := table.FindClosest(rootID, 10)
	if !assert.Equal(t, 1, len(found)) {
		return
	}

	wg.Wait()

	// we expect id in the table to have same public key but updated address
	assert.Equal(t, updated.publicKey, found[0].publicKey)
	assert.Equal(t, addressToChange, found[0].Address())
}
