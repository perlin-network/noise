package network

import (
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)

type IncomingMessage struct {
	msg   proto.Message
	nonce uint64
}

// Represents a single incoming peer client.
type PeerClient struct {
	server *Server

	id     *peer.ID
	conn   *grpc.ClientConn
	client protobuf.Noise_StreamClient

	mailbox chan IncomingMessage
}

// Establishes an outgoing connection a given peer.
func (c *PeerClient) establishConnection() error {
	if c.id != nil && c.conn == nil && c.client == nil {
		conn, err := c.network().dial(c.id.Address)
		if err != nil {
			return err
		}

		client, err := protobuf.NewNoiseClient(conn).Stream(context.Background())

		if err != nil {
			return err
		}

		c.conn = conn
		c.client = client

		return nil
	}

	return status.Errorf(codes.Internal, "either client id is nil or client conn has alreayd been established")
}

func CreatePeerClient(server *Server) *PeerClient {
	return &PeerClient{
		server:  server,
		mailbox: make(chan IncomingMessage),
	}
}

// Refer to current network.
func (c *PeerClient) network() *Network {
	return c.server.network
}

// Event loop for processing through incoming request/responses.
func (c *PeerClient) process() {
	for item := range c.mailbox {
		switch msg := item.msg.(type) {
		case *protobuf.HandshakeRequest:
			// Update routing table w/ peer's ID.
			c.network().Routes.Update(*c.id)

			// Send handshake response to peer.
			err := c.network().Tell(c.client, &protobuf.HandshakeResponse{})

			if err != nil {
				continue
			}
		case *protobuf.HandshakeResponse:
			// Update routing table w/ peer's ID.
			c.network().Routes.Update(*c.id)

			addresses, publicKeys := bootstrapPeers(c.network(), *c.id, dht.BucketSize)

			// Update routing table w/ bootstrapped peers.
			for i := 0; i < len(addresses); i++ {
				c.network().Routes.Update(peer.CreateID(addresses[i], publicKeys[i]))
			}

			log.Info("[handshake] bootstrapped w/ peer(s): " + strings.Join(c.network().Routes.GetPeerAddresses(), ", ") + ".")
		case *protobuf.LookupNodeRequest:
			response := &protobuf.LookupNodeResponse{Peers: []*protobuf.ID{}}

			// Update routing table w/ peer's ID.
			c.network().Routes.Update(*c.id)

			// Respond back with closest peers to a provided target.
			for _, id := range c.network().Routes.FindClosestPeers(peer.ID(*msg.Target), dht.BucketSize) {
				id := protobuf.ID(id)
				response.Peers = append(response.Peers, &id)
			}

			err := c.network().Reply(c.client, item.nonce, response)
			if err != nil {
				continue
			}

			log.Info("[lookup] connected peers: " + strings.Join(c.network().Routes.GetPeerAddresses(), ", "))
		}
	}
}

// Clean up mailbox for peer client.
func (c *PeerClient) close() {
	if c.conn != nil {
		c.conn.Close()
	}

	close(c.mailbox)
}
