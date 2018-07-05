package builders

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/peer"
	"sync"
)

// NetworkBuilder is a Address->processors struct
type NetworkBuilder struct {
	keys    *crypto.KeyPair
	address string

	plugins     *network.PluginList
	pluginCount int
}

// NewNetworkBuilder lets you configure a network to build
func NewNetworkBuilder() *NetworkBuilder {
	return &NetworkBuilder{}
}

// SetKeys pair created from crypto.KeyPair
func (builder *NetworkBuilder) SetKeys(pair *crypto.KeyPair) {
	builder.keys = pair
}

// SetAddress sets the host address for the network.
func (builder *NetworkBuilder) SetAddress(address string) {
	builder.address = address
}

// AddPluginWithPriority register a new plugin onto the network with a set priority.
func (builder *NetworkBuilder) AddPluginWithPriority(priority int, plugin network.PluginInterface) error {
	// Initialize plugin list if not exist.
	if builder.plugins == nil {
		builder.plugins = network.NewPluginList()
	}

	if !builder.plugins.Put(priority, plugin) {
		return fmt.Errorf("plugin %s is already registered", reflect.TypeOf(plugin).String())
	}

	return nil
}

// AddPlugin register a new plugin onto the network.
func (builder *NetworkBuilder) AddPlugin(plugin network.PluginInterface) error {
	err := builder.AddPluginWithPriority(builder.pluginCount, plugin)
	if err == nil {
		builder.pluginCount++
	}
	return err
}

// Build verifies all parameters of the network and returns either an error due to
// misconfiguration, or a noise.network.Network.
func (builder *NetworkBuilder) Build() (*network.Network, error) {
	if builder.keys == nil {
		return nil, errors.New("cryptography keys not provided to Network; cannot create node ID")
	}

	if len(builder.address) == 0 {
		return nil, errors.New("Network requires public server IP for peers to connect to")
	}

	// Initialize plugin list if not exist.
	if builder.plugins == nil {
		builder.plugins = network.NewPluginList()
	} else {
		builder.plugins.SortByPriority()
	}

	unifiedAddress, err := network.ToUnifiedAddress(builder.address)
	if err != nil {
		return nil, err
	}

	id := peer.CreateID(unifiedAddress, builder.keys.PublicKey)

	net := &network.Network{
		ID:      id,
		Keys:    builder.keys,
		Address: unifiedAddress,

		Plugins: builder.plugins,

		Peers: new(network.StringPeerClientSyncMap),

		Connections: new(sync.Map),
		SendQueue:   make(chan *network.Packet, 1024),
		RecvQueue:   make(chan *network.Packet, 1024),

		Listening: make(chan struct{}),
	}

	net.Init()

	return net, nil
}
