package discovery

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/internal/protobuf"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/peer"
)

func queryPeerByID(net *network.Network, peerID peer.ID, targetID peer.ID, responses chan []*protobuf.ID) {
	client, err := net.Client(peerID.Address)
	if err != nil {
		responses <- []*protobuf.ID{}
		return
	}

	targetProtoID := protobuf.ID(targetID)

	msg := &protobuf.LookupNodeRequest{Target: &targetProtoID}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	response, err := client.Request(ctx, msg)

	if err != nil {
		responses <- []*protobuf.ID{}
		return
	}

	if response, ok := response.(*protobuf.LookupNodeResponse); ok {
		responses <- response.Peers
	} else {
		responses <- []*protobuf.ID{}
	}
}

type lookupBucket struct {
	pending int
	queue   []peer.ID
}

func (lookup *lookupBucket) performLookup(net *network.Network, targetID peer.ID, alpha int, visited *sync.Map) (results []peer.ID) {
	responses := make(chan []*protobuf.ID)

	// Go through every peer in the entire queue and queue up what peers believe
	// is closest to a target ID.

	for ; lookup.pending < alpha && len(lookup.queue) > 0; lookup.pending++ {
		go queryPeerByID(net, lookup.queue[0], targetID, responses)

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
			go queryPeerByID(net, lookup.queue[0], targetID, responses)
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
func FindNode(net *network.Network, targetID peer.ID, alpha int, disjointPaths int) (results []peer.ID) {
	plugin, exists := net.Plugin(PluginID)

	// Discovery plugin was not registered. Fail.
	if !exists {
		return
	}

	visited := new(sync.Map)

	var lookups []*lookupBucket

	// Start searching for target from #ALPHA peers closest to target by queuing
	// them up and marking them as visited.
	for i, peerID := range plugin.(*Plugin).Routes.FindClosestPeers(targetID, alpha) {
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
			results = append(results, lookup.performLookup(net, targetID, alpha, visited)...)
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
