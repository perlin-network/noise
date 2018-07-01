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

type PingProcessor struct{}

func (PingProcessor) Handle(ctx *network.MessageContext) error {
	// Send pong to peer.
	err := ctx.Reply(&protobuf.Pong{})

	if err != nil {
		glog.Error(err)
		return err
	}
	return nil
}

type PongProcessor struct{}

func (PongProcessor) Handle(ctx *network.MessageContext) error {
	peers := findNode(ctx.Network(), ctx.Sender(), dht.BucketSize)

	// Update routing table w/ closest peers to self.
	for _, peerID := range peers {
		ctx.Network().Routes.Update(peerID)
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
	for _, peerID := range ctx.Network().Routes.FindClosestPeers(peer.ID(*msg.Target), dht.BucketSize) {
		id := protobuf.ID(peerID)
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
	builder.AddProcessor((*protobuf.Ping)(nil), new(PingProcessor))
	builder.AddProcessor((*protobuf.Pong)(nil), new(PongProcessor))
	builder.AddProcessor((*protobuf.LookupNodeRequest)(nil), new(LookupNodeRequestProcessor))
}
