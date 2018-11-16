package skademlia

import (
	"net"

	"github.com/perlin-network/noise/base"
	"github.com/perlin-network/noise/crypto/blake2b"
	"github.com/perlin-network/noise/protocol"

	"github.com/pkg/errors"
)

var _ protocol.ConnectionAdapter = (*ConnectionAdapter)(nil)

type ConnectionAdapter struct {
	baseConn *base.ConnectionAdapter
	rt       RoutingTable
}

func NewConnectionAdapter(listener net.Listener, dialer base.Dialer, id ID) (*ConnectionAdapter, error) {
	baseConn, err := base.NewConnectionAdapter(listener, dialer)
	if err != nil {
		return nil, err
	}
	table := CreateRoutingTable(id)
	return &ConnectionAdapter{
		baseConn: baseConn,
		rt:       *table,
	}, nil
}

func (a *ConnectionAdapter) EstablishActively(c *protocol.Controller, local []byte, remote []byte) (protocol.MessageAdapter, error) {
	ok, id := a.rt.GetPeerFromPublicKey(remote)
	if !ok {
		return nil, errors.New("skademlia: remote ID not found in routing table")
	}

	conn, err := a.baseConn.Dialer(id.Address)
	if err != nil {
		return nil, err
	}

	return base.NewMessageAdapter(a.baseConn, conn, local, remote, id.Address, false)
}

func (a *ConnectionAdapter) EstablishPassively(c *protocol.Controller, localID []byte) chan protocol.MessageAdapter {
	return a.baseConn.EstablishPassively(c, localID)
}

func (a *ConnectionAdapter) GetConnectionIDs() [][]byte {
	results := [][]byte{}
	for _, peer := range a.rt.GetPeers() {
		results = append(results, peer.id)
	}
	return results
}

func (a *ConnectionAdapter) GetAddressByID(id []byte) (string, error) {
	// TODO:
	return "", errors.New("Not implemented")
}

func (a *ConnectionAdapter) AddConnection(remote []byte, addr string) {
	// TODO:
	id := ID{blake2b.New().HashBytes(remote), addr}
	a.rt.Update(id)
}
