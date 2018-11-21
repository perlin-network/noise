package discovery_test

import (
	"fmt"
	"github.com/perlin-network/noise/base"
	"github.com/perlin-network/noise/base/discovery"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

const (
	serviceID = 42
	numNodes  = 3
	startPort = 5000
	host      = "localhost"
)

// DiscoveryNode buffers all messages into a mailbox for this test.
type DiscoveryNode struct {
	Node        *protocol.Node
	Mailbox     chan string
	ConnAdapter protocol.ConnectionAdapter
}

func (n *DiscoveryNode) receiveHandler(message *protocol.Message) (*protocol.MessageBody, error) {
	if len(message.Body.Payload) == 0 {
		return nil, errors.New("Empty payload")
	}
	reqMsg := string(message.Body.Payload)

	n.Mailbox <- reqMsg
	return nil, nil
}

func dialTCP(addr string) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, 10*time.Second)
}

// TestDiscovery demonstrates using request response.
func TestDiscovery(t *testing.T) {
	var nodes []*DiscoveryNode

	// setup all the nodes
	for i := 0; i < numNodes; i++ {
		idAdapter := base.NewIdentityAdapter()
		addr := fmt.Sprintf("%s:%d", host, startPort+i)

		listener, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		connAdapter, err := base.NewConnectionAdapter(listener, dialTCP)
		if err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		node := &DiscoveryNode{
			Node: protocol.NewNode(
				protocol.NewController(),
				connAdapter,
				idAdapter,
			),
			ConnAdapter: connAdapter,
		}

		node.Node.AddService(serviceID, node.receiveHandler)

		discoveryService := discovery.NewService(
			node.Node,
			peer.CreateID(addr, idAdapter.GetKeyPair().PublicKey),
		)

		node.Node.AddService(discovery.ServiceID, discoveryService.ReceiveHandler)

		nodes = append(nodes, node)
	}

	// Connect everyone to node 0
	for i := 1; i < len(nodes); i++ {
		if i == 0 {
			// skip node 0
			continue
		}
		node0ID := nodes[0].Node.GetIdentityAdapter().MyIdentity()
		nodes[i].ConnAdapter.AddPeerID(node0ID, fmt.Sprintf("%s:%d", host, startPort+0))
	}

	for _, node := range nodes {
		node.Node.Start()
	}

	time.Sleep(time.Duration(len(nodes)) * time.Second)

	// Broadcast out a message from Node 0.
	expected := "This is a broadcasted message from Node 0."
	nodes[0].Node.Broadcast(&protocol.MessageBody{
		Service: serviceID,
		Payload: ([]byte)(expected),
	})

	// Check if message was received by other nodes.
	for i := 1; i < len(nodes); i++ {
		select {
		case received := <-nodes[i].Mailbox:
			assert.Equalf(t, received, expected, "Expected message %s to be received by node %d but got %v\n", expected, i, received)
		case <-time.After(2 * time.Second):
			assert.Fail(t, "Timed out attempting to receive message from Node 0.\n")
		}
	}
}
