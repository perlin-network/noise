package skademlia

import (
	"bytes"
	"container/list"
	"sort"
	"sync"
)

// BucketSize defines the NodeID, Key, and routing table data structures.
const BucketSize = 16

// RoutingTable contains one bucket list for lookups.
type RoutingTable struct {
	// Current node's ID.
	self ID

	buckets []*Bucket
}

// ID is an IdentityAdapter and address pair
type ID struct {
	*IdentityAdapter
	Address string
}

// Bucket holds a list of contacts of this node.
type Bucket struct {
	*list.List
	mutex *sync.RWMutex
}

// NewBucket is a Factory method of Bucket, contains an empty list.
func NewBucket() *Bucket {
	return &Bucket{
		List:  list.New(),
		mutex: &sync.RWMutex{},
	}
}

// CreateRoutingTable is a Factory method of RoutingTable containing empty buckets.
func CreateRoutingTable(id ID) *RoutingTable {
	table := &RoutingTable{
		self:    id,
		buckets: make([]*Bucket, len(id.MyIdentity())*8),
	}
	for i := 0; i < len(id.MyIdentity())*8; i++ {
		table.buckets[i] = NewBucket()
	}

	table.Update(id)

	return table
}

// Self returns the ID of the node hosting the current routing table instance.
func (t *RoutingTable) Self() ID {
	return t.self
}

// Update moves a peer to the front of a bucket in the routing table.
func (t *RoutingTable) Update(target ID) {
	if len(t.self.MyIdentity()) != len(target.MyIdentity()) {
		return
	}

	bucketID := prefixLen(xor(target.MyIdentity(), t.self.MyIdentity()))
	bucket := t.Bucket(bucketID)

	var element *list.Element

	// Find current node in bucket.
	bucket.mutex.Lock()

	for e := bucket.Front(); e != nil; e = e.Next() {
		id := e.Value.(ID)
		if bytes.Equal(id.MyIdentity(), target.MyIdentity()) {
			element = e
			break
		}
	}

	if element == nil {
		// Populate bucket if its not full.
		if bucket.Len() <= BucketSize {
			bucket.PushFront(target)
		}
	} else {
		bucket.MoveToFront(element)
	}

	bucket.mutex.Unlock()
}

// GetPeer retrieves the ID struct in the routing table given a node ID if found.
func (t *RoutingTable) GetPeer(id []byte) *ID {
	bucketID := prefixLen(xor(id, t.self.MyIdentity()))
	bucket := t.Bucket(bucketID)

	bucket.mutex.Lock()

	defer bucket.mutex.Unlock()

	for e := bucket.Front(); e != nil; e = e.Next() {
		found := e.Value.(ID)
		if bytes.Equal(found.MyIdentity(), id) {
			return &found
		}
	}

	return nil
}

// GetPeers returns a randomly-ordered, unique list of all peers within the routing network (excluding itself).
func (t *RoutingTable) GetPeers() (peers []ID) {
	visited := make(map[string]struct{})
	visited[t.self.MyIdentityHex()] = struct{}{}

	for _, bucket := range t.buckets {
		bucket.mutex.RLock()

		for e := bucket.Front(); e != nil; e = e.Next() {
			id := e.Value.(ID)
			if _, seen := visited[id.MyIdentityHex()]; !seen {
				peers = append(peers, id)
				visited[id.MyIdentityHex()] = struct{}{}
			}
		}

		bucket.mutex.RUnlock()
	}

	return
}

// GetPeerAddresses returns a unique list of all peer addresses within the routing network.
func (t *RoutingTable) GetPeerAddresses() (peers []string) {
	visited := make(map[string]struct{})
	visited[t.self.MyIdentityHex()] = struct{}{}

	for _, bucket := range t.buckets {
		bucket.mutex.RLock()

		for e := bucket.Front(); e != nil; e = e.Next() {
			id := e.Value.(ID)
			if _, seen := visited[id.MyIdentityHex()]; !seen {
				peers = append(peers, id.Address)
				visited[id.MyIdentityHex()] = struct{}{}
			}
		}

		bucket.mutex.RUnlock()
	}

	return
}

// RemovePeer removes a peer from the routing table given the node ID with O(bucket_size) time complexity.
func (t *RoutingTable) RemovePeer(id []byte) bool {
	bucketID := prefixLen(xor(id, t.self.MyIdentity()))
	bucket := t.Bucket(bucketID)

	bucket.mutex.Lock()

	for e := bucket.Front(); e != nil; e = e.Next() {
		found := e.Value.(ID)
		if bytes.Equal(found.MyIdentity(), id) {
			bucket.Remove(e)

			bucket.mutex.Unlock()
			return true
		}
	}

	bucket.mutex.Unlock()

	return false
}

// PeerExists checks if a peer exists in the routing table with O(bucket_size) time complexity.
func (t *RoutingTable) PeerExists(target ID) bool {
	bucketID := prefixLen(xor(target.MyIdentity(), t.self.MyIdentity()))
	bucket := t.Bucket(bucketID)

	bucket.mutex.Lock()

	defer bucket.mutex.Unlock()

	for e := bucket.Front(); e != nil; e = e.Next() {
		id := e.Value.(ID)
		if bytes.Equal(id.MyIdentity(), target.MyIdentity()) {
			return true
		}
	}

	return false
}

// FindClosestPeers returns a list of k(count) peers with smallest XorID distance.
func (t *RoutingTable) FindClosestPeers(target ID, count int) (peers []ID) {
	if len(t.self.MyIdentity()) != len(target.MyIdentity()) {
		return []ID{}
	}

	bucketID := prefixLen(xor(target.MyIdentity(), t.self.MyIdentity()))
	bucket := t.Bucket(bucketID)

	bucket.mutex.RLock()

	for e := bucket.Front(); e != nil; e = e.Next() {
		peers = append(peers, e.Value.(ID))
	}

	bucket.mutex.RUnlock()

	for i := 1; len(peers) < count && (bucketID-i >= 0 || bucketID+i < len(t.self.MyIdentity())*8); i++ {
		if bucketID-i >= 0 {
			other := t.Bucket(bucketID - i)
			other.mutex.RLock()
			for e := other.Front(); e != nil; e = e.Next() {
				peers = append(peers, e.Value.(ID))
			}
			other.mutex.RUnlock()
		}

		if bucketID+i < len(t.self.MyIdentity())*8 {
			other := t.Bucket(bucketID + i)
			other.mutex.RLock()
			for e := other.Front(); e != nil; e = e.Next() {
				peers = append(peers, e.Value.(ID))
			}
			other.mutex.RUnlock()
		}
	}

	// Sort peers by XorID distance.
	sort.Slice(peers, func(i, j int) bool {
		left := xor(peers[i].MyIdentity(), target.MyIdentity())
		right := xor(peers[j].MyIdentity(), target.MyIdentity())
		return bytes.Compare(left, right) == -1
	})

	if len(peers) > count {
		peers = peers[:count]
	}

	return peers
}

// Bucket returns a specific Bucket by ID.
func (t *RoutingTable) Bucket(id int) *Bucket {
	if id >= 0 && id < len(t.buckets) {
		return t.buckets[id]
	}
	return nil
}
