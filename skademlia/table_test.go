package skademlia

import (
	"github.com/perlin-network/noise/edwards25519"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/blake2b"
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
