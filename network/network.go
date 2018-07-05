package network

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"github.com/pkg/errors"
	"github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
	"net"
	"net/url"
	"runtime"
	"sync"
	"time"
)

type Packet struct {
	RemoteAddress string
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
	Peers *StringPeerClientSyncMap

	SendQueue chan *Packet
	RecvQueue chan *Packet

	// Map of connection addresses (string) <-> net.Conn
	Connections *sync.Map

	// <-Listening will block a goroutine until this node is listening for peers.
	Listening chan struct{}
}

func (n *Network) Init() {
	workerCount := runtime.NumCPU()

	for i := 0; i < workerCount; i++ {
		// Spawn worker routines for sending queued messages to the networking layer.
		go func() {
			for {
				select {
				case packet := <-n.SendQueue:
					if stream, exists := n.Connections.Load(packet.RemoteAddress); exists {
						err := n.sendMessage(stream.(*smux.Session), packet.Payload)

						if err != nil {
							// Error sending message
							packet.Result <- err
						} else {
							// Sending message is successful.
							packet.Result <- struct{}{}
						}
					} else {
						packet.Result <- fmt.Errorf("cannot send message; not connected to peer %s", packet.RemoteAddress)
					}
				}
			}
		}()

		// Spawn worker routines for receiving and handling messages in the application layer.
		go func() {
			for {
				select {
				case packet := <-n.RecvQueue:
					if client, exists := n.Peers.Load(packet.RemoteAddress); exists {
						var ptr ptypes.DynamicAny
						if err := ptypes.UnmarshalAny(packet.Payload.Message, &ptr); err != nil {
							packet.Result <- err
							continue
						}

						client.IDInit.Do(func() {
							client.ID = (*peer.ID)(packet.Payload.Sender)
						})

						if channel, exists := client.Requests.Load(packet.Payload.Nonce); exists && packet.Payload.Nonce > 0 {
							channel <- ptr.Message
							packet.Result <- struct{}{}
							continue
						}
						
						switch ptr.Message.(type) {
						case *protobuf.StreamPacket: // Handle stream packet message.
							pkt := ptr.Message.(*protobuf.StreamPacket)
							client.handleStreamPacket(pkt.Data)
						default: // Handle other messages.
							ctx := new(MessageContext)
							ctx.client = client
							ctx.message = ptr.Message
							ctx.nonce = packet.Payload.Nonce

							// Execute 'on receive message' callback for all plugins.
							n.Plugins.Each(func(plugin PluginInterface) {
								err := plugin.Receive(ctx)

								if err != nil {
									glog.Error(err)
								}
							})
						}

						packet.Result <- struct{}{}
					}
				}
			}
		}()
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
	address, err := ToUnifiedAddress(address)
	if err != nil {
		return nil, err
	}

	if address == n.Address {
		return nil, errors.New("peer should not dial itself")
	}

	if client, loaded := n.Peers.LoadOrStore(
		address,
		createPeerClient(n, address),
	); loaded {
		return client, nil
	} else {
		client.runInitHooks()
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
		session, err := n.Dial(address)
		if err != nil {
			glog.Error(err)
			continue
		}

		ping, err := n.prepareMessage(&protobuf.Ping{})
		if err != nil {
			glog.Error(err)
			continue
		}

		err = n.sendMessage(session, ping)
		if err != nil {
			glog.Error(err)
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

	n.Connections.Store(address, session)

	_, err = n.Client(address)
	if err != nil {
		panic(err) // TODO: should never fail?
	}

	return session, nil
}

// Accept handles peer registration and processes incoming message streams.
func (n *Network) Accept(conn net.Conn) {
	var incoming *smux.Session
	var outgoing *smux.Session
	var id *peer.ID

	var err error

	// Cleanup connections when we are done with them.
	defer func() {
		if id != nil {
			n.Connections.Delete(id.Address)
		}
	}()

	// Wrap a session around the incoming connection.
	incoming, err = smux.Server(conn, muxConfig())
	if err != nil {
		return
	}
	defer incoming.Close()

	var msg *protobuf.Message

	// Attempt to receive ping/pong.
	if msg, err = n.receiveMessage(incoming, time.Now().Add(1*time.Second)); err != nil {
		return
	}

	id = (*peer.ID)(msg.Sender)

	// Load an outgoing connection, or dial to the incoming peer.
	if session, established := n.Connections.Load(id.Address); established {
		outgoing = session.(*smux.Session)
	} else {
		outgoing, err = n.Dial(id.Address)
		if err != nil {
			return
		}
	}

	// First message in a handshake must be ping/pong. Else reject.

	switch msg.Message.TypeUrl {
	case "type.googleapis.com/protobuf.Ping":
		pong, err := n.prepareMessage(&protobuf.Pong{})
		if err != nil {
			return
		}

		err = n.sendMessage(outgoing, pong)
		if err != nil {
			return
		}
	case "type.googleapis.com/protobuf.Pong":
	default:
		return
	}

	glog.Infof("Handshake completed for peer %s.", id.Address)

	for {
		msg, err := n.receiveMessage(incoming, time.Time{})

		// Will trigger 'broken pipe' on peer disconnection.
		if err != nil {
			return
		}

		// Peer sent message with a completely different ID. Disconnect.
		if !id.Equals(peer.ID(*msg.Sender)) {
			glog.Errorf("Message signed by peer %s but client is %s", peer.ID(*msg.Sender), id.Address)
			return
		}

		n.RecvQueue <- &Packet{RemoteAddress: id.Address, Payload: msg, Result: make(chan interface{}, 1)}
	}
}

// Plugin returns a plugins proxy interface should it be registered with the
// network. The second returning parameter is false otherwise.
//
// Example: network.Plugin((*Plugin)(nil))
func (n *Network) Plugin(key interface{}) (PluginInterface, bool) {
	return n.Plugins.Get(key)
}

func (n *Network) PrepareMessage(message proto.Message) (*protobuf.Message, error) {
	return n.prepareMessage(message)
}

// prepareMessage marshals a message into a *protobuf.Message and signs it with this
// nodes private key. Errors if the message is null.
func (n *Network) prepareMessage(message proto.Message) (*protobuf.Message, error) {
	if message == nil {
		return nil, errors.New("message is null")
	}

	raw, err := ptypes.MarshalAny(message)
	if err != nil {
		return nil, err
	}

	id := protobuf.ID(n.ID)

	signature, err := n.Keys.Sign(raw.Value)
	if err != nil {
		return nil, err
	}

	msg := &protobuf.Message{
		Message:   raw,
		Sender:    &id,
		Signature: signature,
	}

	return msg, nil
}
