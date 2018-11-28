package skademlia

import (
	"bytes"
	"encoding/hex"
	"net"

	"github.com/perlin-network/noise/base"
	"github.com/perlin-network/noise/crypto/blake2b"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protocol"

	"github.com/pkg/errors"
)

var _ protocol.ConnectionAdapter = (*ConnectionAdapter)(nil)

type Dialer func(address string) (net.Conn, error)

type ConnectionAdapter struct {
	listener net.Listener
	dialer   Dialer
	svc      *Service
}

func NewConnectionAdapter(listener net.Listener, dialer Dialer, id peer.ID) (*ConnectionAdapter, error) {
	return &ConnectionAdapter{
		listener: listener,
		dialer:   dialer,
	}, nil
}

func (a *ConnectionAdapter) SetSKademliaService(service *Service) {
	a.svc = service
}

func (a *ConnectionAdapter) EstablishActively(c *protocol.Controller, local []byte, remote []byte) (protocol.MessageAdapter, error) {
	if a.svc == nil {
		return nil, errors.New("skademlia: connection not setup with a service")
	}

	if bytes.Equal(local, remote) {
		return nil, errors.New("skademlia: skip connecting to self pk")
	}

	localPeer := a.svc.Routes.self
	if !bytes.Equal(local, localPeer.PublicKey) {
		return nil, errors.Errorf("skademlia: invalid local peer: %s != %s", hex.EncodeToString(local), a.svc.Routes.self.PublicKeyHex())
	}

	remotePeer, ok := a.svc.Routes.GetPeerFromPublicKey(remote)
	if !ok {
		hexID := hex.EncodeToString(remote)
		return nil, errors.Errorf("skademlia: remote ID %s not found in routing table", hexID)
	}

	if localPeer.Address == remotePeer.Address {
		return nil, errors.New("Skip connecting to self address")
	}

	conn, err := a.dialer(remotePeer.Address)
	if err != nil {
		return nil, err
	}

	return base.NewMessageAdapter(a, conn, local, remote, localPeer.Address, remotePeer.Address, false)
}

func (a *ConnectionAdapter) EstablishPassively(c *protocol.Controller, localID []byte) chan protocol.MessageAdapter {
	if a.svc == nil {
		return nil
	}
	localPeer := a.svc.Routes.self
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

			adapter, err := base.NewMessageAdapter(a, conn, localPeer.PublicKey, nil, localPeer.Address, "", true)
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

// GetPeerIDs returns the public keys of all connected nodes in the routing table
func (a *ConnectionAdapter) GetPeerIDs() [][]byte {
	results := [][]byte{}
	for _, peer := range a.svc.Routes.GetPeers() {
		results = append(results, peer.PublicKey)
	}
	return results
}

func (a *ConnectionAdapter) GetAddressByID(remote []byte) (string, error) {
	if peer, ok := a.svc.Routes.GetPeer(blake2b.New().HashBytes(remote)); ok {
		return peer.Address, nil
	}
	return "", errors.New("skademlia: peer not found")
}

func (a *ConnectionAdapter) AddPeerID(remote []byte, addr string) error {
	hexID := hex.EncodeToString(remote)
	log.Debug().
		Str("local", hex.EncodeToString(a.svc.Routes.Self().PublicKey)).
		Msgf("adding %s to routing table", hexID)
	id := NewID(remote, addr)
	err := a.svc.Routes.Update(id)
	if err == ErrBucketFull {
		if ok, _ := a.svc.EvictLastSeenPeer(id.Id); ok {
			return a.svc.Routes.Update(id)
		}
	}
	return nil
}
