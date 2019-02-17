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
	"sync"
	"sync/atomic"
	"time"
)

type Node struct {
	Keys identity.Keypair

	nat       nat.Provider
	transport transport.Layer

	listener net.Listener
	host     string
	port     uint16

	maxMessageSize uint64

	sendMessageTimeout    time.Duration
	receiveMessageTimeout time.Duration

	sendWorkerBusyTimeout time.Duration

	onListenerErrorCallbacks *callbacks.SequentialCallbackManager
	onPeerConnectedCallbacks *callbacks.SequentialCallbackManager
	onPeerDialedCallbacks    *callbacks.SequentialCallbackManager
	onPeerInitCallbacks      *callbacks.SequentialCallbackManager

	metadata sync.Map

	kill     chan chan struct{}
	killOnce uint32
}

func NewNode(params parameters) (*Node, error) {
	if params.Port != 0 && (params.Port < 1024 || params.Port > 65535) {
		return nil, errors.Errorf("port must be either 0 or between [1024, 65535]; port specified was %d", params.Port)
	}

	if params.Transport == nil {
		return nil, errors.New("no transport layer was registered; try set params.Transport to transport.NewTCP()")
	}

	listener, err := params.Transport.Listen(params.Host, params.Port)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to start listening for peers on port %d", params.Port)
	}

	params.Port = params.Transport.Port(listener.Addr())

	node := Node{
		Keys: params.Keys,

		nat:       params.NAT,
		transport: params.Transport,

		listener: listener,
		host:     params.Host,
		port:     params.Port,

		maxMessageSize: params.MaxMessageSize,

		sendMessageTimeout:    params.SendMessageTimeout,
		receiveMessageTimeout: params.ReceiveMessageTimeout,

		sendWorkerBusyTimeout: params.SendWorkerBusyTimeout,

		onListenerErrorCallbacks: callbacks.NewSequentialCallbackManager(),
		onPeerConnectedCallbacks: callbacks.NewSequentialCallbackManager(),
		onPeerDialedCallbacks:    callbacks.NewSequentialCallbackManager(),
		onPeerInitCallbacks:      callbacks.NewSequentialCallbackManager(),

		kill: make(chan chan struct{}, 1),
	}

	for key, val := range params.Metadata {
		node.Set(key, val)
	}

	if node.nat != nil {
		err = node.nat.AddMapping(node.transport.String(), node.port, node.port, 1*time.Hour)
		if err != nil {
			return nil, errors.Wrap(err, "nat: failed to port-forward")
		}
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
		case signal := <-n.kill:
			close(signal)
			return
		default:
		}

		conn, err := n.listener.Accept()

		if err != nil {
			n.onListenerErrorCallbacks.RunCallbacks(err)
			continue
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
func (n *Node) OnListenerError(c OnErrorCallback) {
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
func (n *Node) OnPeerConnected(c OnPeerInitCallback) {
	n.onPeerConnectedCallbacks.RegisterCallback(func(params ...interface{}) error {
		if len(params) != 1 {
			panic(errors.Errorf("noise: OnPeerConnected received unexpected args %v", params))
		}

		return c(n, params[0].(*Peer))
	})
}

// OnPeerDisconnected registers a callback whenever a peer has been disconnected.
func (n *Node) OnPeerDisconnected(srcCallbacks ...OnPeerDisconnectCallback) {
	n.onPeerInitCallbacks.RegisterCallback(func(params ...interface{}) error {
		if len(params) != 1 {
			panic(errors.Errorf("noise: OnPeerDisconnected received unexpected args %v", params))
		}

		peer := params[0].(*Peer)
		peer.OnDisconnect(srcCallbacks...)
		return nil
	})
}

// OnPeerDialed registers a callback for whenever a peer has been successfully dialed.
func (n *Node) OnPeerDialed(c OnPeerInitCallback) {
	n.onPeerDialedCallbacks.RegisterCallback(func(params ...interface{}) error {
		if len(params) != 1 {
			panic(errors.Errorf("noise: OnPeerDialed received unexpected args %v", params))
		}

		return c(n, params[0].(*Peer))
	})
}

func (n *Node) OnPeerInit(srcCallbacks ...OnPeerInitCallback) {
	targetCallbacks := make([]callbacks.Callback, 0, len(srcCallbacks))

	for _, c := range srcCallbacks {
		c := c
		targetCallbacks = append(targetCallbacks, func(params ...interface{}) error {
			if len(params) != 1 {
				panic(errors.Errorf("noise: OnPeerInit received unexpected args %v", params))
			}

			return c(n, params[0].(*Peer))
		})
	}

	n.onPeerInitCallbacks.RegisterCallback(targetCallbacks...)
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
	if !atomic.CompareAndSwapUint32(&n.killOnce, 0, 1) {
		return
	}

	signal := make(chan struct{})
	n.kill <- signal

	if err := n.listener.Close(); err != nil {
		n.onListenerErrorCallbacks.RunCallbacks(err)
	}

	<-signal
	close(n.kill)

	if n.nat != nil {
		err := n.nat.DeleteMapping(n.transport.String(), n.port, n.port)

		if err != nil {
			panic(errors.Wrap(err, "nat: failed to remove port-forward"))
		}
	}
}

func (n *Node) ExternalAddress() string {
	if n.nat != nil {
		externalIP, err := n.nat.ExternalIP()
		if err != nil {
			panic(err)
		}

		return fmt.Sprintf("%s:%d", externalIP.String(), n.port)
	}

	return fmt.Sprintf("%s:%d", n.host, n.Port())
}
