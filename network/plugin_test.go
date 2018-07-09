package network

import (
	"fmt"
	"testing"
	"time"

	"github.com/perlin-network/noise/crypto/signing/ed25519"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/builders"
	"github.com/perlin-network/noise/network/discovery"
)

var (
	startup        = 0
	receive        = 0
	cleanup        = 0
	peerConnect    = 0
	peerDisconnect = 0
)

type MockPlugin struct {
	*network.Plugin
}

func (state *MockPlugin) Startup(net *network.Network) {
	startup++
}

func (state *MockPlugin) Receive(ctx *network.PluginContext) error {
	receive++
	return nil
}
func (state *MockPlugin) Cleanup(net *network.Network) {
	cleanup++
}
func (state *MockPlugin) PeerConnect(client *network.PeerClient) {
	peerConnect++
}
func (state *MockPlugin) PeerDisconnect(client *network.PeerClient) {
	peerDisconnect++
}

func TestPluginHooks(t *testing.T) {
	host := "localhost"
	port := 10000
	var nodes []*network.Network
	nodeCount := 4
	for i := 0; i < nodeCount; i++ {
		builder := builders.NewNetworkBuilder()
		builder.SetKeys(ed25519.RandomKeyPair())
		builder.SetAddress(FormatAddress("tcp", host, uint16(port+i)))
		builder.AddPlugin(new(discovery.Plugin))
		builder.AddPlugin(new(MockPlugin))

		node, err := builder.Build()
		if err != nil {
			fmt.Println(err)
		}

		go node.Listen()

		nodes = append(nodes, node)
	}
	for _, node := range nodes {
		node.BlockUntilListening()
	}
	for i, node := range nodes {
		if i != 0 {
			node.Bootstrap(nodes[0].Address)
		}
	}
	time.Sleep(1 * time.Second)
	for _, node := range nodes {
		node.Close()
	}
	time.Sleep(1 * time.Second)

	if startup != nodeCount {
		t.Fatalf("startup hooks error, got: %d, expected: %d", startup, nodeCount)
	}
	if receive < nodeCount*2 { //Cannot in specific time
		t.Fatalf("receive hooks error, got: %d, expected at least: %d", receive, nodeCount*2)
	}
	if cleanup != nodeCount {
		t.Fatalf("cleanup hooks error, got: %d, expected: %d", cleanup, nodeCount)
	}
	if peerConnect < nodeCount*2 {
		t.Fatalf("connect hooks error, got: %d, expected at least: %d", peerConnect, nodeCount*2)
	}
	if peerDisconnect < nodeCount*2 {
		t.Fatalf("disconnect hooks error, got: %d, expected at least: %d", peerDisconnect, nodeCount*2)
	}
}
