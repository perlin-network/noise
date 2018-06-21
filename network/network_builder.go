package network

import (
	"errors"
	"strconv"
	"sync"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/peer"
)

type NetworkBuilder struct {
	keys    *crypto.KeyPair
	address string
	port    int
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

	id := peer.CreateID(builder.address+":"+strconv.Itoa(builder.port), builder.keys.PublicKey)

	network := &Network{
		Keys:    builder.keys,
		Address: builder.address,
		Port:    builder.port,
		ID:      id,

		RequestNonce: 0,
		Requests:     &sync.Map{},

		Routes: dht.CreateRoutingTable(id),

		ConnPool: &sync.Map{},
	}

	return network, nil
}
