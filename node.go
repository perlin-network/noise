package noise

import (
	"fmt"
	"github.com/perlin-network/noise/callbacks"
	"github.com/perlin-network/noise/identity"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/nat"
	"github.com/perlin-network/noise/transport"
	"github.com/pkg/errors"
	"net"
	"runtime"
	"strconv"
	"sync"
)

type Node struct {
	ID identity.Manager

	nat       nat.Provider
	transport transport.Layer

	listener net.Listener
	port     uint16

	maxMessageSize uint64

	onListenerErrorCallbacks *callbacks.SequentialCallbackManager
	onPeerConnectedCallbacks *callbacks.SequentialCallbackManager
	onPeerDialedCallbacks    *callbacks.SequentialCallbackManager
	onPeerInitCallbacks      *callbacks.SequentialCallbackManager

	metadata sync.Map

	kill     chan struct{}
	killOnce sync.Once
}

func NewNode(params parameters) (*Node, error) {
	if params.Port != 0 && (params.Port < 1024 || params.Port > 65535) {
		return nil, errors.Errorf("port must be either 0 or between [1024, 65535]; port specified was %d", params.Port)
	}

	if params.Transport == nil {
		return nil, errors.New("no transport layer was registered; try set params.Transport to transport.NewTCP()")
	}

	listener, err := params.Transport.Listen(params.Port)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to start listening for peers on port %d", params.Port)
	}

	params.Port = params.Transport.Port(listener.Addr())

	node := Node{
		ID: params.ID,

		nat:       params.NAT,
		transport: params.Transport,

		listener: listener,
		port:     params.Port,

		maxMessageSize: params.MaxMessageSize,

		onListenerErrorCallbacks: callbacks.NewSequentialCallbackManager(),
		onPeerConnectedCallbacks: callbacks.NewSequentialCallbackManager(),
		onPeerDialedCallbacks:    callbacks.NewSequentialCallbackManager(),
		onPeerInitCallbacks:      callbacks.NewSequentialCallbackManager(),

		kill: make(chan struct{}, 1),
	}

	for key, val := range params.Metadata {
		node.Set(key, val)
	}

	return &node, nil
}

func (n *Node) Port() uint16 {
	return n.port
}

// Listen makes our node start listening for peers.
func (n *Node) Listen() {
	for {
		select {
		case <-n.kill:
			if err := n.listener.Close(); err != nil {
				n.onListenerErrorCallbacks.RunCallbacks(err)
			}

			n.listener = nil
			return
		default:
		}

		conn, err := n.listener.Accept()

		if err != nil {
			n.onListenerErrorCallbacks.RunCallbacks(err)
		}

		peer := newPeer(n, conn)
		peer.init()

		if errs := n.onPeerConnectedCallbacks.RunCallbacks(peer); len(errs) > 0 {
			log.Warn().Errs("errors", errs).Msg("Got errors running OnPeerConnected callbacks.")
		}
		if errs := n.onPeerInitCallbacks.RunCallbacks(peer); len(errs) > 0 {
			log.Warn().Errs("errors", errs).Msg("Got errors running OnPeerInit callbacks.")
		}

	}
}

// Dial has our node attempt to dial and establish a connection with a remote peer.
func (n *Node) Dial(address string) (*Peer, error) {
	if n.ExternalAddress() == address {
		return nil, errors.New("noise: node attempted to dial itself")
	}

	fmt.Println("Dialing", address)

	conn, err := n.transport.Dial(address)

	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to peer %s", conn)
	}

	peer := newPeer(n, conn)
	peer.init()

	if errs := n.onPeerDialedCallbacks.RunCallbacks(peer); len(errs) > 0 {
		log.Error().Errs("errors", errs).Msg("Got errors running OnPeerConnected callbacks.")
	}
	if errs := n.onPeerInitCallbacks.RunCallbacks(peer); len(errs) > 0 {
		log.Error().Errs("errors", errs).Msg("Got errors running OnPeerInit callbacks.")
	}

	return peer, nil
}

