package skademlia

import (
	"bytes"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/protocol"
	"sort"
	"sync"
	"time"
)

func Broadcast(node *noise.Node, opcode noise.Opcode, message noise.Message) error {
	for _, peerID := range FindClosestPeers(Table(node), protocol.NodeID(node).Hash(), DefaultBucketSize) {
		peer := protocol.Peer(node, peerID)

		if peer == nil {
			continue
		}

		if err := peer.SendMessage(opcode, message); err != nil {
			return err
		}
	}

	return nil
}

func queryPeerByID(node *noise.Node, peerID, targetID ID, responses chan []ID) {
	var err error

	if peerID.Equals(protocol.NodeID(node)) {
		responses <- []ID{}
		return
	}

	peer := protocol.Peer(node, peerID)

	if peer == nil {
		peer, err = node.Dial(peerID.address)

		if err != nil {
			responses <- []ID{}
			return
		}
	}

	// Send lookup request.
	err = peer.SendMessage(OpcodeLookupRequest, LookupRequest(targetID))
	if err != nil {
		responses <- []ID{}
		return
	}

	// Handle lookup response.
	for {
		select {
		case msg := <-peer.Receive(OpcodeLookupResponse):
			responses <- msg.(LookupResponse).peers
		case <-time.After(3 * time.Second):
			responses <- []ID{}
			return
		}
	}
}

type lookupBucket struct {
	pending int
	queue   []ID
}

func (lookup *lookupBucket) performLookup(node *noise.Node, table *table, targetID ID, alpha int, visited *sync.Map) (results []ID) {
	responses := make(chan []ID)

	// Go through every peer in the entire queue and queue up what peers believe
	// is closest to a target ID.

	for ; lookup.pending < alpha && len(lookup.queue) > 0; lookup.pending++ {
		go queryPeerByID(node, lookup.queue[0], targetID, responses)

		lookup.queue = lookup.queue[1:]
	}

	// Empty queue.
	lookup.queue = lookup.queue[:0]

	// Asynchronous breadth-first search.
	for lookup.pending > 0 {
		response := <-responses

		lookup.pending--

		// Expand responses containing a peer's belief on the closest peers to target ID.
		for _, id := range response {
			if _, seen := visited.LoadOrStore(string(id.Hash()), struct{}{}); !seen {
				// Append new peer to be queued by the routing table.
				results = append(results, id)
				lookup.queue = append(lookup.queue, id)
			}
		}

		// Queue and request for #ALPHA closest peers to target ID from expanded results.
		for ; lookup.pending < alpha && len(lookup.queue) > 0; lookup.pending++ {
			go queryPeerByID(node, lookup.queue[0], targetID, responses)
			lookup.queue = lookup.queue[1:]
		}

		// Empty queue.
		lookup.queue = lookup.queue[:0]
	}

	return
}

// FindNode queries all peers this current node acknowledges for the closest peers
// to a specified target ID.
//
// All lookups are done under a number of disjoint lookups in parallel.
//
// Queries at most #ALPHA nodes at a time per lookup, and returns all peer IDs closest to a target peer ID.
func FindNode(node *noise.Node, targetID ID, alpha int, disjointPaths int) (results []ID) {
	table, visited := Table(node), new(sync.Map)

	visited.Store(string(protocol.NodeID(node).Hash()), struct{}{})
	visited.Store(string(targetID.Hash()), struct{}{})

	var lookups []*lookupBucket

	// Start searching for target from #ALPHA peers closest to target by queuing
	// them up and marking them as visited.
	for i, peerID := range FindClosestPeers(table, targetID.Hash(), alpha) {
		visited.Store(string(peerID.Hash()), struct{}{})

		if len(lookups) < disjointPaths {
			lookups = append(lookups, new(lookupBucket))
		}

		lookup := lookups[i%disjointPaths]
		lookup.queue = append(lookup.queue, peerID.(ID))

		results = append(results, peerID.(ID))
	}

	wait, mutex := new(sync.WaitGroup), new(sync.Mutex)

	for _, lookup := range lookups {
		go func(lookup *lookupBucket) {
			mutex.Lock()
			results = append(results, lookup.performLookup(node, table, targetID, alpha, visited)...)
			mutex.Unlock()

			wait.Done()
		}(lookup)

		wait.Add(1)
	}

	// Wait until all #D parallel lookups have been completed.
	wait.Wait()

	// Sort resulting peers by XOR distance.
	sort.Slice(results, func(i, j int) bool {
		return bytes.Compare(xor(results[i].Hash(), targetID.Hash()), xor(results[j].Hash(), targetID.Hash())) == -1
	})

	// Cut off list of results to only have the routing table focus on the
	// #BucketSize closest peers to the current node.
	if len(results) > DefaultBucketSize {
		results = results[:DefaultBucketSize]
	}

	return
}
