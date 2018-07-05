package network

import (
	"errors"
	"net"
	"time"

	"sync"
	"sync/atomic"

	"github.com/golang/protobuf/proto"
	"github.com/perlin-network/noise/network/rpc"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
)

// PeerClient represents a single incoming peers client.
type PeerClient struct {
	Network *Network

	ID      *peer.ID
	Address string

	Requests     *sync.Map
	RequestNonce uint64

	stream StreamState
}

type StreamState struct {
	sync.Mutex
	buffer        []byte
	dataAvailable chan struct{}
	closed        bool
}

// createPeerClient creates a stub peer client.
func createPeerClient(network *Network, address string) *PeerClient {
	client := &PeerClient{
		Network:      network,
		Address:      address,
		Requests:     new(sync.Map),
		RequestNonce: 0,
		stream: StreamState{
			buffer:        make([]byte, 0),
			dataAvailable: make(chan struct{}),
		},
	}

	client.Network.Plugins.Each(func(plugin PluginInterface) {
		plugin.PeerConnect(client)
	})

	return client
}

// nextNonce gets the next most available request nonce. TODO: Have nonce recycled over time.
func (c *PeerClient) nextNonce() uint64 {
	return atomic.AddUint64(&c.RequestNonce, 1)
}

// Close stops all sessions/streams and cleans up the nodes
// routing table. Errors if session fails to close.
func (c *PeerClient) Close() error {
	c.stream.Lock()
	c.stream.closed = true
	c.stream.Unlock()

	// Handle 'on peer disconnect' callback for plugins.
	c.Network.Plugins.Each(func(plugin PluginInterface) {
		plugin.PeerDisconnect(c)
	})

	if c.ID != nil {
		c.Network.Peers.Delete(c.ID.Address)
		c.Network.Connections.Delete(c.ID.Address)
	}

	return nil
}

// Write asynchronously emit a message to a given peer.
func (c *PeerClient) Tell(message proto.Message) error {
	signed, err := c.Network.PrepareMessage(message)
	if err != nil {
		return err
	}

	err = c.Network.Write(c.Address, signed)
	if err != nil {
		return err
	}

	return nil
}

// Request requests for a response for a request sent to a given peer.
func (c *PeerClient) Request(req *rpc.Request) (proto.Message, error) {
	signed, err := c.Network.PrepareMessage(req.Message)
	if err != nil {
		return nil, err
	}

	signed.Nonce = c.nextNonce()

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
		return nil, errors.New("request timed out")
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

func (c *PeerClient) handleStreamPacket(pkt []byte) {
	c.stream.Lock()
	wasEmpty := len(c.stream.buffer) == 0
	c.stream.buffer = append(c.stream.buffer, pkt...)
	c.stream.Unlock()

	if wasEmpty {
		select {
		case c.stream.dataAvailable <- struct{}{}:
		default:
		}
	}
}

// Implement net.Conn.
func (c *PeerClient) Read(out []byte) (int, error) {
	for {
		c.stream.Lock()
		closed := c.stream.closed
		n := copy(out, c.stream.buffer)
		c.stream.buffer = c.stream.buffer[n:]
		c.stream.Unlock()

		if closed {
			return n, errors.New("closed")
		}

		if n == 0 {
			select {
			case <-c.stream.dataAvailable:
			case <-time.After(1 * time.Second):
			}
		} else {
			return n, nil
		}
	}
}

func (c *PeerClient) Write(data []byte) (int, error) {
	err := c.Tell(&protobuf.StreamPacket{Data: data})
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

type NoiseAddr struct {
	Address string
}

func (a *NoiseAddr) Network() string {
	return "noise"
}

func (a *NoiseAddr) String() string {
	return a.Address
}

func (c *PeerClient) LocalAddr() net.Addr {
	return &NoiseAddr{Address: "[local]"}
}

func (c *PeerClient) RemoteAddr() net.Addr {
	return &NoiseAddr{Address: c.Address}
}

func (c *PeerClient) SetDeadline(t time.Time) error {
	// TODO
	return nil
}

func (c *PeerClient) SetReadDeadline(t time.Time) error {
	// TODO
	return nil
}

func (c *PeerClient) SetWriteDeadline(t time.Time) error {
	// TODO
	return nil
}
