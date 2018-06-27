package discovery

import (
	"sync"

	"time"

	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/rpc"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
)

func bootstrapPeers(net *network.Network, target peer.ID, count int) (addresses []string, publicKeys [][]byte) {
	queue := []peer.ID{target}

	visited := make(map[string]struct{})
	visited[net.Keys.PublicKeyHex()] = struct{}{}
	visited[target.PublicKeyHex()] = struct{}{}

	for len(queue) > 0 {
		var wait sync.WaitGroup
		wait.Add(len(queue))

		responses := make(chan *protobuf.LookupNodeResponse, len(queue))

		// Queue up all work into worker pools for contacting peers.
		for _, popped := range queue {
			go func(peerId peer.ID) {
				defer wait.Done()

				client, err := net.Dial(peerId.Address)
				if err != nil {
					return
				}

				protoID := protobuf.ID(peerId)

				request := new(rpc.Request)
				request.SetMessage(&protobuf.LookupNodeRequest{Target: &protoID})
				request.SetTimeout(3 * time.Second)

				response, err := client.Request(request)

				if err != nil {
					client.Redial()
					return
				}

				if response, ok := response.(*protobuf.LookupNodeResponse); ok {
					responses <- response
				}
			}(popped)
		}

		// Empty the queue.
		queue = []peer.ID{}

		// Wait until all responses from peers come back.
		wait.Wait()

		// Expand nodes in breadth-first search.
		close(responses)
		for response := range responses {
			// Queue up expanded nodes.
			for _, id := range response.Peers {
				p := peer.ID(*id)

				if _, seen := visited[p.PublicKeyHex()]; !seen && p.Address != net.Address() {

					queue = append(queue, p)
					visited[p.PublicKeyHex()] = struct{}{}

					addresses = append(addresses, p.Address)

					publicKey := make([]byte, peer.IdSize)
					copy(publicKey, p.PublicKey[:])

					publicKeys = append(publicKeys, publicKey)
				}
			}
		}
	}

	return
}
