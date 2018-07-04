package network

import (
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"strings"
	"sync"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"github.com/pkg/errors"
	"github.com/xtaci/kcp-go"
)

// Network represents the current networking state for this node.
type Network struct {
	// Node's keypair.
	Keys *crypto.KeyPair

	// Full address to listen on. `protocol://host:port`
	Address string

	// Map of plugins registered to the network.
	// map[string]Plugin
	Plugins *PluginList

	// Node's cryptographic ID.
	ID peer.ID

	// Map of connection addresses (string) <-> *network.PeerClient
	// so that the Network doesn't dial multiple times to the same ip
	Peers *StringPeerClientSyncMap

	// Map of peer addresses (string) <-> *network.Worker
	Workers      map[string]*Worker
	WorkersMutex sync.Mutex

	// <-Listening will block a goroutine until this node is listening for peers.
	Listening chan struct{}
}

func (n *Network) loadWorker(address string) (*Worker, bool) {
	n.WorkersMutex.Lock()
	defer n.WorkersMutex.Unlock()

	if n.Workers == nil {
		return nil, false
	}

	if state, ok := n.Workers[address]; ok {
		return state, true
	} else {
		return nil, false
	}
}

func (n *Network) WriteMessage(address string, message *protobuf.Message) error {
	worker, available := n.loadWorker(address)
	if !available {
		return fmt.Errorf("worker not found for %s", address)
	}

	worker.sendQueue <- message
	return nil
}

func (n *Network) ReadMessage(address string) (*protobuf.Message, error) {
	worker, available := n.loadWorker(address)
	if !available {
		return nil, fmt.Errorf("worker not found for %s", address)
	}

	message, available := <-worker.recvQueue
	if !available {
		return nil, fmt.Errorf("worker closed for %s", address)
	}

	return message, nil
}

func (n *Network) GetPort() uint16 {
	info, err := ExtractAddressInfo(n.Address)
	if err != nil {
		glog.Fatal(err)
	}

	return info.Port
}

// Listen starts listening for peers on a port.
func (n *Network) Listen() {
	// Handle 'network starts listening' callback for plugins.
	n.Plugins.Each(func(plugin PluginInterface) {
		plugin.Startup(n)
	})

	// Handle 'network stops listening' callback for plugins.
	defer func() {
		n.Plugins.Each(func(plugin PluginInterface) {
			plugin.Cleanup(n)
		})
	}()

	urlInfo, err := url.Parse(n.Address)
	if err != nil {
		glog.Fatal(err)
	}

	var listener net.Listener

	if urlInfo.Scheme == "kcp" {
		listener, err = kcp.ListenWithOptions(urlInfo.Host, nil, 10, 3)
		if err != nil {
			glog.Fatal(err)
		}
	} else if urlInfo.Scheme == "tcp" {
		listener, err = net.Listen("tcp", urlInfo.Host)
	} else {
		err = errors.New("Invalid scheme: " + urlInfo.Scheme)
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

	client := createPeerClient(n, address)
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
			glog.Error(err)
			continue
		}

		err = client.Tell(&protobuf.Ping{})
		if err != nil {
			glog.Error(err)
			continue
		}
	}
}

func (n *Network) spawnWorker(address string) *Worker {
	n.WorkersMutex.Lock()
	defer n.WorkersMutex.Unlock()

	if n.Workers == nil {
		n.Workers = make(map[string]*Worker)
	}

	// Return worker if exists.
	if worker, exists := n.Workers[address]; exists {
		return worker
	}

	// Spawn and cache new worker otherwise.
	n.Workers[address] = &Worker{
		// TODO: Make queue size configurable.
		sendQueue: make(chan *protobuf.Message, 4096),
		recvQueue: make(chan *protobuf.Message, 4096),

		needClose: make(chan struct{}),
		closed:    make(chan struct{}),
	}

	return n.Workers[address]
}

// Dial establishes a bidirectional connection to an address, and additionally handshakes with said address.
// Blocks until the worker responsible for bidirectional communication is gracefully stopped.
func (n *Network) Dial(address string) (*PeerClient, error) {
	client, err := n.Client(address)
	if err != nil {
		return nil, err
	}

	// Establish an outgoing connection.
	outgoing, err := dialAddress(address)

	// Failed to establish an outgoing connection.
	if err != nil {
		return nil, err
	}

	// The worker does not have an incoming connection yet; handled in Ingest().
	// However, have the worker become available in sending messages.
	worker := n.spawnWorker(address)
	go worker.startSender(n, outgoing)
	go n.handleWorker(address, worker)

	return client, nil
}

func (n *Network) handleWorker(address string, worker *Worker) {
	defer func() {
		n.WorkersMutex.Lock()
		delete(n.Workers, address)
		n.WorkersMutex.Unlock()
	}()

	// Wait until worker is closed.
	<-worker.needClose
	close(worker.closed)
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
		n.Tell(address, message)
	}
}

// BroadcastByIDs broadcasts a message to a set of peer clients denoted by their peer IDs.
func (n *Network) BroadcastByIDs(message proto.Message, ids ...peer.ID) {
	for _, id := range ids {
		n.Tell(id.Address, message)
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

// Tell asynchronously emit a message to a denoted target address.
func (n *Network) Tell(targetAddress string, msg proto.Message) error {
	if client, err := n.Client(targetAddress); err == nil {
		err := client.Tell(msg)

		if err != nil {
			return fmt.Errorf("failed to send message to peer %s [err=%s]", targetAddress, err)
		}
	} else {
		return fmt.Errorf("failed to send message to peer %s; peer does not exist. [err=%s]", targetAddress, err)
	}

	return nil
}

// Plugin returns a plugins proxy interface should it be registered with the
// network. The second returning parameter is false otherwise.
//
// Example: network.Plugin((*Plugin)(nil))
func (n *Network) Plugin(key interface{}) (PluginInterface, bool) {
	return n.Plugins.Get(key)
}
