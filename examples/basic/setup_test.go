package basic

import (
	"flag"
	"fmt"
	"time"

	"github.com/perlin-network/noise/examples/basic/messages"
	"github.com/perlin-network/noise/network"
)

type BasicNode struct {
	ClusterNode
	h        string
	p        int
	ps       []string
	net      *network.Network
	Messages []*messages.BasicMessage
}

func (e *BasicNode) Host() string {
	return e.h
}

func (e *BasicNode) Port() int {
	return e.p
}

func (e *BasicNode) Peers() []string {
	return e.ps
}

func (e *BasicNode) Net() *network.Network {
	return e.net
}

func (e *BasicNode) SetNet(n *network.Network) {
	e.net = n
}

// Handle implements the network interface callback
func (e *BasicNode) Handle(client *network.PeerClient, raw *network.IncomingMessage) error {
	message := raw.Message.(*messages.BasicMessage)

	e.Messages = append(e.Messages, message)

	return nil
}

// makes sure the implementation matches the interface at compile time
var _ ClusterNode = (*BasicNode)(nil)
var blockTimeout = 10 * time.Second

// PopMessage returns the oldest message from it's buffer and removes it from the list
func (e *BasicNode) PopMessage() *messages.BasicMessage {
	if len(e.Messages) == 0 {
		return nil
	}
	var retVal *messages.BasicMessage
	retVal, e.Messages = e.Messages[0], e.Messages[1:]
	return retVal
}

// ExampleSetupClusters - example of how to use SetupClusters() to automate tests
func ExampleSetupClusters() {
	// parse to flags to silence the glog library
	flag.Parse()

	host := "localhost"
	startPort := 5000
	numNodes := 3
	var nodes []*BasicNode
	var cn []ClusterNode
	var peers []string

	for i := 0; i < numNodes; i++ {
		node := &BasicNode{}
		node.h = host
		node.p = startPort + i

		nodes = append(nodes, node)
		cn = append(cn, node)
		peers = append(peers, fmt.Sprintf("%s:%d", node.h, node.p))
	}

	for _, node := range nodes {
		node.ps = peers
	}

	if err := SetupCluster(cn); err != nil {
		fmt.Print(err)
		return
	}

	// After all the nodes are started, get them to start talking with each other
	for i, node := range nodes {
		if node.net == nil {
			fmt.Printf("expected %d nodes, but node %d is missing a network", len(nodes), i)
			return
		}

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
		fmt.Printf("expected nothing in node 0, got %v", result)
		return
	}
	for i := 1; i < len(nodes); i++ {
		if result := nodes[i].PopMessage(); result == nil {
			fmt.Printf("expected a message in node %d but it was blank", i)
			return
		} else {
			if result.Message != testMessage {
				fmt.Printf("expected message %s in node %d but got %v", testMessage, i, result)
				return
			}
		}
	}

	fmt.Printf("success")
	// Output: success
}
