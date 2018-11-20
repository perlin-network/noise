package requestresponse

import (
	"context"
	"fmt"
	"github.com/perlin-network/noise/base"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

const (
	serviceID = 42
	numNodes  = 3
	startPort = 5000
	host      = "localhost"
)

// RequestResponseNode buffers all messages into a mailbox for this test.
type RequestResponseNode struct {
	Node        *protocol.Node
	ConnAdapter protocol.ConnectionAdapter
}

func (n *RequestResponseNode) receiveHandler(message *protocol.Message) (*protocol.MessageBody, error) {
	if len(message.Body.Payload) == 0 {
		return nil, errors.New("Empty payload")
	}
	reqMsg := string(message.Body.Payload)

	return &protocol.MessageBody{
		Service: serviceID,
		Payload: ([]byte)(fmt.Sprintf("%s reply", reqMsg)),
	}, nil
}

func dialTCP(addr string) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, 10*time.Second)
}

// TestRequestResponse demonstrates using request response.
func TestRequestResponse(t *testing.T) {
	var nodes []*RequestResponseNode

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

		node := &RequestResponseNode{
			Node: protocol.NewNode(
				protocol.NewController(),
				connAdapter,
				idAdapter,
			),
			ConnAdapter: connAdapter,
		}

		node.Node.AddService(serviceID, node.receiveHandler)

		node.Node.Start()

		nodes = append(nodes, node)
	}

	// Connect all the node routing tables
	for i, srcNode := range nodes {
		for j, otherNode := range nodes {
			if i == j {
				continue
			}
			peerID := otherNode.Node.GetIdentityAdapter().MyIdentity()
			srcNode.ConnAdapter.AddPeerID(peer.CreateID(fmt.Sprintf("%s:%d", host, startPort+j), peerID))
		}
	}

	reqMsg := "This is a request response message from Node 0."
	resp, err := nodes[0].Node.Request(context.Background(),
		nodes[1].Node.GetIdentityAdapter().MyIdentity(),
		&protocol.MessageBody{
			Service: serviceID,
			Payload: ([]byte)(reqMsg),
		},
	)

	assert.Nil(t, err)
	assert.Equal(t, fmt.Sprintf("%s reply", reqMsg), string(resp.Payload))
}
