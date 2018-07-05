package backoff

import (
	"flag"
	"testing"
	"time"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/examples/basic/messages"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/builders"
	"github.com/perlin-network/noise/network/discovery"
	"github.com/pkg/errors"
)

const (
	numNodes  = 2
	protocol  = "tcp"
	host      = "127.0.0.1"
	startPort = 21200
)

// BasicPlugin buffers all messages into a mailbox for this test.
type BasicPlugin struct {
	*network.Plugin
	Mailbox chan *messages.BasicMessage
}

func (state *BasicPlugin) Startup(net *network.Network) {
	// Create mailbox
	state.Mailbox = make(chan *messages.BasicMessage, 1)
}

func (state *BasicPlugin) Receive(ctx *network.MessageContext) error {
	switch msg := ctx.Message().(type) {
	case *messages.BasicMessage:
		state.Mailbox <- msg
	}
	return nil
}

func broadcastAndCheck(nodes []*network.Network, plugins []*BasicPlugin) error {
	// Broadcast out a message from Node 0.
	expected := "This is a broadcasted message from Node 0."
	nodes[0].Broadcast(&messages.BasicMessage{Message: expected})

	// Check if message was received by other nodes.
	for i := 1; i < len(nodes); i++ {
		select {
		case received := <-plugins[i].Mailbox:
			if received.Message != expected {
				return errors.Errorf("Expected message %s to be received by node %d but got %v\n", expected, i, received.Message)
			}
		case <-time.After(2 * time.Second):
			return errors.Errorf("Timed out attempting to receive message from Node 0.\n")
		}
	}

	return nil
}

func newNode(i int, reconnecting bool) (*network.Network, *BasicPlugin, error) {
	builder := builders.NewNetworkBuilder()
	builder.SetKeys(crypto.RandomKeyPair())
	builder.SetAddress(network.FormatAddress(protocol, host, uint16(startPort+i)))

	if !reconnecting {
		builder.AddPlugin(new(discovery.Plugin))
	}
	builder.AddPlugin(new(Plugin))

	plugin := new(BasicPlugin)
	builder.AddPlugin(plugin)

	node, err := builder.Build()
	if err != nil {
		return nil, nil, err
	}

	go node.Listen()

	node.BlockUntilListening()

	// Bootstrap to Node 0.
	if !reconnecting && i != 0 {
		node.Bootstrap(network.FormatAddress(protocol, host, uint16(startPort)))
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
	var plugins []*BasicPlugin

	for i := 0; i < numNodes; i++ {
		node, plugin, err := newNode(i, false)
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

	// disconnect node 2
	nodes[1].HardDisconnect()

	// wait until about the middle of the backoff period
	time.Sleep(initialDelay + (defaultMinInterval * (defaultMaxAttempts / 2)))

	// tests that broadcasting fails
	if err := broadcastAndCheck(nodes, plugins); err == nil {
		t.Fatalf("On disconnect, expected the broadcast to fail")
	}

	// recreate the second node to the cluster
	node, plugin, err := newNode(1, true)
	if err != nil {
		t.Fatal(err)
	}
	nodes[1] = node
	plugins[1] = plugin

	// wait for reconnection
	time.Sleep(3 * time.Second)

	// broad cast should be working again
	if err := broadcastAndCheck(nodes, plugins); err != nil {
		t.Fatal(err)
	}
}
