package builders

import (
	"errors"
	"reflect"
	"strconv"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/peer"
)

// NetworkBuilder is a Address->processors struct
type NetworkBuilder struct {
	keys *crypto.KeyPair
	host string
	port uint16

	// map[string]MessageProcessor
	processors *network.StringMessageProcessorSyncMap
}

// SetKeys pair created from crypto.KeyPair
func (builder *NetworkBuilder) SetKeys(pair *crypto.KeyPair) {
	builder.keys = pair
}

// SetHost of NetworkBuilder e.g. "127.0.0.1"
func (builder *NetworkBuilder) SetHost(host string) {
	builder.host = host
}

// SetPort of NetworkBuilder
func (builder *NetworkBuilder) SetPort(port uint16) {
	builder.port = port
}

// AddProcessor for a given message,
// Example: builder.AddProcessor((*protobuf.LookupNodeRequest)(nil), MessageProcessor{})
func (builder *NetworkBuilder) AddProcessor(message proto.Message, processor network.MessageProcessor) {
	// Initialize map if not exist.
	if builder.processors == nil {
		builder.processors = &network.StringMessageProcessorSyncMap{}
	}

	name := reflect.TypeOf(message).String()

	// Store pointers to message processor only.
	if value := reflect.ValueOf(message); value.Kind() == reflect.Ptr && value.Pointer() == 0 {
		builder.processors.Store(name, processor)
	} else {
		glog.Fatal("message must be nil")
	}
}

// BuildNetwork verifies all parameters of the network and returns either an error due to
// misconfiguration, or a noise.network.Network.
func (builder *NetworkBuilder) BuildNetwork() (*network.Network, error) {
	if builder.keys == nil {
		return nil, errors.New("cryptography keys not provided to Network; cannot create node Id")
	}

	if len(builder.host) == 0 {
		return nil, errors.New("Network requires public server IP for peers to connect to")
	}

	if builder.port <= 0 || builder.port >= 65535 {
		return nil, errors.New("port to listen for peers on must be within the range (0, 65535)")
	}

	// Initialize map if not exist.
	if builder.processors == nil {
		builder.processors = &network.StringMessageProcessorSyncMap{}
	}

	unifiedHost, err := network.ToUnifiedHost(builder.host)
	if err != nil {
		return nil, err
	}

	id := peer.CreateID(unifiedHost+":"+strconv.Itoa(int(builder.port)), builder.keys.PublicKey)

	net := &network.Network{
		Keys: builder.keys,
		Host: unifiedHost,
		Port: builder.port,
		ID:   id,

		Processors: builder.processors,

		Routes: dht.CreateRoutingTable(id),

		Peers: &network.StringPeerClientSyncMap{},

		Listening: make(chan struct{}, 1),
	}

	return net, nil
}
