package network

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"google.golang.org/grpc"
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
	resolvedAddr := resolveAddresses(addresses)
	addresses = FilterPeers(n.Host, n.Port, resolvedAddr)

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
	peer, ok := n.Peers.Load(address)
	if !ok || peer == nil {
		return nil, false
	}

	client := peer.(*PeerClient)

	err := client.open()
	if err != nil {
		return nil, false
	}

	return client, true
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
	if client, ok := n.GetPeer(address); ok {
		return client, nil
	}

	client := createPeerClient(n.server)

	err = client.establishConnection(address)
	if err != nil {
		glog.Infof(fmt.Sprintf("Failed to connect to peer %s err=[%+v]\n", address, err))
		client.close()
		return nil, err
	}

	// Cache the peer's client.
	n.Peers.Store(address, client)

	return client, nil
}

// Asynchronously broadcast a message to all peer clients.
func (n *Network) Broadcast(message proto.Message) {
	n.Peers.Range(func(key, value interface{}) bool {
		client := value.(*PeerClient)
		err := client.Tell(message)

		if err != nil {
			glog.Warningf("Failed to send message to peer %s [err=%s]", client.Id.Address, err)
		}

		return true
	})
}

// Asynchronously broadcast a message to a set of peer clients denoted by their addresses.
func (n *Network) BroadcastByAddresses(message proto.Message, addresses ...string) {
	for _, address := range addresses {
		if client, ok := n.GetPeer(address); ok {
			err := client.Tell(message)

			if err != nil {
				glog.Warningf("Failed to send message to peer %s [err=%s]", client.Id.Address, err)
			}

			client.close()
		} else {
			glog.Warningf("Failed to send message to peer %s; peer does not exist.", address)
		}
	}
}

// Asynchronously broadcast a message to a set of peer clients denoted by their peer IDs.
func (n *Network) BroadcastByIds(message proto.Message, ids ...peer.ID) {
	for _, id := range ids {
		if client, ok := n.GetPeer(id.Address); ok {
			err := client.Tell(message)

			if err != nil {
				glog.Warningf("Failed to send message to peer %s [err=%s]", client.Id.Address, err)
			}

			client.close()
		} else {
			glog.Warningf("Failed to send message to peer %s; peer does not exist.", id)
		}
	}
}

// Asynchronously broadcast message to random selected K peers.
// Does not guarantee broadcasting to exactly K peers.
func (n *Network) BroadcastRandomly(message proto.Message, K int) {
	var addresses []string

	n.Peers.Range(func(key, value interface{}) bool {
		client := value.(*PeerClient)
		addresses = append(addresses, client.Id.Address)

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

func resolveAddresses(addresses []string) []string {
	retVal := []string{}
	for _, address := range addresses {
		resolved, err := ToUnifiedAddress(address)
		if err != nil {
			continue
		}
		retVal = append(retVal, resolved)
	}
	return retVal
}
