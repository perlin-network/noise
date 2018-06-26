package network

import (
	"net"
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
	"strings"
	"fmt"
)

type Network struct {
	// Routing table.
	Routes *dht.RoutingTable

	// Node's keypair.
	Keys *crypto.KeyPair

	// Node's Network information.
	// The Address is `Host:Port`.
	Host string
	Port int

	// Map of incomingStream message processors for the Network.
	// map[string]MessageProcessor
	Processors *StringMessageProcessorSyncMap

	// Node's cryptographic ID.
	ID peer.ID

	listener net.Listener

	// Map of connection addresses (string) <-> *Network.PeerClient
	// so that the Network doesn't dial multiple times to the same ip
	Peers *StringPeerClientSyncMap
}

var (
	dialTimeout = 3 * time.Second
)

func (n *Network) Address() string {
	return n.Host + ":" + strconv.Itoa(n.Port)
}

// Listen starts listening for peers on a port.
func (n *Network) Listen() {
	listener, err := kcp.ListenWithOptions(":"+strconv.Itoa(n.Port), nil, 10, 3)
	if err != nil {
		glog.Fatal(err)
		return
	}

	glog.Infof("Listening for peers on port %d.", n.Port)

	// Handle new clients.
	for {
		if conn, err := listener.AcceptKCP(); err == nil {
			go n.handleMux(conn)
		} else {
			glog.Error(err)
		}
	}
}

func (n *Network) handleMux(conn net.Conn) {
	config := smux.DefaultConfig()
	config.MaxReceiveBuffer = 8192
	config.KeepAliveInterval = 1 * time.Second

	session, err := smux.Server(conn, config)
	if err != nil {
		glog.Error(err)
		return
	}

	defer session.Close()

	client := newPeerClient(n)

	// Handle new streams and process their incoming messages.
	for {
		stream, err := session.AcceptStream()
		if err != nil {
			glog.Error(err)
			return
		}

		// One goroutine per request stream.
		go client.handleIncomingRequest(stream)
	}
}

// Bootstrap with a number of peers and commence a handshake.
func (n *Network) Bootstrap(addresses ...string) {
	addresses = FilterPeers(n.Host, n.Port, addresses)

	for _, address := range addresses {
		client, err := n.Dial(address)
		if err != nil {
			continue
		}

		// Send a handshake request.
		err = client.Tell(&protobuf.HandshakeRequest{})
		if err != nil {
			continue
		}
	}
}

// Loads the peer from n.Peers and opens it
func (n *Network) GetPeer(address string) (*PeerClient, bool) {
	client, ok := n.Peers.Load(address)
	if !ok || client == nil {
		return nil, false
	}

	return client, true
}

func (n *Network) Dial(address string) (*PeerClient, error) {
	address = strings.TrimSpace(address)
	if len(address) == 0 {
		return nil, fmt.Errorf("cannot dial, address was empty")
	}

	address, err := ToUnifiedAddress(address)
	if err != nil {
		return nil, err
	}

	// load a cached connection
	if client, ok := n.GetPeer(address); ok && client != nil {
		return client, nil
	}

	client := newPeerClient(n)

	err = client.establishConnection(address)
	if err != nil {
		glog.Infof(fmt.Sprintf("Failed to connect to peer %s err=[%+v]\n", address, err))
		return nil, err
	}

	// Cache the peer's client.
	n.Peers.Store(address, client)

	return client, nil
}