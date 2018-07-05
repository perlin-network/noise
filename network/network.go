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
	"math/rand"
)

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
	Peers *StringPeerClientSyncMap

	SendQueue chan *Packet
	RecvQueue chan *protobuf.Message

	// Map of connection addresses (string) <-> net.Conn
	Connections *sync.Map

	// <-Listening will block a goroutine until this node is listening for peers.
	Listening chan struct{}
}

// Init starts all network I/O workers.
func (n *Network) Init() {
	workerCount := runtime.NumCPU()

	for i := 0; i < workerCount; i++ {
		// Spawn worker routines for sending queued messages to the networking layer.
		go func() {
			for {
				select {
				case packet := <-n.SendQueue:
					if session, exists := n.Connections.Load(packet.Target); exists {
						stream, err := session.(*smux.Session).OpenStream()
						if err != nil {
							packet.Result <- err
							continue
						}

						err = sendMessage(stream, packet.Payload)
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
						packet.Result <- fmt.Errorf("cannot send message; not connected to peer %s", packet.Target)
					}
				}
			}
		}()

		// Spawn worker routines for receiving and handling messages in the application layer.
		go func() {
			for {
				select {
				case packet := <-n.RecvQueue:
					if client, exists := n.Peers.Load(packet.Sender.Address); exists {
						var ptr ptypes.DynamicAny
						if err := ptypes.UnmarshalAny(packet.Message, &ptr); err != nil {
							continue
						}

						if channel, exists := client.Requests.Load(packet.Nonce); exists && packet.Nonce > 0 {
							channel <- ptr.Message
							continue
						}

						switch ptr.Message.(type) {

						case *protobuf.StreamPacket:
							client.handleStreamPacket(ptr.Message.(*protobuf.StreamPacket).Data)
						default:
							ctx := new(MessageContext)
							ctx.client = client
							ctx.message = ptr.Message
							ctx.nonce = packet.Nonce

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

	if client, existed := n.Peers.Load(address); existed {
		return client, nil
	} else {
		session, err := n.Dial(address)

		if err != nil {
			return nil, err
		}

		n.Connections.Store(address, session)

		client := createPeerClient(n, address)
		n.Peers.Store(address, client)

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

	return session, nil
}

// Accept handles peer registration and processes incoming message streams.
func (n *Network) Accept(conn net.Conn) {
	var incoming *smux.Session
	var outgoing *smux.Session
	var client *PeerClient

	var err error

	// Cleanup connections when we are done with them.
	defer func() {
		if client != nil {
			n.Connections.Delete(client.Address)

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
		// Open a new stream.
		stream, err := incoming.AcceptStream()
		if err != nil {
			break
		}

		go func() {
			defer stream.Close()

			msg, err := receiveMessage(stream)

			// Will trigger 'broken pipe' on peer disconnection.
			if err != nil {
				return
			}

			// Initialize client if not exists.
			if client == nil {
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
					return
				}
			}

			// Peer sent message with a completely different ID. Disconnect.
			if !client.ID.Equals(peer.ID(*msg.Sender)) {
				glog.Errorf("Message signed by peer %s but client is %s", peer.ID(*msg.Sender), client.ID.Address)
				return
			}

			n.RecvQueue <- msg
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

// Write asynchronously sends a message to a denoted target address.
func (n *Network) Write(address string, message *protobuf.Message) error {
	packet := &Packet{Target: address, Payload: message, Result: make(chan interface{}, 1)}

	n.SendQueue <- packet

	select {
	case raw := <-packet.Result:
		switch err := raw.(type) {
		case error:
			return err
		default:
			return nil
		}
	}

	return nil
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
