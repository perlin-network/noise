package builders

import (
	"errors"
	"fmt"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/peer"
)

// NetworkBuilder is a Address->processors struct
type NetworkBuilder struct {
	keys    *crypto.KeyPair
	address string

	// map[string]PluginInterface
	plugins *network.StringPluginInterfaceSyncMap

	pluginOrder pluginPriorities
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

// AddPlugin register a new plugin into the network.
func (builder *NetworkBuilder) AddPlugin(priority int, name string, plugin network.PluginInterface) error {
	// Initialize map if not exist.
	if builder.plugins == nil {
		builder.plugins = &network.StringPluginInterfaceSyncMap{}
	}

	if _, exists := builder.plugins.Load(name); exists {
		return fmt.Errorf("plugin %s is already registered", name)
	}

	builder.pluginOrder = append(builder.pluginOrder, &pluginPriority{
		Priority:  priority,
		Name:      name,
		InsertIdx: len(builder.pluginOrder),
	})
	builder.plugins.Store(name, plugin)
	return nil
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

	// Initialize map if not exist.
	if builder.plugins == nil {
		builder.plugins = &network.StringPluginInterfaceSyncMap{}
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

		PluginOrder: builder.pluginOrder.GetSortedNames(),

		Peers: new(network.StringPeerClientSyncMap),

		Listening: make(chan struct{}),
	}

	return net, nil
}
