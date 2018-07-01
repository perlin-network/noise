package discovery

import (
	"time"

	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/rpc"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"sort"
)

func queryPeerByID(net *network.Network, peerID peer.ID, targetID peer.ID, responses chan []*protobuf.ID) {
	client, err := net.Dial(peerID.Address)
	if err != nil {
		responses <- []*protobuf.ID{}
		return
	}

	targetProtoID := protobuf.ID(targetID)

	request := new(rpc.Request)
	request.SetMessage(&protobuf.LookupNodeRequest{Target: &targetProtoID})
	request.SetTimeout(3 * time.Second)

	response, err := client.Request(request)

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

// FindNode queries all peers this current node acknowledges for the closest peers
// to a specified target ID. Queries at most #ALPHA nodes at a time, and returns
// a bucket filled with the closest peers.
func FindNode(net *network.Network, targetID peer.ID, alpha int) (results []peer.ID) {
	plugin, exists := net.Plugin("discovery")

	// Discovery plugin was not registered. Fail.
	if !exists {
		return
	}

	var queue []peer.ID

	responses, visited := make(chan []*protobuf.ID), make(map[string]struct{})

	// Start searching for target from #ALPHA peers closest to target by queuing
	// them up and marking them as visited.
	for _, peerID := range plugin.(*Plugin).Routes.FindClosestPeers(targetID, alpha) {
		visited[peerID.PublicKeyHex()] = struct{}{}
		queue = append(queue, peerID)

		results = append(results, peerID)
	}

	pending := 0

	// Go through every peer in the entire queue and queue up what peers believe
	// is closest to a target ID.
	for ; pending < alpha && len(queue) > 0; pending++ {
		go queryPeerByID(net, queue[0], targetID, responses)

		results = append(results, queue[0])
		queue = queue[1:]
	}

	// Empty queue.
	queue = []peer.ID{}

	// Asynchronous breadth-first search.
	for pending > 0 {
		response := <-responses

		pending--

		// Expand responses containing a peer's belief on the closest peers to target ID.
		for _, id := range response {
			peerID := peer.ID(*id)

			if _, seen := visited[peerID.PublicKeyHex()]; !seen {
				// Append new peer to be queued by the routing table.
				results = append(results, peerID)

				queue = append(queue, peerID)
				visited[peerID.PublicKeyHex()] = struct{}{}
			}
		}

		// Queue and request for #ALPHA closest peers to target ID from expanded results.
		for ; pending < alpha && len(queue) > 0; pending++ {
			go queryPeerByID(net, queue[0], targetID, responses)
			queue = queue[1:]
		}

		// Empty queue.
		queue = []peer.ID{}
	}

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
