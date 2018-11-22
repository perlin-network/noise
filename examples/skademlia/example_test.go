package skademlia_test

import (
	"fmt"
	"net"
	"time"

	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/skademlia"
)

const (
	serviceID = 42
	numNodes  = 3
	startPort = 5000
	host      = "localhost"
)

// SKNode buffers all messages into a mailbox for this test.
type SKService struct {
	protocol.Service
	Mailbox chan string
}

func (n *SKService) Receive(message *protocol.Message) (*protocol.MessageBody, error) {
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

func makeMessageBody(value string) *protocol.MessageBody {
	return &protocol.MessageBody{
		Service: serviceID,
		Payload: ([]byte)(value),
	}
}

func dialTCP(addr string) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, 10*time.Second)
}

// ExampleSKademlia demonstrates a simple example using SKademlia
func TODOExampleSKademlia() {
	var nodes []*protocol.Node
	var services []*SKService

	// setup all the nodes
	for i := 0; i < numNodes; i++ {
		idAdapter := skademlia.NewIdentityAdapter(8, 8)

		address := fmt.Sprintf("%s:%d", host, startPort+i)
		listener, err := net.Listen("tcp", address)
		if err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		connAdapter, err := skademlia.NewConnectionAdapter(
			listener,
			dialTCP,
			skademlia.NewID(idAdapter.MyIdentity(), address),
		)
		if err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		node := protocol.NewNode(
			protocol.NewController(),
			connAdapter,
			idAdapter,
		)
		svc := &SKService{
			Mailbox: make(chan string, 1),
		}

		node.AddService(svc)

		node.Start()

		nodes = append(nodes, node)
		services = append(services, svc)
	}

	// Connect all the node routing tables
	for i, srcNode := range nodes {
		for j, otherNode := range nodes {
			if i == j {
				continue
			}
			peerID := otherNode.GetIdentityAdapter().MyIdentity()
			srcNode.GetConnectionAdapter().AddPeerID(peerID, fmt.Sprintf("%s:%d", host, startPort+j))
		}
	}

	// Broadcast out a message from Node 0.
	expected := "This is a broadcasted message from Node 0."
	nodes[0].Broadcast(makeMessageBody(expected))

	fmt.Println("Node 0 sent out a message.")

	// Check if message was received by other nodes.
	for i := 1; i < len(services); i++ {
		select {
		case received := <-services[i].Mailbox:
			if received != expected {
				fmt.Printf("Expected message %s to be received by node %d but got %v\n", expected, i, received)
			} else {
				fmt.Printf("Node %d received a message from Node 0.\n", i)
			}
		case <-time.After(3 * time.Second):
			fmt.Printf("Timed out attempting to receive message from Node 0.\n")
		}
	}

	// disconnect a node
	// HACK: why is node[1] node 2?
	nodes[0].ManuallyRemovePeer(nodes[numNodes-2].GetIdentityAdapter().MyIdentity())

	expected = "This is a second broadcasted message from Node 0."
	nodes[0].Broadcast(makeMessageBody(expected))

	fmt.Println("Node 0 sent out a second message.")

	// Check if message was received by other nodes.
	for i := 1; i < len(services); i++ {
		select {
		case received := <-services[i].Mailbox:
			if received != expected {
				fmt.Printf("Expected message %s to be received by node %d but got %v\n", expected, i, received)
			} else {
				fmt.Printf("Node %d received a second message from Node 0.\n", i)
			}
		case <-time.After(3 * time.Second):
			fmt.Printf("Timed out attempting to receive message from Node 0.\n")
		}
	}

	// Output:
	// Node 0 sent out a message.
	// Node 1 received a message from Node 0.
	// Node 2 received a message from Node 0.
	// Node 0 sent out a second message.
	// Node 1 received a second message from Node 0.
	// Timed out attempting to receive message from Node 0.
}
