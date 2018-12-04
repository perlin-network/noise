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
	"github.com/perlin-network/noise/skademlia/dht"
	"github.com/perlin-network/noise/skademlia/discovery"
	"github.com/perlin-network/noise/skademlia/peer"

	"github.com/stretchr/testify/assert"
)

func TestSKademliaEviction(t *testing.T) {
	bucketSize := 4
	startPort := 5500

	self := skademlia.NewIdentityAdapter(8, 8)
	ids := []*skademlia.IdentityAdapter{self}
	// create 5 peers, last peer should not be in table
	peers := generateBucketIDs(self, 5)
	ids = append(ids, peers...)
	nodes, msgServices := makeNodesFromIDs(ids, bucketSize, startPort)

	// being discovery process to connect nodes to each other
	peer0 := peer.CreateID(fmt.Sprintf("%s:%d", host, startPort), nodes[0].GetIdentityAdapter().MyIdentity())

	var discoveryServices []*discovery.Service

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
	time.Sleep(100 * time.Duration(len(nodes)) * time.Millisecond)

	rt := discoveryServices[0].Routes

	skademliaID := peer.CreateID("", ids[1].MyIdentity())
	expectedBucketID := rt.GetBucketID(skademliaID.Id)
	for i := 2; i < len(nodes); i++ {
		skademliaID = peer.CreateID("", ids[i].MyIdentity())
		bucketID := rt.GetBucketID(skademliaID.Id)
		assert.Equalf(t, expectedBucketID, bucketID, "expected bucket ID to be %d, got %d", expectedBucketID, bucketID)
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
			case <-time.After(100 * time.Millisecond):
				if i == 0 && j == 5 {
					// this case should fail because node 5 is not in node 0's routing table
					continue
				}
				assert.Failf(t, "Timed out attempting to receive message", "from Node %d for Node %d", i, j)
			}
		}
	}

	expectedLen := 4
	bucket := discoveryServices[0].Routes.Bucket(expectedBucketID)
	assert.Equalf(t, expectedLen, bucket.Len(), "expected bucket size to be %d, got %d", expectedLen, bucket.Len())

	// stop node 1, bootstrap node 5 and broadcast again
	nodes[1].Stop()
	node5, ok := nodes[5].GetConnectionAdapter().(*skademlia.ConnectionAdapter)
	assert.True(t, ok)
	assert.Nil(t, node5.Bootstrap(peer0))
	// make sure node 0 and node 5 are connected
	time.Sleep(100 * time.Millisecond)

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
			case <-time.After(100 * time.Millisecond):
				if i == 1 || j == 1 {
					// node 1 is disconnected, should not send or receive any messages
					continue
				}
				assert.Failf(t, "Timed out attempting to receive message", "from Node %d for Node %d", i, j)
			}
		}
	}

	// nodes[1] is no longer in node[0]'s routing table, but node[5] is now in it so size is the same
	expectedLen = 4
	assert.Equalf(t, expectedLen, bucket.Len(), "expected bucket size to be %d, got %d", expectedLen, bucket.Len())
}

func generateBucketIDs(id *skademlia.IdentityAdapter, n int) []*skademlia.IdentityAdapter {
	self := peer.CreateID("", id.MyIdentity())
	rt := dht.NewRoutingTable(self)

	var ids []*skademlia.IdentityAdapter

	for len(ids) < n {
		id := skademlia.NewIdentityAdapter(8, 8)
		if rt.GetBucketID(peer.CreateID("", id.MyIdentity()).Id) == 4 {
			ids = append(ids, id)
		}
	}
	return ids
}

func makeNodesFromIDs(ids []*skademlia.IdentityAdapter, bucketSize int, startPort int) ([]*protocol.Node, []*MsgService) {
	var nodes []*protocol.Node
	var msgServices []*MsgService

	// setup all the nodes
	for i := 0; i < len(ids); i++ {
		idAdapter := ids[i]

		port := startPort + i
		address := fmt.Sprintf("%s:%d", host, port)
		listener, err := net.Listen("tcp", address)
		if err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		node := protocol.NewNode(
			protocol.NewController(),
			idAdapter,
		)

		connAdapter, err := skademlia.NewConnectionAdapter(listener, dialTCP, node, address)
		if err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		// override the routes with one with a different bucket size
		peerID := peer.CreateID(address, idAdapter.MyIdentity())
		rt := dht.NewRoutingTableWithOptions(peerID, dht.WithBucketSize(bucketSize))
		connAdapter.Discovery.Routes = rt

		msgSvc := &MsgService{
			Mailbox: make(chan string, 1),
		}

		node.AddService(msgSvc)

		node.Start()

		nodes = append(nodes, node)
		msgServices = append(msgServices, msgSvc)
	}

	return nodes, msgServices
}
