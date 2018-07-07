package network

import (
	"math/rand"
	"net"
	"net/url"
	"runtime"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"github.com/pkg/errors"
	"github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
)

var packetPool = sync.Pool{
	New: func() interface{} {
		return new(Packet)
	},
}

var contextPool = sync.Pool{
	New: func() interface{} {
		return new(PluginContext)
	},
}

type Packet struct {
	Target  string
	Payload *protobuf.Message
	Result  chan interface{}
}

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
	Peers *sync.Map

	SendQueue chan *Packet
	RecvQueue chan *protobuf.Message

	// Map of connection addresses (string) <-> net.Conn
	Connections *sync.Map

	// <-Listening will block a goroutine until this node is listening for peers.
	Listening chan struct{}

	SignaturePolicy crypto.SignaturePolicy
	HashPolicy      crypto.HashPolicy

	// <-Kill will begin the server shutdown process
	Kill chan struct{}
}

// Init starts all network I/O workers.
func (n *Network) Init() {
	workerCount := runtime.NumCPU() + 1

	for i := 0; i < workerCount; i++ {
		// Spawn worker routines for sending queued messages to the networking layer.
		go n.handleSendQueue()

		// Spawn worker routines for receiving and handling messages in the application layer.
		go n.handleRecvQueue()
	}
}

// Send queue worker.
func (n *Network) handleSendQueue() {
	for {
		select {
		case packet := <-n.SendQueue:
			if session, exists := n.Connections.Load(packet.Target); exists {
				stream, err := session.(*smux.Session).OpenStream()
				if err != nil {
					packet.Result <- err
					continue
				}

				err = n.sendMessage(stream, packet.Payload)
				if err != nil {
					packet.Result <- err
					continue
				}

				err = stream.Close()
				if err != nil {
					packet.Result <- err
					continue
				}

				packet.Result <- struct{}{}
			} else {
				packet.Result <- errors.Errorf("cannot send message; not connected to peer %s", packet.Target)
			}
		}
	}
}

// Receive queue worker.
func (n *Network) handleRecvQueue() {
	for {
		select {
		case msg := <-n.RecvQueue:
			if client, exists := n.Peers.Load(msg.Sender.Address); exists {
				client := client.(*PeerClient)

				// Check if the client is ready.
				if !client.IncomingReady() {
					continue
				}

				var ptr ptypes.DynamicAny
				if err := ptypes.UnmarshalAny(msg.Message, &ptr); err != nil {
					continue
				}

				if channel, exists := client.Requests.Load(msg.Nonce); exists && msg.Nonce > 0 {
					channel.(chan proto.Message) <- ptr.Message
					continue
				}

				switch ptr.Message.(type) {
				case *protobuf.Bytes:
					client.handleBytes(ptr.Message.(*protobuf.Bytes).Data)
				default:
					ctx := contextPool.New().(*PluginContext)
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

					contextPool.Put(ctx)
				}
			}
		}
	}
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
		err = errors.New("invalid scheme: " + urlInfo.Scheme)
	}

	if err != nil {
		glog.Fatal(err)
	}

	close(n.Listening)

	glog.Infof("Listening for peers on %s.\n", n.Address)

	// handle server shutdowns
	go func() {
		select {
		case <-n.Kill:
			// cause listener.Accept() to stop blocking so it can continue the loop
			listener.Close()
		}
	}()

	// Handle new clients.
	for {
		if conn, err := listener.Accept(); err == nil {
			go n.Accept(conn)

		} else {
			// if the Shutdown flag is set, no need to continue with the for loop
			select {
			case <-n.Kill:
				glog.Infof("Shutting down server on %s.\n", n.Address)
				return
			default:
				// without the default case the select will block.
			}

			glog.Error(err)
		}
	}
}

// Client either creates or returns a cached peer client given its host address.
func (n *Network) Client(address string) (*PeerClient, error) {
	address, err := ToUnifiedAddress(address)
	if err != nil {
		return nil, err
	}

	if address == n.Address {
		return nil, errors.New("peer should not dial itself")
	}

	client, err := createPeerClient(n, address)
	if err != nil {
		return nil, err
	}

	if client, exists := n.Peers.LoadOrStore(address, client); exists {
		client := client.(*PeerClient)

		if !client.OutgoingReady() {
			return nil, errors.New("peer failed to connect")
		}

		return client, nil
	} else {
		client := client.(*PeerClient)

		defer func() {
			close(client.outgoingReady)
		}()

		session, err := n.Dial(address)

		if err != nil {
			n.Peers.Delete(address)
			return nil, err
		}

		n.Connections.Store(address, session)

		// Execute 'peer connect' callback for all registered plugins.
		n.Plugins.Each(func(plugin PluginInterface) {
			plugin.PeerConnect(client)
		})

		return client, nil
	}
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
		client, err := n.Client(address)

		if err != nil {
			glog.Error(err)
			continue
		}

		err = client.Tell(&protobuf.Ping{})
		if err != nil {
			continue
		}
	}
}

// Dial establishes a bidirectional connection to an address, and additionally handshakes with said address.
func (n *Network) Dial(address string) (*smux.Session, error) {
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
		err = errors.New("invalid scheme: " + urlInfo.Scheme)
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

	return session, nil
}

