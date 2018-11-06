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
	DefaultC2               = 16
	weakSignatureExpiration = 30 * time.Second
)

var (
	defaultPeerID *peer.ID

	// package errors
	// ErrInvalidKeyPair returns if the provided plugin keypair is not valid
	ErrInvalidKeyPair = errors.New("discovery: keypair is not set or does not match public key of node ID")
	// ErrInvalidNodeID occurs if puzzle enforcement is set and the sender node ID is invalid
	ErrInvalidNodeID = errors.New("discovery: invalid sender node ID")
	// ErrSignatureInvalid occurs when the message signature is invalid
	ErrSignatureInvalid = errors.New("discovery: invalid signature")
	// ErrSignatureExpired occurs when the message signature is verified but has expired
	ErrSignatureExpired = errors.New("discovery: signature has expired")
	// ErrSignatureNoSender returns if the message has no sender or sender public key
	ErrSignatureNoSender = errors.New("discovery: no sender or sender public key")

	PluginID                         = (*Plugin)(nil)
	_        network.PluginInterface = (*Plugin)(nil)
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
			log.Fatal().Err(ErrInvalidKeyPair).Msg("")
		}
	}

	if state.signaturePolicy == nil || state.hashPolicy == nil {
		log.Fatal().Msg("discovery: plugin is not properly initialized, please use discovery.New()")
	}

	// Create routing table.
	state.Routes = CreateRoutingTable(net.ID)
	state.id = net.ID
}

func (state *Plugin) Receive(pctx *network.PluginContext) error {
	sender := pctx.Sender()
	if state.enforcePuzzle && !VerifyPuzzle(sender, state.c1, state.c2) {
		return errors.Wrapf(ErrInvalidNodeID, "sender %v is not a valid node ID", sender)
	}
	// Update routing for every incoming message.
	state.Routes.Update(sender)
	// expire signature after 30 seconds
	ctx := context.Background()
	expiration := time.Now().Add(weakSignatureExpiration)
	signature := serializePeerIDAndExpiration(&state.id, &expiration)

	// Handle RPC.
	switch msg := pctx.Message().(type) {
	case *protobuf.Ping:
		if state.disablePing {
			break
		}

		// Send pong to peer.
		err := pctx.Reply(ctx, &protobuf.Pong{}, network.WithReplySignature(signature))

		if err != nil {
			return err
		}
	case *protobuf.Pong:
		if state.disablePong {
			break
		}

		// Verify weak signature
		/*if ok, err := state.verifyWeakSignature(pctx.RawMessage()); !ok && err != nil {
			return err
		}*/

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

		// Verify weak signature
		/*if ok, err := state.verifyWeakSignature(pctx.RawMessage()); !ok && err != nil {
			return err
		}*/

		// Prepare response.
		response := &protobuf.LookupNodeResponse{}

		// Respond back with closest peers to a provided target.
		for _, peerID := range state.Routes.FindClosestPeers(peer.ID(*msg.Target), BucketSize) {
			id := protobuf.ID(peerID)
			response.Peers = append(response.Peers, &id)
		}

		err := pctx.Reply(ctx, response, network.WithReplySignature(signature))
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
func (state Plugin) GetStrongSignature(msg protobuf.Message) ([]byte, error) {
	if len(msg.Signature) != 0 {
		return nil, errors.New("discovery: message signature must be empty")
	}
	raw, err := proto.Marshal(&msg)
	if err != nil {
		return nil, err
	}
	return state.kp.Sign(state.signaturePolicy, state.hashPolicy, serializeMessage(&state.id, raw))
}

// verifyStrongSignature verifies whether the message signature is a valid strong signature
func (state Plugin) verifyStrongSignature(msg protobuf.Message) (bool, error) {
	if len(msg.Signature) == 0 {
		return false, errors.Wrapf(ErrSignatureInvalid, "no message signature provided")
	}
	if msg.Sender == nil || msg.Sender.PublicKey == nil {
		return false, ErrSignatureNoSender
	}

	// clear out message signature prior to serializing
	signature := msg.Signature
	msg.Signature = nil

	id := (*peer.ID)(msg.GetSender())
	raw, err := proto.Marshal(&msg)
	if err != nil {
		return false, err
	}

	if crypto.Verify(state.signaturePolicy,
		state.hashPolicy,
		id.PublicKey,
		serializeMessage(id, raw),
		signature,
	) {
		return true, nil
	}
	return false, ErrSignatureInvalid
}

// GetWeakSignature generates the weak signature of the message.
func (state Plugin) GetWeakSignature(expiration time.Time) ([]byte, error) {
	return state.kp.Sign(state.signaturePolicy, state.hashPolicy, serializePeerIDAndExpiration(&state.id, &expiration))
}

// verifyWeakSignature verifies whether the message signature is a valid weak signature
func (state Plugin) verifyWeakSignature(msg protobuf.Message) (bool, error) {
	if len(msg.Signature) == 0 {
		return false, errors.Wrapf(ErrSignatureInvalid, "no message signature provided")
	}
	if msg.Sender == nil || msg.Sender.PublicKey == nil {
		return false, ErrSignatureNoSender
	}

	id := (*peer.ID)(msg.GetSender())
	expiration := time.Unix(0, msg.GetExpiration())

	if crypto.Verify(state.signaturePolicy,
		state.hashPolicy,
		id.PublicKey,
		serializePeerIDAndExpiration(id, &expiration),
		msg.Signature,
	) {
		if time.Now().Before(expiration) {
			return true, nil
		}
		return false, ErrSignatureExpired
	}
	return false, ErrSignatureInvalid
}
