package discovery

import (
	"strings"

	"github.com/golang/glog"
	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
)

type Plugin struct {
	*network.Plugin

	Routes *dht.RoutingTable

	// DisableDialing when enable overrides dialing out to new nodes (default: false)
	DisableDialing bool
}

var PluginID = (*Plugin)(nil)

func (state *Plugin) Startup(net *network.Network) {
	// Create routing table.
	state.Routes = dht.CreateRoutingTable(net.ID)
}

func (state *Plugin) Receive(ctx *network.MessageContext) error {
	// Update routing for every incoming message.
	state.Routes.Update(ctx.Sender())

	// Handle RPC.
	switch msg := ctx.Message().(type) {
	case *protobuf.Ping:
		// Send pong to peer.
		err := ctx.Reply(&protobuf.Pong{})

		if err != nil {
			glog.Error(err)
			return err
		}
	case *protobuf.Pong:
		if state.DisableDialing {
			// override dial behavior
			break
		}

		peers := FindNode(ctx.Network(), ctx.Sender(), dht.BucketSize)

		// Update routing table w/ closest peers to self.
		for _, peerID := range peers {
			state.Routes.Update(peerID)
		}

		glog.Infof("bootstrapped w/ peer(s): %s.", strings.Join(state.Routes.GetPeerAddresses(), ", "))
	case *protobuf.LookupNodeRequest:
		// Prepare response.
		response := &protobuf.LookupNodeResponse{}

		// Respond back with closest peers to a provided target.
		for _, peerID := range state.Routes.FindClosestPeers(peer.ID(*msg.Target), dht.BucketSize) {
			id := protobuf.ID(peerID)
			response.Peers = append(response.Peers, &id)
		}

		err := ctx.Reply(response)
		if err != nil {
			glog.Error(err)
			return err
		}

		glog.Infof("connected peers: %s.", strings.Join(state.Routes.GetPeerAddresses(), ", "))
	}

	return nil
}

func (state *Plugin) Cleanup(net *network.Network) {
	// TODO: Save routing table?
}

func (state *Plugin) PeerDisconnect(id *peer.ID) {
	// Delete peer if in routing table.
	if state.Routes.PeerExists(*id) {
		state.Routes.RemovePeer(*id)

		glog.Infof("Peer %s has disconnected.", id.Address)
	}
}
