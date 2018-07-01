package builders

import (
	"errors"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/peer"
)

// NetworkBuilder is a Address->processors struct
type NetworkBuilder struct {
	keys        *crypto.KeyPair
	address     string
	upnpEnabled bool

	// map[string]PluginInterface
	plugins *network.StringPluginInterfaceSyncMap
}

func NewNetworkBuilder() *NetworkBuilder {
	return &NetworkBuilder{
		upnpEnabled: false,
	}
}

// SetKeys pair created from crypto.KeyPair
func (builder *NetworkBuilder) SetKeys(pair *crypto.KeyPair) {
	builder.keys = pair
}

func (builder *NetworkBuilder) SetAddress(address string) {
	builder.address = address
}

func (builder *NetworkBuilder) SetUpnpEnabled(enabled bool) {
	builder.upnpEnabled = enabled
}

// AddPlugin register a new plugin into the network.
func (builder *NetworkBuilder) AddPlugin(name string, plugin network.PluginInterface) {
	// Initialize map if not exist.
	if builder.plugins == nil {
		builder.plugins = &network.StringPluginInterfaceSyncMap{}
	}

	builder.plugins.Store(name, plugin)
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
		ID:          id,
		Keys:        builder.keys,
		Address:     unifiedAddress,
		UpnpEnabled: builder.upnpEnabled,

		Plugins: builder.plugins,

		Peers: new(network.StringPeerClientSyncMap),

		Listening: make(chan struct{}),
	}

	return net, nil
}
