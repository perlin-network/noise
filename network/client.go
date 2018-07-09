package network

import (
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/perlin-network/noise/network/rpc"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"github.com/pkg/errors"
)

// PeerClient represents a single incoming peers client.
type PeerClient struct {
	Network *Network

	ID      *peer.ID
	Address string

	Requests     *sync.Map
	RequestNonce uint64

	stream StreamState

	outgoingReady chan struct{}
	incomingReady chan struct{}

	jobQueue chan func()
	jobExecutorInit sync.Once
}

type StreamState struct {
	sync.Mutex
	buffer        []byte
	buffered      chan struct{}
	closed        bool
	readDeadline  time.Time
	writeDeadline time.Time
}

// createPeerClient creates a stub peer client.
func createPeerClient(network *Network, address string) (*PeerClient, error) {
	// Ensure the address is valid.
	if _, err := ParseAddress(address); err != nil {
		return nil, err
	}

	client := &PeerClient{
		Network:      network,
		Address:      address,
		Requests:     new(sync.Map),
		RequestNonce: 0,

		incomingReady: make(chan struct{}),
		outgoingReady: make(chan struct{}),

		stream: StreamState{
			buffer:   make([]byte, 0),
			buffered: make(chan struct{}),
		},
	}

	return client, nil
}

func (c *PeerClient) Submit(job func()) {
	c.jobExecutorInit.Do(func() {
		c.jobQueue = make(chan func(), 128)
		go c.executeJobs()
	})
	c.jobQueue <- job
}

func (c *PeerClient) executeJobs() {
	for job := range c.jobQueue {
		job()
	}
}

// Close stops all sessions/streams and cleans up the nodes
// routing table. Errors if session fails to close.
func (c *PeerClient) Close() error {
	c.stream.Lock()
	c.stream.closed = true
	c.stream.Unlock()

	if c.jobQueue != nil {
		close(c.jobQueue)
	}

	// Handle 'on peer disconnect' callback for plugins.
	c.Network.Plugins.Each(func(plugin PluginInterface) {
		plugin.PeerDisconnect(c)
	})

	// Remove entries from node's network.
	if c.ID != nil {
		// close out connections
		if conn, ok := c.Network.Connections.Load(c.ID.Address); ok {
			if state, ok := conn.(*ConnState); ok && state != nil {
				state.session.Close()
			}
		}

		c.Network.Peers.Delete(c.ID.Address)
		c.Network.Connections.Delete(c.ID.Address)
	}

	return nil
}

// Write asynchronously emit a message to a given peer.
func (c *PeerClient) Tell(message proto.Message) error {
	signed, err := c.Network.PrepareMessage(message)
	if err != nil {
		return errors.Wrap(err, "failed to sign message")
	}

	err = c.Network.Write(c.Address, signed)
	if err != nil {
		return errors.Wrapf(err, "failed to send message to %s", c.Address)
	}

	return nil
}

// Request requests for a response for a request sent to a given peer.
func (c *PeerClient) Request(req *rpc.Request) (proto.Message, error) {
	signed, err := c.Network.PrepareMessage(req.Message)
	if err != nil {
		return nil, err
	}

	signed.Nonce = atomic.AddUint64(&c.RequestNonce, 1)

	err = c.Network.Write(c.Address, signed)
	if err != nil {
		return nil, err
	}

	// Start tracking the request.
	channel := make(chan proto.Message, 1)
	c.Requests.Store(signed.Nonce, channel)

	// Stop tracking the request.
	defer close(channel)
	defer c.Requests.Delete(signed.Nonce)

	select {
	case res := <-channel:
		return res, nil
	case <-time.After(req.Timeout):
	}

	return nil, errors.New("request timed out")
}

// Reply is equivalent to Write() with an appended nonce to signal a reply.
func (c *PeerClient) Reply(nonce uint64, message proto.Message) error {
	signed, err := c.Network.PrepareMessage(message)
	if err != nil {
		return err
	}

	// Set the nonce.
	signed.Nonce = nonce

	err = c.Network.Write(c.Address, signed)
	if err != nil {
		return err
	}

	return nil
}

func (c *PeerClient) handleBytes(pkt []byte) {
	c.stream.Lock()
	empty := len(c.stream.buffer) == 0
	c.stream.buffer = append(c.stream.buffer, pkt...)
	c.stream.Unlock()

	if empty {
		select {
		case c.stream.buffered <- struct{}{}:
		default:
		}
	}
}

// Read implement net.Conn by reading packets of bytes over a stream.
func (c *PeerClient) Read(out []byte) (int, error) {
	for {
		c.stream.Lock()
		closed := c.stream.closed
		n := copy(out, c.stream.buffer)
		c.stream.buffer = c.stream.buffer[n:]
		readDeadline := c.stream.readDeadline
		c.stream.Unlock()

		if closed {
			return n, errors.New("closed")
		}

		if !readDeadline.IsZero() && time.Now().After(readDeadline) {
			return n, errors.New("read deadline exceeded")
		}

		if n == 0 {
			select {
			case <-c.stream.buffered:
			case <-time.After(1 * time.Second):
			}
		} else {
			return n, nil
		}
	}
}

// Write implements net.Conn and sends packets of bytes over a stream.
func (c *PeerClient) Write(data []byte) (int, error) {
	c.stream.Lock()
	writeDeadline := c.stream.writeDeadline
	c.stream.Unlock()

	if !writeDeadline.IsZero() && time.Now().After(writeDeadline) {
		return 0, errors.New("write deadline exceeded")
	}

	err := c.Tell(&protobuf.Bytes{Data: data})
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

// LocalAddr implements net.Conn.
func (c *PeerClient) LocalAddr() net.Addr {
	addr, err := ParseAddress(c.Network.Address)
	if err != nil {
		panic(err) // should never happen
	}
	return addr
}

// RemoteAddr implements net.Conn.
func (c *PeerClient) RemoteAddr() net.Addr {
	addr, err := ParseAddress(c.Address)
	if err != nil {
		panic(err) // should never happen
	}
	return addr
}

// SetDeadline implements net.Conn.
func (c *PeerClient) SetDeadline(t time.Time) error {
	c.stream.Lock()
	c.stream.readDeadline = t
	c.stream.writeDeadline = t
	c.stream.Unlock()
	return nil
}

// SetReadDeadline implements net.Conn.
func (c *PeerClient) SetReadDeadline(t time.Time) error {
	c.stream.Lock()
	c.stream.readDeadline = t
	c.stream.Unlock()
	return nil
}

// SetWriteDeadline implements net.Conn.
func (c *PeerClient) SetWriteDeadline(t time.Time) error {
	c.stream.Lock()
	c.stream.writeDeadline = t
	c.stream.Unlock()
	return nil
}

// IncomingReady returns true should the client have both incoming and outgoing sockets established.
func (c *PeerClient) IncomingReady() bool {
	select {
	case <-c.incomingReady:
	case <-time.After(1 * time.Second):
		return false
	}

	return true
}

// OutgoingReady returns true should the client have an outgoing socket established..
func (c *PeerClient) OutgoingReady() bool {
	select {
	case <-c.outgoingReady:
	case <-time.After(1 * time.Second):
		return false
	}

	return true
}
