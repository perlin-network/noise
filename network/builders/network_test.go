package builders

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/golang/glog"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
)

var (
	keys = crypto.RandomKeyPair()
	host = "localhost"
	port = uint16(12345)
)

// MockProcessor so to keep independency to incoming.go and outgoing.go
type MockProcessor struct{}

func (p *MockProcessor) Handle(ctx *network.MessageContext) error {
	// Send handshake response to peer.
	err := ctx.Reply(&protobuf.HandshakeResponse{})

	if err != nil {
		glog.Error(err)
		return err
	}
	return nil
}

func buildNetwork(port uint16) (*network.Network, error) {
	builder := &NetworkBuilder{}
	builder.SetKeys(keys)
	builder.SetHost(host)
	builder.SetPort(port)

	builder.AddProcessor((*protobuf.HandshakeRequest)(nil), new(MockProcessor))

	return builder.BuildNetwork()
}

func TestBuildNetwork(t *testing.T) {
	_, err := buildNetwork(port)

	if err != nil {
		t.Fatalf("error: %v", err)
	}
}

func TestSetters(t *testing.T) {
	net, _ := buildNetwork(port)
	if net.Address() != fmt.Sprintf("127.0.0.1:%d", port) { // Unified address.
		t.Fatalf("address is wrong: expected %s but got %s", fmt.Sprintf("127.0.0.1:%d", port), net.Address())
	}
	if net.Host != fmt.Sprintf("127.0.0.1") { // Unified address.
		t.Fatal("host is wrong")
	}

	comparee := peer.CreateID("localhost:12345", keys.PublicKey)
	if !net.ID.Equals(comparee) {
		t.Fatalf("address is wrong %s", net.ID)
	}

	if !bytes.Equal(net.Keys.PrivateKey, keys.PrivateKey) {
		t.Fatalf("private key is wrong")
	}

	if !bytes.Equal(net.Keys.PublicKey, keys.PublicKey) {
		t.Fatalf("public key is wrong")
	}

}
func TestPeers(t *testing.T) {

	var nets []*network.Network
	resolvedAddrs := []string{"127.0.0.1:12345", "127.0.0.1:12346", "127.0.0.1:12347"}
	excl := [][2]int{{1, 2}, {0, 2}, {0, 1}}
	// Build
	for i := 0; i < 3; i++ {
		myport := port + uint16(i)
		net, _ := buildNetwork(myport)
		go net.Listen()
		net.BlockUntilListening()
		if i != 0 {
			net.Bootstrap(resolvedAddrs[0])
		}
		nets = append(nets, net)
	}
	for i := 0; i < len(nets); i++ {
		for exc := range []int{0, 1} {
			if _, err := nets[i].Client(resolvedAddrs[excl[i][exc]]); err != nil {
				t.Fatalf("nets[%d] missing peer: %s", i, resolvedAddrs[excl[i][exc]])
			}
		}
	}

}

// Broadcast functions are tested through examples.
