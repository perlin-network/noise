package discovery

import (
	"fmt"
	"strings"
	"testing"

	"github.com/golang/glog"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/builders"
	"github.com/perlin-network/noise/protobuf"
)

var (
	keys = crypto.RandomKeyPair()
	host = "localhost"
	port = 12345
)

// MockProcessor to keep independent from incoming.go and outgoing.go.
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

func buildNetwork(port uint16) *builders.NetworkBuilder {
	builder := &builders.NetworkBuilder{}
	builder.SetKeys(keys)
	builder.SetHost(host)
	builder.SetPort(port)

	builder.AddProcessor((*protobuf.HandshakeRequest)(nil), new(MockProcessor))

	return builder
}

func TestDiscovery(t *testing.T) {
	builder := buildNetwork(uint16(port))

	BootstrapPeerDiscovery(builder)

	network, _ := builder.BuildNetwork()

	expected := []string{
		"*protobuf.HandshakeRequest",
		"*protobuf.HandshakeResponse",
		"*protobuf.LookupNodeRequest",
	}

	processors := fmt.Sprintf("%v", network.Processors)

	for _, name := range expected {
		if !strings.Contains(processors, name) {
			t.Fatalf("processor name is incorrect: %s", name)
		}
	}
}
