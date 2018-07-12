package network

import (
	"reflect"

	"sync"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/crypto/blake2b"
	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"github.com/pkg/errors"
)

// Builder is a Address->processors struct
type Builder struct {
	keys    *crypto.KeyPair
	address string

	plugins     *PluginList
	pluginCount int

	signaturePolicy crypto.SignaturePolicy
	hashPolicy      crypto.HashPolicy
}

// NewBuilder lets you configure a network to build.
func NewBuilder() *Builder {
	return &Builder{
		signaturePolicy: ed25519.New(),
		hashPolicy:      blake2b.New(),
	}
}

// SetKeys pair created from crypto.KeyPair.
func (builder *Builder) SetKeys(pair *crypto.KeyPair) {
	builder.keys = pair
}

// SetAddress sets the host address for the network.
func (builder *Builder) SetAddress(address string) {
	builder.address = address
}

// SetSignaturePolicy sets the signature policy for the network.
func (builder *Builder) SetSignaturePolicy(policy crypto.SignaturePolicy) {
	builder.signaturePolicy = policy
}

// SetHashPolicy sets the hash policy for the network.
func (builder *Builder) SetHashPolicy(policy crypto.HashPolicy) {
	builder.hashPolicy = policy
}

// AddPluginWithPriority register a new plugin onto the network with a set priority.
func (builder *Builder) AddPluginWithPriority(priority int, plugin PluginInterface) error {
	// Initialize plugin list if not exist.
	if builder.plugins == nil {
		builder.plugins = NewPluginList()
	}

	if !builder.plugins.Put(priority, plugin) {
		return errors.Errorf("plugin %s is already registered", reflect.TypeOf(plugin).String())
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
		return nil, errors.New("cryptography keys not provided to Network; cannot create node ID")
	}

	if len(builder.address) == 0 {
		return nil, errors.New("Network requires public server IP for peers to connect to")
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
		ID:      id,
		Keys:    builder.keys,
		Address: unifiedAddress,

		Plugins: builder.plugins,

		Peers: new(sync.Map),

		Connections: new(sync.Map),
		SendQueue:   make(chan *Packet, 4096),
		RecvQueue:   make(chan *protobuf.Message, 4096),

		Listening: make(chan struct{}),

		SignaturePolicy: builder.signaturePolicy,
		HashPolicy:      builder.hashPolicy,

		Kill: make(chan struct{}),
	}

	net.Init()

	return net, nil
}
