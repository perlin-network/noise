package discovery

import (
	"fmt"
	"strings"
	"testing"

	"github.com/golang/glog"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/builders"
	"github.com/perlin-network/noise/network/discovery"
	"github.com/perlin-network/noise/protobuf"
)

var (
	kp   = crypto.RandomKeyPair()
	host = "localhost"
	port = 12345
)

//DummyProcessor so to keep independency to incoming.go and outgoing.go
type DummyProcessor struct{}

func (DummyProcessor) Handle(client *network.PeerClient, message *network.IncomingMessage) error {
	// Send handshake response to peer.
	err := client.Tell(&protobuf.HandshakeResponse{})

	if err != nil {
		glog.Error(err)
		return err
	}
	return nil
}

func buildNet(port int) *builders.NetworkBuilder {
	builder := &builders.NetworkBuilder{}
	builder.SetKeys(kp)
	builder.SetHost(host)
	builder.SetPort(port)

	builder.AddProcessor((*protobuf.HandshakeRequest)(nil), new(DummyProcessor))

	return builder
}

func TestDiscovery(t *testing.T) {
	nb1 := buildNet(port)
	discovery.BootstrapPeerDiscovery(nb1)
	net1, _ := nb1.BuildNetwork()
	expected := []string{
		"*protobuf.HandshakeRequest",
		"*protobuf.HandshakeResponse",
		"*protobuf.LookupNodeRequest",
	}
	processors := fmt.Sprintf("%v", net1.Processors)
	for _, v := range expected {
		if !strings.Contains(processors, v) {
			t.Fatalf("not enough processor: %s", v)
		}
	}
}