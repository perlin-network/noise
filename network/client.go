package network

import (
	"errors"
	"net"
	"net/url"
	"time"

	"sync"
	"sync/atomic"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/perlin-network/noise/network/rpc"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
)

// PeerClient represents a single incoming peers client.
type PeerClient struct {
	Network *Network

	ID      *peer.ID
	Address string

	session          *smux.Session
	sessionInitMutex sync.Mutex

	Requests     *Uint64MessageChannelSyncMap
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
func createPeerClient(network *Network) *PeerClient {
	return &PeerClient{
		Network:      network,
		Requests:     new(Uint64MessageChannelSyncMap),
		RequestNonce: 0,
		stream: StreamState{
			buffer:        make([]byte, 0),
			dataAvailable: make(chan struct{}),
		},
	}
}

// nextNonce gets the next most available request nonce. TODO: Have nonce recycled over time.
func (c *PeerClient) nextNonce() uint64 {
	return atomic.AddUint64(&c.RequestNonce, 1)
}

// establishConnection establishes a session by dialing a peers address. Errors if
// peer is not dial-able, or if the peer client already is connected.
func (c *PeerClient) establishConnection(address string) error {
	c.sessionInitMutex.Lock()
	defer c.sessionInitMutex.Unlock()

	if c.IsConnected() {
		return nil
	}

	urlInfo, err := url.Parse(address)
	if err != nil {
		return err
	}

	var conn net.Conn

	if urlInfo.Scheme == "kcp" {
		conn, err = kcp.DialWithOptions(urlInfo.Host, nil, 10, 3)
	} else if urlInfo.Scheme == "tcp" {
		conn, err = net.Dial("tcp", urlInfo.Host)
	} else {
		err = errors.New("Invalid scheme: " + urlInfo.Scheme)
	}

	// Failed to connect.
	if err != nil {
		glog.Error(err)
		return err
	}

	c.session, err = smux.Client(conn, muxConfig())

	// Failed to open session.
	if err != nil {
		glog.Error(err)
		return err
	}

	// Cache the peer's client.
	c.Network.Peers.Store(address, c)
	c.Address = address

	c.Network.Plugins.Each(func(plugin PluginInterface) {
		plugin.PeerConnect(c)
	})

	return nil
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

	// Disconnect the user.
	if c.ID != nil {
		// Delete peer from network.
		c.Network.Peers.Delete(c.ID.Address)
	}

	if c.session != nil && !c.session.IsClosed() {
		err := c.session.Close()
		if err != nil {
			glog.Error(err)
		}
	}

	return nil
}

// prepareMessage marshals a message into a proto.Tell and signs it with this
// nodes private key. Errors if the message is null.
func (c *PeerClient) prepareMessage(message proto.Message) (*protobuf.Message, error) {
	if message == nil {
		return nil, errors.New("message is null")
	}

	raw, err := ptypes.MarshalAny(message)
	if err != nil {
		return nil, err
	}

	id := protobuf.ID(c.Network.ID)

	signature, err := c.Network.Keys.Sign(raw.Value)
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

// Tell asynchronously emit a message to a given peer.
func (c *PeerClient) Tell(message proto.Message) error {
	// A nonce of 0 indicates a message that is not a request/response.
	return c.Reply(0, message)
}

// Request requests for a response for a request sent to a given peer.
func (c *PeerClient) Request(req *rpc.Request) (proto.Message, error) {
	if !c.IsConnected() {
		return nil, errors.New("client is not connected")
	}

	// Open a new stream.
	stream, err := c.OpenStream()
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	stream.SetDeadline(time.Now().Add(req.Timeout))

	// Prepare message.
	msg, err := c.prepareMessage(req.Message)
	if err != nil {
		return nil, err
	}

	msg.Nonce = c.nextNonce()

	// Send request bytes.
	err = c.Network.sendMessage(stream, msg)
	if err != nil {
		return nil, err
	}

	// Start tracking the request.
	channel := make(MessageChannel, 1)
	c.Requests.Store(msg.Nonce, channel)

	// Stop tracking the request.
	defer close(channel)
	defer c.Requests.Delete(msg.Nonce)

	select {
	case res := <-channel:
		return res, nil
	case <-time.After(req.Timeout):
		return nil, errors.New("request timed out")
	}

	return nil, errors.New("request timed out")
}

// Reply is equivalent to Tell() with an appended nonce to signal a reply.
func (c *PeerClient) Reply(nonce uint64, message proto.Message) error {
	if !c.IsConnected() {
		return errors.New("client is not connected")
	}

	// Open a new stream.
	stream, err := c.OpenStream()
	if err != nil {
		return err
	}
	defer stream.Close()

	// Prepare message.
	msg, err := c.prepareMessage(message)
	if err != nil {
		return err
	}

	msg.Nonce = nonce

	// Send message bytes.
	err = c.Network.sendMessage(stream, msg)
	if err != nil {
		return err
	}

	return nil
}

func (c *PeerClient) IsConnected() bool {
	return c.session != nil
}

// Opens a new stream with preconfigured settings through the clients
// assigned session.
func (c *PeerClient) OpenStream() (*smux.Stream, error) {
	// Open new stream.
	stream, err := c.session.OpenStream()
	if err != nil {
		return nil, err
	}

	// Configure deadlines. TODO: Make configurable.
	err = stream.SetReadDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		return nil, err
	}
	err = stream.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		return nil, err
	}

	return stream, nil
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
	err := c.Tell(&protobuf.StreamPacket{
		Data: data,
	})
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
