package requestresponse

import (
	"context"
	"fmt"
	"github.com/perlin-network/noise/base"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

const (
	simpleServiceID = 50
	numNodes        = 3
	host            = "localhost"
)

func dialTCP(addr string) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, 10*time.Second)
}

func setupNodes(startPort int) []*protocol.Node {
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
			connAdapter,
			idAdapter,
		)

		node.Start()

		nodes = append(nodes, node)
	}

	return nodes
}

type SimpleService struct {
	protocol.Service
}

func (n *SimpleService) Receive(message *protocol.Message) (*protocol.MessageBody, error) {
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
	startPort := 5000

	nodes := setupNodes(startPort)

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
		srcNode.GetConnectionAdapter().AddPeerID(peerID, fmt.Sprintf("%s:%d", host, startPort+j))
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
