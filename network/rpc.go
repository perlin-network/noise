package network

import (
	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"sync"
)

func BootstrapPeers(network *Network, table *dht.RoutingTable, target peer.ID, count int) (addresses []string, publicKeys [][]byte) {
	queue := []peer.ID{target}

	visited := make(map[string]struct{})
	visited[table.Self().Hex()] = struct{}{}
	visited[target.Hex()] = struct{}{}

	for len(queue) > 0 {
		var wait sync.WaitGroup
		wait.Add(len(queue))

		//responses := make(chan *protobuf.LookupNodeResponse, len(queue))

		// Queue up all work into worker pools for contacting peers.
		for _, popped := range queue {
			go func(peerId protobuf.ID) {
				defer wait.Done()

				client, err := network.Dial(peerId.Address)
				if err != nil {
					return
				}

				request := &protobuf.LookupNodeRequest{
					Target: &peerId,
				}

				err = network.Tell(client, request)

				if err != nil {
					return
				}

				// TODO: Create request/response RPC over gRPC.

				//if response, ok := response.(*messages.LookupNodeResponse); ok && response.Verify() {
				//	responses <- response
				//}
			}(protobuf.ID(popped))
		}
	}

	return
}
