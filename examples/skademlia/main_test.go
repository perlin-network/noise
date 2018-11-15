package skademlia_test

import (
	"flag"
	"fmt"
	"net"
	"time"

	"github.com/perlin-network/noise/connection"
	"github.com/perlin-network/noise/identity"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
)

const (
	serviceID = 42
)

// SKNode buffers all messages into a mailbox for this test.
type SKNode struct {
	Node        *protocol.Node
	Mailbox     chan string
	ConnAdapter *connection.AddressableConnectionAdapter
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

// ExampleSKademliaExample demonstrates a simple test using SKademlia
func ExampleSKademliaExample() {
	startPortFlag := flag.Int("port", 5000, "start port to listen to")
	hostFlag := flag.String("host", "localhost", "host to listen to")
	nodesFlag := flag.Int("nodes", 3, "number of nodes to start")
	flag.Parse()

	numNodes := *nodesFlag
	startPort := *startPortFlag
	host := *hostFlag

	var nodes []*SKNode

	// setup all the nodes
	for i := 0; i < numNodes; i++ {
		idAdapter := identity.NewDefaultIdentityAdapter()

		listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, startPort+i))
		if err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		connAdapter, err := connection.StartAddressableConnectionAdapter(listener, dialTCP)
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
			peerID := (*otherNode.Node.GetIdentityAdapter()).MyIdentity()
			srcNode.ConnAdapter.MapIDToAddress(peerID, fmt.Sprintf("%s:%d", host, startPort+j))
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

	// Output:
	// Node 0 sent out a message.
	// Node 1 received a message from Node 0.
	// Node 2 received a message from Node 0.
}
