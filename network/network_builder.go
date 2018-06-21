package network

import (
	"errors"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/peer"
	"strconv"
	"sync"
	"github.com/golang/protobuf/proto"
)

type NetworkBuilder struct {
	keys    *crypto.KeyPair
	address string
	port    int

	// map[proto.Message]MessageProcessor
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
func (builder *NetworkBuilder) AddProcessor(message proto.Message, processor MessageProcessor) {
	// Initialize map if not exist.
	if builder.processors == nil {
		builder.processors = &sync.Map{}
	}

	builder.processors.Store(message, processor)
}

func (builder *NetworkBuilder) BuildNetwork() (*Network, error) {
	if builder.keys == nil {
		return nil, errors.New("cryptography keypair not provided to network; cannot create node id")
	}

	if len(builder.address) == 0 {
		return nil, errors.New("network requires public server IP for peers to connect to")
	}

	if builder.port <= 0 || builder.port >= 65535 {
		return nil, errors.New("port to listen for peers on must be within the range (0, 65535)")
	}

	// Initialize map if not exist.
	if builder.processors == nil {
		builder.processors = &sync.Map{}
	}

	id := peer.CreateID(builder.address+":"+strconv.Itoa(builder.port), builder.keys.PublicKey)

	network := &Network{
		Keys:    builder.keys,
		Address: builder.address,
		Port:    builder.port,
		ID:      id,

		RequestNonce: 0,
		Requests:     &sync.Map{},

		Processors: builder.processors,

		Routes: dht.CreateRoutingTable(id),
	}

	return network, nil
}
