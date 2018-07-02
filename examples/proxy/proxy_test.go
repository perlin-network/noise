package proxy

import (
	"fmt"
	"time"

	"github.com/perlin-network/noise/network/discovery"

	"errors"
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

var ids = make(map[string]int)

// ProxyPlugin buffers all messages into a mailbox for this test.
type ProxyPlugin struct {
	*network.Plugin
	Mailbox chan *messages.ProxyMessage
}

func (n *ProxyPlugin) Startup(net *network.Network) {
	// Create mailbox
	n.Mailbox = make(chan *messages.ProxyMessage, 1)
}

// Handle implements the network interface callback
func (n *ProxyPlugin) Receive(ctx *network.MessageContext) error {
	// Handle the proxy message.
	switch msg := ctx.Message().(type) {
	case *messages.ProxyMessage:
		fmt.Printf("Node %d received a message from node %d.\n", ids[ctx.Network().Address], ids[ctx.Sender().Address])
		n.Mailbox <- msg
		if err := n.ProxyBroadcast(ctx.Network(), ctx.Sender(), msg); err != nil {
			fmt.Println(err)
		}
	}
	return nil
}

// ProxyBroadcast forwards messages to nodes
func (n *ProxyPlugin) ProxyBroadcast(node *network.Network, sender peer.ID, msg *messages.ProxyMessage) error {
	targetID := peer.ID{
		PublicKey: msg.Destination.PublicKey,
		Address:   msg.Destination.Address,
	}

	// check if we've reached the target
	if node.ID.Equals(targetID) {
		// success
		return nil
	}

	plugin, registered := node.Plugin(discovery.PluginID)
	if !registered {
		return errors.New("discovery plugin not registered")
	}

	routes := plugin.(*discovery.Plugin).Routes

	// check if the target is a directly connected peer
	if routes.PeerExists(targetID) {
		// if it is already in the routing table, then send messages directly there
		node.BroadcastByIDs(msg, targetID)
		return nil
	}

	// find the 2 closest peer with the Kademlia table
	closestPeers := routes.FindClosestPeers(targetID, 2)

	// if one of the 2 is the sender, remove it from the list
	for i, peer := range closestPeers {
		if peer.Equals(sender) {
			closestPeers = append(closestPeers[:i], closestPeers[i+1:]...)
			break
		}
	}

	// if no valid peers, may not be able to propagate
	if len(closestPeers) == 0 {
		return fmt.Errorf("could not found route from node %d to node %d", ids[node.Address], ids[targetID.Address])
	}

	// propagate the message it to the closest peer
	node.BroadcastByIDs(msg, closestPeers[0])

	return nil
}

// ExampleProxy demonstrates how to send a message to nodes which do not directly have connections.
// Messages are proxied to closer nodes using the Kademlia table.
func ExampleProxy() {
	numNodes := 5
	sender := 0
	target := numNodes - 1

	var nodes []*network.Network
	var processors []*ProxyPlugin

	for i := 0; i < numNodes; i++ {
		addr := fmt.Sprintf("kcp://%s:%d", host, startPort+i)
		ids[addr] = i

		builder := builders.NewNetworkBuilder()
		builder.SetKeys(crypto.RandomKeyPair())
		builder.SetAddress(addr)

		builder.AddPlugin(discovery.PluginID, new(discovery.Plugin))

		processors = append(processors, new(ProxyPlugin))
		builder.AddPlugin("proxy", processors[i])

		node, err := builder.Build()
		if err != nil {
			fmt.Println(err)
		}
		nodes = append(nodes, node)

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
	time.Sleep(300 * time.Millisecond)

	fmt.Println("Nodes setup as a line topology.")

	// Broadcast is an asynchronous call to send a message to other nodes
	expected := &messages.ProxyMessage{
		Message: fmt.Sprintf("This is a proxy message from Node %d", sender),
		Destination: &messages.ID{
			Address:   nodes[target].ID.Address,
			PublicKey: nodes[target].ID.PublicKey,
		},
	}
	processors[sender].ProxyBroadcast(nodes[sender], nodes[sender].ID, expected)

	fmt.Printf("Node %d sent out a message to node %d.\n", sender, target)

	// Check if message was received by target node.
	select {
	case received := <-processors[target].Mailbox:
		if received.Message != expected.Message {
			fmt.Printf("Expected message (%v) to be received by node %d but got (%v).\n", expected, target, received)
		} else {
			fmt.Printf("Node %d successfully proxied a message to node %d.\n", sender, target)
		}
	case <-time.After(1 * time.Second):
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
