package skademlia

import (
	"github.com/perlin-network/noise/connection"
	"github.com/perlin-network/noise/protocol"
)

var _ protocol.ConnectionAdapter = (*ConnectionAdapter)(nil)

// ConnectionAdapter implements the protocol.ConnectionAdapter
type ConnectionAdapter struct {
	connection.AddressableConnectionAdapter
}

// NewConnectionAdapter creates a new instance.
func NewConnectionAdapter() *ConnectionAdapter {
	return &ConnectionAdapter{}
}

// EstablishPassively implements the protocol.ConnectionAdapter interface
func (ca *ConnectionAdapter) EstablishPassively(c *protocol.Controller, local []byte) chan protocol.MessageAdapter {
	return nil
}

// EstablishActively implements the protocol.ConnectionAdapter interface
func (ca *ConnectionAdapter) EstablishActively(c *protocol.Controller, local []byte, remote []byte) (protocol.MessageAdapter, error) {
	return nil, nil
}
