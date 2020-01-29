package noise

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"github.com/oasislabs/ed25519"
	"go.uber.org/zap"
	"io"
	"net"
	"sync"
	"time"
)

type clientSide bool

const (
	clientSideInbound  clientSide = false
	clientSideOutbound clientSide = true
)

// Client represents an pooled inbound/outbound connection under some node. Should a client successfully undergo
// noise's protocol handshake, information about the peer representative of this client, such as its ID is available.
//
// A clients connection may be closed through (*Client).Close, through the result of a failed handshake, through
// exceeding the max inbound/outbound connection count configured on the clients associated node, through a node
// gracefully being stopped, through a Handler configured on the node returning an error upon recipient of some data,
// or through receiving unexpected/suspicious data.
//
// The lifecycle of a client may be controlled through (*Client).WaitUntilReady, and (*Client).WaitUntilClosed. It
// provably has been useful in writing unit tests where a client instance is used under high concurrency scenarios.
//
// A client in total has four goroutines associated to it: a goroutine responsible for handling writing messages, a
// goroutine responsible for handling the recipient of messages, a goroutine for timing out the client connection
// should there be no further read/writes after some configured timeout on the clients associatd node, and a goroutine
// for handling protocol logic such as handshaking/executing Handler's.
type Client struct {
	node *Node

	id ID

	addr string
	conn net.Conn
	side clientSide

	suite cipher.AEAD

	logger struct {
		sync.RWMutex
		*zap.Logger
	}

	timeout struct {
		reset chan struct{}
		timer *time.Timer
	}

	reader *connReader
	writer *connWriter

	requests *requestMap

	ready       chan struct{}
	writerDone  chan struct{}
	readerDone  chan struct{}
	handlerDone chan struct{}
	clientDone  chan struct{}

	err struct {
		sync.Mutex
		error
	}

	closeOnce sync.Once
}

func newClient(node *Node) *Client {
	c := &Client{
		node:        node,
		reader:      newConnReader(),
		writer:      newConnWriter(),
		requests:    newRequestMap(),
		ready:       make(chan struct{}),
		writerDone:  make(chan struct{}),
		readerDone:  make(chan struct{}),
		handlerDone: make(chan struct{}),

		clientDone: make(chan struct{}),
	}

	c.logger.Logger = node.logger

	return c
}

// ID returns an immutable copy of the ID of this client, which is established once the client has successfully
// completed the handshake protocol configured from this clients associated node.
//
// ID may be called concurrently.
func (c *Client) ID() ID {
	return c.id
}

// Logger returns the underlying logger associated to this client. It may optionally be set via (*Client).SetLogger.
//
// Logger may be called concurrently.
func (c *Client) Logger() *zap.Logger {
	c.logger.RLock()
	defer c.logger.RUnlock()

	return c.logger.Logger
}

// SetLogger updates the logger instance of this client.
//
// SetLogger may be called concurrently.
func (c *Client) SetLogger(logger *zap.Logger) {
	c.logger.Lock()
	defer c.logger.Unlock()

	c.logger.Logger = logger
}

// Close asynchronously kills the underlying connection and signals all goroutines to stop underlying this client.
//
// Close may be called concurrently.
func (c *Client) Close() {
	c.close()
}

// WaitUntilReady pauses the goroutine to which it was called within until/unless the client has successfully
// completed/failed the handshake protocol configured under the node instance to which this peer was derived from.
//
// It pauses the goroutine by reading from a channel that is closed when the client has successfully completed/failed
// the aforementioned handshake protocol.
//
// WaitUntilReady may be called concurrently.
func (c *Client) WaitUntilReady() {
	c.waitUntilReady()
}

// WaitUntilClosed pauses the goroutine to which it was called within until all goroutines associated to this client
// has been closed. The goroutines associated to this client would only close should:
//
// 1) handshaking failed/succeeded,
// 2) the connection was dropped, or
// 3) (*Client).Close was called.
//
// WaitUntilReady may be called concurrently.
func (c *Client) WaitUntilClosed() {
	c.waitUntilClosed()
}

// Error returns the very first error that has caused this clients connection to have dropped.
//
// Error may be called concurrently.
func (c *Client) Error() error {
	c.err.Lock()
	defer c.err.Unlock()

	return c.err.error
}

func (c *Client) reportError(err error) {
	c.err.Lock()
	defer c.err.Unlock()

	if c.err.error == nil {
		c.err.error = err
	}
}

