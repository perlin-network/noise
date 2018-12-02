package basic

import (
	"context"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/perlin-network/noise/base"
	"github.com/perlin-network/noise/examples/basic/messages"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
	"net"
	"time"
)

const (
	serviceID = 42
	numNodes  = 3
	startPort = 5000
	host      = "localhost"
)

// BasicService buffers all messages into a mailbox for this test.
type BasicService struct {
	protocol.Service
	Mailbox chan *messages.BasicMessage
}

func (n *BasicService) Receive(ctx context.Context, message *protocol.Message) (*protocol.MessageBody, error) {
	if message.Body.Service != serviceID {
		// early exit if not the matching service
		return nil, nil
	}
	if len(message.Body.Payload) == 0 {
		return nil, errors.New("Empty payload")
	}
	var basicMessage messages.BasicMessage
	if err := proto.Unmarshal(message.Body.Payload, &basicMessage); err != nil {
		return nil, errors.Wrap(err, "Unable to unmarshal payload")
	}
	n.Mailbox <- &basicMessage
	return nil, nil
}

func makeMessageBody(value string) *protocol.MessageBody {
	msg := &messages.BasicMessage{
		Message: value,
	}
	payload, err := proto.Marshal(msg)
	if err != nil {
		return nil
	}
	body := &protocol.MessageBody{
		Service: serviceID,
		Payload: payload,
	}
	return body
}

func dialTCP(addr string) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, 10*time.Second)
}

// ExampleBasic demonstrates how to broadcast a message to a set of peers that discover
// each other through peer discovery.
func ExampleBasic() {
	var services []*BasicService
	var nodes []*protocol.Node

	// setup all the nodes
	for i := 0; i < numNodes; i++ {
		idAdapter := base.NewIdentityAdapter()

		listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, startPort+i))
		if err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		connAdapter, err := base.NewConnectionAdapter(listener, dialTCP)
		if err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		node := protocol.NewNode(
			protocol.NewController(),
			idAdapter,
		)
		node.SetConnectionAdapter(connAdapter)

		service := &BasicService{
			Mailbox: make(chan *messages.BasicMessage, 1),
		}

		node.AddService(service)

		nodes = append(nodes, node)
		services = append(services, service)
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

	for _, node := range nodes {
		node.Listen()
	}

	time.Sleep(time.Duration(len(nodes)*100) * time.Millisecond)

	// Broadcast out a message from Node 0.
	expected := "This is a broadcasted message from Node 0."
	nodes[0].Broadcast(context.Background(), makeMessageBody(expected))

	fmt.Println("Node 0 sent out a message.")

	// Check if message was received by other nodes.
	for i := 1; i < len(services); i++ {
		select {
		case received := <-services[i].Mailbox:
			if received.Message != expected {
				fmt.Printf("Expected message %s to be received by node %d but got %v\n", expected, i, received.Message)
			} else {
				fmt.Printf("Node %d received a message from Node 0.\n", i)
			}
		case <-time.After(2 * time.Second):
			fmt.Printf("Timed out attempting to receive message from Node 0.\n")
		}
	}

	// Output:
	// Node 0 sent out a message.
	// Node 1 received a message from Node 0.
	// Node 2 received a message from Node 0.
}
