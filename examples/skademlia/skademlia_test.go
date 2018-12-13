package skademlia_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/utils"

	"github.com/stretchr/testify/assert"
)

const (
	serviceID = 42
	host      = "localhost"
)

func TestSKademliaBootstrap(t *testing.T) {
	numNodes := 3
	nodes := makeNodes(numNodes)

	// Connect other nodes to node 0
	for _, svc := range nodes {
		assert.Nil(t, svc.Bootstrap(nodes[0].Self()))
	}

	// make sure nodes are connected
	time.Sleep(time.Duration(len(nodes)) * time.Second)

	// assert broadcasts goes to everyone
	for i := 0; i < len(nodes); i++ {
		expected := fmt.Sprintf("This is a broadcasted message from Node %d.", i)
		assert.Nil(t, nodes[i].Messenger().Broadcast(context.Background(), &protocol.MessageBody{
			Service: serviceID,
			Payload: ([]byte)(expected),
		}))

		// Check if message was received by other nodes.
		for j := 0; j < len(nodes); j++ {
			if i == j {
				continue
			}
			select {
			case received := <-nodes[j].Mailbox:
				assert.Equalf(t, expected, received, "Expected message '%s' to be received by node %d but got '%v'", expected, j, received)
			case <-time.After(2 * time.Second):
				assert.Failf(t, "Timed out attempting to receive message", "from Node %d for Node %d", i, j)
			}
		}
	}
}

type MsgService struct {
	*noise.Noise
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

func makeNodes(numNodes int) []*MsgService {
	var services []*MsgService

	// setup all the nodes
	for i := 0; i < numNodes; i++ {

		// setup the node
		config := &noise.Config{
			Host:            host,
			Port:            utils.GetRandomUnusedPort(),
			EnableSKademlia: true,
		}
		n, err := noise.NewNoise(config)
		if err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		// create service
		service := &MsgService{
			Noise:   n,
			Mailbox: make(chan string, 1),
		}
		// register the callback
		service.OnReceive(noise.OpCode(serviceID), service.Receive)

		services = append(services, service)
	}

	return services
}
