package discovery

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/internal/protobuf"
	"github.com/perlin-network/noise/peer"
)

const (
	reqTimeoutInSec = 3
)

func queryPeerByID(reqAdapter RequestAdapter, peerID peer.ID, targetID peer.ID, responses chan []*protobuf.ID) {
	targetProtoID := protobuf.ID(targetID)

	content := &protobuf.LookupNodeRequest{Target: &targetProtoID}

	msg, err := toProtobufMessage(opCodeLookupRequest, content)
	if err != nil {
		responses <- []*protobuf.ID{}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), reqTimeoutInSec*time.Second)
	defer cancel()

	response, err := reqAdapter.Request(ctx, peerID, msg)
	if err != nil {
		responses <- []*protobuf.ID{}
		return
	}

	pm, err := fromProtobufMessage(response)
	if err != nil {
		responses <- []*protobuf.ID{}
		return
	}

	if resp, ok := pm.(*protobuf.LookupNodeResponse); ok {
		responses <- resp.Peers
	} else {
		responses <- []*protobuf.ID{}
	}
}

type lookupBucket struct {
	pending int
	queue   []peer.ID
}

func (lookup *lookupBucket) performLookup(reqAdapter RequestAdapter, targetID peer.ID, alpha int, visited *sync.Map) (results []peer.ID) {
	responses := make(chan []*protobuf.ID)

	// Go through every peer in the entire queue and queue up what peers believe
	// is closest to a target ID.

	for ; lookup.pending < alpha && len(lookup.queue) > 0; lookup.pending++ {
		go queryPeerByID(reqAdapter, lookup.queue[0], targetID, responses)

		results = append(results, lookup.queue[0])
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
			peerID := peer.ID(*id)

			if _, seen := visited.LoadOrStore(peerID.PublicKeyHex(), struct{}{}); !seen {
				// Append new peer to be queued by the routing table.
				results = append(results, peerID)
				lookup.queue = append(lookup.queue, peerID)
			}
		}

		// Queue and request for #ALPHA closest peers to target ID from expanded results.
		for ; lookup.pending < alpha && len(lookup.queue) > 0; lookup.pending++ {
			go queryPeerByID(reqAdapter, lookup.queue[0], targetID, responses)
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
func FindNode(rt *dht.RoutingTable, reqAdapter RequestAdapter, targetID peer.ID, alpha int, disjointPaths int) (results []peer.ID) {

	visited := new(sync.Map)

	var lookups []*lookupBucket

	// Start searching for target from #ALPHA peers closest to target by queuing
	// them up and marking them as visited.
	for i, peerID := range rt.FindClosestPeers(targetID, alpha) {
		visited.Store(peerID.PublicKeyHex(), struct{}{})

		if len(lookups) < disjointPaths {
			lookups = append(lookups, new(lookupBucket))
		}

		lookup := lookups[i%disjointPaths]
		lookup.queue = append(lookup.queue, peerID)

		results = append(results, peerID)
	}

	wait, mutex := &sync.WaitGroup{}, &sync.Mutex{}

	for _, lookup := range lookups {
		go func(lookup *lookupBucket) {
			mutex.Lock()
			results = append(results, lookup.performLookup(reqAdapter, targetID, alpha, visited)...)
			mutex.Unlock()

			wait.Done()
		}(lookup)

		wait.Add(1)
	}

	// Wait until all #D parallel lookups have been completed.
	wait.Wait()

	// Sort resulting peers by XOR distance.
	sort.Slice(results, func(i, j int) bool {
		left := results[i].Xor(targetID)
		right := results[j].Xor(targetID)
		return left.Less(right)
	})

	// Cut off list of results to only have the routing table focus on the
	// #dht.BucketSize closest peers to the current node.
	if len(results) > dht.BucketSize {
		results = results[:dht.BucketSize]
	}

	return
}
