package discovery

import (
	"context"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/internal/protobuf"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/peer"

	"github.com/pkg/errors"
)

const (
	defaultDisablePing   = false
	defaultDisablePong   = false
	defaultDisableLookup = false
	defaultEnforcePuzzle = false
	DefaultC1            = 16
	DefaultC2            = 16
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
	// enforcePuzzle checks whether node IDs satisfy S/Kademlia cryptopuzzles
	enforcePuzzle bool
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

// WithPuzzleEnabled sets the plugin to enforce S/Kademlia peer node IDs for c1 and c2
func WithPuzzleEnabled(c1, c2 int) PluginOption {
	return func(o *Plugin) {
		o.c1 = c1
		o.c2 = c2
	}
}

// WithDisablePing sets whether to reply to ping messages
func WithDisablePing(v bool) PluginOption {
	return func(o *Plugin) {
		o.disablePing = v
	}
}

// WithDisablePong sets whether to reply to pong messages
func WithDisablePong(v bool) PluginOption {
	return func(o *Plugin) {
		o.disablePong = v
	}
}

// WithDisableLookup sets whether to reply to node lookup messages
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
		o.enforcePuzzle = defaultEnforcePuzzle
		o.c1 = DefaultC1
		o.c2 = DefaultC2
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

// PerformPuzzle returns an S/Kademlia compatible keypair and node ID nonce
func (state *Plugin) PerformPuzzle() (*crypto.KeyPair, []byte) {
	return generateKeyPairAndNonce(state.c1, state.c2)
}

func (state *Plugin) Startup(net *network.Network) {
	if state.enforcePuzzle {
		// verify the provided nonce is valid
		if !VerifyPuzzle(net.ID, state.c1, state.c2) {
			log.Fatal().Msg("discovery: provided node ID nonce does not solve the cryptopuzzle.")
		}
	}

	// Create routing table.
	state.Routes = CreateRoutingTable(net.ID)
}

func (state *Plugin) Receive(ctx *network.PluginContext) error {
	sender := ctx.Sender()
	if state.enforcePuzzle && !VerifyPuzzle(sender, state.c1, state.c2) {
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
