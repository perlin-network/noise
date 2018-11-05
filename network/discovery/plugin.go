package discovery

import (
	"bytes"
	"context"
	"github.com/gogo/protobuf/proto"
	"time"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/crypto/blake2b"
	"github.com/perlin-network/noise/crypto/ed25519"
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
	// DefaultC1 is the prefix-matching length for the static cryptopuzzle.
	DefaultC1 = 16
	// DefaultC2 is the prefix-matching length for the dynamic cryptopuzzle.
	DefaultC2 = 16
)

var (
	defaultPeerID *peer.ID
)

// Plugin defines the discovery plugin struct.
type Plugin struct {
	*network.Plugin

	// Plugin options.
	disablePing   bool
	disablePong   bool
	disableLookup bool
	// enforcePuzzle checks whether node IDs satisfy S/Kademlia cryptopuzzles.
	enforcePuzzle bool
	// c1 is the number of preceding bits of 0 in the H(H(key_public)) for NodeID generation.
	c1 int
	// c2 is the number of preceding bits of 0 in the H(NodeID xor X) for checking if dynamic cryptopuzzle is solved.
	c2 int
	// signaturePolicy for signing messages
	signaturePolicy crypto.SignaturePolicy
	// hashPolicy for hashing messages
	hashPolicy crypto.HashPolicy

	Routes *RoutingTable
	id     peer.ID
	kp     *crypto.KeyPair
}

const (
	weakSignatureExpiration = 30 * time.Second
)

var (
	PluginID                         = (*Plugin)(nil)
	_        network.PluginInterface = (*Plugin)(nil)
)

// PluginOption are configurable options for the discovery plugin.
type PluginOption func(*Plugin)

// WithPuzzleEnabled sets the plugin to enforce S/Kademlia peer node IDs for c1 and c2.
func WithPuzzleEnabled(c1, c2 int) PluginOption {
	return func(o *Plugin) {
		o.c1 = c1
		o.c2 = c2
	}
}

// WithDisablePing sets whether to reply to ping messages.
func WithDisablePing(v bool) PluginOption {
	return func(o *Plugin) {
		o.disablePing = v
	}
}

// WithDisablePong sets whether to reply to pong messages.
func WithDisablePong(v bool) PluginOption {
	return func(o *Plugin) {
		o.disablePong = v
	}
}

// WithDisableLookup sets whether to reply to node lookup messages.
func WithDisableLookup(v bool) PluginOption {
	return func(o *Plugin) {
		o.disableLookup = v
	}
}

// WithSignaturePolicy sets the signature policy for signing messages.
func WithSignaturePolicy(sp crypto.SignaturePolicy) PluginOption {
	return func(o *Plugin) {
		o.signaturePolicy = sp
	}
}

// WithHashPolicy sets the signature policy for signing messages.
func WithHashPolicy(hp crypto.HashPolicy) PluginOption {
	return func(o *Plugin) {
		o.hashPolicy = hp
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
		o.signaturePolicy = ed25519.New()
		o.hashPolicy = blake2b.New()
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

// PerformPuzzle returns an S/Kademlia compatible keypair and node ID nonce and sets it in the plugin.
func (state *Plugin) PerformPuzzle() (*crypto.KeyPair, []byte) {
	kp, nonce := generateKeyPairAndNonce(state.c1, state.c2)
	state.kp = kp
	return kp, nonce
}

// SetKeyPair sets the plugin's keypair for signing messages.
func (state *Plugin) SetKeyPair(kp *crypto.KeyPair) error {
	if kp == nil {
		return errors.New("discovery: keypair cannot be nil")
	}
	state.kp = kp
	return nil
}

func (state *Plugin) Startup(net *network.Network) {
	if state.enforcePuzzle {
		// verify the provided nonce is valid
		if !VerifyPuzzle(net.ID, state.c1, state.c2) {
			log.Fatal().Msg("discovery: provided node ID nonce does not solve the cryptopuzzle.")
		}
		// verify that the keypair set matches net.ID publicy key
		if state.kp == nil || !bytes.Equal(state.kp.PublicKey, net.ID.PublicKey) {
			log.Fatal().Msg("discovery: keypair is not set or does not match public key of node ID")
		}
	}

	// Create routing table.
	state.Routes = CreateRoutingTable(net.ID)
	state.id = net.ID
}

func (state *Plugin) Receive(pctx *network.PluginContext) error {
	sender := pctx.Sender()
	if state.enforcePuzzle && !VerifyPuzzle(sender, state.c1, state.c2) {
		return errors.Errorf("Sender %v is not a valid node ID", sender)
	}
	// Update routing for every incoming message.
	state.Routes.Update(sender)
	// expire signature after 30 seconds
	ctx := context.Background()
	/*expiration := time.Now().Add(weakSignatureExpiration)
	signature := serializePeerIDAndExpiration(&state.id, &expiration)*/

	// Handle RPC.
	switch msg := pctx.Message().(type) {
	case *protobuf.Ping:
		if state.disablePing {
			break
		}

		// Verify weak signature

		// Send pong to peer.
		err := pctx.Reply(ctx, &protobuf.Pong{})

		if err != nil {
			return err
		}
	case *protobuf.Pong:
		if state.disablePong {
			break
		}

		peers := FindNode(pctx.Network(), pctx.Sender(), BucketSize, 8)

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

		err := pctx.Reply(ctx, response)
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

// GetStrongSignature generates the strong signature of the message.
func (state Plugin) GetStrongSignature(msg *protobuf.Message) ([]byte, error) {
	raw, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return state.kp.Sign(state.signaturePolicy, state.hashPolicy, serializeMessage(&state.id, raw))
}
