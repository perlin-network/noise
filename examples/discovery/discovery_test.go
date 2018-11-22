package discovery_test

import (
	"fmt"
	"github.com/perlin-network/noise/base"
	"github.com/perlin-network/noise/kademlia"
	"github.com/perlin-network/noise/kademlia/discovery"
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
	serviceID = 56
	numNodes  = 3
	startPort = 5000
	host      = "localhost"
)

// MessageService buffers all messages into a mailbox for this test.
type MessageService struct {
	protocol.Service
	Mailbox chan string
}

func (s *MessageService) Receive(message *protocol.Message) (*protocol.MessageBody, error) {
	if message.Body.Service != serviceID {
		return nil, nil
	}
	if len(message.Body.Payload) == 0 {
		return nil, errors.New("Empty payload")
	}
	reqMsg := string(message.Body.Payload)

	s.Mailbox <- reqMsg
	return nil, nil
}

func dialTCP(addr string) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, 10*time.Second)
}

func TODOTestDiscovery(t *testing.T) {
	var nodes []*protocol.Node
	var services []*MessageService

	// setup all the nodes
	for i := 0; i < numNodes; i++ {
		idAdapter := base.NewIdentityAdapter()
		addr := fmt.Sprintf("%s:%d", host, startPort+i)

		listener, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		connAdapter, err := kademlia.NewConnectionAdapter(listener, dialTCP)
		if err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		node := protocol.NewNode(
			protocol.NewController(),
			connAdapter,
			idAdapter,
		)

		service := &MessageService{
			Mailbox: make(chan string, 1),
		}

		node.AddService(service)

		discoveryService := discovery.NewService(
			node,
			peer.CreateID(addr, idAdapter.MyIdentity()),
		)

		connAdapter.SetDiscoveryService(discoveryService)

		node.AddService(discoveryService)

		nodes = append(nodes, node)
		services = append(services, service)
	}

	// Connect everyone to node 0
	for i := 1; i < len(nodes); i++ {
		if i == 0 {
			// skip node 0
			continue
		}
		node0ID := nodes[0].GetIdentityAdapter().MyIdentity()
		nodes[i].GetConnectionAdapter().AddPeerID(node0ID, fmt.Sprintf("%s:%d", host, startPort+0))
	}

	for _, node := range nodes {
		node.Start()
	}

	time.Sleep(time.Duration(len(nodes)*100) * time.Millisecond)

	// Broadcast out a message from Node 0.
	expected := "This is a broadcasted message from Node 0."
	nodes[0].Broadcast(&protocol.MessageBody{
		Service: serviceID,
		Payload: ([]byte)(expected),
	})

	// Check if message was received by other nodes.
	for i := 1; i < len(services); i++ {
		select {
		case received := <-services[i].Mailbox:
			assert.Equalf(t, received, expected, "Expected message %s to be received by node %d but got %v\n", expected, i, received)
		case <-time.After(2 * time.Second):
			assert.Fail(t, "Timed out attempting to receive message from Node 0.\n")
		}
	}
}
