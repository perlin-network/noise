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
)

// PeerClient represents a single incoming peers client.
type PeerClient struct {
	Network *Network

	Id *peer.ID

	Session *smux.Session

	Backoff *backoff.Backoff
}

// createPeerClient creates a stub peer client.
func createPeerClient(network *Network) *PeerClient {
	return &PeerClient{Network: network, Backoff: &backoff.Backoff{}}
}

// Dial attempts to establish or reestablish a session by dialing a peer's address.
// If peer is not dial-able, will attempt to retry using backoff until max attempts
// reached (which will then error).
func (c *PeerClient) Dial(address string) error {
	if c.Session != nil && !c.Session.IsClosed() {
		err := c.Session.Close()
		if err != nil {
			glog.Error(err)
			return err
		}
	}

	var dialer *kcp.UDPSession
	var err error
	c.Backoff.Reset()
	for !c.Backoff.TimeoutExceeded() {
		dialer, err = kcp.DialWithOptions(address, nil, 10, 3)
		// Failed to open session. Retry.
		if err != nil {
			glog.Error(err)
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

	c.Session, err = smux.Client(dialer, muxConfig())
	if err != nil {
		glog.Error(err)
		return err
	}

	// Cache the peer's client.
	c.Network.Peers.Store(address, c)

	return nil
}

func (c *PeerClient) Redial() error {
	return c.Dial(c.Id.Address)
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

// PrepareMessage marshals a message into a proto.Message and signs it with this
// nodes private key. Errors if the message is null.
func (c *PeerClient) PrepareMessage(message proto.Message) (*protobuf.Message, error) {
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
	if c.Session == nil {
		return errors.New("client session nil")
	}

	// Open a new stream.
	stream, err := c.Session.OpenStream()
	if err != nil {
		return err
	}
	defer stream.Close()

	// Send message bytes.
	err = c.sendMessage(stream, message)
	if err != nil {
		if err.Error() == "broken pipe" {
			c.Redial()
		}
		return err
	}

	return nil
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

	// Send request bytes.
	err = c.sendMessage(stream, req.Message)
	if err != nil {
		if err.Error() == "broken pipe" {
			c.Redial()
		}
		return nil, err
	}

	// Await for response message.
	res, err := c.receiveMessage(stream)
	if err != nil {
		if err.Error() == "broken pipe" {
			c.Redial()
		}
		return nil, err
	}

	// Unmarshal response protobuf.
	var ptr ptypes.DynamicAny
	if err := ptypes.UnmarshalAny(res.Message, &ptr); err != nil {
		return nil, err
	}

	return ptr.Message, nil
}
