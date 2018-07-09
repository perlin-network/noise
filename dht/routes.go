package dht

import (
	"container/list"
	"sort"
	"sync"

	"github.com/perlin-network/noise/peer"
)

// BucketSize defines the NodeID, Key, and routing table data structures.
const BucketSize = 16

// RoutingTable contains one bucket list for lookups
type RoutingTable struct {
	// Current node's ID.
	self peer.ID

	buckets []*Bucket
}

// Bucket holds a list of contacts of this node
type Bucket struct {
	*list.List
	mutex *sync.RWMutex
}

// NewBucket is a Factory method of Bucket, contains an empty list
func NewBucket() *Bucket {
	return &Bucket{
		List:  list.New(),
		mutex: &sync.RWMutex{},
	}
}

// CreateRoutingTable is a Factory method of RoutingTable
// , contains empty buckets
func CreateRoutingTable(id peer.ID) *RoutingTable {
	table := &RoutingTable{
		self:    id,
		buckets: make([]*Bucket, len(id.PublicKey)*8),
	}
	for i := 0; i < len(id.PublicKey)*8; i++ {
		table.buckets[i] = NewBucket()
	}

	table.Update(id)

	return table
}

// Self returns the ID of the node hosting the current routing table instance.
func (t *RoutingTable) Self() peer.ID {
	return t.self
}

// Update moves a peer to the front of a bucket int he routing table.
func (t *RoutingTable) Update(target peer.ID) {
	if len(t.self.PublicKey) != len(target.PublicKey) {
		return
	}

	bucketID := target.Xor(t.self).PrefixLen()
	bucket := t.Bucket(bucketID)

	var element *list.Element

	// Find current node in bucket.
	bucket.mutex.Lock()

	for e := bucket.Front(); e != nil; e = e.Next() {
		if e.Value.(peer.ID).Equals(target) {
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

// GetPeers returns an unique list of all peers within the routing network (excluding yourself).
func (t *RoutingTable) GetPeers() (peers []peer.ID) {
	visited := make(map[string]struct{})
	visited[t.self.PublicKeyHex()] = struct{}{}

	for _, bucket := range t.buckets {
		bucket.mutex.RLock()

		for e := bucket.Front(); e != nil; e = e.Next() {
			id := e.Value.(peer.ID)
			if _, seen := visited[id.PublicKeyHex()]; !seen {
				peers = append(peers, id)
				visited[id.PublicKeyHex()] = struct{}{}
			}
		}

		bucket.mutex.RUnlock()
	}

	return
}

// GetPeerAddresses returns an unique list of all peer addresses within the routing network.
func (t *RoutingTable) GetPeerAddresses() (peers []string) {
	visited := make(map[string]struct{})
	visited[t.self.PublicKeyHex()] = struct{}{}

	for _, bucket := range t.buckets {
		bucket.mutex.RLock()

		for e := bucket.Front(); e != nil; e = e.Next() {
			id := e.Value.(peer.ID)
			if _, seen := visited[id.PublicKeyHex()]; !seen {
				peers = append(peers, id.Address)
				visited[id.PublicKeyHex()] = struct{}{}
			}
		}

		bucket.mutex.RUnlock()
	}

	return
}

// RemovePeer removes a peer from the routing table. O(bucket_size).
func (t *RoutingTable) RemovePeer(target peer.ID) bool {
	bucketID := target.Xor(t.self).PrefixLen()
	bucket := t.Bucket(bucketID)

	bucket.mutex.Lock()

	for e := bucket.Front(); e != nil; e = e.Next() {
		if e.Value.(peer.ID).Equals(target) {
			bucket.Remove(e)

			bucket.mutex.Unlock()
			return true
		}
	}

	bucket.mutex.Unlock()

	return false
}

// PeerExists check if a peer exists in the routing table. O(bucket_size).
func (t *RoutingTable) PeerExists(target peer.ID) bool {
	bucketID := target.Xor(t.self).PrefixLen()
	bucket := t.Bucket(bucketID)

	bucket.mutex.Lock()

	defer bucket.mutex.Unlock()

	for e := bucket.Front(); e != nil; e = e.Next() {
		if e.Value.(peer.ID).Equals(target) {
			return true
		}
	}

	return false
}

// FindClosestPeers returns a list of k(count param) peers with smallest XOR distance
func (t *RoutingTable) FindClosestPeers(target peer.ID, count int) (peers []peer.ID) {
	if len(t.self.PublicKey) != len(target.PublicKey) {
		return []peer.ID{}
	}

	bucketID := target.Xor(t.self).PrefixLen()
	bucket := t.Bucket(bucketID)

	bucket.mutex.RLock()

	for e := bucket.Front(); e != nil; e = e.Next() {
		peers = append(peers, e.Value.(peer.ID))
	}

	bucket.mutex.RUnlock()

	for i := 1; len(peers) < count && (bucketID-i >= 0 || bucketID+i < len(t.self.PublicKey)*8); i++ {
		if bucketID-i >= 0 {
			other := t.Bucket(bucketID - i)
			other.mutex.RLock()
			for e := other.Front(); e != nil; e = e.Next() {
				peers = append(peers, e.Value.(peer.ID))
			}
			other.mutex.RUnlock()
		}

		if bucketID+i < len(t.self.PublicKey)*8 {
			other := t.Bucket(bucketID + i)
			other.mutex.RLock()
			for e := other.Front(); e != nil; e = e.Next() {
				peers = append(peers, e.Value.(peer.ID))
			}
			other.mutex.RUnlock()
		}
	}

	// Sort peers by XOR distance.
	sort.Slice(peers, func(i, j int) bool {
		left := peers[i].Xor(target)
		right := peers[j].Xor(target)
		return left.Less(right)
	})

	if len(peers) > count {
		peers = peers[:count]
	}

	return peers
}

// Bucket returns a specific Bucket by id
func (t *RoutingTable) Bucket(id int) *Bucket {
	if id >= 0 && id < len(t.buckets) {
		return t.buckets[id]
	}
	return nil
}
