package skademlia_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/skademlia"
	"github.com/perlin-network/noise/skademlia/dht"
	"github.com/perlin-network/noise/skademlia/peer"
	"github.com/perlin-network/noise/utils"

	"github.com/stretchr/testify/assert"
)

const (
	evictOpcode = 43
	c1          = 8
	c2          = 8
)

func TestSKademliaEviction(t *testing.T) {
	bucketSize := 4

	self := skademlia.NewIdentityAdapter(c1, c2)
	ids := []*skademlia.IdentityAdapter{self}
	// create 5 peers, last peer should not be in table
	peers := generateBucketIDs(self, 5)
	ids = append(ids, peers...)
	services := makeNodesFromIDs(ids, bucketSize)

	// Connect other nodes to node 0
	for i, svc := range services {
		if i == 0 {
			continue
		}

		// bootstrap
		assert.Nil(t, svc.Bootstrap(services[0].Self()))
	}

	// make sure nodes are connected
	time.Sleep(100 * time.Duration(len(services)) * time.Millisecond)

	rt := services[0].Metadata()["connection_adapter"].(*skademlia.ConnectionAdapter).Discovery.Routes

	skademliaID := peer.CreateID("", ids[1].MyIdentity())
	expectedBucketID := rt.GetBucketID(skademliaID.Id)
	for i := 2; i < len(services); i++ {
		skademliaID = peer.CreateID("", ids[i].MyIdentity())
		bucketID := rt.GetBucketID(skademliaID.Id)
		assert.Equalf(t, expectedBucketID, bucketID, "expected bucket ID to be %d, got %d", expectedBucketID, bucketID)
	}

	// assert broadcasts goes to everyone
	for i := 0; i < len(services); i++ {
		expected := fmt.Sprintf("This is a broadcasted message from Node %d.", i)
		assert.Nil(t, services[i].Broadcast(context.Background(), &noise.MessageBody{
			Service: evictOpcode,
			Payload: ([]byte)(expected),
		}))

		// Check if message was received by other nodes.
		for j := 0; j < len(services); j++ {
			if i == j {
				continue
			}
			select {
			case received := <-services[j].Mailbox:
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
	bucket := rt.Bucket(expectedBucketID)
	assert.Equalf(t, expectedLen, bucket.Len(), "expected bucket size to be %d, got %d", expectedLen, bucket.Len())

	// stop node 1, bootstrap node 5 and broadcast again
	services[1].Shutdown()
	assert.Nil(t, services[5].Bootstrap(services[0].Self()))
	// make sure node 0 and node 5 are connected
	time.Sleep(100 * time.Millisecond)

	// assert broadcasts goes to everyone
	for i := 0; i < len(services); i++ {
		expected := fmt.Sprintf("This is a broadcasted message from Node %d.", i)
		assert.Nil(t, services[i].Broadcast(context.Background(), &noise.MessageBody{
			Service: evictOpcode,
			Payload: ([]byte)(expected),
		}))

		// Check if message was received by other nodes.
		for j := 0; j < len(services); j++ {
			if i == j {
				continue
			}
			select {
			case received := <-services[j].Mailbox:
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
		id := skademlia.NewIdentityAdapter(c1, c2)
		if rt.GetBucketID(peer.CreateID("", id.MyIdentity()).Id) == 4 {
			ids = append(ids, id)
		}
	}
	return ids
}

func dialTCP(addr string) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, 10*time.Second)
}

type EvictService struct {
	*noise.Noise
	Mailbox chan string
}

func (n *EvictService) Receive(ctx context.Context, message *noise.Message) (*noise.MessageBody, error) {
	if message.Body.Service != evictOpcode {
		return nil, nil
	}
	if len(message.Body.Payload) == 0 {
		return nil, nil
	}
	payload := string(message.Body.Payload)
	n.Mailbox <- payload
	return nil, nil
}

func makeNodesFromIDs(ids []*skademlia.IdentityAdapter, bucketSize int) []*EvictService {
	var services []*EvictService

	// setup all the nodes
	for i := 0; i < len(ids); i++ {

		// setup the node
		config := &noise.Config{
			Host:            host,
			Port:            utils.GetRandomUnusedPort(),
			PrivateKeyHex:   ids[i].GetKeyPair().PrivateKeyHex(),
			EnableSKademlia: true,
			SKademliaC1:     c1,
			SKademliaC2:     c2,
		}
		n, err := noise.NewNoise(config)
		if err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		// create service
		svc := &EvictService{
			Noise:   n,
			Mailbox: make(chan string, 1),
		}

		// override the routes with one with a different bucket size
		peerID := peer.ID(svc.Self())
		rt := dht.NewRoutingTableWithOptions(peerID, dht.WithBucketSize(bucketSize))
		svc.Metadata()["connection_adapter"].(*skademlia.ConnectionAdapter).Discovery.Routes = rt

		// register the callback
		svc.OnReceive(noise.OpCode(bootstrapOpcode), svc.Receive)

		services = append(services, svc)
	}

	return services
}
