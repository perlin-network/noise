package topologies

import (
	"fmt"
	"time"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/examples/basic/messages"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/builders"
)

const (
	host = "127.0.0.1"
)

// TopologyProcessor implements the message handler
type TopologyProcessor struct {
	Mailbox chan *messages.BasicMessage
}

// Handle implements the network interface callback
func (n *TopologyProcessor) Handle(ctx *network.MessageContext) error {
	message := ctx.Message().(*messages.BasicMessage)
	n.Mailbox <- message
	return nil
}

// broadcastTest will broadcast a message from the sender node, checks if the right peers receive it
func broadcastTest(nodes []*network.Network, processors []*TopologyProcessor, peers map[string]map[string]struct{}, sender int) {
	timeout := 250 * time.Millisecond

	// Broadcast is an asynchronous call to send a message to other nodes
	expected := fmt.Sprintf("This is a broadcasted message from Node %d", sender)
	nodes[sender].Broadcast(&messages.BasicMessage{Message: expected})

	// check the messages
	for i := 0; i < len(nodes); i++ {
		if _, isPeer := peers[nodes[i].Address][nodes[sender].Address]; !isPeer || i == sender {
			// if not a peer or not the sender, should not receive anything
			select {
			case received := <-processors[sender].Mailbox:
				fmt.Printf("Expected nothing in sending node %d, got %v\n", sender, received)
			case <-time.After(timeout):
				// this is the good case, don't want to receive anything
			}
		} else {
			// this is a connected peer, it should receive something
			select {
			case received := <-processors[i].Mailbox:
				// this is a receiving node, it should have just the one message buffered up
				if received.Message != expected {
					fmt.Printf("Expected message '%s' for node %d --> %d, but got %v\n", expected, sender, i, received)
				}
			case <-time.After(timeout):
				fmt.Printf("Expected a message for node %d --> %d, but it timed out\n", sender, i)
			}
		}
	}
}

func ExampleProxy() {
	numNodes := 5
	host := "127.0.0.1"
	startPort := 5070

	//----------------

	var nodes []*network.Network
	var processors []*TopologyProcessor

	for i := 0; i < numNodes; i++ {
		builder := &builders.NetworkBuilder{}
		builder.SetKeys(crypto.RandomKeyPair())
		builder.SetAddress(fmt.Sprintf("kcp://%s:%d", host, startPort+i))

		// excluding peer discovery to test non-fully connected topology
		//discovery.BootstrapPeerDiscovery(builder)

		processor := &TopologyProcessor{Mailbox: make(chan *messages.BasicMessage, 1)}
		builder.AddProcessor((*messages.BasicMessage)(nil), processor)

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

	timeout := 250 * time.Millisecond
	sender := 0
	dest := numNodes - 1

	// Broadcast is an asynchronous call to send a message to other nodes
	expected := fmt.Sprintf("This is a broadcasted message from Node %d", sender)
	nodes[sender].Broadcast(&messages.BasicMessage{Message: expected})

	// check the messages
	for i := 0; i < len(nodes); i++ {
		if _, isPeer := peers[nodes[i].Address][nodes[sender].Address]; !isPeer || i == sender {
			// if not a peer or not the sender, should not receive anything
			select {
			case received := <-processors[sender].Mailbox:
				fmt.Printf("Expected nothing in sending node %d, got %v\n", sender, received)
			case <-time.After(timeout):
				// this is the good case, don't want to receive anything
			}
		} else {
			// this is a connected peer, it should receive something
			select {
			case received := <-processors[i].Mailbox:
				// this is a receiving node, it should have just the one message buffered up
				if received.Message != expected {
					fmt.Printf("Expected message '%s' for node %d --> %d, but got %v\n", expected, sender, i, received)
				}
			case <-time.After(timeout):
				fmt.Printf("Expected a message for node %d --> %d, but it timed out\n", sender, i)
			}
		}
	}

	fmt.Printf("Messages sent from each node.")

	// Output:
	// Nodes setup as a line topology.
	// Messages sent from each node.
}
