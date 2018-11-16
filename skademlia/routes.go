package skademlia

import (
	"bytes"
	"container/list"
	"encoding/hex"
	"sort"
	"sync"

	"github.com/perlin-network/noise/crypto/blake2b"
)

// BucketSize defines the NodeID, Key, and routing table data structures.
const BucketSize = 16

// RoutingTable contains one bucket list for lookups.
type RoutingTable struct {
	// Current node's ID.
	self ID
	id   []byte

	publicKeyToNodeID sync.Map

	buckets []*Bucket
}

// ID is a public key and address pair
type ID struct {
	PublicKey []byte
	Address   string
}

// Bucket holds a list of peers of this node.
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
		buckets: make([]*Bucket, len(id.PublicKey)*8),
	}
	for i := 0; i < len(id.PublicKey)*8; i++ {
		table.buckets[i] = NewBucket()
	}
	nodeID := blake2b.New().HashBytes(id.PublicKey)
	table.publicKeyToNodeID.Store(id.PublicKey, nodeID)
	table.id = nodeID

	table.Update(id)

	return table
}

// Self returns the ID of the node hosting the current routing table instance.
func (t *RoutingTable) Self() ID {
	return t.self
}

// Update moves a peer to the front of a bucket in the routing table.
func (t *RoutingTable) Update(target ID) {
	var targetID []byte
	idInt, ok := t.publicKeyToNodeID.Load(target.PublicKey)
	if !ok {
		targetID = blake2b.New().HashBytes(target.PublicKey)
		t.publicKeyToNodeID.Store(target.PublicKey, targetID)
	} else {
		targetID = idInt.([]byte)
	}

	if len(t.id) != len(targetID) {
		return
	}

	bucketID := prefixLen(xor(targetID, t.id))
	bucket := t.Bucket(bucketID)

	var element *list.Element

	// Find current peer in bucket.
	bucket.mutex.Lock()

	for e := bucket.Front(); e != nil; e = e.Next() {
		id := e.Value.(ID)
		idInt, ok = t.publicKeyToNodeID.Load(id.PublicKey)
		nodeID := idInt.([]byte)
		if bytes.Equal(nodeID, targetID) {
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

// GetPeer retrieves the ID struct in the routing table given a peer ID if found.
func (t *RoutingTable) GetPeer(id []byte) (bool, *ID) {
	bucketID := prefixLen(xor(id, t.id))
	bucket := t.Bucket(bucketID)

	bucket.mutex.Lock()

	defer bucket.mutex.Unlock()

	for e := bucket.Front(); e != nil; e = e.Next() {
		found := e.Value.(ID)
		idInt, _ := t.publicKeyToNodeID.Load(found.PublicKey)
		nodeID := idInt.([]byte)
		if bytes.Equal(nodeID, id) {
			return true, &found
		}
	}

	return false, nil
}

// GetPeerFromPublicKey retrieves the ID struct in the routing table given a peer's public key if found.
func (t *RoutingTable) GetPeerFromPublicKey(publicKey []byte) (bool, *ID) {
	id := blake2b.New().HashBytes(publicKey)
	return t.GetPeer(id)
}

// GetPeers returns a randomly-ordered, unique list of all peers within the routing network (excluding itself).
func (t *RoutingTable) GetPeers() (peers []ID) {
	visited := make(map[string]struct{})
	visited[hex.EncodeToString(t.id)] = struct{}{}

	for _, bucket := range t.buckets {
		bucket.mutex.RLock()

		for e := bucket.Front(); e != nil; e = e.Next() {
			id := e.Value.(ID)
			idInt, _ := t.publicKeyToNodeID.Load(id.PublicKey)
			nodeID := idInt.([]byte)
			idHex := hex.EncodeToString(nodeID)
			if _, seen := visited[idHex]; !seen {
				peers = append(peers, id)
				visited[idHex] = struct{}{}
			}
		}

		bucket.mutex.RUnlock()
	}

	return
}

// GetPeerAddresses returns a unique list of all peer addresses within the routing network.
func (t *RoutingTable) GetPeerAddresses() (peers []string) {
	visited := make(map[string]struct{})
	visited[hex.EncodeToString(t.id)] = struct{}{}

	for _, bucket := range t.buckets {
		bucket.mutex.RLock()

		for e := bucket.Front(); e != nil; e = e.Next() {
			id := e.Value.(ID)
			idInt, _ := t.publicKeyToNodeID.Load(id.PublicKey)
			nodeID := idInt.([]byte)
			idHex := hex.EncodeToString(nodeID)
			if _, seen := visited[idHex]; !seen {
				peers = append(peers, id.Address)
				visited[idHex] = struct{}{}
			}
		}

		bucket.mutex.RUnlock()
	}

	return
}

// RemovePeer removes a peer from the routing table given the peer ID with O(bucket_size) time complexity.
func (t *RoutingTable) RemovePeer(id []byte) bool {
	bucketID := prefixLen(xor(id, t.id))
	bucket := t.Bucket(bucketID)

	bucket.mutex.Lock()

	for e := bucket.Front(); e != nil; e = e.Next() {
		found := e.Value.(ID)
		idInt, _ := t.publicKeyToNodeID.Load(found.PublicKey)
		nodeID := idInt.([]byte)
		if bytes.Equal(nodeID, id) {
			bucket.Remove(e)

			bucket.mutex.Unlock()
			return true
		}
	}

	bucket.mutex.Unlock()

	return false
}

// FindClosestPeers returns a list of k(count) peers with smallest XorID distance.
func (t *RoutingTable) FindClosestPeers(target ID, count int) (peers []ID) {
	var targetID []byte
	idInt, ok := t.publicKeyToNodeID.Load(target.PublicKey)
	if !ok {
		targetID = blake2b.New().HashBytes(target.PublicKey)
		t.publicKeyToNodeID.Store(target.PublicKey, targetID)
	} else {
		targetID = idInt.([]byte)
	}

	if len(t.id) != len(targetID) {
		return []ID{}
	}

	bucketID := prefixLen(xor(targetID, t.id))
	bucket := t.Bucket(bucketID)

	bucket.mutex.RLock()

	for e := bucket.Front(); e != nil; e = e.Next() {
		peers = append(peers, e.Value.(ID))
	}

	bucket.mutex.RUnlock()

	for i := 1; len(peers) < count && (bucketID-i >= 0 || bucketID+i < len(t.id)*8); i++ {
		if bucketID-i >= 0 {
			other := t.Bucket(bucketID - i)
			other.mutex.RLock()
			for e := other.Front(); e != nil; e = e.Next() {
				peers = append(peers, e.Value.(ID))
			}
			other.mutex.RUnlock()
		}

		if bucketID+i < len(t.id)*8 {
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
		left := xor(peers[i].id, targetID)
		right := xor(peers[j].id, targetID)
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
