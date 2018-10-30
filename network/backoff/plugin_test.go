package backoff

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/examples/basic/messages"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/discovery"
	"github.com/perlin-network/noise/types/opcode"

	"github.com/pkg/errors"
)

const (
	numNodes = 2
	protocol = "tcp"
	host     = "127.0.0.1"
)

var (
	idToPort = make(map[int]uint16)
	keys     = make(map[string]*crypto.KeyPair)
)

// mockPlugin buffers all messages into a mailbox for this test.
type mockPlugin struct {
	*network.Plugin
	Mailbox chan *messages.BasicMessage
}

// Startup implements the network interface callback
func (state *mockPlugin) Startup(net *network.Network) {
	// Create mailbox
	state.Mailbox = make(chan *messages.BasicMessage, 1)
}

// Receive implements the network interface callback
func (state *mockPlugin) Receive(ctx *network.PluginContext) error {
	switch msg := ctx.Message().(type) {
	case *messages.BasicMessage:
		state.Mailbox <- msg
	}
	return nil
}

// broadcastAndCheck will send a message from node 0 to other nodes and check if it's received
func broadcastAndCheck(nodes []*network.Network, plugins []*mockPlugin) error {
	// Broadcast out a message from Node 0.
	expected := "This is a broadcasted message from Node 0."
	nodes[0].Broadcast(context.Background(), &messages.BasicMessage{Message: expected})

	// Check if message was received by other nodes.
	for i := 1; i < len(nodes); i++ {
		select {
		case received := <-plugins[i].Mailbox:
			if received.Message != expected {
				return errors.Errorf("Expected message %s to be received by node %d but got %v", expected, i, received.Message)
			}
		case <-time.After(2 * time.Second):
			return errors.Errorf("Timed out attempting to receive message from Node 0.")
		}
	}

	return nil
}

// newNode creates a new node and and adds it to the cluster, allows adding certain plugins if needed
func newNode(i int, addDiscoveryPlugin bool, addBackoffPlugin bool) (*network.Network, *mockPlugin, error) {
	port := uint16(0)
	ok := false
	// get random port
	if port, ok = idToPort[i]; !ok {
		port = uint16(network.GetRandomUnusedPort())
		idToPort[i] = port
	}
	// restore the key if it was created in the past
	addr := network.FormatAddress(protocol, host, port)
	if _, ok := keys[addr]; !ok {
		keys[addr] = ed25519.RandomKeyPair()
	}

	builder := network.NewBuilder()
	builder.SetKeys(keys[addr])
	builder.SetAddress(addr)

	if addDiscoveryPlugin {
		builder.AddPlugin(new(discovery.Plugin))
	}
	if addBackoffPlugin {
		builder.AddPlugin(New())
	}

	plugin := new(mockPlugin)
	builder.AddPlugin(plugin)

	node, err := builder.Build()
	if err != nil {
		return nil, nil, err
	}

	go node.Listen()

	node.BlockUntilListening()

	// Bootstrap to Node 0
	if addDiscoveryPlugin && i != 0 {
		node.Bootstrap(network.FormatAddress(protocol, host, uint16(idToPort[0])))
	}

	return node, plugin, nil
}

// TestPlugin tests the functionality of the exponential backoff as a plugin.
func TestPlugin(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping backoff plugin test in short mode")
	}

	flag.Parse()

	var nodes []*network.Network
	var plugins []*mockPlugin

	opcode.RegisterMessageType(opcode.Opcode(1000), &messages.BasicMessage{})

	for i := 0; i < numNodes; i++ {
		node, plugin, err := newNode(i, true, i == 0)
		if err != nil {
			t.Error(err)
		}
		plugins = append(plugins, plugin)
		nodes = append(nodes, node)
	}

	// Wait for all nodes to finish discovering other peers.
	time.Sleep(1 * time.Second)

	// chack that broadcasts are working
	if err := broadcastAndCheck(nodes, plugins); err != nil {
		t.Fatal(err)
	}

	// disconnect the node from the cluster
	nodes[1].Close()

	// wait until about the middle of the backoff period
	time.Sleep(defaultPluginInitialDelay + defaultMinInterval*2)

	// tests that broadcasting fails
	if err := broadcastAndCheck(nodes, plugins); err == nil {
		t.Fatalf("On disconnect, expected the broadcast to fail")
	}

	// recreate the second node and add back to the cluster
	node, plugin, err := newNode(1, false, false)
	if err != nil {
		t.Fatal(err)
	}
	nodes[1] = node
	plugins[1] = plugin

	// wait for reconnection
	time.Sleep(5 * time.Second)

	// broad cast should be working again
	if err := broadcastAndCheck(nodes, plugins); err != nil {
		t.Fatal(err)
	}
}
