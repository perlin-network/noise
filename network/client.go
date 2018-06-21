package network

import (
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"reflect"
)

type IncomingMessage struct {
	Message proto.Message
	Nonce   uint64
}

// Represents a single incoming peer Client.
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

// Refer to current Network.
func (c *PeerClient) Network() *Network {
	return c.server.network
}

// Event loop for processing through incoming request/responses.
func (c *PeerClient) process() {
	for item := range c.mailbox {
		processor, exists := c.Network().Processors.Load(reflect.TypeOf(item.Message).String())

		if exists {
			processor := processor.(MessageProcessor)
			processor.Handle(c, &item)
		} else {
			log.Debug("Unknown message type received:", reflect.TypeOf(item.Message).String())
		}
	}
}

// Clean up mailbox for peer Client.
func (c *PeerClient) close() {
	if c.Conn != nil {
		c.Conn.Close()
	}

	close(c.mailbox)
}
