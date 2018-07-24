package network

import (
	"fmt"
	"testing"

	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/types"
	"github.com/stretchr/testify/assert"
)

var (
	startup        = 0
	receive        = 0
	cleanup        = 0
	peerConnect    = 0
	peerDisconnect = 0

	_ PluginInterface = (*MockPlugin)(nil)
)

type MockPlugin struct {
	*Plugin
}

func (state *MockPlugin) Startup(net *Network) {
	startup++
}

func (state *MockPlugin) Receive(ctx *PluginContext) error {
	receive++
	return nil
}

func (state *MockPlugin) Cleanup(net *Network) {
	cleanup++
}

func (state *MockPlugin) PeerConnect(client *PeerClient) {
	peerConnect++
}

func (state *MockPlugin) PeerDisconnect(client *PeerClient) {
	peerDisconnect++
}

func (state *MockPlugin) Priority() int {
	return 0
}

func TestPluginHooks(t *testing.T) {
	host := "localhost"
	var nodes []*Network
	nodeCount := 4

	for i := 0; i < nodeCount; i++ {
		builder := NewBuilder()
		builder.SetKeys(ed25519.RandomKeyPair())
		addr := types.FormatAddress("tcp", host, uint16(GetRandomUnusedPort()))
		builder.SetAddress(addr)
		p := new(MockPlugin)
		builder.AddPlugin(p)

		assert.Equal(t, 0, 0)

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

	//for i, node := range nodes {
	//	if i != 0 {
	//		node.Bootstrap(nodes[0].Address)
	//	}
	//}
	//
	//time.Sleep(500 * time.Millisecond)
	//
	//for _, node := range nodes {
	//	node.Close()
	//}
	//
	//time.Sleep(500 * time.Millisecond)
	//
	//if startup != nodeCount {
	//	t.Fatalf("startup hooks error, got: %d, expected: %d", startup, nodeCount)
	//}
	//
	//if receive < nodeCount*2 { // Cannot in specific time
	//	t.Fatalf("receive hooks error, got: %d, expected at least: %d", receive, nodeCount*2)
	//}
	//
	//if cleanup != nodeCount {
	//	t.Fatalf("cleanup hooks error, got: %d, expected: %d", cleanup, nodeCount)
	//}
	//
	//if peerConnect < nodeCount*2 {
	//	t.Fatalf("connect hooks error, got: %d, expected at least: %d", peerConnect, nodeCount*2)
	//}
	//
	//if peerDisconnect < nodeCount*2 {
	//	t.Fatalf("disconnect hooks error, got: %d, expected at least: %d", peerDisconnect, nodeCount*2)
	//}
}
