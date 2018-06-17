package dht

import (
	"container/list"
	"github.com/perlin-network/noise/peer"
	"sort"
	"sync"
)

const BucketSize = 20

type RoutingTable struct {
	// Current node's ID.
	self peer.ID

	buckets *sync.Map
}

type Bucket struct {
	*list.List
	mutex *sync.RWMutex
}

func NewBucket() *Bucket {
	return &Bucket{
		List:  list.New(),
		mutex: &sync.RWMutex{},
	}
}

func CreateRoutingTable(id peer.ID) *RoutingTable {
	table := &RoutingTable{self: id, buckets: &sync.Map{}}
	for i := 0; i < peer.IdSize*8; i++ {
		table.buckets.Store(i, NewBucket())
	}

	table.Update(id)

	return table
}

// Returns the ID of the node hosting the current routing table instance.
func (t *RoutingTable) Self() peer.ID {
	return t.self
}

// Moves a peer to the front of a bucket int he routing table.
func (t *RoutingTable) Update(target peer.ID) {
	bucketId := target.Xor(t.self).PrefixLen()
	bucket := t.Bucket(bucketId)

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

// Returns an unique list of all peers within the routing network (excluding yourself).
func (t *RoutingTable) GetPeers() (peers []peer.ID) {
	visited := make(map[string]struct{})
	visited[t.self.Hex()] = struct{}{}

	t.buckets.Range(func(key, value interface{}) bool {
		bucket := value.(*Bucket)

		bucket.mutex.RLock()

		for e := bucket.Front(); e != nil; e = e.Next() {
			id := e.Value.(peer.ID)
			if _, seen := visited[id.Hex()]; !seen {
				peers = append(peers, id)
				visited[id.Hex()] = struct{}{}
			}
		}

		bucket.mutex.RUnlock()
		return true
	})

	return
}

// Returns an unique list of all peer addresses within the routing network.
func (t *RoutingTable) GetPeerAddresses() (peers []string) {
	visited := make(map[string]struct{})
	visited[t.self.Hex()] = struct{}{}

	t.buckets.Range(func(key, value interface{}) bool {
		bucket := value.(*Bucket)

		bucket.mutex.RLock()

		for e := bucket.Front(); e != nil; e = e.Next() {
			id := e.Value.(peer.ID)
			if _, seen := visited[id.Hex()]; !seen {
				peers = append(peers, id.Address)
				visited[id.Hex()] = struct{}{}
			}
		}

		bucket.mutex.RUnlock()
		return true
	})

	return
}

// Removes a peer from the routing table. O(bucket_size).
func (t *RoutingTable) RemovePeer(target peer.ID) {
	bucketId := target.Xor(t.self).PrefixLen()
	bucket := t.Bucket(bucketId)

	bucket.mutex.Lock()

	for e := bucket.Front(); e != nil; e = e.Next() {
		if e.Value.(peer.ID).Equals(target) {
			bucket.Remove(e)
		}
	}

	bucket.mutex.Unlock()
}

// Determines if a peer exists in the routing table. O(bucket_size).
func (t *RoutingTable) PeerExists(target peer.ID) bool {
	bucketId := target.Xor(t.self).PrefixLen()
	bucket := t.Bucket(bucketId)

	bucket.mutex.Lock()

	for e := bucket.Front(); e != nil; e = e.Next() {
		if e.Value.(peer.ID).Equals(target) {
			return true
		}
	}

	bucket.mutex.Unlock()

	return false
}

func (t *RoutingTable) FindClosestPeers(target peer.ID, count int) (peers []peer.ID) {
	bucketId := target.Xor(t.self).PrefixLen()
	bucket := t.Bucket(bucketId)

	bucket.mutex.RLock()

	for e := bucket.Front(); e != nil; e = e.Next() {
		peers = append(peers, e.Value.(peer.ID))
	}

	for i := 1; len(peers) < count && (bucketId-i >= 0 || bucketId+i < peer.IdSize*8); i++ {
		if bucketId-i >= 0 {
			for e := t.Bucket(bucketId - i).Front(); e != nil; e = e.Next() {
				peers = append(peers, e.Value.(peer.ID))
			}
		}

		if bucketId+i < peer.IdSize*8 {
			for e := t.Bucket(bucketId + i).Front(); e != nil; e = e.Next() {
				peers = append(peers, e.Value.(peer.ID))
			}
		}
	}

	bucket.mutex.RUnlock()

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

func (t *RoutingTable) Bucket(id int) *Bucket {
	if bucket, exists := t.buckets.Load(id); exists {
		if bucket, ok := bucket.(*Bucket); ok {
			return bucket
		}
	}
	return nil
}
