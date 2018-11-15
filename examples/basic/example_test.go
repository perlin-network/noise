package basic

import (
	"flag"
	"fmt"
	"net"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/perlin-network/noise/connection"
	"github.com/perlin-network/noise/examples/basic/messages"
	"github.com/perlin-network/noise/identity"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
)

const (
	serviceID = 42
)

// BasicNode buffers all messages into a mailbox for this test.
type BasicNode struct {
	Node        *protocol.Node
	Mailbox     chan *messages.BasicMessage
	ConnAdapter *connection.AddressableConnectionAdapter
}

func (n *BasicNode) Service(message *protocol.Message) {
	if message.Body.Service != serviceID {
		return
	}
	if len(message.Body.Payload) == 0 {
		return
	}
	var basicMessage messages.BasicMessage
	if err := proto.Unmarshal(message.Body.Payload, &basicMessage); err != nil {
		return
	}
	n.Mailbox <- &basicMessage
}

func makeMessageBody(value string) *protocol.MessageBody {
	basicMessage := &messages.BasicMessage{
		Message: value,
	}
	payload, err := proto.Marshal(basicMessage)
	if err != nil {
		return nil
	}
	pMsg := &protocol.MessageBody{
		Service: serviceID,
		Payload: payload,
	}
	return pMsg
}

func dialTCP(addr string) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, 10*time.Second)
}

// ExampleBasic demonstrates how to broadcast a message to a set of peers that discover
// each other through peer discovery.
func ExampleBasic() {
	startPortFlag := flag.Int("port", 5000, "start port to listen to")
	hostFlag := flag.String("host", "localhost", "host to listen to")
	nodesFlag := flag.Int("nodes", 3, "number of nodes to start")
	flag.Parse()

	numNodes := *nodesFlag
	startPort := *startPortFlag
	host := *hostFlag

	var nodes []*BasicNode

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

		node := &BasicNode{
			Node: protocol.NewNode(
				protocol.NewController(),
				connAdapter,
				idAdapter,
			),
			Mailbox:     make(chan *messages.BasicMessage, 1),
			ConnAdapter: connAdapter,
		}

		node.Node.AddService(serviceID, node.Service)

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
			if received.Message != expected {
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
