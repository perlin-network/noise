package skademlia

import (
	"bytes"
	"context"
	"encoding/hex"
	"github.com/perlin-network/noise/skademlia/dht"
	"net"

	"github.com/perlin-network/noise/base"
	"github.com/perlin-network/noise/crypto/blake2b"
	"github.com/perlin-network/noise/internal/protobuf"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/skademlia/peer"

	"github.com/pkg/errors"
)

var _ protocol.ConnectionAdapter = (*ConnectionAdapter)(nil)

type Dialer func(address string) (net.Conn, error)

type ConnectionAdapter struct {
	listener    net.Listener
	dialer      Dialer
	Discovery   *DiscoveryService
	sendAdapter protocol.SendAdapter
}

func NewConnectionAdapter(listener net.Listener, dialer Dialer, node *protocol.Node, localAddr string) (*ConnectionAdapter, error) {
	a := &ConnectionAdapter{
		listener:    listener,
		dialer:      dialer,
		sendAdapter: node,
		Discovery:   NewDiscoveryService(node, dht.NewID(node.GetIdentityAdapter().MyIdentity(), localAddr)),
	}

	if ia, ok := node.GetIdentityAdapter().(*IdentityAdapter); ok {
		node.SetCustomHandshakeProcessor(NewHandshakeProcessor(ia))
	} else {
		return nil, errors.New("Node identity adapter type should be skademlia type")
	}
	node.SetConnectionAdapter(a)
	node.AddService(a.Discovery)

	return a, nil
}

func (a *ConnectionAdapter) Dial(c *protocol.Controller, local []byte, remote []byte) (protocol.MessageAdapter, error) {
	if a.Discovery == nil {
		return nil, errors.New("skademlia: connection not setup with a service")
	}

	if bytes.Equal(local, remote) {
		return nil, errors.New("skademlia: skip connecting to self pk")
	}

	localPeer := a.Discovery.Routes.Self()
	if !bytes.Equal(local, localPeer.PublicKey) {
		return nil, errors.Errorf("skademlia: invalid local peer: %s != %s", hex.EncodeToString(local), a.Discovery.Routes.Self().PublicKeyHex())
	}

	remotePeer, ok := a.Discovery.Routes.GetPeerFromPublicKey(remote)
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

	return base.NewMessageAdapterActive(a, conn, local, remote, localPeer.Address, remotePeer.Address)
}

func (a *ConnectionAdapter) Accept(c *protocol.Controller, local []byte) chan protocol.MessageAdapter {
	if a.Discovery == nil {
		return nil
	}
	localPeer := a.Discovery.Routes.Self()
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

			adapter, err := base.NewMessageAdapterPassive(a, conn, localPeer.PublicKey, localPeer.Address)
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
	for _, peer := range a.Discovery.Routes.GetPeers() {
		results = append(results, peer.PublicKey)
	}
	return results
}

func (a *ConnectionAdapter) GetAddressByID(remote []byte) (string, error) {
	if peer, ok := a.Discovery.Routes.GetPeer(blake2b.New().HashBytes(remote)); ok {
		return peer.Address, nil
	}
	return "", errors.New("skademlia: peer not found")
}

func (a *ConnectionAdapter) AddPeerID(remote []byte, addr string) error {
	id := dht.NewID(remote, addr)
	err := a.Discovery.Routes.Update(id)
	if err == dht.ErrBucketFull {
		if ok, _ := a.Discovery.EvictLastSeenPeer(id.Id); ok {
			return a.Discovery.Routes.Update(id)
		}
	}
	return nil
}

func (a *ConnectionAdapter) Bootstrap(peers ...peer.ID) error {
	if a.sendAdapter == nil {
		return errors.New("node not setup properly")
	}
	if a.Discovery == nil {
		return errors.New("discovery not setup properly")
	}
	if len(peers) == 0 {
		return nil
	}
	// add all the peers
	for _, peer := range peers {
		if err := a.AddPeerID(peer.PublicKey, peer.Address); err != nil {
			return err
		}
	}
	body, err := ToMessageBody(ServiceID, OpCodePing, &protobuf.Ping{})
	if err != nil {
		return err
	}
	// broadcast a ping to all the peers
	return a.sendAdapter.Broadcast(context.Background(), body)
}
