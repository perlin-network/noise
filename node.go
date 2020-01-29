package noise

import (
	"context"
	"errors"
	"fmt"
	"github.com/oasislabs/ed25519"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"net"
	"strconv"
	"time"
)

type Node struct {
	logger *zap.Logger

	host net.IP
	port uint16
	addr string

	publicKey  PublicKey
	privateKey PrivateKey

	id ID

	maxDialAttempts        int
	maxInboundConnections  int
	maxOutboundConnections int

	idleTimeout time.Duration

	listener  net.Listener
	listening atomic.Bool

	outbound *clientMap
	inbound  *clientMap

	codec    *Codec
	binders  []Binder
	handlers []Handler

	kill chan error
}

// NewNode instantiates a new node instance, and pre-configures the node with provided options.
// Default values for some non-specified options are instantiated as well, which may yield an error.
func NewNode(opts ...NodeOption) (*Node, error) {
	n := &Node{kill: make(chan error, 1)}

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

	if n.maxDialAttempts == 0 {
		n.maxDialAttempts = 3
	}

	if n.maxInboundConnections == 0 {
		n.maxInboundConnections = 128
	}

	if n.maxOutboundConnections == 0 {
		n.maxOutboundConnections = 128
	}

	if n.idleTimeout == 0 {
		n.idleTimeout = 10 * time.Second
	}

	n.inbound = newClientMap(n.maxInboundConnections)
	n.outbound = newClientMap(n.maxOutboundConnections)

	n.codec = NewCodec()

	return n, nil
}

// Listen has the node start listening for new peers. If an error occurs while starting the listener due to
// misconfigured options or resource exhaustion, an error is returned.
//
// Listen may be called concurrently.
func (n *Node) Listen() error {
	if !n.listening.CAS(false, true) {
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
		err = fmt.Errorf("did not listen for tcp: %w", n.listener.Close())
		return err
	}

	n.host = addr.IP
	n.port = uint16(addr.Port)

	if n.addr == "" {
		n.addr = net.JoinHostPort(normalizeIP(n.host), strconv.FormatUint(uint64(n.port), 10))
	}

	n.id = NewID(n.publicKey, n.host, n.port)

	for _, binder := range n.binders {
		if err = binder.Bind(n); err != nil {
			return err
		}
	}

	go func() {
		defer close(n.kill)
		defer n.inbound.release()
		defer n.listening.Store(false)

		for {
			conn, err := n.listener.Accept()
			if err != nil {
				n.kill <- err
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

func (n *Node) RegisterMessage(ser Serializable, de interface{}) uint16 {
	return n.codec.Register(ser, de)
}

func (n *Node) EncodeMessage(msg Serializable) ([]byte, error) {
	return n.codec.Encode(msg)
}

func (n *Node) DecodeMessage(data []byte) (Serializable, error) {
	return n.codec.Decode(data)
}

func (n *Node) SendMessage(ctx context.Context, addr string, msg Serializable) error {
	data, err := n.EncodeMessage(msg)
	if err != nil {
		return err
	}

	return n.Send(ctx, addr, data)
}

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

func (n *Node) Send(ctx context.Context, addr string, data []byte) error {
	if !n.listening.Load() {
		return errors.New("node must be listening before it can send a message")
	}

	c, err := n.dialIfNotExists(ctx, addr)
	if err != nil {
		return err
	}

	if err := c.send(0, data); err != nil {
		return err
	}

	for _, binder := range c.node.binders {
		binder.OnMessageSent(c)
	}

	return nil
}

func (n *Node) Request(ctx context.Context, addr string, data []byte) ([]byte, error) {
	if !n.listening.Load() {
		return nil, errors.New("node must be listening before it can send a request")
	}

	c, err := n.dialIfNotExists(ctx, addr)
	if err != nil {
		return nil, err
	}

	msg, err := c.request(ctx, data)
	if err != nil {
		return nil, err
	}

	for _, binder := range c.node.binders {
		binder.OnMessageSent(c)
	}

	return msg.data, nil
}

func (n *Node) Ping(ctx context.Context, addr string) (*Client, error) {
	if !n.listening.Load() {
		return nil, errors.New("node must be listening before it can ping")
	}

	return n.dialIfNotExists(ctx, addr)
}

// Close gracefully stops all live inbound/outbound peer connections registered on this node, and stops the node
// from handling/accepting new incoming peer connections. It returns an error if an error occurs closing the nodes
// listener.
//
// Close may be called concurrently.
func (n *Node) Close() error {
	if n.listening.CAS(true, false) {
		if err := n.listener.Close(); err != nil {
			return err
		}
	}

	<-n.kill

	return nil
}

func (n *Node) dialIfNotExists(ctx context.Context, addr string) (*Client, error) {
	var err error

	if addr, err = ResolveAddress(addr); err != nil {
		return nil, err
	}

	for i := 0; i < n.maxDialAttempts; i++ {
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
			return nil, err
		}
	}

	return nil, fmt.Errorf("attempted to dial %s several times but failed: %w", addr, err)
}

func (n *Node) Bind(binders ...Binder) {
	n.binders = append(n.binders, binders...)
}

func (n *Node) Handle(handlers ...Handler) {
	n.handlers = append(n.handlers, handlers...)
}

func (n *Node) Sign(data []byte) []byte {
	return n.privateKey.Sign(data)
}

func (n *Node) Inbound() []*Client {
	return n.inbound.slice()
}

func (n *Node) Outbound() []*Client {
	return n.outbound.slice()
}

func (n *Node) Addr() string {
	return n.addr
}

func (n *Node) Logger() *zap.Logger {
	return n.logger
}

func (n *Node) ID() ID {
	return n.id
}
