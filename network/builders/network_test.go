package builders

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang/glog"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/builders"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
)

var (
	kp   = crypto.RandomKeyPair()
	host = "localhost"
	port = uint16(12345)
)

//MockProcessor so to keep independency to incoming.go and outgoing.go
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

func buildNet(port uint16) (*network.Network, error) {
	builder := &builders.NetworkBuilder{}
	builder.SetKeys(kp)
	builder.SetHost(host)
	builder.SetPort(port)

	builder.AddProcessor((*protobuf.HandshakeRequest)(nil), new(MockProcessor))

	return builder.BuildNetwork()
}

func TestBuildNetwork(t *testing.T) {
	_, err := buildNet(port)

	if err != nil {
		t.Fatalf("testbuildnetwork error: %v", err)
	}
}

func TestSetters(t *testing.T) {
	net, _ := buildNet(port)
	if net.Address() != fmt.Sprintf("127.0.0.1:%d", port) { //unified
		t.Fatalf("address is wrong: expected %s but got %s", fmt.Sprintf("127.0.0.1:%d", port), net.Address())
	}
	if net.Host != fmt.Sprintf("127.0.0.1") { //unified
		t.Fatal("host is wrong")
	}

	comparee := peer.CreateID("localhost:12345", kp.PublicKey)
	if !net.ID.Equals(comparee) {
		t.Fatalf("address is wrong %s", net.ID)
	}

	if !bytes.Equal(net.Keys.PrivateKey, kp.PrivateKey) {
		t.Fatalf("private key is wrong")
	}

	if !bytes.Equal(net.Keys.PublicKey, kp.PublicKey) {
		t.Fatalf("public key is wrong")
	}

}
func TestPeers(t *testing.T) {
	net1, _ := buildNet(port)

	net2, _ := buildNet(12346)
	net3, _ := buildNet(12347)

	go net1.Listen()
	go net2.Listen()
	go net3.Listen()

	//goroutines race for some reason
	time.Sleep(1 * time.Millisecond)
	peers := []string{}
	peers = append(peers, "localhost:12346")
	peers = append(peers, "localhost:12347")
	net1.Bootstrap(peers...)
	net2.Bootstrap(peers...)
	net3.Bootstrap(peers...)
	//Give some time for discovery
	time.Sleep(1 * time.Second)

	resolvedHost := "127.0.0.1"
	resolvedAddr1 := fmt.Sprintf("%s:12345", resolvedHost)
	resolvedAddr2 := fmt.Sprintf("%s:12346", resolvedHost)
	resolvedAddr3 := fmt.Sprintf("%s:12347", resolvedHost)

	if !strings.Contains(fmt.Sprintf("%v", net1.Peers), resolvedAddr2) ||
		!strings.Contains(fmt.Sprintf("%v", net1.Peers), resolvedAddr3) {
		t.Fatalf("missing Peers 0")
	}

	if _, err := net1.Client(resolvedAddr2); err != nil {
		t.Fatalf("net1 missing peer: %s", resolvedAddr2)
	}
	if _, err := net1.Client(resolvedAddr3); err != nil {
		t.Fatalf("net1 missing peer: %s", resolvedAddr3)
	}
	if _, err := net2.Client(resolvedAddr1); err != nil {
		t.Fatalf("net2 missing peer: %s", resolvedAddr1)
	}
	if _, err := net2.Client(resolvedAddr3); err != nil {
		t.Fatalf("net2 missing peer: %s", resolvedAddr3)
	}
	if _, err := net3.Client(resolvedAddr1); err != nil {
		t.Fatalf("net3 missing peer: %s", resolvedAddr1)
	}
	if _, err := net3.Client(resolvedAddr2); err != nil {
		t.Fatalf("net3 missing peer: %s", resolvedAddr2)
	}
}

//Boardcast functions can be tested using examples
