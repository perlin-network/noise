package requestresponse

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/perlin-network/noise/base"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/utils"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const (
	simpleServiceID = 50
	numNodes        = 3
	host            = "localhost"
)

func dialTCP(addr string) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, 10*time.Second)
}

func setupNodes() ([]*protocol.Node, []int) {
	var nodes []*protocol.Node
	var ports []int

	// setup all the nodes
	for i := 0; i < numNodes; i++ {
		idAdapter := base.NewIdentityAdapter()

		port := utils.GetRandomUnusedPort()
		ports = append(ports, port)
		address := fmt.Sprintf("%s:%d", host, port)
		listener, err := net.Listen("tcp", address)
		if err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		node := protocol.NewNode(
			protocol.NewController(),
			idAdapter,
		)

		if _, err := base.NewConnectionAdapter(listener, dialTCP, node); err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		node.Start()

		nodes = append(nodes, node)
	}

	return nodes, ports
}

type SimpleService struct {
	protocol.Service
}

func (n *SimpleService) Receive(ctx context.Context, message *protocol.Message) (*protocol.MessageBody, error) {
	if message.Body.Service != simpleServiceID {
		// not the matching service id
		return nil, nil
	}
	if len(message.Body.Payload) == 0 {
		return nil, errors.New("Empty payload")
	}
	reqMsg := string(message.Body.Payload)

	return &protocol.MessageBody{
		Service: simpleServiceID,
		Payload: ([]byte)(fmt.Sprintf("%s reply", reqMsg)),
	}, nil
}

func TestSimpleRequestResponse(t *testing.T) {
	nodes, ports := setupNodes()

	for _, node := range nodes {
		node.AddService(&SimpleService{})
	}

	// Connect node 0's routing table
	i, srcNode := 0, nodes[0]
	for j, otherNode := range nodes {
		if i == j {
			continue
		}
		peerID := otherNode.GetIdentityAdapter().MyIdentity()
		srcNode.GetConnectionAdapter().AddPeerID(peerID, fmt.Sprintf("%s:%d", host, ports[j]))
	}

	reqMsg0 := "Request response message from Node 0 to Node 1."
	resp, err := nodes[0].Request(context.Background(),
		nodes[1].GetIdentityAdapter().MyIdentity(),
		&protocol.MessageBody{
			Service: simpleServiceID,
			Payload: ([]byte)(reqMsg0),
		},
	)
	assert.Nil(t, err)
	assert.Equal(t, fmt.Sprintf("%s reply", reqMsg0), string(resp.Payload))

	reqMsg1 := "Request response message from Node 1 to Node 2."
	resp, err = nodes[1].Request(context.Background(),
		nodes[2].GetIdentityAdapter().MyIdentity(),
		&protocol.MessageBody{
			Service: simpleServiceID,
			Payload: ([]byte)(reqMsg1),
		},
	)
	assert.NotNil(t, err, "Should fail, nodes are not connected")
}
