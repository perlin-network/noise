package network

import (
	"net"
	"time"
	"net/url"
	"errors"

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

type MessageChannel chan proto.Message

// PeerClient represents a single incoming peers client.
type PeerClient struct {
	Network *Network

	ID *peer.ID

	session *smux.Session

	Requests     *Uint64MessageChannelSyncMap
	RequestNonce uint64
}

// createPeerClient creates a stub peer client.
func createPeerClient(network *Network) *PeerClient {
	return &PeerClient{Network: network, Requests: new(Uint64MessageChannelSyncMap), RequestNonce: 0}
}

// nextNonce gets the next most available request nonce. TODO: Have nonce recycled over time.
func (c *PeerClient) nextNonce() uint64 {
	return atomic.AddUint64(&c.RequestNonce, 1)
}

// establishConnection establishes a session by dialing a peers address. Errors if
// peer is not dial-able, or if the peer client already is connected.
func (c *PeerClient) establishConnection(address string) error {
	if c.session != nil {
		return nil
	}

	uInfo, err := url.Parse(address)
	if err != nil {
		return err
	}

	var conn net.Conn

	if uInfo.Scheme == "kcp" {
		conn, err = kcp.DialWithOptions(uInfo.Host, nil, 10, 3)
	} else if uInfo.Scheme == "tcp" {
		conn, err = net.Dial("tcp", uInfo.Host)
	} else {
		err = errors.New("Invalid scheme: " + uInfo.Scheme)
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

	return nil
}

// Close stops all sessions/streams and cleans up the nodes
// routing table. Errors if session fails to close.
func (c *PeerClient) Close() {
	// Disconnect the user.
	if c.ID != nil {
		if c.Network.Routes != nil && c.Network.Routes.PeerExists(*c.ID) {
			c.Network.Routes.RemovePeer(*c.ID)
			c.Network.Peers.Delete(c.ID.Address)

			glog.Infof("Peer %s has disconnected.", c.ID.Address)
		}
	}

	if c.session != nil && !c.session.IsClosed() {
		err := c.session.Close()
		if err != nil {
			glog.Error(err)
		}
	}
}

// prepareMessage marshals a message into a proto.Message and signs it with this
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
	if c.session == nil {
		return nil, errors.New("client session nil")
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
	if c.session == nil {
		return errors.New("client session nil")
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
