package network_test

import (
	"context"
	"testing"
	"time"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/crypto/blake2b"
	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/internal/test/protobuf"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/discovery"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/types/opcode"
)

func init() {
	opcode.RegisterMessageType(opcode.Opcode(1000), &protobuf.TestMessage{})
}

type env struct {
	name        string
	networkType string
	hash        crypto.HashPolicy
	signature   crypto.SignaturePolicy
}

var (
	kcpEnv          = env{name: "kcp-blake2b-ed25519", networkType: "kcp", hash: blake2b.New(), signature: ed25519.New()}
	tcpEnv          = env{name: "tcp-blake2b-ed25519", networkType: "tcp", hash: blake2b.New(), signature: ed25519.New()}
	allEnvs         = []env{kcpEnv, tcpEnv}
	mailboxPluginID = (*MailBoxPlugin)(nil)
)

type testSuite struct {
	t *testing.T
	e env

	builderOptions []network.BuilderOption
	bootstrapNode  *network.Network
	nodes          []*network.Network
	plugins        []*network.Plugin
}

func newTest(t *testing.T, e env, opts ...network.BuilderOption) *testSuite {
	te := &testSuite{
		t:              t,
		e:              e,
		builderOptions: opts,
	}
	return te
}

func (te *testSuite) startBoostrap(numNodes int, plugins ...network.PluginInterface) {
	for i := 0; i < numNodes; i++ {
		builder := network.NewBuilderWithOptions(te.builderOptions...)
		builder.SetKeys(te.e.signature.RandomKeyPair())
		builder.SetAddress(network.FormatAddress(te.e.networkType, "localhost", uint16(network.GetRandomUnusedPort())))

		builder.AddPlugin(new(discovery.Plugin))
		builder.AddPlugin(new(MailBoxPlugin))

		for _, plugin := range plugins {
			builder.AddPlugin(plugin)
		}

		node, err := builder.Build()
		if err != nil {
			te.t.Fatalf("Build() = expected no error, got %v", err)
		}

		if node == nil {
			te.t.Fatalf("Build() expected node to be not nil")
		}

		go node.Listen()

		if i == 0 {
			te.bootstrapNode = node
			node.BlockUntilListening()
		} else {
			te.nodes = append(te.nodes, node)
		}
	}

	for _, node := range te.nodes {
		node.Bootstrap(te.bootstrapNode.Address)
	}

	// wait for nodes to discover other peers
	for _, node := range te.nodes {
		pluginInt, ok := node.Plugin(discovery.PluginID)
		if !ok {
			te.t.Fatalf("Plugin() expected true, got false")
		}

		plugin := pluginInt.(*discovery.Plugin)
		routes := plugin.Routes
		peers := routes.GetPeers()

		for len(peers) < numNodes-1 {
			peers = routes.GetPeers()
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func (te *testSuite) tearDown() {
	for _, node := range te.nodes {
		node.Close()
	}
	te.bootstrapNode.Close()
}

func (te *testSuite) getMailbox(n *network.Network) *MailBoxPlugin {
	if n == nil {
		return nil
	}
	pluginInt, ok := n.Plugin(mailboxPluginID)
	if !ok {
		te.t.Errorf("Plugin(mailboxPluginID) expected true, got false")
	}
	return pluginInt.(*MailBoxPlugin)
}

func (te *testSuite) getPeers(n *network.Network) []peer.ID {
	if n == nil {
		return nil
	}
	pluginInt, ok := n.Plugin(discovery.PluginID)
	if !ok {
		te.t.Errorf("Plugin() expected true, got false")
	}
	plugin := pluginInt.(*discovery.Plugin)
	routes := plugin.Routes
	return routes.GetPeers()
}

// MailBoxPlugin buffers all messages into a mailbox for test validation.
type MailBoxPlugin struct {
	*network.Plugin
	RecvMailbox chan *protobuf.TestMessage
	SendMailbox chan *protobuf.TestMessage
}

// Startup creates a mailbox channel
func (state *MailBoxPlugin) Startup(net *network.Network) {
	state.RecvMailbox = make(chan *protobuf.TestMessage, 100)
	state.SendMailbox = make(chan *protobuf.TestMessage, 100)
}

// Send puts a sent message into the SendMailbox channel
func (state *MailBoxPlugin) Send(ctx *network.PluginContext) error {
	switch msg := ctx.Message().(type) {
	case *protobuf.TestMessage:
		state.SendMailbox <- msg
	}
	return nil
}

// Receive puts a received message into the RecvMailbox channel
func (state *MailBoxPlugin) Receive(ctx *network.PluginContext) error {
	switch msg := ctx.Message().(type) {
	case *protobuf.TestMessage:
		state.RecvMailbox <- msg
	}
	return nil
}

func isIn(address string, ids ...peer.ID) bool {
	for _, a := range ids {
		if a.Address == address {
			return true
		}
	}
	return false
}

func isInAddress(address string, addresses ...string) bool {
	for _, a := range addresses {
		if a == address {
			return true
		}
	}
	return false
}

// Plugin for client test
type clientTestPlugin struct {
	*network.Plugin
}

// Receive takes in *messages.ProxyMessage and replies with *messages.ID
func (p *clientTestPlugin) Receive(ctx *network.PluginContext) error {
	switch msg := ctx.Message().(type) {
	case *protobuf.TestMessage:
		response := &protobuf.TestMessage{Message: msg.Message}
		time.Sleep(time.Duration(msg.Duration) * time.Second)
		ctx.Reply(context.Background(), response)
	}

	return nil
}
