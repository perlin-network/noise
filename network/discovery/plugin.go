package discovery

import (
	"context"

	"github.com/perlin-network/noise/internal/protobuf"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/peer"

	"github.com/pkg/errors"
)

const (
	defaultDisablePing             = false
	defaultDisablePong             = false
	defaultDisableLookup           = false
	defaultEnforceSKademliaNodeIDs = false
	defaultC1                      = 16
	defaultC2                      = 16
)

var (
	defaultPeerID *peer.ID
)

type Plugin struct {
	*network.Plugin

	// Plugin options
	disablePing   bool
	disablePong   bool
	disableLookup bool
	//eEnforceSkademliaNodeIDs checks whether node IDs satisfy S/Kademlia cryptopuzzles
	enforceSkademliaNodeIDs bool
	// id is an S/Kademlia-compatible ID
	id *peer.ID
	// c1 is the number of preceding bits of 0 in the H(H(key_public)) for NodeID generation
	c1 int
	// c2 is the number of preceding bits of 0 in the H(NodeID xor X) for checking if dynamic cryptopuzzle is solved
	c2 int

	Routes *RoutingTable
}

var (
	PluginID                         = (*Plugin)(nil)
	_        network.PluginInterface = (*Plugin)(nil)
)

// PluginOption are configurable options for the discovery plugin
type PluginOption func(*Plugin)

// WithEnforceSKademliaNodeIDs sets the plugin to enforce S/Kademlia peer node IDs
func WithEnforceSKademliaNodeIDs(v bool) PluginOption {
	return func(o *Plugin) {
		o.enforceSkademliaNodeIDs = v
	}
}

// WithSKademliaID sets the current node ID to a S/Kademlia-compatible node ID
func WithSKademliaID(id *peer.ID) PluginOption {
	return func(o *Plugin) {
		o.id = id
	}
}

// WithStaticPuzzleConstant sets the prefix matching length for the static S/Kademlia cryptopuzzle
func WithStaticPuzzleConstant(c1 int) PluginOption {
	return func(o *Plugin) {
		o.c1 = c1
	}
}

// WithDynamicPuzzleConstant sets the prefix matching length for the static S/Kademlia cryptopuzzle
func WithDynamicPuzzleConstant(c2 int) PluginOption {
	return func(o *Plugin) {
		o.c2 = c2
	}
}

func WithDisablePing(v bool) PluginOption {
	return func(o *Plugin) {
		o.disablePing = v
	}
}

func WithDisablePong(v bool) PluginOption {
	return func(o *Plugin) {
		o.disablePong = v
	}
}

func WithDisableLookup(v bool) PluginOption {
	return func(o *Plugin) {
		o.disableLookup = v
	}
}

func defaultOptions() PluginOption {
	return func(o *Plugin) {
		o.disablePing = defaultDisablePing
		o.disablePong = defaultDisablePong
		o.disableLookup = defaultDisableLookup
		o.enforceSkademliaNodeIDs = defaultEnforceSKademliaNodeIDs
		o.id = defaultPeerID
		o.c1 = defaultC1
		o.c2 = defaultC2
	}
}

// New returns a new discovery plugin with specified options.
func New(opts ...PluginOption) *Plugin {
	p := new(Plugin)
	defaultOptions()(p)

	for _, opt := range opts {
		opt(p)
	}

	return p
}

func (state *Plugin) Startup(net *network.Network) {
	if state.id != nil {
		net.ID = *state.id
	}

	// Create routing table.
	state.Routes = CreateRoutingTable(net.ID)
}

func (state *Plugin) Receive(ctx *network.PluginContext) error {
	sender := ctx.Sender()
	if state.enforceSkademliaNodeIDs && !VerifyPuzzle(sender, state.c1, state.c2) {
		return errors.Errorf("Sender %v is not a valid node ID", sender)
	}
	// Update routing for every incoming message.
	state.Routes.Update(sender)
	gCtx := network.WithSignMessage(context.Background(), true)

	// Handle RPC.
	switch msg := ctx.Message().(type) {
	case *protobuf.Ping:
		if state.disablePing {
			break
		}

		// Send pong to peer.
		err := ctx.Reply(gCtx, &protobuf.Pong{})

		if err != nil {
			return err
		}
	case *protobuf.Pong:
		if state.disablePong {
			break
		}

		peers := FindNode(ctx.Network(), ctx.Sender(), BucketSize, 8)

		// Update routing table w/ closest peers to self.
		for _, peerID := range peers {
			state.Routes.Update(peerID)
		}

		log.Debug().
			Strs("peers", state.Routes.GetPeerAddresses()).
			Msg("bootstrapped w/ peer(s)")
	case *protobuf.LookupNodeRequest:
		if state.disableLookup {
			break
		}

		// Prepare response.
		response := &protobuf.LookupNodeResponse{}

		// Respond back with closest peers to a provided target.
		for _, peerID := range state.Routes.FindClosestPeers(peer.ID(*msg.Target), BucketSize) {
			id := protobuf.ID(peerID)
			response.Peers = append(response.Peers, &id)
		}

		err := ctx.Reply(gCtx, response)
		if err != nil {
			return err
		}

		log.Debug().
			Strs("peers", state.Routes.GetPeerAddresses()).
			Msg("connected to peer(s)")
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

			log.Debug().
				Str("address", client.Network.ID.Address).
				Str("peer_address", client.ID.Address).
				Msg("peer has disconnected")
		}
	}
}
