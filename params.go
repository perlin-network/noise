package noise

import (
	"github.com/perlin-network/noise/identity"
	"github.com/perlin-network/noise/nat"
	"github.com/perlin-network/noise/transport"
)

type parameters struct {
	Host string
	Port uint16

	NAT       nat.Provider
	ID        identity.Manager
	Transport transport.Layer

	Metadata map[string]interface{}

	MaxMessageSize uint64
}

func DefaultParams() parameters {
	return parameters{
		Host:           "127.0.0.1",
		Transport:      transport.NewTCP(),
		Metadata:       map[string]interface{}{},
		MaxMessageSize: 1048576,
	}
}