// Accept handles peer registration and processes incoming message streams.
func (n *Network) Accept(conn net.Conn) {
	var incoming *smux.Session
	var outgoing *smux.Session

	var client *PeerClient
	var clientInit sync.Once

	var err error

	// Cleanup connections when we are done with them.
	defer func() {
		if client != nil {
			client.Close()
		}

		if incoming != nil {
			incoming.Close()
		}

		if outgoing != nil {
			outgoing.Close()
		}
	}()

	// Wrap a session around the incoming connection.
	incoming, err = smux.Server(conn, muxConfig())
	if err != nil {
		glog.Error(err)
		return
	}

	for {
		stream, err := incoming.AcceptStream()
		if err != nil {
			break
		}

		go func() {
			defer stream.Close()

			var err error

			// Receive a message from the stream.
			msg, err := n.receiveMessage(stream)

			// Will trigger 'broken pipe' on peer disconnection.
			if err != nil {
				return
			}

			// Initialize client if not exists.
			clientInit.Do(func() {
				client, err = n.Client(msg.Sender.Address)
				if err != nil {
					glog.Error(err)
					return
				}

				client.ID = (*peer.ID)(msg.Sender)

				// Load an outgoing connection.
				if session, established := n.Connections.Load(client.ID.Address); established {
					outgoing = session.(*smux.Session)
				} else {
					err = errors.New("failed to load session")
				}

				// Signal that the client is ready.
				close(client.incomingReady)
			})

			if err != nil {
				return
			}

			// Peer sent message with a completely different ID. Disconnect.
			if !client.ID.Equals(peer.ID(*msg.Sender)) {
				glog.Errorf("Message signed by peer %s but client is %s", peer.ID(*msg.Sender), client.ID.Address)
				return
			}

			select {
			case n.RecvQueue <- msg:
			default:
				//glog.Errorf("recv queue full, dropping messages")
			}
		}()

	}
}

// Plugin returns a plugins proxy interface should it be registered with the
// network. The second returning parameter is false otherwise.
//
// Example: network.Plugin((*Plugin)(nil))
func (n *Network) Plugin(key interface{}) (PluginInterface, bool) {
	return n.Plugins.Get(key)
}

// PrepareMessage marshals a message into a *protobuf.Message and signs it with this
// nodes private key. Errors if the message is null.
func (n *Network) PrepareMessage(message proto.Message) (*protobuf.Message, error) {
	if message == nil {
		return nil, errors.New("message is null")
	}

	raw, err := ptypes.MarshalAny(message)
	if err != nil {
		return nil, err
	}

	id := protobuf.ID(n.ID)

	signature, err := n.Keys.Sign(
		n.SignaturePolicy,
		n.HashPolicy,
		raw.Value,
	)
	if err != nil {
		return nil, err
	}

	msg := &protobuf.Message{}
	msg.Message = raw
	msg.Sender = &id
	msg.Signature = signature

	return msg, nil
}

// Write asynchronously sends a message to a denoted target address.
func (n *Network) Write(address string, message *protobuf.Message) error {
	packet := packetPool.New().(*Packet)
	defer packetPool.Put(packet)

	packet.Target = address
	packet.Payload = message
	packet.Result = make(chan interface{}, 1)

	select {
	case n.SendQueue <- packet:
	default:
		return errors.New("send queue full")
	}

	select {
	case raw := <-packet.Result:
		switch err := raw.(type) {
		case error:
			return errors.Wrapf(err, "failed to send message to %s", address)
		default:
			return nil
		}
	case <-time.After(3 * time.Second):
		return errors.Errorf("worker must be too busy; failed to send message to %s", address)
	}

	return nil
}

// Broadcast asynchronously broadcasts a message to all peer clients.
func (n *Network) Broadcast(message proto.Message) {
	n.Peers.Range(func(key, value interface{}) bool {
		client := value.(*PeerClient)

		err := client.Tell(message)

		if err != nil {
			glog.Warningf("Failed to send message to peer %v [err=%s]", client.ID, err)
		}

		return true
	})
}

// BroadcastByAddresses broadcasts a message to a set of peer clients denoted by their addresses.
func (n *Network) BroadcastByAddresses(message proto.Message, addresses ...string) {
	signed, err := n.PrepareMessage(message)
	if err != nil {
		return
	}

	for _, address := range addresses {
		n.Write(address, signed)
	}
}

// BroadcastByIDs broadcasts a message to a set of peer clients denoted by their peer IDs.
func (n *Network) BroadcastByIDs(message proto.Message, ids ...peer.ID) {
	signed, err := n.PrepareMessage(message)
	if err != nil {
		return
	}

	for _, id := range ids {
		n.Write(id.Address, signed)
	}
}

// BroadcastRandomly asynchronously broadcasts a message to random selected K peers.
// Does not guarantee broadcasting to exactly K peers.
func (n *Network) BroadcastRandomly(message proto.Message, K int) {
	var addresses []string

	n.Peers.Range(func(key, value interface{}) bool {
		client := value.(*PeerClient)

		addresses = append(addresses, client.Address)

		// Limit total amount of addresses in case we have a lot of peers.
		return len(addresses) <= K*3
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

// Close shuts down the entire network.
func (n *Network) Close() {
	// Kill the listener.
	close(n.Kill)

	// Clean out client connections.
	n.Peers.Range(func(key, value interface{}) bool {
		value.(*PeerClient).Close()
		return true
	})
}
