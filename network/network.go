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

	// Map of connection addresses (string) <-> *Network.PeerClient
	// so that the Network doesn't dial multiple times to the same ip
	Peers *StringPeerClientSyncMap

	Connections map[string]*ConnectionState
	ConnectionsMutex sync.Mutex

	// <-Listening will block a goroutine until this node is listening for peers.
	Listening chan struct{}
}

type ConnectionState struct {
	sendQueue chan *protobuf.Message
	recvQueue chan *protobuf.Message
	needClose chan struct{}
	closed chan struct{}
}

func dialAddress(address string) (net.Conn, error) {
	urlInfo, err := url.Parse(address)
	if err != nil {
		return nil, err
	}

	var conn net.Conn

	if urlInfo.Scheme == "kcp" {
		conn, err = kcp.DialWithOptions(urlInfo.Host, nil, 10, 3)
	} else if urlInfo.Scheme == "tcp" {
		conn, err = net.Dial("tcp", urlInfo.Host)
	} else {
		err = errors.New("Invalid scheme: " + urlInfo.Scheme)
	}

	// Failed to connect.
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (s *ConnectionState) IsClosed() bool {
	select {
	case <-s.closed:
		return true
	default:
		return false
	}
}

func (s *ConnectionState) RequestClose() {
	select {
	case s.needClose <- struct{}{}:
	default:
	}
}

func (s *ConnectionState) startRecv(n *Network, c net.Conn) {
	defer close(s.recvQueue)

	for {
		if s.IsClosed() {
			return
		}

		message, err := n.receiveMessage(c)
		if err != nil {
			glog.Error(err)
			s.RequestClose()
			return
		}

		s.recvQueue <- message
	}
}

func (s *ConnectionState) startSend(n *Network, c net.Conn) {
	for {
		if s.IsClosed() {
			return
		}

		var message *protobuf.Message

		select {
		case message = <-s.sendQueue:
		case <-s.closed:
			return
		}

		err := n.sendMessage(c, message)
		if err != nil {
			glog.Error(err)
			s.RequestClose()
			return
		}
	}
}

func (n *Network) HandleState(address string, state *ConnectionState, conn net.Conn) {
	defer func() {
		if conn != nil {
			conn.Close()
		}

		n.ConnectionsMutex.Lock()
		delete(n.Connections, address)
		n.ConnectionsMutex.Unlock()
	}()

	if conn == nil {
		var err error
		conn, err = dialAddress(address)
		if err != nil {
			glog.Error(err)
			return
		}
	}

	go state.startSend(n, conn)
	go state.startRecv(n, conn)

	<-state.needClose
	close(state.closed)
}

func (n *Network) EnsureConnectionState(address string, conn net.Conn) *ConnectionState {
	var state *ConnectionState

	n.ConnectionsMutex.Lock()
	if n.Connections == nil {
		n.Connections = make(map[string]*ConnectionState)
	}
	if _, ok := n.Connections[address]; !ok {
		n.Connections[address] = &ConnectionState {
			sendQueue: make(chan *protobuf.Message, 4096),
			recvQueue: make(chan *protobuf.Message, 4096),
			needClose: make(chan struct{}),
			closed: make(chan struct{}),
		}
		state = n.Connections[address]
		go n.HandleState(address, state, conn)
		if conn == nil {
			glog.Infof("Ensure: New connection with %s", address)
		}
	} else {
		state = n.Connections[address]
	}
	n.ConnectionsMutex.Unlock()

	return state
}

func (n *Network) GetConnectionState(address string) (*ConnectionState, bool) {
	n.ConnectionsMutex.Lock()
	defer n.ConnectionsMutex.Unlock()

	if n.Connections == nil {
		return nil, false
	}

	if state, ok := n.Connections[address]; ok {
		return state, true
	} else {
		return nil, false
	}
}

func (n *Network) WriteMessage(address string, message *protobuf.Message) error {
	state := n.EnsureConnectionState(address, nil)
	state.sendQueue <- message
	return nil
}

func (n *Network) ReadMessage(address string) (*protobuf.Message, error) {
	state, ok := n.GetConnectionState(address)
	if !ok {
		return nil, fmt.Errorf("State not found for %s", address)
	}
	message, ok := <-state.recvQueue
	if !ok {
		return nil, fmt.Errorf("State closed for %s", address)
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

	n.EnsureConnectionState(address, nil)

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
