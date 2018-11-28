package requestresponse

import (
	"context"
	"fmt"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const (
	nestedServiceID   = 51
	maxNestedMessages = 5
)

var (
	keys [][]byte
)

type NestedService struct {
	protocol.Service
	node          *protocol.Node
	id            int
	requestCount  int
	responseCount int
}

func (n *NestedService) Receive(message *protocol.Message) (*protocol.MessageBody, error) {
	if message.Body.Service != nestedServiceID {
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
		target := keys[(n.id+1)%numNodes]
		return n.node.Request(ctx,
			target,
			&protocol.MessageBody{
				Service: nestedServiceID,
				Payload: ([]byte)(fmt.Sprintf("%s %d", reqMsg, n.id)),
			},
		)
	} else if n.requestCount > n.responseCount {

		// after a certain number of request/response, only send the responses
		n.responseCount++
		return &protocol.MessageBody{
			Service: nestedServiceID,
			Payload: ([]byte)(reqMsg),
		}, nil
	}

	// after all the responses are back, no more replies
	return nil, nil
}

func TestNestedRequestResponse(t *testing.T) {
	startPort := 5100
	nodes := setupNodes(startPort)

	for i, node := range nodes {
		node.AddService(&NestedService{
			id:   i,
			node: node,
		})
		keys = append(keys, node.GetIdentityAdapter().MyIdentity())
	}

	// Connect all node's routing table
	for i, node := range nodes {
		for j := range nodes {
			if i == j {
				continue
			}
			node.GetConnectionAdapter().AddPeerID(keys[j], fmt.Sprintf("%s:%d", host, startPort+j))
		}
	}

	msg := "init"
	resp, err := nodes[0].Request(context.Background(),
		keys[1],
		&protocol.MessageBody{
			Service: nestedServiceID,
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
