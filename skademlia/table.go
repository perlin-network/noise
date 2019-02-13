package skademlia

import (
	"bytes"
	"container/list"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
	"sort"
	"sync"
	"time"
)

var (
	bucketSize    = 16
	ErrBucketFull = errors.New("kademlia: cannot add ID, bucket is full")
)

type table struct {
	self protocol.ID

	numBuckets int
	buckets    []*bucket
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

	numBuckets := len(self.Hash()) * 8
	table := table{
		self:       self,
		numBuckets: numBuckets,
		buckets:    make([]*bucket, numBuckets),
	}

	for i := 0; i < numBuckets; i++ {
		table.buckets[i] = newBucket()
	}

	table.Update(self)

	return &table
}

func BucketSize() int {
	return bucketSize
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
		if bucket.Len() < BucketSize() {
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

func Table(node *noise.Node) *table {
	t := node.Get(keyKademliaTable)

	if t == nil {
		panic("kademlia: node has not enforced identity policy, and thus has no table associated to it")
	}

	if t, ok := t.(*table); ok {
		return t
	}

	panic("kademlia: table associated to node is not an instance of a kademlia table")
}

// FindClosestPeers returns a list of K peers with in order of ascending XOR distance.
func FindClosestPeers(t *table, target []byte, K int) (peers []protocol.ID) {
	bucketID := t.bucketID(xor(target, t.self.Hash()))
	bucket := t.bucket(bucketID)

	bucket.RLock()

	for e := bucket.Front(); e != nil; e = e.Next() {
		if !e.Value.(protocol.ID).Equals(t.self) {
			peers = append(peers, e.Value.(protocol.ID))
		}
	}

	bucket.RUnlock()

	for i := 1; len(peers) < K && (bucketID-i >= 0 || bucketID+i < t.numBuckets); i++ {
		if bucketID-i >= 0 {
			other := t.bucket(bucketID - i)
			other.RLock()
			for e := other.Front(); e != nil; e = e.Next() {
				if !e.Value.(protocol.ID).Equals(t.self) {
					peers = append(peers, e.Value.(protocol.ID))
				}
			}
			other.RUnlock()
		}

		if bucketID+i < t.numBuckets {
			other := t.bucket(bucketID + i)
			other.RLock()
			for e := other.Front(); e != nil; e = e.Next() {
				if !e.Value.(protocol.ID).Equals(t.self) {
					peers = append(peers, e.Value.(protocol.ID))
				}
			}
			other.RUnlock()
		}
	}

	// Sort peers by XOR distance.
	sort.Slice(peers, func(i, j int) bool {
		return bytes.Compare(xor(peers[i].Hash(), target), xor(peers[j].Hash(), target)) == -1
	})

	if len(peers) > K {
		peers = peers[:K]
	}

	return peers
}

func UpdateTable(node *noise.Node, target protocol.ID) (err error) {
	table := Table(node)

	if err = table.Update(target); err != nil {
		switch err {
		case ErrBucketFull:
			bucket := table.bucket(table.bucketID(target.Hash()))

			last := bucket.Back()
			lastPeer := protocol.Peer(node, last.Value.(protocol.ID))

			if lastPeer == nil {
				return errors.New("kademlia: last peer in bucket was not actually connected to our node")
			}

			// If the candidate peer to-be-evicted responds with an 'evict' message back, move him to the front of the bucket
			// and do not push the target ID into the bucket. Else, evict the candidate peer and push the target ID to the
			// front of the bucket.
			evictLastPeer := func() {
				lastPeer.Disconnect()

				bucket.Remove(last)
				bucket.PushFront(target)
			}

			// Send an 'evict' message to the candidate peer to-be-evicted.
			err := lastPeer.SendMessage(OpcodeEvict, Evict{})

			if err != nil {
				evictLastPeer()
				return nil
			}

			select {
			case <-lastPeer.Receive(OpcodeEvict):
				bucket.MoveToFront(last)
			case <-time.After(3 * time.Second):
				evictLastPeer()
			}
		default:
			return err
		}
	}

	return nil
}
