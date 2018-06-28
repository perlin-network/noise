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
	keypair = crypto.RandomKeyPair()
	host    = "localhost"
	port    = 12345
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

func buildNetwork(port uint16) *builders.NetworkBuilder {
	builder := &builders.NetworkBuilder{}
	builder.SetKeys(keypair)
	builder.SetHost(host)
	builder.SetPort(port)

	builder.AddProcessor((*protobuf.HandshakeRequest)(nil), new(MockProcessor))

	return builder
}

func TestDiscovery(t *testing.T) {
	builder := buildNetwork(uint16(port))
	discovery.BootstrapPeerDiscovery(builder)
	network, _ := builder.BuildNetwork()
	expectedProcessors := []string{
		"*protobuf.HandshakeRequest",
		"*protobuf.HandshakeResponse",
		"*protobuf.LookupNodeRequest",
	}
	processors := fmt.Sprintf("%v", network.Processors)
	// expected: &{{{0 0} {{map[] true}} map[*protobuf.HandshakeRequest:0xc4200c4068 *protobuf.HandshakeResponse:0xc4200c4070 *protobuf.LookupNodeRequest:0xc4200c4078] 0}}
	for _, processor := range expectedProcessors {
		if !strings.Contains(processors, processor) {
			t.Fatalf("not enough processor: %s", processor)
		}
	}
}
