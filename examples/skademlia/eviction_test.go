package skademlia_test

import (
	"context"
	"fmt"
	"net"
	"sort"
	"testing"
	"time"

	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/skademlia"
	"github.com/perlin-network/noise/utils"

	"github.com/stretchr/testify/assert"
)

func TODOTestSKademliaEviction(t *testing.T) {
	self := skademlia.NewIdentityAdapter(8, 8)
	ids := []*skademlia.IdentityAdapter{self}
	// create 5 peers, last peer should not be in table
	peers := generateBucketIDs(self, 5)
	ids = append(ids, peers...)
	// make max bucket size 4
	nodes, msgServices, ports := makeNodesFromIDs(ids, 4)

	// being discovery process to connect nodes to each other
	peer0 := skademlia.NewID(nodes[0].GetIdentityAdapter().MyIdentity(), fmt.Sprintf("%s:%d", host, ports[0]))

	var discoveryServices []*skademlia.DiscoveryService

	// Connect other nodes to node 0
	for _, node := range nodes {
		ca, ok := node.GetConnectionAdapter().(*skademlia.ConnectionAdapter)
		assert.True(t, ok)

		// pull out the discovery services so you can test it
		discoveryServices = append(discoveryServices, ca.Discovery)

		// bootstrap
		assert.Nil(t, ca.Bootstrap(peer0))
	}

	// make sure nodes are connected
	time.Sleep(250 * time.Duration(len(nodes)) * time.Millisecond)

	rt := discoveryServices[0].Routes

	skademliaID := skademlia.NewID(ids[1].MyIdentity(), "")
	expectedBucketID := rt.GetBucketID(skademliaID.Id)
	for i := 2; i < len(nodes); i++ {
		skademliaID = skademlia.NewID(ids[i].MyIdentity(), "")
		bucketID := rt.GetBucketID(skademliaID.Id)
		assert.Equalf(t, expectedBucketID, bucketID, "expected bucket ID to be %d, got %d", expectedBucketID, bucketID)
		fmt.Printf("bucket id: %d\n", bucketID)
	}

	// for debugging, print out each node's routes
	for i := 0; i < len(discoveryServices); i++ {
		peers := discoveryServices[i].Routes.GetPeerAddresses()
		sort.Strings(peers)
		fmt.Printf("Node %d: Self: %s Routes: %v\n", i, discoveryServices[i].Routes.Self().Address, peers)
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

func makeNodesFromIDs(ids []*skademlia.IdentityAdapter, bucketSize int) ([]*protocol.Node, []*MsgService, []int) {
	var nodes []*protocol.Node
	var msgServices []*MsgService
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

		node := protocol.NewNode(
			protocol.NewController(),
			idAdapter,
		)

		connAdapter, err := skademlia.NewConnectionAdapter(
			listener,
			dialTCP,
		)
		if err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		id := skademlia.NewID(idAdapter.MyIdentity(), address)
		connAdapter.RegisterNode(node, id)

		rt := skademlia.NewRoutingTableWithOptions(id, skademlia.WithBucketSize(bucketSize))
		connAdapter.Discovery.Routes = rt

		msgSvc := &MsgService{
			Mailbox: make(chan string, 1),
		}

		node.AddService(msgSvc)

		node.Listen()

		nodes = append(nodes, node)
		msgServices = append(msgServices, msgSvc)
		ports = append(ports, port)
	}

	return nodes, msgServices, ports
}
