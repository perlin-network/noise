package network

import (
	"fmt"
	"math/rand"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"github.com/pkg/errors"
	"github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
)

type Network struct {
	// Routing table.
	Routes *dht.RoutingTable

	// Node's keypair.
	Keys *crypto.KeyPair

	// Node's Network information.
	// The Address is `Host:Port`.
	Host string
	Port uint16

	// Map of incomingStream message processors for the Network.
	// map[string]MessageProcessor
	Processors *StringMessageProcessorSyncMap

	// Node's cryptographic ID.
	ID peer.ID

	listener net.Listener

	// Map of connection addresses (string) <-> *Network.PeerClient
	// so that the Network doesn't dial multiple times to the same ip
	Peers *StringPeerClientSyncMap

	Listening chan struct{}
}

// Address returns a formated host:port string
func (n *Network) Address() string {
	return n.Host + ":" + strconv.Itoa(int(n.Port))
}

// Listen starts listening for peers on a port.
func (n *Network) Listen() {
	listener, err := kcp.ListenWithOptions(":"+strconv.Itoa(int(n.Port)), nil, 10, 3)
	if err != nil {
		glog.Fatal(err)
		return
	}

	n.Listening <- struct{}{}

	glog.Infof("Listening for peers on port %d.", n.Port)

	// Handle new clients.
	for {
		if conn, err := listener.Accept(); err == nil {
			go n.handleMux(conn)
		} else {
			glog.Error(err)
		}
	}
}

func (n *Network) handleMux(conn net.Conn) {
	session, err := smux.Server(conn, muxConfig())
	if err != nil {
		glog.Error(err)
		return
	}

	defer session.Close()

	client := createPeerClient(n)

	// Handle new streams and process their incoming messages.
	for {
		stream, err := session.AcceptStream()
		if err != nil {
			if err.Error() == "broken pipe" {
				client.Close()
			}
			break
		}

		// One goroutine per request stream.
		go client.handleMessage(stream)
	}
}

// Bootstrap with a number of peers and commence a handshake.
func (n *Network) Bootstrap(addresses ...string) {
	addresses = FilterPeers(n.Host, n.Port, addresses)

	for _, address := range addresses {
		client, err := n.Dial(address)
		if err != nil {
			glog.Warning(err)
			continue
		}

		// Send a handshake request.
		err = client.Tell(&protobuf.HandshakeRequest{})
		if err != nil {
			glog.Error(err)
			continue
		}
	}
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

	if address == n.Address() {
		return nil, errors.New("peer should not dial itself")
	}

	// load a cached connection
	if client, exists := n.Peers.Load(address); exists && client != nil {
		return client, nil
	}

	client := createPeerClient(n)

	err = client.establishConnection(address)
	if err != nil {
		glog.Warningf("Failed to connect to peer %s err=[%+v]\n", address, err)
		return nil, err
	}

	// Cache the peer's client.
	n.Peers.Store(address, client)

	return client, nil
}

// Broadcast asynchronously sends a message to all peer clients.
func (n *Network) Broadcast(message proto.Message) {
	var peerList []string
	var routeList []string

	// get a list of peers in the routing table
	for _, peer := range n.Routes.GetPeers() {
		routeList = append(routeList, peer.Address)
	}

	// get a list of peers in the peer list
	n.Peers.Range(func(key string, client *PeerClient) bool {
		peerList = append(peerList, key)
		return true
	})

	// add missing peers before dialing
	n.dialMissingPeers(peerList, routeList)

	// get a list of peers in the peer list
	n.Peers.Range(func(key string, client *PeerClient) bool {
		if err := client.Tell(message); err != nil {
			glog.Warningf("Failed to send message to peer %v [err=%s]", client.Id, err)
		}

		return true
	})
}

// Asynchronously broadcast a message to a set of peer clients denoted by their addresses.
func (n *Network) BroadcastByAddresses(message proto.Message, addresses ...string) {
	for _, address := range addresses {
		if client, ok := n.Peers.Load(address); ok {
			err := client.Tell(message)

			if err != nil {
				glog.Warningf("Failed to send message to peer %s [err=%s]", client.Id.Address, err)
			}

			client.Close()
		} else {
			glog.Warningf("Failed to send message to peer %s; peer does not exist.", address)
		}
	}
}

// Asynchronously broadcast a message to a set of peer clients denoted by their peer IDs.
func (n *Network) BroadcastByIds(message proto.Message, ids ...peer.ID) {
	for _, id := range ids {
		if client, ok := n.Peers.Load(id.Address); ok {
			err := client.Tell(message)

			if err != nil {
				glog.Warningf("Failed to send message to peer %s [err=%s]", client.Id.Address, err)
			}

			client.Close()
		} else {
			glog.Warningf("Failed to send message to peer %s; peer does not exist.", id)
		}
	}
}

// Asynchronously broadcast message to random selected K peers.
// Does not guarantee broadcasting to exactly K peers.
func (n *Network) BroadcastRandomly(message proto.Message, K int) {
	var addresses []string

	n.Peers.Range(func(key string, client *PeerClient) bool {
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

func (n *Network) dialMissingPeers(peerList []string, routeList []string) {
	// TODO: can cache the listHashes and skip the check

	// make sure every route has a peer
	sort.Strings(peerList)
	sort.Strings(routeList)

	for r, p := 0, 0; r < len(routeList); r++ {
		if p >= len(peerList) || routeList[r] != peerList[p] {
			// this is a missing peer, add it to the peers
			if _, err := n.Dial(routeList[r]); err != nil {
				glog.Infof("Could not dial address: %s", routeList[r])
			}
		} else {
			p++
		}
	}
}
