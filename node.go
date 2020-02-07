package noise

import (
	"context"
	"errors"
	"fmt"
	"github.com/oasislabs/ed25519"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"net"
	"runtime"
	"strconv"
	"sync"
	"time"
)

// Node keeps track of a users ID, all of a users outgoing/incoming connections to/from peers as *Client instances
// under a bounded connection pool whose bounds may be configured, the TCP listener which accepts new incoming peer
// connections, and all Go types that may be serialized/deserialized at will on-the-wire or through a Handler.
//
// A node at most will only have one goroutine + num configured worker goroutines associated to it which represents
// the listener looking to accept new incoming peer connections, and workers responsible for handling incoming peer
// messages. A node once closed or once started (as in, (*Node).Listen was called) should not be reused.
type Node struct {
	logger *zap.Logger

	host net.IP
	port uint16
	addr string

	publicKey  PublicKey
	privateKey PrivateKey

	id ID

	maxDialAttempts        uint
	maxInboundConnections  uint
	maxOutboundConnections uint
	maxRecvMessageSize     uint32
	numWorkers             uint

	idleTimeout time.Duration

	listener  net.Listener
	listening atomic.Bool

	outbound *clientMap
	inbound  *clientMap

	codec     *codec
	protocols []Protocol
	handlers  []Handler

	workers sync.WaitGroup
	work    chan HandlerContext

	listenerDone chan error
}

// NewNode instantiates a new node instance, and pre-configures the node with provided options.
// Default values for some non-specified options are instantiated as well, which may yield an error.
func NewNode(opts ...NodeOption) (*Node, error) {
	n := &Node{
		listenerDone: make(chan error, 1),

		maxDialAttempts:        3,
		maxInboundConnections:  128,
		maxOutboundConnections: 128,
		maxRecvMessageSize:     4 << 20,
		numWorkers:             uint(runtime.NumCPU()),
	}

	for _, opt := range opts {
		opt(n)
	}

	if n.logger == nil {
		n.logger = zap.NewNop()
	}

	if n.privateKey == ZeroPrivateKey {
		_, privateKey, err := ed25519.GenerateKey(nil)
		if err != nil {
			return nil, err
		}

		copy(n.privateKey[:], privateKey)
	}

	copy(n.publicKey[:], ed25519.PrivateKey(n.privateKey[:]).Public().(ed25519.PublicKey)[:])

	if n.id.ID == ZeroPublicKey && n.host != nil && n.port > 0 {
		n.id = NewID(n.publicKey, n.host, n.port)
	}

	n.inbound = newClientMap(n.maxInboundConnections)
	n.outbound = newClientMap(n.maxOutboundConnections)

	n.codec = newCodec()

	return n, nil
}

// Listen has the node start listening for new peers. If an error occurs while starting the listener due to
// misconfigured options or resource exhaustion, an error is returned. If the node is already listening
// for new connections, an error is thrown.
//
// Listen must not be called concurrently, and should only ever be called once per node instance.
func (n *Node) Listen() error {
	if n.listening.Load() {
		return errors.New("node is already listening")
	}

	var err error

	defer func() {
		if err != nil {
			n.listening.Store(false)
		}
	}()

	n.listener, err = net.Listen("tcp", net.JoinHostPort(normalizeIP(n.host), strconv.FormatUint(uint64(n.port), 10)))
	if err != nil {
		return err
	}

	addr, ok := n.listener.Addr().(*net.TCPAddr)
	if !ok {
		n.listener.Close()
		return errors.New("did not bind to a tcp addr")
	}

	n.host = addr.IP
	n.port = uint16(addr.Port)

	if n.addr == "" {
		n.addr = net.JoinHostPort(normalizeIP(n.host), strconv.FormatUint(uint64(n.port), 10))
		n.id = NewID(n.publicKey, n.host, n.port)
	} else {
		resolved, err := ResolveAddress(n.addr)
		if err != nil {
			n.listener.Close()
			return err
		}

		hostStr, portStr, err := net.SplitHostPort(resolved)
		if err != nil {
			n.listener.Close()
			return err
		}

		host := net.ParseIP(hostStr)
		if host == nil {
			n.listener.Close()
			return errors.New("host in provided public address is invalid (must be IPv4/IPv6)")
		}

		port, err := strconv.ParseUint(portStr, 10, 16)
		if err != nil {
			n.listener.Close()
			return err
		}

		n.id = NewID(n.publicKey, host, uint16(port))
	}

	for _, protocol := range n.protocols {
		if protocol.Bind == nil {
			continue
		}

		if err = protocol.Bind(n); err != nil {
			n.listener.Close()
			return err
		}
	}

	n.work = make(chan HandlerContext, int(n.numWorkers))
	n.workers.Add(int(n.numWorkers))

	for i := uint(0); i < n.numWorkers; i++ {
		go func() {
			defer n.workers.Done()

			for ctx := range n.work {
				for _, handler := range n.handlers {
					if err := handler(ctx); err != nil {
						ctx.client.Logger().Warn("Got an error executing a message handler.", zap.Error(err))
						ctx.client.reportError(err)
						ctx.client.close()

						return
					}
				}
			}
		}()
	}

	go func() {
		n.listening.Store(true)

		defer func() {
			n.inbound.release()

			close(n.work)
			n.workers.Wait()

			n.listening.Store(false)
			close(n.listenerDone)
		}()

		n.logger.Info("Listening for incoming peers.",
			zap.String("bind_addr", addr.String()),
			zap.String("id_addr", n.id.Address),
			zap.String("public_key", n.publicKey.String()),
			zap.String("private_key", n.privateKey.String()),
		)

		for {
			conn, err := n.listener.Accept()
			if err != nil {
				n.listenerDone <- err
				break
			}

			addr := conn.RemoteAddr().String()

			client, exists := n.inbound.get(n, addr)
			if !exists {
				go client.inbound(conn, addr)
			}
		}
	}()

	return nil
}

