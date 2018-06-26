package network

import (
	"github.com/gogo/protobuf/proto"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
	"io"
)

// Represents a single incomingStream peer client.
type PeerClient struct {
	network *Network

	Id *peer.ID

	incomingSession *smux.Session
	incomingStream  *smux.Stream

	outgoingSession *smux.Session
	outgoingStream  *smux.Stream

	Mailbox chan proto.Message
	Outbox  chan proto.Message

	// To do with handling request/responses.
	requestNonce uint64
	// map[uint64]MessageChan
	requests *Uint64MessageChanSyncMap
}

func CreatePeerClient(network *Network, session *smux.Session) (*PeerClient, error) {
	client := &PeerClient{
		network:         network,
		incomingSession: session,
	}

	// Open a main incomingStream.
	stream, err := client.openIncomingStream()
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	client.incomingStream = stream

	return client, nil
}

func (c *PeerClient) openIncomingStream() (*smux.Stream, error) {
	stream, err := c.incomingSession.AcceptStream()
	if err != nil {
		return nil, err
	}

	return stream, err
}

func (c *PeerClient) openOutgoingStream() (*smux.Stream, error) {
	stream, err := c.outgoingSession.AcceptStream()
	if err != nil {
		return nil, err
	}

	return stream, err
}

func (c *PeerClient) processIncomingMessages() {
	buffer := make([]byte, 8192)
	for {
		n, err := c.incomingStream.Read(buffer)

		// Packet size overflows buffer. Continue.
		if n == 8192 {
			continue
		}

		if err == io.EOF || err != nil {
			// Disconnect the user.
			if c.Id != nil {
				if c.network.Routes.PeerExists(*c.Id) {
					c.network.Routes.RemovePeer(*c.Id)
					glog.Infof("Peer %s has disconnected.", c.Id.Address)
				}
			}
			break
		}

		// Deserialize message.
		raw := new(protobuf.Message)
		err = proto.Unmarshal(buffer[0:n], raw)

		// Check if any of the message headers are invalid or null.
		if raw.Message == nil || raw.Sender == nil || raw.Sender.PublicKey == nil || len(raw.Sender.Address) == 0 || raw.Signature == nil {
			glog.Info("Received an invalid message (either no message, no sender, or no signature) from a peer.")
			continue
		}

		// Verify signature of message.
		if !crypto.Verify(raw.Sender.PublicKey, raw.Message.Value, raw.Signature) {
			continue
		}

		// Derive, set the peer ID, connect to the peer, and additionally
		// store the peer.
		id := peer.ID(*raw.Sender)

		if c.Id == nil {
			dialer, err := kcp.DialWithOptions(c.Id.Address, nil, 10, 3)

			// Failed to connect. continue.
			if err != nil {
				glog.Warning(err)
				continue
			}

			session, err := smux.Client(dialer, nil)

			// Failed to open session. continue.
			if err != nil {
				glog.Error(err)
				continue
			}

			c.Id = &id
			c.outgoingSession = session

			c.outgoingStream, err = c.openOutgoingStream()

			// Failed to open a stream. continue.
			if err != nil {
				glog.Warning(err)
				continue
			}

			c.network.Peers.Store(id.Address, c)
		}

		// Update routing table w/ peer's ID.
		c.network.Routes.Update(id)

		// Unmarshal protobuf messages.
		var ptr ptypes.DynamicAny
		if err := ptypes.UnmarshalAny(raw.Message, &ptr); err != nil {
			continue
		}

		// Stream received messages to mailbox.
		c.Mailbox <- ptr.Message
	}
}

func (c *PeerClient) processOutgoingMessages() {

}