func (c *Client) close() {
	c.closeOnce.Do(func() {
		c.writer.close()

		if c.conn != nil {
			c.conn.Close()
		}
	})
}

func (c *Client) waitUntilReady() {
	<-c.ready
}

func (c *Client) waitUntilClosed() {
	<-c.clientDone
}

func (c *Client) startTimeout(ctx context.Context) {
	c.timeout.timer = time.NewTimer(c.node.idleTimeout)
	c.timeout.reset = make(chan struct{}, 1)

	go func() {
		defer c.timeout.timer.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-c.clientDone:
				return
			case <-c.timeout.reset:
				if !c.timeout.timer.Stop() {
					<-c.timeout.timer.C
				}

				c.timeout.timer.Reset(c.node.idleTimeout)
			case <-c.timeout.timer.C:
				c.reportError(context.DeadlineExceeded)
				c.close()
				return
			}
		}
	}()
}

func (c *Client) resetTimeout() {
	select {
	case c.timeout.reset <- struct{}{}:
	default:
	}
}

func (c *Client) outbound(ctx context.Context, addr string) {
	c.addr = addr
	c.side = clientSideInbound

	defer func() {
		c.node.outbound.remove(addr)
		close(c.clientDone)
	}()

	var dialer net.Dialer

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		c.reportError(err)
		close(c.ready)
		close(c.writerDone)
		close(c.readerDone)
		close(c.handlerDone)
		return
	}

	conn.(*net.TCPConn).SetNoDelay(false)
	conn.(*net.TCPConn).SetWriteBuffer(10000)
	conn.(*net.TCPConn).SetReadBuffer(10000)

	c.conn = conn
	c.startTimeout(ctx)

	go c.readLoop(conn)
	go c.writeLoop(conn)

	c.handshake(ctx)

	c.handleLoop()

	for _, binder := range c.node.binders {
		binder.OnPeerLeave(c)
	}
}

func (c *Client) inbound(conn net.Conn, addr string) {
	c.addr = addr
	c.conn = conn
	c.side = clientSideOutbound

	defer func() {
		c.node.inbound.remove(addr)
		close(c.clientDone)
	}()

	ctx := context.Background()
	c.startTimeout(ctx)

	go c.readLoop(conn)
	go c.writeLoop(conn)

	c.handshake(ctx)

	if _, open := <-c.ready; !open && c.Error() != nil {
		close(c.handlerDone)
		c.close()
		return
	}

	c.handleLoop()

	for _, binder := range c.node.binders {
		binder.OnPeerLeave(c)
	}
}

func (c *Client) request(ctx context.Context, data []byte) (message, error) {
	// Figure out an available request nonce.

	ch, nonce, err := c.requests.nextNonce()
	if err != nil {
		return message{}, err
	}

	// Send request.

	if err := c.send(nonce, data); err != nil {
		c.requests.markRequestFailed(nonce)
		return message{}, err
	}

	c.resetTimeout()

	// Await response.

	var msg message

	select {
	case msg = <-ch:
	case <-ctx.Done():
		return message{}, ctx.Err()
	}

	c.resetTimeout()

	return msg, nil
}

func (c *Client) send(nonce uint64, data []byte) error {
	data = message{nonce: nonce, data: data}.marshal()

	if c.suite != nil {
		var err error

		if data, err = encryptAEAD(c.suite, data); err != nil {
			return err
		}
	}

	c.writer.write(data)

	c.resetTimeout()

	return nil
}

func (c *Client) recv(ctx context.Context) (message, error) {
	select {
	case data, open := <-c.reader.pending:
		if !open {
			return message{}, io.EOF
		}

		if c.suite != nil {
			var err error

			if data, err = decryptAEAD(c.suite, data); err != nil {
				return message{}, err
			}
		}

		msg, err := unmarshalMessage(data)
		if err != nil {
			return message{}, err
		}

		c.resetTimeout()

		return msg, nil
	case <-ctx.Done():
		return message{}, ctx.Err()
	}
}

