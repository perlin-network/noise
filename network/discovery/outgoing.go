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

func queryPeerById(net *network.Network, peerId peer.ID, targetId peer.ID, responses chan []*protobuf.ID) {
	client, err := net.Dial(peerId.Address)
	if err != nil {
		responses <- []*protobuf.ID{}
		return
	}

	targetProtoId := protobuf.ID(targetId)

	request := new(rpc.Request)
	request.SetMessage(&protobuf.LookupNodeRequest{Target: &targetProtoId})
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

func findNode(net *network.Network, targetId peer.ID, alpha int) (results []peer.ID) {
	var queue []peer.ID

	responses, visited := make(chan []*protobuf.ID), make(map[string]struct{})

	// Start searching for target from #ALPHA peers closest to target by queuing
	// them up and marking them as visited.
	for _, peerId := range net.Routes.FindClosestPeers(targetId, alpha) {
		visited[peerId.PublicKeyHex()] = struct{}{}
		queue = append(queue, peerId)

		results = append(results, peerId)
	}

	pending := 0

	// Go through every peer in the entire queue and queue up what peers believe
	// is closest to a target ID.
	for ; pending < alpha && len(queue) > 0; pending++ {
		go queryPeerById(net, queue[0], targetId, responses)

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
			peerId := peer.ID(*id)

			if _, seen := visited[peerId.PublicKeyHex()]; !seen {
				// Append new peer to be queued by the routing table.
				results = append(results, peerId)

				queue = append(queue, peerId)
				visited[peerId.PublicKeyHex()] = struct{}{}
			}
		}

		// Queue and request for #ALPHA closest peers to target ID from expanded results.
		for ; pending < alpha && len(queue) > 0; pending++ {
			go queryPeerById(net, queue[0], targetId, responses)
			queue = queue[1:]
		}

		// Empty queue.
		queue = []peer.ID{}
	}

	// Sort resulting peers by XOR distance.
	sort.Slice(results, func(i, j int) bool {
		left := results[i].Xor(targetId)
		right := results[j].Xor(targetId)
		return left.Less(right)
	})

	// Cut off list of results to only have the routing table focus on the
	// #dht.BucketSize closest peers to the current node.
	if len(results) > dht.BucketSize {
		results = results[:dht.BucketSize]
	}

	return
}
