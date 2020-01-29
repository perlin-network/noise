package kademlia

import (
	"fmt"
	"github.com/perlin-network/noise"
	"sync"
)

type Table struct {
	sync.RWMutex

	entries [noise.SizePublicKey * 8][]noise.ID
	self    noise.ID
}

func NewTable(self noise.ID) *Table {
	table := &Table{self: self}

	if err := table.Update(self); err != nil {
		panic(err)
	}

	return table
}

func (t *Table) Self() noise.ID {
	return t.self
}

func (t *Table) Bucket(target noise.PublicKey) []noise.ID {
	t.RLock()
	defer t.RUnlock()

	return t.entries[t.getBucketIndex(target)]
}

func (t *Table) Update(target noise.ID) error {
	if target.ID == noise.ZeroPublicKey {
		return nil
	}

	t.Lock()
	defer t.Unlock()

	idx := t.getBucketIndex(target.ID)

	for i, id := range t.entries[idx] {
		if id.ID == target.ID { // Found the target ID already inside the routing table.
			t.entries[idx][0], t.entries[idx][i] = t.entries[idx][i], t.entries[idx][0]
			return nil
		}
	}

	if len(t.entries[idx]) < BucketSize {
		t.entries[idx] = append([]noise.ID{target}, t.entries[idx]...)
		return nil
	}

	return fmt.Errorf("cannot insert id %x into routing table: %w", target.ID, ErrBucketFull)
}

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

func (t *Table) Peers() []noise.ID {
	return t.FindClosest(t.self.ID, BucketSize)
}

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
