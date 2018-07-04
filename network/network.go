package network

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"

	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"github.com/pkg/errors"
	"github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
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
			go n.Accept(conn)
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

// Dial establishes a bidirectional connection to an address, and additionally handshakes with said address.
// Blocks until the worker responsible for bidirectional communication is gracefully stopped.
func (n *Network) Dial(address string) (*PeerClient, error) {
	client, err := n.Client(address)
	if err != nil {
		return nil, err
	}

	// Establish an outgoing connection.
	outgoing, err := n.dial(address)

	// Failed to establish an outgoing connection.
	if err != nil {
		return nil, err
	}

	// The worker does not have an incoming connection yet; handled in Accept().
	// However, have the worker become available in sending messages.
	worker := n.spawnWorker(address)
	go worker.startSender(n, outgoing)
	go n.handleWorker(address, worker)

	return client, nil
}

func (n *Network) dial(address string) (net.Conn, error) {
	urlInfo, err := url.Parse(address)
	if err != nil {
		return nil, err
	}

	var conn net.Conn

	// Choose scheme.
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

	// Wrap a session around the outgoing connection.
	session, err := smux.Client(conn, muxConfig())
	if err != nil {
		return nil, err
	}

	// Open an outgoing stream.
	stream, err := session.OpenStream()
	if err != nil {
		return nil, err
	}

	return stream, nil
}

// Accept handles peer registration and processes incoming message streams.
func (n *Network) Accept(conn net.Conn) {
	id := n.processHandshake(conn)

	// Handshake failed.
	if id == nil {
		return
	}

	// Lets now setup our peer client.
	client, err := n.Client(id.Address)
	if err != nil {
		glog.Error(err)
		return
	}
	client.ID = id

	defer client.Close()

	for {
		msg, err := n.ReadMessage(id.Address)

		// Disconnections will occur here.
		if err != nil {
			return
		}

		id := (peer.ID)(*msg.Sender)

		// Peer sent message with a completely different ID. Destroy.
		if !client.ID.Equals(id) {
			glog.Errorf("Message signed by peer %s but client is %s", client.ID.Address, id.Address)
			return
		}

		// Unmarshal message.
		var ptr ptypes.DynamicAny
		if err := ptypes.UnmarshalAny(msg.Message, &ptr); err != nil {
			glog.Error(err)
			return
		}

		// Check if the incoming message is a response.
		if channel, exists := client.Requests.Load(msg.Nonce); exists && msg.Nonce > 0 {
			channel <- ptr.Message
			return
		}

		switch ptr.Message.(type) {
		case *protobuf.StreamPacket: // Handle stream packet message.
			pkt := ptr.Message.(*protobuf.StreamPacket)
			client.handleStreamPacket(pkt.Data)
		default: // Handle other messages.
			ctx := new(MessageContext)
			ctx.client = client
			ctx.message = ptr.Message
			ctx.nonce = msg.Nonce

			// Execute 'on receive message' callback for all plugins.
			n.Plugins.Each(func(plugin PluginInterface) {
				err := plugin.Receive(ctx)

				if err != nil {
					glog.Error(err)
				}
			})
		}
	}
}

// Plugin returns a plugins proxy interface should it be registered with the
// network. The second returning parameter is false otherwise.
//
// Example: network.Plugin((*Plugin)(nil))
func (n *Network) Plugin(key interface{}) (PluginInterface, bool) {
	return n.Plugins.Get(key)
}
