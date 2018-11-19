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
type SKNode struct {
	Node        *protocol.Node
	Mailbox     chan string
	ConnAdapter protocol.ConnectionAdapter
}

func (n *SKNode) service(message *protocol.Message) {
	if message.Body.Service != serviceID {
		return
	}
	if len(message.Body.Payload) == 0 {
		return
	}
	payload := string(message.Body.Payload)
	n.Mailbox <- payload
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
	var nodes []*SKNode

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
			skademlia.ID{
				IdentityAdapter: idAdapter,
				Address:         address},
		)
		if err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		node := &SKNode{
			Node: protocol.NewNode(
				protocol.NewController(),
				connAdapter,
				idAdapter,
			),
			Mailbox:     make(chan string, 1),
			ConnAdapter: connAdapter,
		}

		node.Node.AddService(serviceID, node.service)

		node.Node.Start()

		nodes = append(nodes, node)
	}

	// Connect all the node routing tables
	for i, srcNode := range nodes {
		for j, otherNode := range nodes {
			if i == j {
				continue
			}
			peerID := otherNode.Node.GetIdentityAdapter().MyIdentity()
			srcNode.ConnAdapter.AddConnection(peerID, fmt.Sprintf("%s:%d", host, startPort+j))
		}
	}

	// Broadcast out a message from Node 0.
	expected := "This is a broadcasted message from Node 0."
	nodes[0].Node.Broadcast(makeMessageBody(expected))

	fmt.Println("Node 0 sent out a message.")

	// Check if message was received by other nodes.
	for i := 1; i < len(nodes); i++ {
		select {
		case received := <-nodes[i].Mailbox:
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
	nodes[0].Node.ManuallyRemovePeer(nodes[numNodes-2].Node.GetIdentityAdapter().MyIdentity())

	expected = "This is a second broadcasted message from Node 0."
	nodes[0].Node.Broadcast(makeMessageBody(expected))

	fmt.Println("Node 0 sent out a second message.")

	// Check if message was received by other nodes.
	for i := 1; i < len(nodes); i++ {
		select {
		case received := <-nodes[i].Mailbox:
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
