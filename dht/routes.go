package dht

import (
	"container/list"
	"github.com/perlin-network/noise/peer"
	"sort"
)

const BucketSize = 20

type RoutingTable struct {
	// Current node's ID.
	self peer.ID

	buckets [peer.IdSize * 8]*list.List
}

func CreateRoutingTable(id peer.ID) *RoutingTable {
	table := &RoutingTable{self: id}
	for i := 0; i < peer.IdSize*8; i++ {
		table.buckets[i] = list.New()
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
	bucket := t.buckets[bucketId]

	var element *list.Element

	// Find current node in bucket.
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

		// TODO: Remove nodes that don't respond to a ping.
	} else {
		bucket.MoveToFront(element)
	}
}

// Returns an unique list of all peers within the routing network (excluding yourself).
func (t *RoutingTable) GetPeers() (peers []peer.ID) {
	visited := make(map[string]struct{})
	visited[t.self.Hex()] = struct{}{}

	for _, bucket := range t.buckets {
		for e := bucket.Front(); e != nil; e = e.Next() {
			id := e.Value.(peer.ID)
			if _, seen := visited[id.Hex()]; !seen {
				peers = append(peers, id)
				visited[id.Hex()] = struct{}{}
			}
		}
	}

	return
}

// Returns an unique list of all peer addresses within the routing network.
func (t *RoutingTable) GetPeerAddresses() (peers []string) {
	visited := make(map[string]struct{})
	visited[t.self.Hex()] = struct{}{}

	for _, bucket := range t.buckets {
		for e := bucket.Front(); e != nil; e = e.Next() {
			id := e.Value.(peer.ID)
			if _, seen := visited[id.Hex()]; !seen {
				peers = append(peers, id.Address)
				visited[id.Hex()] = struct{}{}
			}
		}
	}

	return
}

// Removes a peer from the routing table. O(bucket_size).
func (t *RoutingTable) RemovePeer(target peer.ID) {
	bucketId := target.Xor(t.self).PrefixLen()

	for e := t.buckets[bucketId].Front(); e != nil; e = e.Next() {
		if e.Value.(peer.ID).Equals(target) {
			t.buckets[bucketId].Remove(e)
		}
	}
}

// Determines if a peer exists in the routing table. O(bucket_size).
func (t *RoutingTable) PeerExists(target peer.ID) bool {
	bucketId := target.Xor(t.self).PrefixLen()

	for e := t.buckets[bucketId].Front(); e != nil; e = e.Next() {
		if e.Value.(peer.ID).Equals(target) {
			return true
		}
	}

	return false
}

func (t *RoutingTable) FindClosestPeers(target peer.ID, count int) (peers []peer.ID) {
	bucketId := target.Xor(t.self).PrefixLen()

	for e := t.buckets[bucketId].Front(); e != nil; e = e.Next() {
		peers = append(peers, e.Value.(peer.ID))
	}

	for i := 1; len(peers) < count && (bucketId-i >= 0 || bucketId+i < peer.IdSize*8); i++ {
		if bucketId-i >= 0 {
			for e := t.buckets[bucketId-i].Front(); e != nil; e = e.Next() {
				peers = append(peers, e.Value.(peer.ID))
			}
		}

		if bucketId+i < peer.IdSize*8 {
			for e := t.buckets[bucketId+i].Front(); e != nil; e = e.Next() {
				peers = append(peers, e.Value.(peer.ID))
			}
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