// OnListenerError registers a callback for whenever our nodes listener fails to accept an incoming peer.
func (n *Node) OnListenerError(c onErrorCallback) {
	n.onListenerErrorCallbacks.RegisterCallback(func(params ...interface{}) error {
		if len(params) != 1 {
			panic(errors.Errorf("noise: OnListenerError received unexpected args %v", params))
		}

		err, ok := params[0].(error)

		if !ok {
			return nil
		}

		return c(n, errors.Wrap(err, "failed to accept an incoming peer"))
	})
}

// OnPeerConnected registers a callback for whenever a peer has successfully been accepted by our node.
func (n *Node) OnPeerConnected(c onPeerInitCallback) {
	n.onPeerConnectedCallbacks.RegisterCallback(func(params ...interface{}) error {
		if len(params) != 1 {
			panic(errors.Errorf("noise: OnPeerConnected received unexpected args %v", params))
		}

		return c(n, params[0].(*Peer))
	})
}

// OnPeerDisconnected registers a callback whenever a peer has been disconnected.
func (n *Node) OnPeerDisconnected(c onPeerDisconnectCallback) {
	n.onPeerInitCallbacks.RegisterCallback(func(params ...interface{}) error {
		if len(params) != 1 {
			panic(errors.Errorf("noise: OnPeerDisconnected received unexpected args %v", params))
		}

		peer := params[0].(*Peer)
		peer.OnDisconnect(c)

		return nil
	})
}

// OnPeerDialed registers a callback for whenever a peer has been successfully dialed.
func (n *Node) OnPeerDialed(c onPeerInitCallback) {
	n.onPeerDialedCallbacks.RegisterCallback(func(params ...interface{}) error {
		if len(params) != 1 {
			panic(errors.Errorf("noise: OnPeerDialed received unexpected args %v", params))
		}

		return c(n, params[0].(*Peer))
	})
}

func (n *Node) OnPeerInit(c onPeerInitCallback) {
	_, file, no, ok := runtime.Caller(1)
	if ok {
		log.Debug().Msgf("OnPeerInit() called from %s#%d.", file, no)
	}

	n.onPeerInitCallbacks.RegisterCallback(func(params ...interface{}) error {
		if len(params) != 1 {
			panic(errors.Errorf("noise: OnPeerInit received unexpected args %v", params))
		}

		return c(n, params[0].(*Peer))
	})
}

// OnMessageReceived registers a callback for whenever any peer sends a message to our node.
// Returning noise.DeRegisterCallback will deregister the callback only from a single peer.
func (n *Node) OnMessageReceived(o Opcode, c onMessageReceivedCallback) {
	n.OnPeerInit(func(node *Node, peer *Peer) error {
		peer.OnMessageReceived(o, c)
		return nil
	})
}

// Set sets a metadata entry given a key-value pair on our node.
func (n *Node) Set(key string, val interface{}) {
	n.metadata.Store(key, val)
}

// Get returns the value to a metadata key from our node, or otherwise returns nil should
// there be no corresponding value to a provided key.
func (n *Node) Get(key string) interface{} {
	val, _ := n.metadata.Load(key)
	return val
}

func (n *Node) LoadOrStore(key string, val interface{}) interface{} {
	val, _ = n.metadata.LoadOrStore(key, val)
	return val
}

func (n *Node) Has(key string) bool {
	_, exists := n.metadata.Load(key)
	return exists
}

func (n *Node) Delete(key string) {
	n.metadata.Delete(key)
}

// Fence blocks the current goroutine until the node stops listening for peers.
func (n *Node) Fence() {
	<-n.kill
}

func (n *Node) Kill() {
	n.killOnce.Do(func() {
		n.kill <- struct{}{}
	})
}

func (n *Node) ExternalAddress() string {
	//return n.nat.ExternalIP().String() + ":" + strconv.Itoa(int(n.port))
	return "127.0.0.1" + ":" + strconv.Itoa(int(n.port))
}
