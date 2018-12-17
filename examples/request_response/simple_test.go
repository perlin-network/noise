package requestresponse

import (
	"context"
	"fmt"
	"testing"

	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/utils"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const (
	simpleOpCode = 50
	numNodes     = 3
	host         = "localhost"
)

type SimpleService struct {
	*noise.Noise
}

func (s *SimpleService) receive(ctx context.Context, message *noise.Message) (*noise.MessageBody, error) {
	if message.Body.Service != simpleOpCode {
		// not the matching service id
		return nil, nil
	}
	if len(message.Body.Payload) == 0 {
		return nil, errors.New("Empty payload")
	}
	reqMsg := string(message.Body.Payload)

	return &noise.MessageBody{
		Service: simpleOpCode,
		Payload: ([]byte)(fmt.Sprintf("%s reply", reqMsg)),
	}, nil
}

func TestSimpleRequestResponse(t *testing.T) {
	// setup the services
	var services []*SimpleService
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
		s := &SimpleService{
			Noise: n,
		}
		s.OnReceive(noise.OpCode(simpleOpCode), s.receive)

		services = append(services, s)
	}

	// Connect others to node 0's routing table
	var peerIDs []noise.PeerID
	for i, other := range services {
		if i == 0 {
			continue
		}
		peerIDs = append(peerIDs, other.Self())
	}
	err := services[0].Bootstrap(peerIDs...)
	assert.Nil(t, err)

	reqMsg0 := "Request response message from Node 0 to Node 1."
	resp, err := services[0].Request(context.Background(),
		services[1].Self().PublicKey,
		&noise.MessageBody{
			Service: simpleOpCode,
			Payload: ([]byte)(reqMsg0),
		},
	)
	assert.Nil(t, err)
	assert.Equal(t, fmt.Sprintf("%s reply", reqMsg0), string(resp.Payload))

	reqMsg1 := "Request response message from Node 1 to Node 2."
	resp, err = services[1].Request(context.Background(),
		services[2].Self().PublicKey,
		&noise.MessageBody{
			Service: simpleOpCode,
			Payload: ([]byte)(reqMsg1),
		},
	)
	assert.NotNil(t, err, "Should fail, nodes are not connected")
}
