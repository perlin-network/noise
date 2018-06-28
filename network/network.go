package network

import (
	"fmt"
	"math/rand"
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
)

// Network represents the current networking state for this node.
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

	// Map of connection addresses (string) <-> *Network.PeerClient
	// so that the Network doesn't dial multiple times to the same ip
	Peers *StringPeerClientSyncMap

	// <-Listening will block a goroutine until this node is listening for peers.
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

	if address == n.Address() {
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
	for len(n.Listening) == 0 {}
}

// Bootstrap with a number of peers and commence a handshake.
func (n *Network) Bootstrap(addresses ...string) {
	n.BlockUntilListening()

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
	fmt.Printf("[debug] broadcasting ...\n")

	var peers []string
	n.Peers.Range(func(key string, client *PeerClient) bool {
		peers = append(peers, fmt.Sprintf("%s(%s)", key, client.Id.Address))
		err := client.Tell(message)

		if err != nil {
			glog.Warningf("Failed to send message to peer %v [err=%s]", client.Id, err)
		}

		return true
	})

	fmt.Printf("[debug] peers %v\n", peers)

}

// BroadcastByAddresses broadcasts a message to a set of peer clients denoted by their addresses.
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

// BroadcastByIds broadcasts a message to a set of peer clients denoted by their peer IDs.
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

// BroadcastRandomly asynchronously broadcast message to random selected K peers.
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
