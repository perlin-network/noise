package proxy

import (
	"fmt"
	"time"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/examples/proxy/messages"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/builders"
)

const (
	host = "127.0.0.1"
)

// TopologyProcessor implements the message handler
type TopologyProcessor struct {
	Mailbox chan *messages.ProxyMessage
}

// Handle implements the network interface callback
func (n *TopologyProcessor) Handle(ctx *network.MessageContext) error {
	message := ctx.Message().(*messages.ProxyMessage)
	n.Mailbox <- message
	return nil
}

func ExampleProxy() {
	numNodes := 5
	host := "127.0.0.1"
	startPort := 20070

	//----------------

	var nodes []*network.Network
	var processors []*TopologyProcessor

	for i := 0; i < numNodes; i++ {
		builder := &builders.NetworkBuilder{}
		builder.SetKeys(crypto.RandomKeyPair())
		builder.SetAddress(fmt.Sprintf("kcp://%s:%d", host, startPort+i))

		// excluding peer discovery to test non-fully connected topology
		//discovery.BootstrapPeerDiscovery(builder)

		processor := &TopologyProcessor{Mailbox: make(chan *messages.ProxyMessage, 1)}
		builder.AddProcessor((*messages.ProxyMessage)(nil), processor)

		node, err := builder.BuildNetwork()
		if err != nil {
			fmt.Println(err)
		}
		nodes = append(nodes, node)
		processors = append(processors, processor)

		go node.Listen()
	}

	// make sure all the servers are listening
	for _, node := range nodes {
		node.BlockUntilListening()
	}

	//----------------

	for i := 0; i < numNodes; i++ {
		var peerList []string
		if i > 0 {
			peerList = append(peerList, nodes[i-1].Address)
		}
		if i < numNodes-1 {
			peerList = append(peerList, nodes[i+1].Address)
		}

		// get nodes to start talking with each other
		nodes[i].Bootstrap(peerList...)

	}

	// Wait for all nodes to finish discovering other peers.
	time.Sleep(time.Duration(100*len(nodes)) * time.Millisecond)

	fmt.Println("Nodes setup as a line topology.")

	//----------------

	sender := 0
	target := numNodes - 1
	target = 1

	// Broadcast is an asynchronous call to send a message to other nodes
	expectedMsg := &messages.ProxyMessage{
		Message: fmt.Sprintf("This is a proxy message from Node %d", sender),
	}
	nodes[sender].Broadcast(expectedMsg)

	fmt.Printf("Node %d sent out a message to node %d.\n", sender, target)

	// Check if message was received by target node.
	select {
	case received := <-processors[target].Mailbox:
		if received.Message != expectedMsg.Message {
			fmt.Printf("Expected message (%v) to be received by node %d but got (%v).\n", expectedMsg, target, received)
		} else {
			fmt.Printf("Node %d received a message from Node %d.\n", target, sender)
		}
	case <-time.After(time.Duration(numNodes+1) * time.Second):
		fmt.Printf("Timed out attempting to receive message from Node %d.\n", sender)
	}

	// Output:
	// Nodes setup as a line topology.
	// Node 0 sent out a message to node 4.
	// Node 1 received a message from Node 0.
	// Node 2 received a message from Node 1.
	// Node 3 received a message from Node 2.
	// Node 4 received a message from Node 3.
}
