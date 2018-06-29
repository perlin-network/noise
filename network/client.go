package network

import (
	"time"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/perlin-network/noise/network/backoff"
	"github.com/perlin-network/noise/network/rpc"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"github.com/pkg/errors"
	"github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
	"sync/atomic"
)

type MessageChannel chan proto.Message

// PeerClient represents a single incoming peers client.
type PeerClient struct {
	Network *Network

	Id *peer.ID

	Session *smux.Session

	Backoff *backoff.Backoff
	redialLock uint32

	Requests     *Uint64MessageChannelSyncMap
	RequestNonce uint64
}

// createPeerClient creates a stub peer client.
func createPeerClient(network *Network) *PeerClient {
	return &PeerClient{Network: network, Backoff: backoff.DefaultBackoff(), Requests: new(Uint64MessageChannelSyncMap), RequestNonce: 0}
}

// nextNonce gets the next most available request nonce. TODO: Have nonce recycled over time.
func (c *PeerClient) nextNonce() uint64 {
	return atomic.AddUint64(&c.RequestNonce, 1)
}

// establishConnection establishes a session by dialing a peers address. Errors if
// peer is not dial-able, or if the peer client already is connected.
func (c *PeerClient) establishConnection(address string) error {
	if c.Session != nil {
		return nil
	}

	dialer, err := kcp.DialWithOptions(address, nil, 10, 3)

	// Failed to connect.
	if err != nil {
		glog.Error(err)
		return err
	}

	c.Session, err = smux.Client(dialer, muxConfig())

	// Failed to open session.
	if err != nil {
		glog.Error(err)
		return err
	}

	// Cache the peer's client.
	c.Network.Peers.Store(address, c)

	return nil
}

// Attempts to re-establish a connect with a client that was
// already previously connected to.
func (c *PeerClient) reestablishConnection() error {
	// Don't attempt to redial if already in progress.
	if !atomic.CompareAndSwapUint32(&c.redialLock, 0, 1) {
		return nil
	}
	defer atomic.StoreUint32(&c.redialLock, 0)

	glog.Infof("Lost connection with %s, attempting to reestablish.", c.Id.Address)

	if c.Id == nil {
		glog.Infof("asdsadfdfds")
		return errors.New("client id is null")
	}

	// Attempt to reconnect to the original client.
	c.Backoff.Reset()
	for !c.Backoff.TimeoutExceeded() {
		// Close any previous session.
		if c.Session != nil && !c.Session.IsClosed() {
			err := c.Session.Close()
			if err != nil {
				glog.Error(err)
				return err
			}
		}
		c.Session = nil

		err := c.establishConnection(c.Id.Address)
		if err != nil {
			glog.Error(err)
			return err
		}

		// Attempt to handshake to verify connection.
		request := new(rpc.Request)
		request.SetMessage(&protobuf.HandshakeRequest{})
		request.SetTimeout(3 * time.Second)
		_, err = c.Request(request)

		// Handshake failed. Will retry until limit.
		if err != nil {
			d := c.Backoff.NextDuration()
			time.Sleep(d)
			continue
		}

		break
	}
	if c.Backoff.TimeoutExceeded() {
		c.Close()
		return errors.New("max connection attempts exceeded")
	}

	return nil
}

// Close stops all sessions/streams and cleans up the nodes
// routing table. Errors if session fails to close.
func (c *PeerClient) Close() {
	// Disconnect the user.
	if c.Id != nil {
		if c.Network.Routes != nil && c.Network.Routes.PeerExists(*c.Id) {
			c.Network.Routes.RemovePeer(*c.Id)
			c.Network.Peers.Delete(c.Id.Address)

			glog.Infof("Peer %s has disconnected.", c.Id.Address)
		}
	}

	if c.Session != nil && !c.Session.IsClosed() {
		err := c.Session.Close()
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
	if c.Session == nil {
		return nil, errors.New("client session nil")
	}

	// Open a new stream.
	stream, err := c.Session.OpenStream()
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
	if c.Session == nil {
		return errors.New("client session nil")
	}

	// Open a new stream.
	stream, err := c.Session.OpenStream()
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
