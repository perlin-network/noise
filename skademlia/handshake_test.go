package skademlia

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/utils"
)

const (
	serviceID = 42
	host      = "localhost"
)

// SKService buffers all messages into a mailbox for this test.
type SKService struct {
	protocol.Service
	Mailbox chan string
}

func (n *SKService) Receive(message *protocol.Message) (*protocol.MessageBody, error) {
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

func makeMessageBody(value string) *protocol.MessageBody {
	return &protocol.MessageBody{
		Service: serviceID,
		Payload: ([]byte)(value),
	}
}

func dialTCP(addr string) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, 10*time.Second)
}

func TestHandshake(t *testing.T) {
	var nodes []*protocol.Node
	var services []*SKService
	var ports []int
	numNodes := 3

	// setup all the nodes
	for i := 0; i < numNodes; i++ {
		//var idAdapter *IdentityAdapter
		idAdapter := NewIdentityAdapter(8, 8)

		// sending to node[2] will fail due to handshake verification
		if i == 2 {
			idAdapter = NewIdentityAdapter(4, 4)
		}

		port := utils.GetRandomUnusedPort()
		ports = append(ports, port)
		address := fmt.Sprintf("%s:%d", host, port)
		listener, err := net.Listen("tcp", address)
		if err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		id := NewID(idAdapter.MyIdentity(), address)
		s := NewService(nil, id)
		connAdapter, err := NewConnectionAdapter(
			listener,
			dialTCP,
			id,
		)
		if err != nil {
			log.Fatal().Msgf("%+v", err)
		}
		connAdapter.SetSKademliaService(s)

		node := protocol.NewNode(
			protocol.NewController(),
			connAdapter,
			idAdapter,
		)
		node.SetCustomHandshakeProcessor(NewHandshakeProcessor(idAdapter))

		service := &SKService{
			Mailbox: make(chan string, 1),
		}

		node.AddService(service)

		node.Start()

		nodes = append(nodes, node)
		services = append(services, service)
	}

	nodeA := nodes[0]
	nodeB := nodes[1]
	nodeC := nodes[2]

	// Connect all the node routing tables
	for i, srcNode := range nodes {
		for j, otherNode := range nodes {
			if i == j {
				continue
			}
			peerID := otherNode.GetIdentityAdapter().MyIdentity()
			srcNode.GetConnectionAdapter().AddPeerID(peerID, fmt.Sprintf("%s:%d", host, ports[j]))
		}
	}

	body := makeMessageBody("hello")
	msg := protocol.Message{
		Sender:    nodeA.GetIdentityAdapter().MyIdentity(),
		Recipient: nodeB.GetIdentityAdapter().MyIdentity(),
		Body:      body,
	}
	err := nodeA.Send(&msg)
	if err != nil {
		t.Errorf("Send() expected no error, got: %+v", err)
	}

	// nodeC sending to nodeA should error
	msg.Sender = nodeC.GetIdentityAdapter().MyIdentity()
	msg.Recipient = nodeA.GetIdentityAdapter().MyIdentity()
	err = nodeC.Send(&msg)
	if err == nil {
		t.Errorf("Send() expected error")
	}

	// nodeA sending to nodeC should fail handshake
	msg.Sender = nodeA.GetIdentityAdapter().MyIdentity()
	msg.Recipient = nodeC.GetIdentityAdapter().MyIdentity()
	err = nodeA.Send(&msg)
	if err == nil {
		t.Errorf("Send() expected error")
	}

	// nodeB sending to nodeC should fail handshake
	msg.Sender = nodeB.GetIdentityAdapter().MyIdentity()
	err = nodeB.Send(&msg)
	if err == nil {
		t.Errorf("Send() expected error")
	}
}
