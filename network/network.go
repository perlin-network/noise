package network

import (
	"fmt"
	"math/rand"
	"strings"
	"net"
	"net/url"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"github.com/pkg/errors"
	"github.com/xtaci/kcp-go"
)

// Network represents the current networking state for this node.
type Network struct {
	// Routing table.
	Routes *dht.RoutingTable

	// Node's keypair.
	Keys *crypto.KeyPair

	// Full address to listen on. `protocol://host:port`
	Address string

	// Map of incomingStream message processors for the Network.
	// map[string]MessageProcessor
	Processors *StringMessageProcessorSyncMap

	// Node's cryptographic ID.
	ID peer.ID

	// Map of connection addresses (string) <-> *Network.PeerClient
	// so that the Network doesn't dial multiple times to the same ip
	Peers *StringPeerClientSyncMap

	// <-Listening will block a goroutine until this node is listening for peers.
	Listening chan struct{}
}

type Acceptable interface {
	Accept() (net.Conn, error)
}

type KCPAcceptor struct {
	listener *kcp.Listener
}

func (a *KCPAcceptor) Accept() (net.Conn, error) {
	return a.listener.Accept()
}

// Listen starts listening for peers on a port.
func (n *Network) Listen() {
	uInfo, err := url.Parse(n.Address)
	if err != nil {
		glog.Fatal(err)
	}

	var listener Acceptable

	if uInfo.Scheme == "kcp" {
		var rawListener *kcp.Listener
		rawListener, err = kcp.ListenWithOptions(uInfo.Host, nil, 10, 3)
		if err == nil {
			listener = &KCPAcceptor { listener: rawListener, }
		}
	} else if uInfo.Scheme == "tcp" {
		listener, err = net.Listen("tcp", uInfo.Host)
	} else {
		err = errors.New("Invalid scheme: " + uInfo.Scheme)
	}

	if err != nil {
		glog.Fatal(err)
	}

	close(n.Listening)

	glog.Infof("Listening for peers on %s.", n.Address)

	// Handle new clients.
	for {
		if conn, err := listener.Accept(); err == nil {
			go n.Ingest(conn)
		} else {
			glog.Error(err)
		}
	}
}

// Client either creates or returns a cached peer client given its host address.
func (n *Network) Client(address string) (*PeerClient, error) {
	address = strings.TrimSpace(address)
	if len(address) == 0 {
		return nil, fmt.Errorf("cannot dial, address was empty")
	}

	address, err := ToUnifiedAddress(address)
	if err != nil {
		return nil, err
	}

	if address == n.Address {
		return nil, errors.New("peer should not dial itself")
	}

	if client, exists := n.Peers.Load(address); exists {
		return client, nil
	}

	client := createPeerClient(n)
	n.Peers.Store(address, client)

	return client, nil
}

// BlockUntilListening blocks until this node is listening for new peers.
func (n *Network) BlockUntilListening() {
	<-n.Listening
}

// Bootstrap with a number of peers and commence a handshake.
func (n *Network) Bootstrap(addresses ...string) {
	n.BlockUntilListening()

	addresses = FilterPeers(n.Address, addresses)

	for _, address := range addresses {
		client, err := n.Dial(address)
		if err != nil {
			glog.Warning(err)
			continue
		}

		// Send a ping.
		err = client.Tell(&protobuf.Ping{})
		if err != nil {
			glog.Error(err)
			continue
		}
	}
}

// Dial establishes a connection to a peer and returns a PeerClient instance to it.
func (n *Network) Dial(address string) (*PeerClient, error) {
	client, err := n.Client(address)
	if err != nil {
		return nil, err
	}

	err = client.establishConnection(address)
	if err != nil {
		glog.Warningf("Failed to connect to peer %s err=[%+v]\n", address, err)
		return nil, err
	}

	return client, nil
}

// Broadcast asynchronously broadcasts a message to all peer clients.
func (n *Network) Broadcast(message proto.Message) {
	n.Peers.Range(func(key string, client *PeerClient) bool {
		err := client.Tell(message)

		if err != nil {
			glog.Warningf("Failed to send message to peer %v [err=%s]", client.ID, err)
		}

		return true
	})
}

// BroadcastByAddresses broadcasts a message to a set of peer clients denoted by their addresses.
func (n *Network) BroadcastByAddresses(message proto.Message, addresses ...string) {
	for _, address := range addresses {
		if client, err := n.Client(address); err == nil {
			err := client.Tell(message)

			if err != nil {
				glog.Warningf("Failed to send message to peer %s [err=%s]", client.ID.Address, err)
			}
		} else {
			glog.Warningf("Failed to send message to peer %s; peer does not exist. [err=%s]", address, err)
		}
	}
}

// BroadcastByIDs broadcasts a message to a set of peer clients denoted by their peer IDs.
func (n *Network) BroadcastByIDs(message proto.Message, ids ...peer.ID) {
	for _, id := range ids {
		if client, err := n.Client(id.Address); err == nil {
			err := client.Tell(message)

			if err != nil {
				glog.Warningf("Failed to send message to peer %s [err=%s]", client.ID.Address, err)
			}
		} else {
			glog.Warningf("Failed to send message to peer %s; peer does not exist. [err=%s]", id, err)
		}
	}
}

// BroadcastRandomly asynchronously broadcast message to random selected K peers.
// Does not guarantee broadcasting to exactly K peers.
func (n *Network) BroadcastRandomly(message proto.Message, K int) {
	var addresses []string

	n.Peers.Range(func(key string, client *PeerClient) bool {
		addresses = append(addresses, client.ID.Address)

		// Limit total amount of addresses in case we have a lot of peers.
		if len(addresses) > K*3 {
			return false
		}

		return true
	})

	// Flip a coin and shuffle :).
	rand.Shuffle(len(addresses), func(i, j int) {
		addresses[i], addresses[j] = addresses[j], addresses[i]
	})

	if len(addresses) < K {
		K = len(addresses)
	}

	n.BroadcastByAddresses(message, addresses[:K]...)
}