func (c *Client) handshake(ctx context.Context) {
	defer close(c.ready)

	// Generate Ed25519 ephemeral keypair to perform a Diffie-Hellman handshake.

	pub, sec, err := GenerateKeys(nil)
	if err != nil {
		c.reportError(err)
		return
	}

	// Send our Ed25519 ephemeral public key and signature of the message '.__noise_handshake'.

	if err := c.send(0, append(pub[:], sec.Sign([]byte(".__noise_handshake"))...)); err != nil {
		c.reportError(fmt.Errorf("failed to send session handshake: %w", err))
		return
	}

	// Read from our peer their Ed25519 ephemeral public key and signature of the message '.__noise_handshake'.

	msg, err := c.recv(ctx)
	if err != nil {
		c.reportError(err)
		return
	}

	if msg.nonce != 0 {
		c.reportError(fmt.Errorf("got session handshake with nonce %d, but expected nonce to be 0", msg.nonce))
		return
	}

	if len(msg.data) != ed25519.PublicKeySize+ed25519.SignatureSize {
		c.reportError(fmt.Errorf("received invalid number of bytes opening a session: expected %d byte(s), but got %d byte(s)",
			ed25519.PublicKeySize+ed25519.SignatureSize,
			len(msg.data),
		))

		return
	}

	var peerPublicKey PublicKey
	copy(peerPublicKey[:], msg.data[:ed25519.PublicKeySize])

	// Verify ownership of our peers Ed25519 public key by verifying the signature they sent.

	if !peerPublicKey.Verify([]byte(".__noise_handshake"), msg.data[ed25519.PublicKeySize:]) {
		c.reportError(errors.New("could not verify session handshake"))
		return
	}

	// Transform all Ed25519 points to Curve25519 points and perform a Diffie-Hellman handshake
	// to derive a shared key.

	shared, err := ECDH(sec, peerPublicKey)

	// Use the derived shared key from Diffie-Hellman to encrypt/decrypt all future communications
	// with AES-256 Galois Counter Mode (GCM).

	core, err := aes.NewCipher(shared[:])
	if err != nil {
		c.reportError(fmt.Errorf("could not instantiate aes: %w", err))
		return
	}

	suite, err := cipher.NewGCM(core)
	if err != nil {
		c.reportError(fmt.Errorf("could not instantiate aes-gcm: %w", err))
		return
	}

	c.suite = suite

	// Send to our peer our overlay ID.

	buf := c.node.id.Marshal()
	buf = append(buf, c.node.Sign(append(buf, shared...))...)

	if err := c.send(0, buf); err != nil {
		c.reportError(fmt.Errorf("failed to send session handshake: %w", err))
		return
	}

	// Read and parse from our peer their overlay ID.

	msg, err = c.recv(ctx)
	if err != nil {
		c.reportError(fmt.Errorf("failed to read overlay handshake: %w", err))
		return
	}

	if msg.nonce != 0 {
		c.reportError(fmt.Errorf("got overlay handshake with nonce %d, but expected nonce to be 0", msg.nonce))
		return
	}

	id, err := UnmarshalID(msg.data)
	if err != nil {
		c.reportError(fmt.Errorf("failed to parse peer id while handling overlay handshake: %w", err))
		return
	}

	// Validate the peers ownership of the overlay ID.

	data := make([]byte, id.Size())
	copy(data, msg.data)

	if !id.ID.Verify(append(data, shared...), msg.data[len(data):]) {
		c.reportError(errors.New("overlay handshake signature is malformed"))
		return
	}

	c.id = id

	for _, binder := range c.node.binders {
		binder.OnPeerJoin(c)
	}
}

func (c *Client) handleLoop() {
	defer close(c.handlerDone)

	for {
		msg, err := c.recv(context.Background())
		if err != nil {
			c.logger.Warn("Got an error deserializing a message from a peer.", zap.Error(err))
			c.reportError(err)
			break
		}

		for _, binder := range c.node.binders {
			binder.OnMessageRecv(c)
		}

		if ch := c.requests.findRequest(msg.nonce); ch != nil {
			ch <- msg
			close(ch)
			continue
		}

		for _, handler := range c.node.handlers {
			if err = handler(HandlerContext{client: c, msg: msg}); err != nil {
				c.logger.Warn("Got an error executing a message handler.", zap.Error(err))
				c.reportError(err)
				break
			}
		}

		if err != nil {
			break
		}
	}

	c.close()
}

func (c *Client) writeLoop(conn net.Conn) {
	defer close(c.writerDone)

	if err := c.writer.loop(conn); err != nil {
		c.logger.Warn("Got an error while sending messages.", zap.Error(err))
		c.reportError(err)
		c.close()
	}
}

func (c *Client) readLoop(conn net.Conn) {
	defer close(c.readerDone)

	if err := c.reader.loop(conn); err != nil {
		c.logger.Warn("Got an error while reading incoming messages.", zap.Error(err))
		c.reportError(err)
		c.close()
	}
}
