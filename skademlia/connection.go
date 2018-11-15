package skademlia

import (
	"github.com/perlin-network/noise/base"
	"github.com/perlin-network/noise/protocol"
	"net"
)

var _ protocol.ConnectionAdapter = (*ConnectionAdapter)(nil)

type ConnectionAdapter struct {
	baseConn *base.ConnectionAdapter
	RoutingTable
}

func NewConnectionAdapter(listener net.Listener, dialer base.Dialer, id ID) (*ConnectionAdapter, error) {
	baseConn, err := base.NewConnectionAdapter(listener, dialer)
	if err != nil {
		return nil, err
	}
	table := CreateRoutingTable(id)
	return &ConnectionAdapter{
		baseConn:     baseConn,
		RoutingTable: *table,
	}, nil
}

func (a *ConnectionAdapter) EstablishActively(c *protocol.Controller, local []byte, remote []byte) (protocol.MessageAdapter, error) {
	return a.baseConn.EstablishActively(c, local, remote)
}

func (a *ConnectionAdapter) EstablishPassively(c *protocol.Controller, local []byte) chan protocol.MessageAdapter {
	return a.baseConn.EstablishPassively(c, local)
}

func (a *ConnectionAdapter) GetConnectionIDs() [][]byte {
	return a.baseConn.GetConnectionIDs()
}

func (a *ConnectionAdapter) AddPeer(id []byte, addr string) error {
	return nil
}
