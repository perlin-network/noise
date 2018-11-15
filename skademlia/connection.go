package skademlia

import (
	"github.com/perlin-network/noise/connection"
	"github.com/perlin-network/noise/protocol"
	"net"
)

var _ protocol.ConnectionAdapter = (*ConnectionAdapter)(nil)

type ConnectionAdapter struct {
	aca *connection.AddressableConnectionAdapter
	RoutingTable
}

func NewConnectionAdapter(listener net.Listener, dialer connection.Dialer) (*ConnectionAdapter, error) {
	aca, err := connection.StartAddressableConnectionAdapter(listener, dialer)
	if err != nil {
		return nil, err
	}
	return &ConnectionAdapter{
		aca: aca,
	}, nil
}

func (a *ConnectionAdapter) EstablishActively(c *protocol.Controller, local []byte, remote []byte) (protocol.MessageAdapter, error) {
	return a.aca.EstablishActively(c, local, remote)
}

func (a *ConnectionAdapter) EstablishPassively(c *protocol.Controller, local []byte) chan protocol.MessageAdapter {
	return a.aca.EstablishPassively(c, local)
}

func (a *ConnectionAdapter) GetConnectionIDs() [][]byte {
	return a.aca.GetConnectionIDs()
}

func (a *ConnectionAdapter) AddPeer(id []byte, addr string) error {
	return nil
}
