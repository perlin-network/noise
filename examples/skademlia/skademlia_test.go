package skademlia_test

import (
	"encoding/hex"
	"fmt"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/skademlia"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

const (
	serviceID = 66
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

func TestSKademlia(t *testing.T) {
	var nodes []*protocol.Node
	var msgServices []*MessageService
	var skServices []*skademlia.Service

	// setup all the nodes
	for i := 0; i < numNodes; i++ {
		idAdapter := skademlia.NewIdentityAdapter(8, 8)
		addr := fmt.Sprintf("%s:%d", host, startPort+i)

		log.Info().
			Str("addr", addr).
			Str("pub_key", hex.EncodeToString(idAdapter.MyIdentity())).
			Msgf("Setting up node %d", i)

		listener, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		connAdapter, err := skademlia.NewConnectionAdapter(
			listener,
			dialTCP,
			skademlia.NewID(idAdapter.MyIdentity(), addr))
		if err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		node := protocol.NewNode(
			protocol.NewController(),
			connAdapter,
			idAdapter,
		)

		msgSvc := &MessageService{
			Mailbox: make(chan string, 1),
		}

		node.AddService(msgSvc)

		skSvc := skademlia.NewService(
			node,
			peer.CreateID(addr, idAdapter.MyIdentity()),
		)

		connAdapter.SetSKademliaService(skSvc)

		node.AddService(skSvc)

		node.Start()

		nodes = append(nodes, node)
		msgServices = append(msgServices, msgSvc)
		skServices = append(skServices, skSvc)
	}

	// make sure all the nodes can listen for incoming connections
	time.Sleep(time.Duration(len(nodes)) * time.Second)

	// Connect other nodes to node 0
	for i := 1; i < len(nodes); i++ {
		if i == 0 {
			// skip node 0
			continue
		}
		node0ID := nodes[0].GetIdentityAdapter().MyIdentity()
		nodes[i].GetConnectionAdapter().AddPeerID(node0ID, fmt.Sprintf("%s:%d", host, startPort+0))
	}

	// being discovery process to connect nodes to each other
	for _, sk := range skServices {
		sk.Bootstrap()
	}

	// make sure nodes are connected
	time.Sleep(time.Duration(len(nodes)) * time.Second)

	// assert broadcasts goes to everyone
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

	// disconnect a node 1 from node 0
	disconnect := []int{0, 1}
	nodes[disconnect[0]].ManuallyRemovePeer(nodes[disconnect[1]].GetIdentityAdapter().MyIdentity())

	// assert broadcasts goes to everyone
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
			if (i == disconnect[0] && j == disconnect[1]) || (i == disconnect[1] && j == disconnect[0]) {
				// this is a disconnected situation, should not deliver message
				select {
				case received := <-msgServices[j].Mailbox:
					assert.Failf(t, "Should not have received message", "from Node %d for Node %d but got '%v'", i, j, received)
				case <-time.After(2 * time.Second):
					// success, no message should have passed
				}
			} else {
				// message should have been delivered
				select {
				case received := <-msgServices[j].Mailbox:
					assert.Equalf(t, expected, received, "Expected message '%s' to be received by node %d but got '%v'", expected, j, received)
				case <-time.After(2 * time.Second):
					assert.Failf(t, "Timed out attempting to receive message", "from Node %d for Node %d", i, j)
				}
			}
		}
	}
}
