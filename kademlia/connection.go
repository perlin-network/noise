package kademlia

import (
	"encoding/hex"
	"github.com/perlin-network/noise/base"
	"github.com/perlin-network/noise/kademlia/discovery"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
	"net"
)

const (
	NumClosestPeers = 10
)

var _ protocol.ConnectionAdapter = (*ConnectionAdapter)(nil)

type Dialer func(address string) (net.Conn, error)

type ConnectionAdapter struct {
	listener  net.Listener
	Dialer    Dialer
	discovery *discovery.Service
}

func NewConnectionAdapter(listener net.Listener, dialer Dialer) (*ConnectionAdapter, error) {
	return &ConnectionAdapter{
		listener: listener,
		Dialer:   dialer,
	}, nil
}

func (a *ConnectionAdapter) SetDiscoveryService(discovery *discovery.Service) {
	a.discovery = discovery
}

func (a *ConnectionAdapter) EstablishActively(c *protocol.Controller, local []byte, remote []byte) (protocol.MessageAdapter, error) {
	if a.discovery == nil {
		return nil, errors.New("Connection not setup with discovery")
	}

	localPeer := a.discovery.Routes.Self()

	remotePeer, ok := a.discovery.Routes.LookupPeer(remote)
	if !ok {
		return nil, errors.Errorf("peer cannot be looked up: %s", hex.EncodeToString(remote))
	}

	conn, err := a.Dialer(remotePeer.Address)
	if err != nil {
		return nil, err
	}

	return base.NewMessageAdapter(a, conn, localPeer.PublicKey, remotePeer.PublicKey, localPeer.Address, remotePeer.Address, false)
}

func (a *ConnectionAdapter) EstablishPassively(c *protocol.Controller, local []byte) chan protocol.MessageAdapter {
	if a.discovery == nil {
		return nil
	}
	localPeer := a.discovery.Routes.Self()
	ch := make(chan protocol.MessageAdapter)
	go func() {
		defer close(ch)
		for {
			select {
			case <-c.Cancellation:
				return
			default:
			}

			conn, err := a.listener.Accept() // TODO: timeout
			if err != nil {
				log.Error().Err(err).Msg("unable to accept connection")
				continue
			}

			adapter, err := base.NewMessageAdapter(a, conn, local, nil, localPeer.Address, "", true)
			if err != nil {
				log.Error().Err(err).Msg("unable to start message adapter")
				continue
			}

			// update the local peer address
			localPeer.Address = adapter.Metadata()["localAddr"]

			ch <- adapter
		}
	}()
	return ch
}

func (a *ConnectionAdapter) AddPeerID(id []byte, addr string) {
	if a.discovery == nil || len(id) == 0 || len(addr) == 0 {
		return
	}
	a.discovery.Routes.Update(peer.CreateID(addr, id))
}

func (a *ConnectionAdapter) GetPeerIDs() [][]byte {
	if a.discovery == nil {
		return nil
	}
	var results [][]byte
	peers := a.discovery.Routes.FindClosestPeers(a.discovery.Routes.Self(), NumClosestPeers)
	for _, peer := range peers {
		results = append(results, peer.PublicKey)
	}
	return results
}
