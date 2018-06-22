package network

import (
	"context"
	"fmt"
	"reflect"

	"github.com/golang/protobuf/proto"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type IncomingMessage struct {
	Message proto.Message
	Nonce   uint64
}

// Represents a single incoming peer client.
type PeerClient struct {
	server *Server

	Id     *peer.ID
	Conn   *grpc.ClientConn
	Client protobuf.Noise_StreamClient

	mailbox chan IncomingMessage
}

// Establishes an outgoing connection a given peer.
func (c *PeerClient) establishConnection() error {
	if c.Id != nil && c.Conn == nil && c.Client == nil {
		conn, err := c.Network().dial(c.Id.Address)
		if err != nil {
			return err
		}

		client, err := protobuf.NewNoiseClient(conn).Stream(context.Background())

		if err != nil {
			return err
		}

		c.Conn = conn
		c.Client = client

		return nil
	}

	return status.Errorf(codes.Internal, "either Client Id is nil or Client Conn has alreayd been established")
}

func CreatePeerClient(server *Server) *PeerClient {
	return &PeerClient{
		server:  server,
		mailbox: make(chan IncomingMessage),
	}
}

// Refer to current network.
func (c *PeerClient) Network() *Network {
	return c.server.network
}

// Event loop for processing through incoming request/responses.
func (c *PeerClient) process() {
	for item := range c.mailbox {
		name := reflect.TypeOf(item.Message).String()
		processor, exists := c.Network().Processors.Load(name)

		if exists {
			processor := processor.(MessageProcessor)
			err := processor.Handle(c, &item)
			if err != nil {
				log.Debug(fmt.Sprintf("An error occurred handling %x: %x", name, err))
			}
		} else {
			log.Debug("Unknown message type received:", name)
		}
	}
}

// Clean up mailbox for peer client.
func (c *PeerClient) close() {
	if c.Conn != nil {
		c.Network().SocketPool.Delete(c.Id.Address)
		c.Conn.Close()
	}

	close(c.mailbox)
}
