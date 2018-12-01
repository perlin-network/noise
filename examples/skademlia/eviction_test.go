package skademlia_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/skademlia"
	"github.com/perlin-network/noise/utils"

	"github.com/stretchr/testify/assert"
)

func generateBucketIDs(id *skademlia.IdentityAdapter, n int) []*skademlia.IdentityAdapter {
	self := skademlia.NewID(id.MyIdentity(), "")
	rt := skademlia.NewRoutingTable(self)

	var ids []*skademlia.IdentityAdapter

	for len(ids) < n {
		id := skademlia.NewIdentityAdapter(8, 8)
		if rt.GetBucketID(skademlia.NewID(id.MyIdentity(), "").Id) == 4 {
			ids = append(ids, id)
		}
	}
	return ids
}

func TestSKademliaEviction(t *testing.T) {
	self := skademlia.NewIdentityAdapter(8, 8)
	ids := []*skademlia.IdentityAdapter{self}
	// create 5 peers, last peer should not be in table
	peers := generateBucketIDs(self, 5)
	ids = append(ids, peers...)
	// make max bucket size 4
	nodes, msgServices, discoveryServices, ports := makeNodesFromIDs(ids, 4)

	node0ID := nodes[0].GetIdentityAdapter().MyIdentity()
	rt := discoveryServices[0].Routes

	// Connect other nodes to node 0
	for i := 1; i < len(nodes); i++ {
		if i == 0 {
			// skip node 0
			continue
		}
		assert.Nil(t, nodes[i].GetConnectionAdapter().AddPeerID(node0ID, fmt.Sprintf("%s:%d", host, ports[0])))
	}

	// being discovery process to connect nodes to each other
	for _, d := range discoveryServices {
		assert.Nil(t, d.Bootstrap())
	}

	// make sure nodes are connected
	time.Sleep(250 * time.Duration(len(nodes)) * time.Millisecond)

	skademliaID := skademlia.NewID(ids[1].MyIdentity(), "")
	expectedBucketID := rt.GetBucketID(skademliaID.Id)
	for i := 2; i < len(nodes); i++ {
		skademliaID = skademlia.NewID(ids[i].MyIdentity(), "")
		bucketID := rt.GetBucketID(skademliaID.Id)
		assert.Equalf(t, expectedBucketID, bucketID, "expected bucket ID to be %d, got %d", expectedBucketID, bucketID)
		fmt.Printf("bucket id: %d\n", bucketID)
	}

	// assert broadcasts goes to everyone
	for i := 0; i < len(nodes); i++ {
		expected := fmt.Sprintf("This is a broadcasted message from Node %d.", i)
		assert.Nil(t, nodes[i].Broadcast(context.Background(), &protocol.MessageBody{
			Service: serviceID,
			Payload: ([]byte)(expected),
		}))

		// Check if message was received by other nodes.
		for j := 0; j < len(msgServices); j++ {
			if i == j {
				continue
			}
			select {
			case received := <-msgServices[j].Mailbox:
				assert.Equalf(t, expected, received, "Expected message '%s' to be received by node %d but got '%v'", expected, j, received)
			case <-time.After(2 * time.Second):
				assert.Failf(t, "Timed out attempting to receive message", "from Node %d for Node %d", i, j)
			}
		}
	}

	expectedLen := 4
	bucket := discoveryServices[0].Routes.Bucket(expectedBucketID)
	assert.Equalf(t, expectedLen, bucket.Len(), "expected bucket size to be %d, got %d", expectedLen, bucket.Len())
}

func makeNodesFromIDs(ids []*skademlia.IdentityAdapter, bucketSize int) ([]*protocol.Node, []*MsgService, []*skademlia.Service, []int) {
	var nodes []*protocol.Node
	var msgServices []*MsgService
	var discoveryServices []*skademlia.Service
	var ports []int

	// setup all the nodes
	for i := 0; i < len(ids); i++ {
		idAdapter := ids[i]

		port := utils.GetRandomUnusedPort()
		address := fmt.Sprintf("%s:%d", host, port)
		listener, err := net.Listen("tcp", address)
		if err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		id := skademlia.NewID(idAdapter.MyIdentity(), address)

		connAdapter, err := skademlia.NewConnectionAdapter(
			listener,
			dialTCP,
			id,
		)
		if err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		node := protocol.NewNode(
			protocol.NewController(),
			connAdapter,
			idAdapter,
		)
		node.SetCustomHandshakeProcessor(skademlia.NewHandshakeProcessor(idAdapter))

		skSvc := skademlia.NewService(node, id)
		rt := skademlia.NewRoutingTableWithOptions(id, skademlia.WithBucketSize(bucketSize))
		skSvc.Routes = rt
		connAdapter.SetSKademliaService(skSvc)

		msgSvc := &MsgService{
			Mailbox: make(chan string, 1),
		}

		node.AddService(msgSvc)
		node.AddService(skSvc)

		node.Start()

		nodes = append(nodes, node)
		msgServices = append(msgServices, msgSvc)
		discoveryServices = append(discoveryServices, skSvc)
		ports = append(ports, port)
	}

	return nodes, msgServices, discoveryServices, ports
}
