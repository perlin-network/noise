package basic

import (
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/perlin-network/noise/connection"
	"github.com/perlin-network/noise/identity"
	"github.com/perlin-network/noise/log"
	"net"
	"time"

	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/examples/basic/messages"
	"github.com/perlin-network/noise/protocol"
)

// BasicNode buffers all messages into a mailbox for this test.
type BasicNode struct {
	Node    *protocol.Node
	Mailbox chan *protocol.MessageBody
}

// ExampleBasicPlugin demonstrates how to broadcast a message to a set of peers that discover
// each other through peer discovery.
func ExampleBasicPlugin() {
	startPortFlag := flag.Int("port", 5000, "start port to listen to")
	hostFlag := flag.String("host", "localhost", "host to listen to")
	nodesFlag := flag.Int("nodes", 3, "number of nodes to start")
	flag.Parse()

	numNodes := *nodesFlag
	startPort := *startPortFlag
	host := *hostFlag

	var nodes []*BasicNode
	var node2keys map[string]int
	var connAdapters []*connection.ConnectionAdapter

	for i := 0; i < numNodes; i++ {
		keyPair := ed25519.RandomKeyPair()
		idAdapter := identity.NewDefaultIdentityAdapter(keyPair)

		listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, startPort+i))
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}

		connAdapter, err := connection.StartAddressableConnectionAdapter(listener, func(addr string) (net.Conn, error) {
			return net.DialTimeout("tcp", addr, 10*time.Second)
		})
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}

		node := protocol.NewNode(
			protocol.NewController(),
			connAdapter,
			idAdapter,
		)

		node.AddService(42, func(message *protocol.Message) {
			node.Mailbox <- message.Body
		})

		nodes = append(nodes, &BasicNode{node,
			Mailbox: make(chan *messages.BasicMessage, 1),
		})
		node2keys[keyPair.PublicKeyHex()] = i
		connAdapters = append(connAdapters, connAdapter)
	}

	// Connect all the node routing tables
	for _, adapter := range connAdapters {
		for peerID, portOffset := range node2keys {
			adapter.MapIDToAddress(peerID, fmt.Sprintf("%s:%d", host, startPort+portOffset))
		}
	}

	// Bootstrap all the nodes
	for _, node := range nodes {
		node.Start()
	}

	// Broadcast out a message from Node 0.
	expected := "This is a broadcasted message from Node 0."
	nodes[0].Send(&protocol.Message{
		Sender:    kp.PublicKey,
		Recipient: peerID,
		Body: &protocol.MessageBody{
			Service: 42,
			Payload: []byte(expected),
		},
	})

	fmt.Println("Node 0 sent out a message.")

	// Check if message was received by other nodes.
	for i := 1; i < len(nodes); i++ {
		select {
		case received := <-nodes[i].Mailbox:
			if received.Payload != expected {
				fmt.Printf("Expected message %s to be received by node %d but got %v\n", expected, i, received.Message)
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
