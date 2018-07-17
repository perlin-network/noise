package network

import (
	"errors"
	"sync"
	"time"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/crypto/blake2b"
	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
)

const (
	defaultAddress = "tcp://localhost:8588"
)

var (
	// ErrDuplicatePlugin returns if the plugin has already been registered
	// with the builder
	ErrDuplicatePlugin = errors.New("builder: plugin is already registered")
	// ErrNoAddress returns if no address was given to the builder
	ErrNoAddress = errors.New("builder: network requires public server IP for peers to connect to")
	// ErrNoKeyPair returns if no keypair was given to the builder
	ErrNoKeyPair = errors.New("builder: cryptography keys not provided to Network; cannot create node ID")
)

// Builder is a Address->processors struct
type Builder struct {
	opts options

	keys    *crypto.KeyPair
	address string

	plugins     *PluginList
	pluginCount int
}

var defaultBuilderOptions = options{
	connectionTimeout: defaultConnectionTimeout,
	signaturePolicy:   ed25519.New(),
	hashPolicy:        blake2b.New(),
	recvWindowSize:    defaultReceiveWindowSize,
	sendWindowSize:    defaultSendWindowSize,
}

// A BuilderOption sets options such as connection timeout and cryptographic // policies for the network
type BuilderOption func(*options)

// ConnectionTimeout returns a NetworkOption that sets the timeout for
// establishing new connections (default: 60 seconds).
func ConnectionTimeout(d time.Duration) BuilderOption {
	return func(o *options) {
		o.connectionTimeout = d
	}
}

// SignaturePolicy returns a BuilderOption that sets the signature policy
// for the network (default: ed25519).
func SignaturePolicy(policy crypto.SignaturePolicy) BuilderOption {
	return func(o *options) {
		o.signaturePolicy = policy
	}
}

// HashPolicy returns a BuilderOption that sets the hash policy for the network
// (default: blake2b).
func HashPolicy(policy crypto.HashPolicy) BuilderOption {
	return func(o *options) {
		o.hashPolicy = policy
	}
}

// RecvWindowSize returns a BuilderOption that sets the receive buffer window
// size (default: 4096).
func RecvWindowSize(recvWindowSize int) BuilderOption {
	return func(o *options) {
		o.recvWindowSize = recvWindowSize
	}
}

// SendWindowSize returns a BuilderOption that sets the send buffer window
// size (default: 4096).
func SendWindowSize(sendWindowSize int) BuilderOption {
	return func(o *options) {
		o.sendWindowSize = sendWindowSize
	}
}

// NewBuilder returns a new builder with default options.
func NewBuilder() *Builder {
	return &Builder{
		opts:    defaultBuilderOptions,
		address: defaultAddress,
		keys:    ed25519.RandomKeyPair(),
	}
}

// NewBuilderWithOptions returns a new builder with specified options.
func NewBuilderWithOptions(opt ...BuilderOption) *Builder {
	builder := NewBuilder()

	for _, o := range opt {
		o(&builder.opts)
	}

	return builder
}

// SetKeys pair created from crypto.KeyPair.
func (builder *Builder) SetKeys(pair *crypto.KeyPair) {
	builder.keys = pair
}

// SetAddress sets the host address for the network.
func (builder *Builder) SetAddress(address string) {
	builder.address = address
}

// AddPluginWithPriority registers a new plugin onto the network with a set priority.
func (builder *Builder) AddPluginWithPriority(priority int, plugin PluginInterface) error {
	// Initialize plugin list if not exist.
	if builder.plugins == nil {
		builder.plugins = NewPluginList()
	}

	if !builder.plugins.Put(priority, plugin) {
		return ErrDuplicatePlugin
	}

	return nil
}

// AddPlugin register a new plugin onto the network.
func (builder *Builder) AddPlugin(plugin PluginInterface) error {
	err := builder.AddPluginWithPriority(builder.pluginCount, plugin)
	if err == nil {
		builder.pluginCount++
	}
	return err
}

// Build verifies all parameters of the network and returns either an error due to
// misconfiguration, or a *Network.
func (builder *Builder) Build() (*Network, error) {
	if builder.keys == nil {
		return nil, ErrNoKeyPair
	}

	if len(builder.address) == 0 {
		return nil, ErrNoAddress
	}

	// Initialize plugin list if not exist.
	if builder.plugins == nil {
		builder.plugins = NewPluginList()
	} else {
		builder.plugins.SortByPriority()
	}

	unifiedAddress, err := ToUnifiedAddress(builder.address)
	if err != nil {
		return nil, err
	}

	id := peer.CreateID(unifiedAddress, builder.keys.PublicKey)

	net := &Network{
		opts:    builder.opts,
		ID:      id,
		keys:    builder.keys,
		Address: unifiedAddress,

		Plugins: builder.plugins,

		Peers: new(sync.Map),

		Connections: new(sync.Map),
		SendQueue:   make(chan *Packet, 4096),
		RecvQueue:   make(chan *protobuf.Message, 4096),

		Listening: make(chan struct{}),

		kill: make(chan struct{}),
	}

	net.Init()

	return net, nil
}
