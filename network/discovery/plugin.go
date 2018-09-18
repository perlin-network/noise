package discovery

import (
	"strings"

	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/internal/protobuf"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/peer"
	"github.com/rs/zerolog/log"
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

func (state *Plugin) Startup(net *network.Network) {
	// Create routing table.
	state.Routes = dht.CreateRoutingTable(net.ID)
}

func (state *Plugin) Receive(ctx *network.PluginContext) error {
	// Update routing for every incoming message.
	state.Routes.Update(ctx.Sender())

	// Handle RPC.
	switch msg := ctx.Message().(type) {
	case *protobuf.Ping:
		if state.DisablePing {
			break
		}

		// Send pong to peer.
		err := ctx.Reply(&protobuf.Pong{})

		if err != nil {
			return err
		}
	case *protobuf.Pong:
		if state.DisablePong {
			break
		}

		peers := FindNode(ctx.Network(), ctx.Sender(), dht.BucketSize, 8)

		// Update routing table w/ closest peers to self.
		for _, peerID := range peers {
			state.Routes.Update(peerID)
		}

		log.Info().Msgf("bootstrapped w/ peer(s): %s.", strings.Join(state.Routes.GetPeerAddresses(), ", "))
	case *protobuf.LookupNodeRequest:
		if state.DisableLookup {
			break
		}

		// Prepare response.
		response := &protobuf.LookupNodeResponse{}

		// Respond back with closest peers to a provided target.
		for _, peerID := range state.Routes.FindClosestPeers(peer.ID(*msg.Target), dht.BucketSize) {
			id := protobuf.ID(peerID)
			response.Peers = append(response.Peers, &id)
		}

		err := ctx.Reply(response)
		if err != nil {
			return err
		}

		log.Info().Msgf("connected peers: %s.", strings.Join(state.Routes.GetPeerAddresses(), ", "))
	}

	return nil
}

func (state *Plugin) Cleanup(net *network.Network) {
	// TODO: Save routing table?
}

func (state *Plugin) PeerDisconnect(client *network.PeerClient) {
	// Delete peer if in routing table.
	if client.ID != nil {
		if state.Routes.PeerExists(*client.ID) {
			state.Routes.RemovePeer(*client.ID)

			log.Info().Msgf("Peer %s has disconnected from %s.", client.ID.Address, client.Network.ID.Address)
		}
	}
}
