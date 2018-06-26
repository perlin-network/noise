package discovery

import (
	"strings"

	"github.com/golang/glog"
	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/builders"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
)

type HandshakeRequestProcessor struct{}

func (HandshakeRequestProcessor) Handle(ctx *network.MessageContext) error {
	// Send handshake response to peer.
	err := ctx.Send(&protobuf.HandshakeResponse{})

	if err != nil {
		glog.Error(err)
		return err
	}
	return nil
}

type HandshakeResponseProcessor struct{}

func (HandshakeResponseProcessor) Handle(ctx *network.MessageContext) error {
	addresses, publicKeys := bootstrapPeers(ctx.Network(), ctx.Self(), dht.BucketSize)

	// Update routing table w/ bootstrapped peers.
	for i := 0; i < len(addresses); i++ {
		ctx.Network().Routes.Update(peer.CreateID(addresses[i], publicKeys[i]))
	}

	glog.Infof("bootstrapped w/ peer(s): %s.", strings.Join(ctx.Network().Routes.GetPeerAddresses(), ", "))

	return nil
}

type LookupNodeRequestProcessor struct{}

func (LookupNodeRequestProcessor) Handle(ctx *network.MessageContext) error {
	// Deserialize received request.
	msg := ctx.Message().(*protobuf.LookupNodeRequest)

	// Prepare response.
	response := &protobuf.LookupNodeResponse{}

	// Respond back with closest peers to a provided target.
	for _, id := range ctx.Network().Routes.FindClosestPeers(peer.ID(*msg.Target), dht.BucketSize) {
		id := protobuf.ID(id)
		response.Peers = append(response.Peers, &id)
	}

	err := ctx.Reply(response)
	if err != nil {
		glog.Error(err)
		return err
	}

	glog.Infof("connected peers: %s.", strings.Join(ctx.Network().Routes.GetPeerAddresses(), ", "))

	return nil
}

// Registers necessary message processors for peer discovery.
func BootstrapPeerDiscovery(builder *builders.NetworkBuilder) {
	builder.AddProcessor((*protobuf.HandshakeRequest)(nil), new(HandshakeRequestProcessor))
	builder.AddProcessor((*protobuf.HandshakeResponse)(nil), new(HandshakeResponseProcessor))
	builder.AddProcessor((*protobuf.LookupNodeRequest)(nil), new(LookupNodeRequestProcessor))
}
