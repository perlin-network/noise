package builders

import (
	"errors"
	"reflect"
	"strconv"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/peer"
)

type NetworkBuilder struct {
	keys    *crypto.KeyPair
	address string
	port    int

	// map[string]MessageProcessor
	processors *sync.Map
}

func (builder *NetworkBuilder) SetKeys(pair *crypto.KeyPair) {
	builder.keys = pair
}

func (builder *NetworkBuilder) SetAddress(address string) {
	builder.address = address
}

func (builder *NetworkBuilder) SetPort(port int) {
	builder.port = port
}

// Sets a processor for a given message,
// Example: builder.AddProcessor((*protobuf.LookupNodeRequest)(nil), MessageProcessor{})
func (builder *NetworkBuilder) AddProcessor(message proto.Message, processor network.MessageProcessor) {
	// Initialize map if not exist.
	if builder.processors == nil {
		builder.processors = &sync.Map{}
	}

	name := reflect.TypeOf(message).String()

	// Store pointers to message processor only.
	if value := reflect.ValueOf(message); value.Kind() == reflect.Ptr && value.Pointer() == 0 {
		builder.processors.Store(name, processor)
	} else {
		builder.processors.Store(name, reflect.Zero(reflect.TypeOf(message)).Interface().(proto.Message))
	}
}

func (builder *NetworkBuilder) BuildNetwork() (*network.Network, error) {
	if builder.keys == nil {
		return nil, errors.New("cryptography keypair not provided to Network; cannot create node Id")
	}

	if len(builder.address) == 0 {
		return nil, errors.New("Network requires public server IP for peers to connect to")
	}

	if builder.port <= 0 || builder.port >= 65535 {
		return nil, errors.New("port to listen for peers on must be within the range (0, 65535)")
	}

	// Initialize map if not exist.
	if builder.processors == nil {
		builder.processors = &sync.Map{}
	}

	id := peer.CreateID(builder.address+":"+strconv.Itoa(builder.port), builder.keys.PublicKey)

	network := &network.Network{
		Keys:    builder.keys,
		Address: builder.address,
		Port:    builder.port,
		ID:      id,

		Processors: builder.processors,

		Routes: dht.CreateRoutingTable(id),

		Peers: &sync.Map{},
	}

	return network, nil
}
