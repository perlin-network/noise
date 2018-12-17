package requestresponse

import (
	"context"
	"fmt"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/utils"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const (
	nestedOpCode      = 51
	maxNestedMessages = 5
)

var (
	services []*NestedService
)

type NestedService struct {
	*noise.Noise
	id            int
	requestCount  int
	responseCount int
}

func (n *NestedService) Receive(ctx context.Context, message *protocol.Message) (*protocol.MessageBody, error) {
	if message.Body.Service != nestedOpCode {
		// not the matching service id
		return nil, nil
	}
	if len(message.Body.Payload) == 0 {
		return nil, errors.New("Empty payload")
	}
	reqMsg := string(message.Body.Payload)

	if n.requestCount < maxNestedMessages {

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		n.requestCount++

		// make another request/response
		target := services[(n.id+1)%numNodes].Self().PublicKey
		return n.Messenger().Request(ctx,
			target,
			&protocol.MessageBody{
				Service: nestedOpCode,
				Payload: ([]byte)(fmt.Sprintf("%s %d", reqMsg, n.id)),
			},
		)
	} else if n.requestCount > n.responseCount {

		// after a certain number of request/response, only send the responses
		n.responseCount++
		return &protocol.MessageBody{
			Service: nestedOpCode,
			Payload: ([]byte)(reqMsg),
		}, nil
	}

	// after all the responses are back, no more replies
	return nil, nil
}

func TestNestedRequestResponse(t *testing.T) {
	// setup the services
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
		s := &NestedService{
			Noise: n,
			id:    i,
		}
		s.OnReceive(noise.OpCode(nestedOpCode), s.Receive)

		services = append(services, s)
	}

	// Connect all node's routing table
	for i, s := range services {
		var peerIDs []noise.PeerID
		for j := range services {
			if i == j {
				continue
			}
			peerIDs = append(peerIDs, services[j].Self())
		}
		s.Bootstrap(peerIDs...)
	}

	msg := "init"
	resp, err := services[0].Messenger().Request(context.Background(),
		services[1].Self().PublicKey,
		&protocol.MessageBody{
			Service: nestedOpCode,
			Payload: ([]byte)(msg),
		},
	)
	assert.Nil(t, err)

	expected := fmt.Sprintf("%s 1", msg)
	for i := 1; i < maxNestedMessages*numNodes; i++ {
		expected = fmt.Sprintf("%s %d", expected, (i+1)%numNodes)
	}
	assert.Equal(t, expected, string(resp.Payload))
}
