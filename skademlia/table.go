package skademlia

import (
	"bytes"
	"container/list"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
	"sync"
)

const DefaultBucketSize = 16

var (
	ErrBucketFull = errors.New("kademlia: cannot add ID, bucket is full")
)

type table struct {
	self protocol.ID

	buckets []*bucket
}

type bucket struct {
	sync.RWMutex
	list.List
}

func newBucket() *bucket {
	return &bucket{}
}

func newTable(self protocol.ID) *table {
	if self == nil {
		panic("kademlia: self ID must not be nil")
	}

	table := table{
		self:    self,
		buckets: make([]*bucket, len(self.Hash())*8),
	}

	for i := 0; i < len(self.Hash())*8; i++ {
		table.buckets[i] = newBucket()
	}

	table.Update(self)

	return &table
}

func (t *table) Update(target protocol.ID) error {
	if len(t.self.Hash()) != len(target.Hash()) {
		return errors.New("kademlia: got invalid hash size for target ID on update")
	}

	bucket := t.bucket(t.bucketID(target.Hash()))

	bucket.Lock()
	defer bucket.Unlock()

	var element *list.Element

	// Find current peer in bucket.
	for e := bucket.Front(); e != nil; e = e.Next() {
		id := e.Value.(protocol.ID)

		if bytes.Equal(id.Hash(), target.Hash()) {
			element = e
			break
		}
	}

	if element == nil {
		// Populate bucket if its not full.
		if bucket.Len() < DefaultBucketSize {
			bucket.PushFront(target)
		} else {
			return ErrBucketFull
		}
	} else {
		bucket.MoveToFront(element)
	}

	return nil
}

func (t *table) Get(target protocol.ID) (protocol.ID, bool) {
	bucket := t.bucket(t.bucketID(target.Hash()))

	bucket.RLock()
	defer bucket.RUnlock()

	for e := bucket.Front(); e != nil; e = e.Next() {
		if found := e.Value.(protocol.ID); bytes.Equal(found.Hash(), target.Hash()) {
			return found, true
		}
	}

	return nil, false
}

func (t *table) Delete(target protocol.ID) bool {
	bucket := t.bucket(t.bucketID(target.Hash()))

	bucket.Lock()
	defer bucket.Unlock()

	for e := bucket.Front(); e != nil; e = e.Next() {
		if found := e.Value.(protocol.ID); bytes.Equal(found.Hash(), target.Hash()) {
			bucket.Remove(e)
			return true
		}
	}

	return false
}

// GetPeers returns a unique list of all peers within the routing network.
func (t *table) GetPeers() (addresses []string) {
	visited := make(map[string]struct{})
	visited[string(t.self.Hash())] = struct{}{}

	for _, bucket := range t.buckets {
		bucket.RLock()

		for e := bucket.Front(); e != nil; e = e.Next() {
			id := e.Value.(protocol.ID)

			if _, seen := visited[string(id.Hash())]; !seen {
				addresses = append(addresses, id.(ID).address)

				visited[string(id.Hash())] = struct{}{}
			}
		}

		bucket.RUnlock()
	}

	return
}

// bucketID returns the corresponding bucket ID based on the ID.
func (t *table) bucketID(id []byte) int {
	return prefixLen(xor(id, t.self.Hash()))
}

// bucket returns a specific bucket by ID.
func (t *table) bucket(id int) *bucket {
	if id >= 0 && id < len(t.buckets) {
		return t.buckets[id]
	}

	return nil
}
