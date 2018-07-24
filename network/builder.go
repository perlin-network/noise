package network

import (
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/crypto/blake2b"
	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/types"
	"github.com/pkg/errors"
)

const (
	defaultAddress = "tcp://localhost:8588"
)

var (
	// ErrStrDuplicatePlugin returns if the plugin has already been registered
	// with the builder
	ErrStrDuplicatePlugin = "builder: plugin %s is already registered"
	// ErrStrNoAddress returns if no address was given to the builder
	ErrStrNoAddress = "builder: network requires public server IP for peers to connect to"
	// ErrStrNoKeyPair returns if no keypair was given to the builder
	ErrStrNoKeyPair = "builder: cryptography keys not provided to Network; cannot create node ID"
)

// Builder is a Address->processors struct
type Builder struct {
	opts options

	keys    *crypto.KeyPair
	address string

	plugins          *PluginList
	transportPlugins *TransportList

	listener net.Listener
}

var defaultBuilderOptions = options{
	connectionTimeout: defaultConnectionTimeout,
	signaturePolicy:   ed25519.New(),
	hashPolicy:        blake2b.New(),
	recvWindowSize:    defaultReceiveWindowSize,
	sendWindowSize:    defaultSendWindowSize,
	writeBufferSize:   defaultWriteBufferSize,
	writeFlushLatency: defaultWriteFlushLatency,
	writeTimeout:      defaultWriteTimeout,
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

// WriteBufferSize returns a BuilderOption that sets the write buffer size
// (default: 4096 bytes).
func WriteBufferSize(byteSize int) BuilderOption {
	return func(o *options) {
		o.writeBufferSize = byteSize
	}
}

// WriteFlushLatency returns a BuilderOption that sets the write flush interval
// (default: 50ms).
func WriteFlushLatency(d time.Duration) BuilderOption {
	return func(o *options) {
		o.writeFlushLatency = d
	}
}

// WriteTimeout returns a BuilderOption that sets the write timeout
// (default: 4096).
func WriteTimeout(d time.Duration) BuilderOption {
	return func(o *options) {
		o.writeTimeout = d
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
	b := NewBuilder()

	for _, o := range opt {
		o(&b.opts)
	}

	return b
}

// SetKeys pair created from crypto.KeyPair.
func (b *Builder) SetKeys(pair *crypto.KeyPair) {
	b.keys = pair
}

// SetAddress sets the host address for the network.
func (b *Builder) SetAddress(address string) {
	b.address = address
}

// AddPluginWithPriority registers a new plugin onto the network with a set priority.
func (b *Builder) AddPluginWithPriority(priority int, plugin PluginInterface) error {
	// Initialize plugin list if not exist.
	if b.plugins == nil {
		b.plugins = NewPluginList()
	}

	if !b.plugins.Put(priority, plugin) {
		return errors.Errorf(ErrStrDuplicatePlugin, reflect.TypeOf(plugin).String())
	}

	return nil
}

// AddPlugin register a new plugin onto the network.
func (b *Builder) AddPlugin(plugin PluginInterface) error {
	err := b.AddPluginWithPriority(plugin.Priority(), plugin)
	return err
}

// RegisterTransportLayer registers a new transport layer protocol
func (b *Builder) RegisterTransportLayer(t TransportInterface) error {
	// Initialize plugin list if not exist.
	if b.transportPlugins == nil {
		b.transportPlugins = NewTransportList()
	}

	if !b.transportPlugins.Put(0, t) {
		return errors.Errorf(ErrStrDuplicatePlugin, reflect.TypeOf(t).String())
	}

	return nil
}

// Build verifies all parameters of the network and returns either an error due to
// misconfiguration, or a *Network.
func (b *Builder) Build() (*Network, error) {
	if b.keys == nil {
		return nil, errors.New(ErrStrNoKeyPair)
	}

	if len(b.address) == 0 {
		return nil, errors.New(ErrStrNoAddress)
	}

	// Initialize plugin list if not exist.
	if b.plugins == nil {
		b.plugins = NewPluginList()
	} else {
		b.plugins.SortByPriority()
	}

	if b.transportPlugins == nil {
		b.transportPlugins = NewTransportList()
	} else {
		b.transportPlugins.SortByPriority()
	}

	// no transport plugins registered, register one based on address
	if b.transportPlugins.Len() == 0 {
		addr, err := types.ParseAddress(b.address)
		if err != nil {
			return nil, err
		}

		switch addr.Protocol {
		case "tcp":
			t, err := NewTCPTransport(b.address)
			if err != nil {
				return nil, err
			}
			b.RegisterTransportLayer(t)
		case "kcp":
			t, err := NewKCPTransport(b.address)
			if err != nil {
				return nil, err
			}
			b.RegisterTransportLayer(t)
		default:
			return nil, errors.Errorf("builder: no default protocol found for %s", addr.Network())
		}
	}

	unifiedAddress, err := types.ToUnifiedAddress(b.address)
	if err != nil {
		return nil, err
	}

	id := peer.CreateID(unifiedAddress, b.keys.PublicKey)

	net := &Network{
		opts:    b.opts,
		ID:      id,
		keys:    b.keys,
		Address: unifiedAddress,

		Plugins:    b.plugins,
		Transports: b.transportPlugins,

		Peers: new(sync.Map),

		Connections: new(sync.Map),

		Listening: make(chan struct{}),

		kill: make(chan struct{}),
	}

	net.Init()

	return net, nil
}
