package proxy

import (
	"context"
	"fmt"
	"time"

	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/examples/proxy/messages"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/discovery"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/types/opcode"

	"github.com/pkg/errors"
)

const (
	host      = "127.0.0.1"
	startPort = 20070
)

// A map of addresses to node IDs.
var ids = make(map[string]int)

// ProxyPlugin buffers all messages into a mailbox for this test.
type ProxyPlugin struct {
	*network.Plugin
	Mailbox chan *messages.ProxyMessage
}

func (n *ProxyPlugin) Startup(net *network.Network) {
	// Create mailbox.
	n.Mailbox = make(chan *messages.ProxyMessage, 1)
}

// Handle implements the network interface callback
func (n *ProxyPlugin) Receive(ctx *network.PluginContext) error {
	// Handle the proxy message.
	switch msg := ctx.Message().(type) {
	case *messages.ProxyMessage:
		n.Mailbox <- msg

		//fmt.Fprintf(os.Stderr, "Node %d received a message from node %d.\n", ids[ctx.Network().Address], ids[ctx.Sender().Address])

		if err := n.ProxyBroadcast(ctx.Network(), ctx.Sender(), msg); err != nil {
			panic(err)
		}
	}
	return nil
}

// ProxyBroadcast proxies a message until it reaches a target ID destination.
func (n *ProxyPlugin) ProxyBroadcast(node *network.Network, sender peer.ID, msg *messages.ProxyMessage) error {
	targetID := peer.ID{
		Id:      msg.Destination.Id,
		Address: msg.Destination.Address,
	}

	// Check if we are the target.
	if node.ID.Equals(targetID) {
		return nil
	}

	plugin, registered := node.Plugin(discovery.PluginID)
	if !registered {
		return errors.New("discovery plugin not registered")
	}

	routes := plugin.(*discovery.Plugin).Routes

	// If the target is in our routing table, directly proxy the message to them.
	if routes.PeerExists(targetID) {
		node.BroadcastByAddresses(context.Background(), msg, targetID.Address)
		return nil
	}

	// Find the 2 closest peers from a nodes point of view (might include us).
	closestPeers := routes.FindClosestPeers(targetID, 2)

	// Remove sender from the list.
	for i, id := range closestPeers {
		if id.Equals(sender) {
			closestPeers = append(closestPeers[:i], closestPeers[i+1:]...)
			break
		}
	}

	// Seems we have ran out of peers to attempt to propagate to.
	if len(closestPeers) == 0 {
		return errors.Errorf("could not found route from node %d to node %d", ids[node.Address], ids[targetID.Address])
	}

	// Propagate message to the closest peer.
	node.BroadcastByAddresses(context.Background(), msg, closestPeers[0].Address)
	return nil
}

// ExampleProxyPlugin demonstrates how to send a message to nodes which do not directly have connections
// to their desired messaging target.
//
// Messages are proxied to closer nodes using the Kademlia routing table.
func ExampleProxyPlugin() {
	numNodes := 5
	sender := 0
	target := numNodes - 1

	var nodes []*network.Network
	var plugins []*ProxyPlugin

	opcode.RegisterMessageType(opcode.Opcode(1000), &messages.ProxyMessage{})

	for i := 0; i < numNodes; i++ {
		addr := fmt.Sprintf("tcp://%s:%d", host, startPort+i)
		ids[addr] = i

		builder := network.NewBuilder()
		builder.SetKeys(ed25519.RandomKeyPair())
		builder.SetAddress(addr)

		// DisablePong will preserve the line topology
		builder.AddPlugin(&discovery.Plugin{
			DisablePong: true,
		})

		plugins = append(plugins, new(ProxyPlugin))
		builder.AddPlugin(plugins[i])

		node, err := builder.Build()
		if err != nil {
			fmt.Println(err)
		}
		nodes = append(nodes, node)

		go node.Listen()
	}

	// Make sure all nodes are listening for incoming peers.
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

		// Bootstrap nodes to their assignd peers.
		nodes[i].Bootstrap(peerList...)

	}

	// Wait for all nodes to finish discovering other peers.
	time.Sleep(1 * time.Second)

	fmt.Println("Nodes setup as a line topology.")

	// Broadcast is an asynchronous call to send a message to other nodes
	expected := &messages.ProxyMessage{
		Message: fmt.Sprintf("This is a proxy message from Node %d", sender),
		Destination: &messages.ID{
			Address: nodes[target].ID.Address,
			Id:      nodes[target].ID.Id,
		},
	}
	plugins[sender].ProxyBroadcast(nodes[sender], nodes[sender].ID, expected)

	fmt.Printf("Node %d sent out a message targeting for node %d.\n", sender, target)

	// Check if message was received by target node.
	select {
	case received := <-plugins[target].Mailbox:
		if received.Message != expected.Message {
			fmt.Printf("Expected message (%v) to be received by node %d but got (%v).\n", expected, target, received)
		} else {
			fmt.Printf("Node %d successfully proxied a message to node %d.\n", sender, target)
		}
	case <-time.After(3 * time.Second):
		fmt.Printf("Timed out attempting to receive message from Node %d.\n", sender)
	}

	// Output:
	// Nodes setup as a line topology.
	// Node 0 sent out a message targeting for node 4.
	// Node 0 successfully proxied a message to node 4.

}
