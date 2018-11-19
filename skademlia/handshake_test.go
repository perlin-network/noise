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

// SKNode buffers all messages into a mailbox for this test.
type SKNode struct {
	Node        *protocol.Node
	Mailbox     chan string
	ConnAdapter protocol.ConnectionAdapter
}

func (n *SKNode) service(message *protocol.Message) {
	if message.Body.Service != serviceID {
		return
	}
	if len(message.Body.Payload) == 0 {
		return
	}
	payload := string(message.Body.Payload)
	n.Mailbox <- payload
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
	var nodes []*SKNode
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

		connAdapter, err := NewConnectionAdapter(
			listener,
			dialTCP,
			NewID(idAdapter.MyIdentity(), address),
		)
		if err != nil {
			log.Fatal().Msgf("%+v", err)
		}

		pNode := protocol.NewNode(
			protocol.NewController(),
			connAdapter,
			idAdapter,
		)
		pNode.SetCustomHandshakeProcessor(NewHandshakeProcessor(idAdapter))

		node := &SKNode{
			Node:        pNode,
			Mailbox:     make(chan string, 1),
			ConnAdapter: connAdapter,
		}

		node.Node.AddService(serviceID, node.service)

		node.Node.Start()

		nodes = append(nodes, node)
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
			peerID := otherNode.Node.GetIdentityAdapter().MyIdentity()
			srcNode.ConnAdapter.AddConnection(peerID, fmt.Sprintf("%s:%d", host, ports[j]))
		}
	}

	body := makeMessageBody("hello")
	msg := protocol.Message{
		Sender:    nodeA.Node.GetIdentityAdapter().MyIdentity(),
		Recipient: nodeB.Node.GetIdentityAdapter().MyIdentity(),
		Body:      body,
	}
	err := nodeA.Node.Send(&msg)
	if err != nil {
		t.Errorf("Send() expected no error, got: %+v", err)
	}

	// nodeC sending to nodeA should error
	msg.Sender = nodeC.Node.GetIdentityAdapter().MyIdentity()
	msg.Recipient = nodeA.Node.GetIdentityAdapter().MyIdentity()
	err = nodeC.Node.Send(&msg)
	if err == nil {
		t.Errorf("Send() expected error")
	}

	// nodeA sending to nodeC should fail handshake
	msg.Sender = nodeA.Node.GetIdentityAdapter().MyIdentity()
	msg.Recipient = nodeC.Node.GetIdentityAdapter().MyIdentity()
	err = nodeA.Node.Send(&msg)
	if err == nil {
		t.Errorf("Send() expected error")
	}

	// nodeB sending to nodeC should fail handshake
	msg.Sender = nodeB.Node.GetIdentityAdapter().MyIdentity()
	err = nodeB.Node.Send(&msg)
	if err == nil {
		t.Errorf("Send() expected error")
	}
}
