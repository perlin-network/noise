package discovery_test

import (
	"encoding/hex"
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
	log.Info().Str("Address", addr).Msg("Listening for peers")
	return net.DialTimeout("tcp", addr, 10*time.Second)
}

func TestDiscovery(t *testing.T) {
	var nodes []*protocol.Node
	var msgServices []*MessageService
	var discoveries []*discovery.Service

	// setup all the nodes
	for i := 0; i < numNodes; i++ {
		idAdapter := base.NewIdentityAdapter()
		addr := fmt.Sprintf("%s:%d", host, startPort+i)

		log.Info().
			Str("addr", addr).
			Str("pub_key", hex.EncodeToString(idAdapter.MyIdentity())).
			Msgf("Setting up node %d", i)

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

		msgService := &MessageService{
			Mailbox: make(chan string, 1),
		}

		node.AddService(msgService)

		discoveryService := discovery.NewService(
			node,
			peer.CreateID(addr, idAdapter.MyIdentity()),
		)

		connAdapter.SetDiscoveryService(discoveryService)

		node.AddService(discoveryService)

		node.Start()

		nodes = append(nodes, node)
		msgServices = append(msgServices, msgService)
		discoveries = append(discoveries, discoveryService)
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

	time.Sleep(time.Duration(len(nodes)) * time.Second)

	for _, d := range discoveries {
		d.Bootstrap()
	}

	time.Sleep(time.Duration(len(nodes)) * time.Second)

	for i := 0; i < len(nodes); i++ {
		// Broadcast out a message from Node 0.
		expected := fmt.Sprintf("This is a broadcasted message from Node %d.", i)
		nodes[i].Broadcast(&protocol.MessageBody{
			Service: serviceID,
			Payload: ([]byte)(expected),
		})

		// Check if message was received by other nodes.
		for j := 0; j < len(msgServices); j++ {
			if i == j {
				continue
			}
			select {
			case received := <-msgServices[j].Mailbox:
				assert.Equalf(t, expected, received, "Expected message '%s' to be received by node %d but got '%v'", expected, j, received)
			case <-time.After(2 * time.Second):
				assert.Failf(t, "Timed out attempting to receive message", "from Node %d for Node %d", i, j)
			}
		}
	}
}
