package kademlia

import (
	"fmt"
	"github.com/perlin-network/noise"
	"sync"
)

// Table represents a Kademlia routing table.
type Table struct {
	sync.RWMutex

	entries [noise.SizePublicKey * 8][]noise.ID
	self    noise.ID
}

// NewTable instantiates a new routing table whose XOR distance metric is defined with respect to some
// given ID.
func NewTable(self noise.ID) *Table {
	table := &Table{self: self}

	if err := table.Update(self); err != nil {
		panic(err)
	}

	return table
}

// Self returns the ID which this routing table's XOR distance metric is defined with respect to.
func (t *Table) Self() noise.ID {
	return t.self
}

// Bucket returns all IDs in the bucket where target resides within.
func (t *Table) Bucket(target noise.PublicKey) []noise.ID {
	t.RLock()
	defer t.RUnlock()

	return t.entries[t.getBucketIndex(target)]
}

// Update attempts to insert the target node/peer ID into this routing table. If the bucket it was expected
// to be inserted within is full, ErrBucketFull is returned. If the ID already exists in its respective routing
// table bucket, it is moved to the head of the bucket.
func (t *Table) Update(target noise.ID) error {
	if target.ID == noise.ZeroPublicKey {
		return nil
	}

	t.Lock()
	defer t.Unlock()

	idx := t.getBucketIndex(target.ID)

	for i, id := range t.entries[idx] {
		if id.ID == target.ID { // Found the target ID already inside the routing table.
			t.entries[idx] = append(append([]noise.ID{target}, t.entries[idx][:i]...), t.entries[idx][i+1:]...)
			return nil
		}
	}

	if len(t.entries[idx]) < BucketSize { // The bucket is not yet under full capacity.
		t.entries[idx] = append([]noise.ID{target}, t.entries[idx]...)
		return nil
	}

	// The bucket is at full capacity. Return ErrBucketFull.

	return fmt.Errorf("cannot insert id %x into routing table: %w", target.ID, ErrBucketFull)
}

// Recorded returns true if target is already recorded in this routing table.
func (t *Table) Recorded(target noise.PublicKey) bool {
	t.RLock()
	defer t.RUnlock()

	for _, id := range t.entries[t.getBucketIndex(target)] {
		if id.ID == target {
			return true
		}
	}

	return false
}

// Delete removes target from this routing table.
func (t *Table) Delete(target noise.PublicKey) bool {
	t.Lock()
	defer t.Unlock()

	idx := t.getBucketIndex(target)

	for i, id := range t.entries[idx] {
		if id.ID == target {
			t.entries[idx] = append(t.entries[idx][:i], t.entries[idx][i+1:]...)
			return true
		}
	}

	return false
}

// Peers returns BucketSize closest peer IDs to the ID which this routing table's distance metric is defined against.
func (t *Table) Peers() []noise.ID {
	return t.FindClosest(t.self.ID, BucketSize)
}

// FindClosest returns the k closest peer IDs to target, and sorts them based on how close they are.
func (t *Table) FindClosest(target noise.PublicKey, k int) []noise.ID {
	var closest []noise.ID

	f := func(bucket []noise.ID) {
		for _, id := range bucket {
			if id.ID != target {
				closest = append(closest, id)
			}
		}
	}

	t.RLock()
	defer t.RUnlock()

	idx := t.getBucketIndex(target)

	f(t.entries[idx])

	for i := 1; len(closest) < k && (idx-i >= 0 || idx+i < len(t.entries)); i++ {
		if idx-i >= 0 {
			f(t.entries[idx-i])
		}

		if idx+i < len(t.entries) {
			f(t.entries[idx+i])
		}
	}

	closest = SortByDistance(target, closest)

	if len(closest) > k {
		closest = closest[:k]
	}

	return closest
}

func (t *Table) getBucketIndex(target noise.PublicKey) int {
	l := PrefixLen(XOR(target[:], t.self.ID[:]))
	if l == noise.SizePublicKey*8 {
		return l - 1
	}

	return l
}