// RegisterMessage registers a Go type T that implements the Serializable interface with an associated deserialize
// function whose signature comprises of func([]byte) (T, error). RegisterMessage should be called in the following
// manner:
//
//  RegisterMessage(T{}, func([]byte) (T, error) { ... })
//
// It returns a 16-bit unsigned integer (opcode) that is associated to the type T on-the-wire. Once a Go type has been
// registered, it may be used in a Handler, or via (*Node).EncodeMessage, (*Node).DecodeMessage, (*Node).SendMessage,
// and (*Node).RequestMessage.
//
// The wire format of a type registered comprises of
// append([]byte{16-bit big-endian integer (opcode)}, ser.Marshal()...).
//
// RegisterMessage may be called concurrently, though is discouraged.
func (n *Node) RegisterMessage(ser Serializable, de interface{}) uint16 {
	return n.codec.register(ser, de)
}

// EncodeMessage encodes msg which must be a registered Go type T into its wire representation. It throws an error
// if the Go type of msg has not yet been registered through (*Node).RegisterMessage. For more details, refer to
// (*Node).RegisterMessage.
//
// EncodeMessage may be called concurrently.
func (n *Node) EncodeMessage(msg Serializable) ([]byte, error) {
	return n.codec.encode(msg)
}

// DecodeMessage decodes data into its registered Go type T should it be well-formed. It throws an error if the opcode
// at the head of data has yet to be registered/associated to a Go type via (*Node).RegisterMessage. For more details,
// refer to (*Node).RegisterMessage.
//
// DecodeMessage may be called concurrently.
func (n *Node) DecodeMessage(data []byte) (Serializable, error) {
	return n.codec.decode(data)
}

// SendMessage encodes msg which is a Go type registered via (*Node).RegisterMessage, and sends it to addr. For more
// details, refer to (*Node).Send and (*Node).RegisterMessage.
func (n *Node) SendMessage(ctx context.Context, addr string, msg Serializable) error {
	data, err := n.EncodeMessage(msg)
	if err != nil {
		return err
	}

	return n.Send(ctx, addr, data)
}

// RequestMessage encodes msg which is a Go type registered via (*Node).RegisterMessage, and sends it as a request
// to addr, and returns a decoded response from the peer at addr. For more details, refer to (*Node).Request
// and (*Node).RegisterMessage.
func (n *Node) RequestMessage(ctx context.Context, addr string, req Serializable) (Serializable, error) {
	data, err := n.EncodeMessage(req)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	data, err = n.Request(ctx, addr, data)
	if err != nil {
		return nil, err
	}

	res, err := n.DecodeMessage(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode request: %w", err)
	}

	return res, nil
}

// Send takes an available connection from this nodes connection pool if the peer at addr has never been connected
// to before, connects to it, handshakes with the peer, and sends it data.
//
// If there already exists a live connection to the peer at addr, no new connection is established and data will be
// sent through. An error is returned if connecting to the peer should it not have been connected to before
// fails, or if handshaking fails, or if the connection is closed.
//
// If there is no available connection from this nodes connection pool, the connection that is at the tail of the pool
// is closed and evicted and used to send data to addr.
func (n *Node) Send(ctx context.Context, addr string, data []byte) error {
	c, err := n.dialIfNotExists(ctx, addr)
	if err != nil {
		return err
	}

	if err := c.send(0, data); err != nil {
		return err
	}

	return nil
}

// Request takes an available connection from this nodes connection pool if the peer at addr has never been connected
// to before, connects to it, handshakes with the peer, and sends it a request should the entire process
// be successful.
//
// Once the request has been sent, the current goroutine Request was called in will block until either
// a response has been received which will be subsequently returned, ctx was canceled/expired, or the connection was
// dropped.
//
// If there already exists a live connection to the peer at addr, no new connection is established and the request
// will follow through. An error is returned if connecting to the peer should it not have been connected to before
// fails, or if handshaking fails.
//
// If there is no available connection from this nodes connection pool, the connection that is at the tail of the pool
// is closed and evicted and used to send a request to addr.
func (n *Node) Request(ctx context.Context, addr string, data []byte) ([]byte, error) {
	c, err := n.dialIfNotExists(ctx, addr)
	if err != nil {
		return nil, err
	}

	msg, err := c.request(ctx, data)
	if err != nil {
		return nil, err
	}

	return msg.data, nil
}

