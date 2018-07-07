package builders

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/perlin-network/noise/crypto/signing/ed25519"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/discovery"
	"github.com/perlin-network/noise/protobuf"
)

var (
	keys     = ed25519.RandomKeyPair()
	host     = "localhost"
	protocol = "tcp"
	port     = uint16(12345)
)

type MockPlugin struct {
	*network.Plugin
}

func (*MockPlugin) Receive(ctx *network.PluginContext) error {
	switch ctx.Message().(type) {
	case *protobuf.Ping:
		err := ctx.Reply(&protobuf.Pong{})

		if err != nil {
			return err
		}
	}

	return nil
}

func buildNetwork(port uint16) (*network.Network, error) {
	builder := NewNetworkBuilder()
	builder.SetKeys(keys)
	builder.SetAddress(
		fmt.Sprintf("%s://%s:%d", protocol, host, port),
	)

	builder.AddPluginWithPriority(1, new(discovery.Plugin))
	builder.AddPluginWithPriority(2, new(MockPlugin))

	return builder.Build()
}

func TestBuildNetwork(t *testing.T) {
	_, err := buildNetwork(port)

	if err != nil {
		t.Fatal(err)
	}
}

func TestSetters(t *testing.T) {
	net, err := buildNetwork(port)
	if err != nil {
		t.Fatal(err)
	}

	if net.Address != fmt.Sprintf("tcp://127.0.0.1:%d", port) { // Unified address.
		t.Fatalf("address is wrong: expected %s but got %s", fmt.Sprintf("tcp://127.0.0.1:%d", port), net.Address)
	}

	if !bytes.Equal(net.Keys.PrivateKey, keys.PrivateKey) {
		t.Fatalf("private key is wrong")
	}

	if !bytes.Equal(net.Keys.PublicKey, keys.PublicKey) {
		t.Fatalf("public key is wrong")
	}

}

func TestPeers(t *testing.T) {
	var nodes []*network.Network
	addresses := []string{"tcp://127.0.0.1:12345", "tcp://127.0.0.1:12346", "tcp://127.0.0.1:12347"}
	peers := [][2]int{{1, 2}, {0, 2}, {0, 1}}

	for i := 0; i < 3; i++ {
		net, err := buildNetwork(port + uint16(i))
		if err != nil {
			t.Fatal(err)
		}

		go net.Listen()
		net.BlockUntilListening()

		if i != 0 {
			net.Bootstrap(addresses[0])
		}
		nodes = append(nodes, net)
	}

	for x := 0; x < len(nodes); x++ {
		for y := 0; y < 2; y++ {
			if _, err := nodes[x].Client(addresses[peers[x][y]]); err != nil {
				t.Fatalf("nodes[%d] missing peer: %s", x, addresses[peers[x][y]])
			}
		}
	}
}

// Broadcast functions are tested through examples.
