package network

import (
	"github.com/golang/protobuf/proto"
	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"strings"
)

type IncomingMessage struct {
	msg   proto.Message
	nonce uint64
}

// Represents a single incoming peer client.
type PeerClient struct {
	server *Server

	id   *peer.ID
	conn protobuf.Noise_StreamClient

	mailbox chan IncomingMessage
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
			err := c.network().Tell(c.conn, &protobuf.HandshakeResponse{})

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

			log.Info("[handshake] bootstrapped w/ peer(s): " + strings.Join(addresses, ", ") + ".")
		case *protobuf.LookupNodeRequest:
			response := &protobuf.LookupNodeResponse{Peers: []*protobuf.ID{}}

			// Update routing table w/ peer's ID.
			c.network().Routes.Update(*c.id)

			// Respond back with closest peers to a provided target.
			for _, id := range c.network().Routes.FindClosestPeers(peer.ID(*msg.Target), dht.BucketSize) {
				id := protobuf.ID(id)
				response.Peers = append(response.Peers, &id)
			}

			err := c.network().Reply(c.conn, item.nonce, response)
			if err != nil {
				continue
			}

			log.Info("[lookup] connected peers: " + strings.Join(c.network().Routes.GetPeerAddresses(), ", "))
		}
	}
}

// Clean up mailbox for peer client.
func (c *PeerClient) close() {
	close(c.mailbox)
}