// Ping takes an available connection from this nodes connection pool if the peer at addr has never been connected
// to before, connects to it, handshakes with the peer, and returns a *Client instance should the entire process
// be successful.
//
// If there already exists a live connection to the peer at addr, no new connection is established and
// the *Client instance associated to the peer is returned. An error is returned if connecting to the peer should it
// not have been connected to before fails, or if ctx was canceled/expired, or if handshaking fails.
//
// If there is no available connection from this nodes connection pool, the connection that is at the tail of the pool
// is closed and evicted and used to ping addr.
//
// It is safe to call Ping concurrently.
func (n *Node) Ping(ctx context.Context, addr string) (*Client, error) {
	return n.dialIfNotExists(ctx, addr)
}

// Close gracefully stops all live inbound/outbound peer connections registered on this node, and stops the node
// from handling/accepting new incoming peer connections. It returns an error if an error occurs closing the nodes
// listener. Nodes that are closed should not ever be re-used.
//
// Close may be called concurrently.
func (n *Node) Close() error {
	if n.listening.CAS(true, false) {
		if err := n.listener.Close(); err != nil {
			return err
		}
	}

	<-n.listenerDone

	return nil
}

func (n *Node) dialIfNotExists(ctx context.Context, addr string) (*Client, error) {
	var err error

	for i := uint(0); i < n.maxDialAttempts; i++ {
		client, exists := n.outbound.get(n, addr)
		if !exists {
			go client.outbound(ctx, addr)
		}

		select {
		case <-ctx.Done():
			err = fmt.Errorf("failed to dial peer: %w", ctx.Err())
		case <-client.ready:
			err = client.Error()
		case <-client.readerDone:
			err = client.Error()
		case <-client.writerDone:
			err = client.Error()
		}

		if err == nil {
			return client, nil
		}

		client.close()
		client.waitUntilClosed()

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			for _, protocol := range n.protocols {
				if protocol.OnPingFailed == nil {
					continue
				}

				protocol.OnPingFailed(addr, err)
			}

			return nil, err
		}
	}

	err = fmt.Errorf("attempted to dial %s several times but failed: %w", addr, err)

	for _, protocol := range n.protocols {
		if protocol.OnPingFailed == nil {
			continue
		}

		protocol.OnPingFailed(addr, err)
	}

	return nil, err
}

// Bind registers a Protocol to this node, which implements callbacks for all events this node can emit throughout
// its lifecycle. For more information on how to implement Protocol, refer to the documentation for Protocol. Bind
// only registers Protocol's should the node not yet be listening for new connections. If the node is already listening
// for new peers, Bind silently returns and does nothing.
//
// Bind may be called concurrently.
func (n *Node) Bind(protocols ...Protocol) {
	if n.listening.Load() {
		return
	}

	n.protocols = append(n.protocols, protocols...)
}

// Handle registers a Handler to this node, which is executed every time this node receives a message from an
// inbound/outbound connection. For more information on how to write a Handler, refer to the documentation for
// Handler. Handle only registers Handler's should the node not yet be listening for new connections. If the node
// is already listening for new peers, Handle silently returns and does nothing.
//
// Handle may be called concurrently.
func (n *Node) Handle(handlers ...Handler) {
	if n.listening.Load() {
		return
	}

	n.handlers = append(n.handlers, handlers...)
}

// Sign uses the nodes private key to sign data and return its cryptographic signature as a slice of bytes.
func (n *Node) Sign(data []byte) Signature {
	return n.privateKey.Sign(data)
}

// Inbound returns a cloned slice of all inbound connections to this node as Client instances. It is useful
// while writing unit tests where you would want to block the current goroutine via (*Client).WaitUntilReady and
// (*Client).WaitUntilClosed to test scenarios where you want to be sure some inbound client is open/closed.
func (n *Node) Inbound() []*Client {
	return n.inbound.slice()
}

// Outbound returns a cloned slice of all outbound connections to this node as Client instances. It is useful
// while writing unit tests where you would want to block the current goroutine via (*Client).WaitUntilClosed to
// test scenarios where you want to be sure some outbound client has resources associated to it completely released.
func (n *Node) Outbound() []*Client {
	return n.outbound.slice()
}

// Addr returns the public address of this node. The public address, should it not be configured through the
// WithNodeAddress functional option when calling NewNode, is initialized to 'host:port' after a successful
// call to (*Node).Listen.
//
// Addr may be called concurrently.
func (n *Node) Addr() string {
	return n.addr
}

// Logger returns the underlying logger associated to this node. The logger, should it not be configured through the
// WithNodeLogger functional option when calling NewNode, is by default zap.NewNop().
//
// Logger may be called concurrently.
func (n *Node) Logger() *zap.Logger {
	return n.logger
}

// ID returns an immutable copy of the ID of this node. The ID of the node is set after a successful call to
// (*Node).Listen, or otherwise passed through the WithNodeID functional option when calling NewNode.
//
// ID may be called concurrently.
func (n *Node) ID() ID {
	return n.id
}
