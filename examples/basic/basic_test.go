package basic

import (
	"context"
	"fmt"
	"time"

	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/examples/basic/messages"
	"github.com/perlin-network/noise/utils"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
)

const (
	opCode   = 43
	numNodes = 3
	host     = "localhost"
)

// BasicService buffers all messages into a mailbox for this test.
type BasicService struct {
	*noise.Noise
	Mailbox chan *messages.BasicMessage
}

func (n *BasicService) Receive(ctx context.Context, message *noise.Message) (*noise.MessageBody, error) {
	if message.Body.Service != opCode {
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

func makeMessageBody(value string) *noise.MessageBody {
	msg := &messages.BasicMessage{
		Message: value,
	}
	payload, err := proto.Marshal(msg)
	if err != nil {
		return nil
	}
	body := &noise.MessageBody{
		Service: opCode,
		Payload: payload,
	}
	return body
}

// ExampleBasic demonstrates how to broadcast a message to a set of peers that discover
// each other through peer discovery.
func ExampleBasic() {
	var services []*BasicService

	// setup all the nodes
	for i := 0; i < numNodes; i++ {
		// setup the node
		config := &noise.Config{
			Host:            host,
			Port:            utils.GetRandomUnusedPort(),
			EnableSKademlia: false,
		}
		n, err := noise.NewNoise(config)
		if err != nil {
			panic(err)
		}

		// create service
		service := &BasicService{
			Noise:   n,
			Mailbox: make(chan *messages.BasicMessage, 1),
		}
		// register the callback
		service.OnReceive(opCode, service.Receive)

		services = append(services, service)
	}

	// Connect all the node routing tables
	for i, svc := range services {
		var peers []noise.PeerID
		for j, other := range services {
			if i == j {
				continue
			}
			peers = append(peers, other.Self())
		}
		svc.Bootstrap(peers...)
	}

	// Broadcast out a message from Node 0.
	expected := "This is a broadcasted message from Node 0."
	services[0].Broadcast(context.Background(), makeMessageBody(expected))

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
