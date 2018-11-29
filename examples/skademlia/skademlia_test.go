package skademlia_test

import (
	"fmt"
	"github.com/perlin-network/noise/skademlia"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"

	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/utils"
)

const (
	serviceID = 42
	host      = "localhost"
	numNodes  = 3
)

// MsgService buffers all messages into a mailbox for this test.
type MsgService struct {
	protocol.Service
	Mailbox chan string
}

func (n *MsgService) Receive(message *protocol.Message) (*protocol.MessageBody, error) {
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

func TestSKademliaBasic(t *testing.T) {
	var nodes []*protocol.Node
	var msgServices []*MsgService
	var discoveryServices []*skademlia.Service
	var ports []int

	// setup all the nodes
	for i := 0; i < numNodes; i++ {
		//var idAdapter *IdentityAdapter
		idAdapter := skademlia.NewIdentityAdapter(8, 8)

		port := utils.GetRandomUnusedPort()
		ports = append(ports, port)
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
		connAdapter.SetSKademliaService(skSvc)

		msgSvc := &MsgService{
			Mailbox: make(chan string, 1),
		}

		node.AddService(msgSvc)

		node.Start()

		nodes = append(nodes, node)
		msgServices = append(msgServices, msgSvc)
		discoveryServices = append(discoveryServices, skSvc)
	}

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
		assert.Nil(t, nodes[i].Broadcast(&protocol.MessageBody{
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
			assert.Nil(t, nodes[i].Send(&protocol.Message{
				Sender:    nodes[i].GetIdentityAdapter().MyIdentity(),
				Recipient: nodes[j].GetIdentityAdapter().MyIdentity(),
				Body: &protocol.MessageBody{
					Service: serviceID,
					Payload: ([]byte)(expected),
				},
			}))
			select {
			case received := <-msgServices[j].Mailbox:
				assert.Equalf(t, expected, received, "Expected message '%s' to be received by node %d but got '%v'", expected, j, received)
			case <-time.After(2 * time.Second):
				assert.Failf(t, "Timed out attempting to receive message", "from Node %d for Node %d", i, j)
			}
		}
	}
}

func TODOTestSKademliaBootstrap(t *testing.T) {
	var nodes []*protocol.Node
	var msgServices []*MsgService
	var discoveryServices []*skademlia.Service
	var ports []int

	// setup all the nodes
	for i := 0; i < numNodes; i++ {
		//var idAdapter *IdentityAdapter
		idAdapter := skademlia.NewIdentityAdapter(8, 8)

		port := utils.GetRandomUnusedPort()
		ports = append(ports, port)
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
		connAdapter.SetSKademliaService(skSvc)

		msgSvc := &MsgService{
			Mailbox: make(chan string, 1),
		}

		node.AddService(msgSvc)

		node.Start()

		nodes = append(nodes, node)
		msgServices = append(msgServices, msgSvc)
		discoveryServices = append(discoveryServices, skSvc)
	}

	// Connect other nodes to node 0
	for i := 1; i < len(nodes); i++ {
		if i == 0 {
			// skip node 0
			continue
		}
		node0ID := nodes[0].GetIdentityAdapter().MyIdentity()
		assert.Nil(t, nodes[i].GetConnectionAdapter().AddPeerID(node0ID, fmt.Sprintf("%s:%d", host, ports[0])))
	}

	// being discovery process to connect nodes to each other
	for _, d := range discoveryServices {
		assert.Nil(t, d.Bootstrap())
	}

	// make sure nodes are connected
	time.Sleep(time.Duration(len(nodes)) * time.Second)

	// assert broadcasts goes to everyone
	for i := 0; i < len(nodes); i++ {
		expected := fmt.Sprintf("This is a broadcasted message from Node %d.", i)
		assert.Nil(t, nodes[i].Broadcast(&protocol.MessageBody{
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
