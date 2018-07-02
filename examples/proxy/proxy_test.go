package proxy

import (
	"fmt"
	"time"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/examples/proxy/messages"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/builders"
	"github.com/perlin-network/noise/peer"
)

const (
	host      = "127.0.0.1"
	startPort = 20070
)

var addrToID map[string]int

// MockProcessor implements the message handler
type MockProcessor struct {
	Mailbox chan *messages.ProxyMessage
}

// Handle implements the network interface callback
func (n *MockProcessor) Handle(ctx *network.MessageContext) error {
	message := ctx.Message().(*messages.ProxyMessage)

	fmt.Printf("Node %d received a message from node %d.\n", addrToID[ctx.Network().Address], addrToID[ctx.Sender().Address])

	n.Mailbox <- message

	if err := n.ProxyBroadcast(ctx.Network(), ctx.Sender(), message); err != nil {
		fmt.Println(err)
	}
	return nil
}

func (n *MockProcessor) ProxyBroadcast(node *network.Network, sender peer.ID, msg *messages.ProxyMessage) error {
	targetID := peer.ID{
		PublicKey: msg.Destination.PublicKey,
		Address:   msg.Destination.Address,
	}
	if node.ID.Equals(targetID) {
		// success
		return nil
	}

	// find closest node
	if node.Routes.PeerExists(targetID) {
		// if it is already in the routing table, then send messages directly there
		node.BroadcastByIDs(msg, targetID)
		return nil
	}

	// find a closest peer that is not the sender
	closestPeers := node.Routes.FindClosestPeers(targetID, 2)

	// remove the sender from the closest peers list
	for i, peer := range closestPeers {
		if peer.Equals(sender) {
			closestPeers = append(closestPeers[:i], closestPeers[i+1:]...)
			break
		}
	}

	if len(closestPeers) == 0 {
		return fmt.Errorf("could not found route from node %d to node %d", addrToID[node.Address], addrToID[targetID.Address])
	}

	// propagate the message it to the closest peer
	node.BroadcastByIDs(msg, closestPeers[0])

	return nil
}

func ExampleProxy() {
	numNodes := 5
	sender := 0
	target := numNodes - 1

	var nodes []*network.Network
	var processors []*MockProcessor
	addrToID = map[string]int{}

	for i := 0; i < numNodes; i++ {
		addr := fmt.Sprintf("kcp://%s:%d", host, startPort+i)
		addrToID[addr] = i

		builder := builders.NewNetworkBuilder()
		builder.SetKeys(crypto.RandomKeyPair())
		builder.SetAddress(addr)

		// excluding peer discovery to test non-fully connected topology
		//discovery.BootstrapPeerDiscovery(builder)

		processor := &MockProcessor{
			Mailbox: make(chan *messages.ProxyMessage, 1),
		}
		builder.AddProcessor((*messages.ProxyMessage)(nil), processor)

		node, err := builder.Build()
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

	// Broadcast is an asynchronous call to send a message to other nodes
	expectedMsg := &messages.ProxyMessage{
		Message: fmt.Sprintf("This is a proxy message from Node %d", sender),
		Destination: &messages.ID{
			Address:   nodes[target].ID.Address,
			PublicKey: nodes[target].ID.PublicKey,
		},
	}
	processors[sender].ProxyBroadcast(nodes[sender], nodes[sender].ID, expectedMsg)

	fmt.Printf("Node %d sent out a message to node %d.\n", sender, target)

	// Check if message was received by target node.
	select {
	case received := <-processors[target].Mailbox:
		if received.Message != expectedMsg.Message {
			fmt.Printf("Expected message (%v) to be received by node %d but got (%v).\n", expectedMsg, target, received)
		} else {
			fmt.Printf("Node %d successfully proxied a message to node %d.\n", sender, target)
		}
	case <-time.After(time.Duration(numNodes+1) * time.Second):
		fmt.Printf("Timed out attempting to receive message from Node %d.\n", sender)
	}

	// Output:
	// Nodes setup as a line topology.
	// Node 0 sent out a message to node 4.
	// Node 1 received a message from node 0.
	// Node 2 received a message from node 1.
	// Node 3 received a message from node 2.
	// Node 4 received a message from node 3.
	// Node 0 successfully proxied a message to node 4.

}
