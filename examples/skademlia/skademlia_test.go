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
	"github.com/perlin-network/noise/utils"

	"github.com/stretchr/testify/assert"
)

const (
	serviceID = 42
	host      = "localhost"
)

func TestSKademliaFixedPeers(t *testing.T) {
	numNodes := 3
	nodes, msgServices, ports := makeNodes(numNodes)

	// Connect all the node routing tables
	for i, srcNode := range nodes {
		for j, otherNode := range nodes {
			if i == j {
				continue
			}
			peerID := otherNode.GetIdentityAdapter().MyIdentity()
			srcNode.GetConnectionAdapter().AddPeerID(peerID, fmt.Sprintf("%s:%d", host, ports[j]))
		}
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

	// sends can go to everyone
	for i := 0; i < len(nodes); i++ {
		// Check if message was received by other nodes.
		for j := 0; j < len(msgServices); j++ {
			if i == j {
				continue
			}
			expected := fmt.Sprintf("Sending a msg message from Node %d to Node %d.", i, j)
			assert.Nil(t, nodes[i].Send(context.Background(),
				nodes[j].GetIdentityAdapter().MyIdentity(),
				&protocol.MessageBody{
					Service: serviceID,
					Payload: ([]byte)(expected),
				},
			))
			select {
			case received := <-msgServices[j].Mailbox:
				assert.Equalf(t, expected, received, "Expected message '%s' to be received by node %d but got '%v'", expected, j, received)
			case <-time.After(2 * time.Second):
				assert.Failf(t, "Timed out attempting to receive message", "from Node %d for Node %d", i, j)
			}
		}
	}
}

func TestSKademliaBootstrap(t *testing.T) {
	numNodes := 3
	nodes, msgServices, ports := makeNodes(numNodes)

	peer0 := dht.NewID(nodes[0].GetIdentityAdapter().MyIdentity(), fmt.Sprintf("%s:%d", host, ports[0]))

	// Connect other nodes to node 0
	for _, node := range nodes {
		ca, ok := node.GetConnectionAdapter().(*skademlia.ConnectionAdapter)
		assert.True(t, ok)
		assert.Nil(t, ca.Bootstrap(peer0))
	}

	// make sure nodes are connected
	time.Sleep(time.Duration(len(nodes)) * time.Second)

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
}

type MsgService struct {
	protocol.Service
	Mailbox chan string
}

func (n *MsgService) Receive(ctx context.Context, message *protocol.Message) (*protocol.MessageBody, error) {
	if message.Body.Service != serviceID {
		return nil, nil
	}
	if len(message.Body.Payload) == 0 {
		return nil, nil
	}
	payload := string(message.Body.Payload)
	n.Mailbox <- payload
	return nil, nil
}

func dialTCP(addr string) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, 10*time.Second)
}

func makeNodes(numNodes int) ([]*protocol.Node, []*MsgService, []int) {
	var nodes []*protocol.Node
	var msgServices []*MsgService
	var ports []int

	// setup all the nodes
	for i := 0; i < numNodes; i++ {
		idAdapter := skademlia.NewIdentityAdapter(8, 8)

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

		if _, err := skademlia.NewConnectionAdapter(listener, dialTCP, node, address); err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		msgSvc := &MsgService{
			Mailbox: make(chan string, 1),
		}

		node.AddService(msgSvc)

		node.Start()

		nodes = append(nodes, node)
		msgServices = append(msgServices, msgSvc)
		ports = append(ports, port)
	}

	return nodes, msgServices, ports
}
