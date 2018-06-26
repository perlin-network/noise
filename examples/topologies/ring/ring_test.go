package basic

import (
	"flag"
	"fmt"
	"testing"
	"time"

	"github.com/perlin-network/noise/examples/basic"
	"github.com/perlin-network/noise/examples/basic/messages"
	"github.com/perlin-network/noise/network"
)

type RingNode struct {
	h        string
	p        int
	ps       []string
	net      *network.Network
	Messages []*messages.BasicMessage
}

func (e *RingNode) Host() string {
	return e.h
}

func (e *RingNode) Port() int {
	return e.p
}

func (e *RingNode) Peers() []string {
	return e.ps
}

func (e *RingNode) Net() *network.Network {
	return e.net
}

func (e *RingNode) SetNet(n *network.Network) {
	e.net = n
}

// Handle implements the network interface callback
func (e *RingNode) Handle(client *network.PeerClient, raw *network.IncomingMessage) error {
	message := raw.Message.(*messages.BasicMessage)

	e.Messages = append(e.Messages, message)

	return nil
}

// PopMessage returns the oldest message from it's buffer and removes it from the list
func (e *RingNode) PopMessage() *messages.BasicMessage {
	if len(e.Messages) == 0 {
		return nil
	}
	var retVal *messages.BasicMessage
	retVal, e.Messages = e.Messages[0], e.Messages[1:]
	return retVal
}

// makes sure the implementation matches the interface at compile time
var _ basic.ClusterNode = (*RingNode)(nil)

func setupNodes() []*RingNode {

	host := "localhost"
	startPort := 5000
	numNodes := 4
	var nodes []*RingNode

	for i := 0; i < numNodes; i++ {
		node := &RingNode{}
		node.h = host
		node.p = startPort + i

		// in a ring, each node is only connected to 2 others
		node.ps = append(node.ps, fmt.Sprintf("%s:%d", node.h, (node.p+1)%numNodes))

		nodes = append(nodes, node)
	}

	return nodes
}

func ringNode2ClusterNode(ring []*RingNode) []basic.ClusterNode {
	var cluster []basic.ClusterNode
	for _, node := range ring {
		cluster = append(cluster, node)
	}
	return cluster
}

func TestRing(t *testing.T) {
	// parse to flags to silence the glog library
	flag.Parse()

	nodes := setupNodes()

	if err := basic.SetupCluster(ringNode2ClusterNode(nodes)); err != nil {
		t.Fatal(err)
	}

	for i, node := range nodes {
		if node.net == nil {
			t.Fatalf("expected %d nodes, but node %d is missing a network", len(nodes), i)
		}

		// get nodes to start talking with each other
		node.Net().Bootstrap(node.Peers()...)

		// TODO: seems there's another race condition with Bootstrap, use a sleep for now
		time.Sleep(1 * time.Second)
	}

	// Broadcast is an asynchronous call to send a message to other nodes
	testMessage := "message from node 0"
	nodes[0].Net().Broadcast(&messages.BasicMessage{Message: testMessage})

	// Simplificiation: message broadcasting is asynchronous, so need the messages to settle
	time.Sleep(1 * time.Second)

	// check if you can send a message from node 1 and will it be received only in node 2,3
	if result := nodes[0].PopMessage(); result != nil {
		t.Errorf("expected nothing in node 0, got %v", result)
	}
	if len(nodes[0].Messages) > 0 {
		t.Errorf("expected no messages buffered in node 0, found: %v", nodes[0].Messages)
	}
	for i := 1; i < len(nodes); i++ {
		if result := nodes[i].PopMessage(); result == nil {
			t.Errorf("expected a message in node %d but it was blank", i)
		} else {
			if result.Message != testMessage {
				t.Errorf("expected message %s in node %d but got %v", testMessage, i, result)
			}
		}
		if len(nodes[i].Messages) > 0 {
			t.Errorf("expected no messages buffered in node %d, found: %v", i, nodes[i].Messages)
		}
	}
}
