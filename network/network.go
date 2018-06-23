package network

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
	"google.golang.org/grpc"
	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/peer"
	"github.com/golang/glog"
	"github.com/perlin-network/noise/protobuf"
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

	// Map of incoming message processors for the Network.
	// map[string]MessageProcessor
	Processors *sync.Map

	// Node's cryptographic ID.
	ID peer.ID

	listener net.Listener
	server   *Server

	// Map of connection addresses (string) <-> *network.PeerClient
	// so that the network doesn't dial multiple times to the same ip
	Peers *sync.Map
}

var (
	dialTimeout = 3 * time.Second
)

func (n *Network) Address() string {
	return n.Host + ":" + strconv.Itoa(n.Port)
}

// Listen for peers on a port specified on instantation of Network{}.
func (n *Network) Listen() {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(n.Port))
	if err != nil {
		glog.Fatal(err)
		return
	}

	client := grpc.NewServer()
	server := createServer(n)

	protobuf.RegisterNoiseServer(client, server)

	n.listener = listener
	n.server = server

	glog.Infof("Listening for peers on port %d.", n.Port)

	err = client.Serve(listener)
	if err != nil {
		glog.Fatal(err)
		return
	}
}

// Bootstrap with a number of peers and commence a handshake.
func (n *Network) Bootstrap(addresses ...string) {
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

// Dials a peer via. gRPC.
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
	if client, ok := n.Peers.Load(address); ok && client != nil {
		return client.(*PeerClient), nil
	}

	client := CreatePeerClient(n.server)

	err = client.establishConnection(address)
	if err != nil {
		glog.Infof(fmt.Sprintf("Failed to connect to peer %s err=[%+v]\n", address, err))
		return nil, err
	}

	// Cache the peer's client.
	n.Peers.Store(address, client)

	return client, nil
}

type Sendable interface {
	Send(*protobuf.Message) error
}
