package discovery

import (
	"strings"

	"github.com/golang/glog"
	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
)

const (
	defaultPriority = -2000
)

type Plugin struct {
	*network.Plugin

	DisablePing   bool
	DisablePong   bool
	DisableLookup bool

	Routes *dht.RoutingTable
}

var (
	PluginID                         = (*Plugin)(nil)
	_        network.PluginInterface = (*Plugin)(nil)
)

func (p *Plugin) Startup(net *network.Network) {
	// Create routing table.
	p.Routes = dht.CreateRoutingTable(net.ID)
}

func (p *Plugin) Receive(ctx *network.PluginContext) error {
	// Update routing for every incoming message.
	p.Routes.Update(ctx.Sender())

	// Handle RPC.
	switch msg := ctx.Message().(type) {
	case *protobuf.Ping:
		if p.DisablePing {
			break
		}

		// Send pong to peer.
		err := ctx.Reply(&protobuf.Pong{})

		if err != nil {
			return err
		}
	case *protobuf.Pong:
		if p.DisablePong {
			break
		}

		peers := FindNode(ctx.Network(), ctx.Sender(), dht.BucketSize, 8)

		// Update routing table w/ closest peers to self.
		for _, peerID := range peers {
			p.Routes.Update(peerID)
		}

		glog.Infof("bootstrapped w/ peer(s): %s.", strings.Join(p.Routes.GetPeerAddresses(), ", "))
	case *protobuf.LookupNodeRequest:
		if p.DisableLookup {
			break
		}

		// Prepare response.
		response := &protobuf.LookupNodeResponse{}

		// Respond back with closest peers to a provided target.
		for _, peerID := range p.Routes.FindClosestPeers(peer.ID(*msg.Target), dht.BucketSize) {
			id := protobuf.ID(peerID)
			response.Peers = append(response.Peers, &id)
		}

		err := ctx.Reply(response)
		if err != nil {
			return err
		}

		glog.Infof("connected peers: %s.", strings.Join(p.Routes.GetPeerAddresses(), ", "))
	}

	return nil
}

func (p *Plugin) Cleanup(net *network.Network) {
	// TODO: Save routing table?
}

func (p *Plugin) PeerDisconnect(client *network.PeerClient) {
	// Delete peer if in routing table.
	if client.ID != nil {
		if p.Routes.PeerExists(*client.ID) {
			p.Routes.RemovePeer(*client.ID)

			glog.Infof("Peer %s has disconnected from %s.", client.ID.Address, client.Network.ID.Address)
		}
	}
}

// Priority returns the plugin priority (default: -2000).
func (p *Plugin) Priority() int {
	return defaultPriority
}
