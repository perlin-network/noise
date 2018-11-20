package skademlia

import (
	"encoding/hex"
	"net"

	"github.com/perlin-network/noise/base"
	"github.com/perlin-network/noise/crypto/blake2b"
	"github.com/perlin-network/noise/log"
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
	id, ok := a.rt.GetPeerFromPublicKey(remote)
	if !ok {
		hexID := hex.EncodeToString(remote)
		return nil, errors.Errorf("skademlia: remote ID %s not found in routing table", hexID)
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

// GetPeerIDs returns the public keys of all connected nodes in the routing table
func (a *ConnectionAdapter) GetPeerIDs() [][]byte {
	results := [][]byte{}
	for _, peer := range a.rt.GetPeers() {
		results = append(results, peer.PublicKey)
	}
	return results
}

func (a *ConnectionAdapter) GetAddressByID(remote []byte) (string, error) {
	if peer, ok := a.rt.GetPeer(blake2b.New().HashBytes(remote)); ok {
		return peer.Address, nil
	}
	return "", errors.New("skademlia: peer not found")
}

func (a *ConnectionAdapter) AddPeerID(remote []byte, addr string) {
	hexID := hex.EncodeToString(remote)
	log.Debug().
		Str("local", hex.EncodeToString(a.rt.Self().PublicKey)).
		Msgf("adding %s to routing table", hexID)
	a.rt.Update(NewID(remote, addr))
}
