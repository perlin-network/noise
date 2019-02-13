package skademlia

import (
	"bytes"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/protocol"
	"sort"
	"sync"
	"time"
)

// Broadcast sends a message denoted by its opcode and content to all S/Kademlia IDs
// closest in terms of XOR distance to that of a specified node instances ID.
//
// It returns a list of errors which have occurred in sending any messages to peers
// closest to a given node instance.
func Broadcast(node *noise.Node, message noise.Message) (errs []error) {
	for _, peerID := range FindClosestPeers(Table(node), protocol.NodeID(node).Hash(), BucketSize()) {
		peer := protocol.Peer(node, peerID)

		if peer == nil {
			continue
		}

		if err := peer.SendMessage(message); err != nil {
			errs = append(errs, err)
		}
	}

	return
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

	opcodeLookupResponse, err := noise.OpcodeFromMessage((*LookupResponse)(nil))
	if err != nil {
		panic("skademlia: response opcode not registered")
	}

	// Send lookup request.
	err = peer.SendMessage(LookupRequest{targetID})
	if err != nil {
		responses <- []ID{}
		return
	}

	// Handle lookup response.
	for {
		select {
		case msg := <-peer.Receive(opcodeLookupResponse):
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

// FindNode implements the `FIND_NODE` RPC method denoted in
// Section 4.4 of the S/Kademlia paper: `Lookup over disjoint paths`.
//
// Given a node instance N, and a S/Kademlia ID as a target T, α disjoint lookups
// take place in parallel over all closest peers to N to target T, with at most D
// lookups happening at once.
//
// Each disjoint lookup queries at most α peers.
//
// It returns at most BUCKET_SIZE S/Kademlia peer IDs closest to that of a
// specifiedtarget T.
func FindNode(node *noise.Node, targetID ID, alpha int, numDisjointPaths int) (results []ID) {
	table, visited := Table(node), new(sync.Map)

	visited.Store(string(protocol.NodeID(node).Hash()), struct{}{})
	visited.Store(string(targetID.Hash()), struct{}{})

	var lookups []*lookupBucket

	// Start searching for target from α peers closest to T by queuing
	// them up and marking them as visited.
	for i, peerID := range FindClosestPeers(table, targetID.Hash(), alpha) {
		visited.Store(string(peerID.Hash()), struct{}{})

		if len(lookups) < numDisjointPaths {
			lookups = append(lookups, new(lookupBucket))
		}

		lookup := lookups[i%numDisjointPaths]
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

	// Wait until all D parallel lookups have been completed.
	wait.Wait()

	// Sort resulting peers by XOR distance.
	sort.Slice(results, func(i, j int) bool {
		return bytes.Compare(xor(results[i].Hash(), targetID.Hash()), xor(results[j].Hash(), targetID.Hash())) == -1
	})

	// Cut off list of results to only have the routing table focus on the
	// BUCKET_SIZE closest peers to the current node.
	if len(results) > BucketSize() {
		results = results[:BucketSize()]
	}

	return
}
