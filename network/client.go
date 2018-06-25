package network

import (
	"context"
	"reflect"

	"errors"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/perlin-network/noise/network/rpc"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"google.golang.org/grpc"
	"sync"
	"sync/atomic"
	"time"
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
	Stream protobuf.Noise_StreamClient

	// To do with handling request/responses.
	requestNonce uint64
	// map[uint64]*proto.Message
	requests *sync.Map

	mailbox chan IncomingMessage

	refCount uint64
	refMutex *sync.Mutex
}

// Establishes an outgoing connection a given peer should one not exist already.
func (c *PeerClient) establishConnection(address string) error {
	if c.Conn == nil && c.Stream == nil {
		// Block in case the server on the other side isn't erady to respond.
		ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
		defer cancel()

		opts := []grpc.DialOption{
			grpc.WithInsecure(),
			grpc.WithBlock(),
		}
		conn, err := grpc.DialContext(ctx, address, opts...)

		// If the connection failed...
		if err != nil {
			return err
		}

		// Setup a RPC client and initialize an one-way stream to the client.
		client, err := protobuf.NewNoiseClient(conn).Stream(context.Background())

		if err != nil {
			return err
		}

		// Keep reference of both gRPC connection and stream.
		c.Conn = conn
		c.Stream = client

		return nil
	}

	return nil
}

// Creates a client representing a single peer, and has peers start
// processing for incoming messages through channels.
func createPeerClient(server *Server) *PeerClient {
	client := &PeerClient{
		server:  server,
		mailbox: make(chan IncomingMessage),

		requestNonce: 0,
		requests:     &sync.Map{},

		refCount: 1,
		refMutex: &sync.Mutex{},
	}

	// Have peers start processing for incoming messages.
	go client.processIncomingMessages()

	return client
}

// Refer to current network.
func (c *PeerClient) Network() *Network {
	return c.server.network
}

// Event loop for processing through incoming request/responses.
func (c *PeerClient) processIncomingMessages() {
	for item := range c.mailbox {
		name := reflect.TypeOf(item.Message).String()
		processor, exists := c.Network().Processors.Load(name)

		if exists {
			processor := processor.(MessageProcessor)
			err := processor.Handle(c, &item)
			if err != nil {
				glog.Infof("An error occurred handling %x: %x", name, err)
			}
		} else {
			glog.Info("Unknown message type received:", name)
		}
	}
}

// Marshals message into proto.Message and signs it with this node's private key.
// Errors if the message is null.
func (c *PeerClient) prepareMessage(message proto.Message) (*protobuf.Message, error) {
	if message == nil {
		return nil, errors.New("message is null")
	}

	raw, err := ptypes.MarshalAny(message)
	if err != nil {
		return nil, err
	}

	id := protobuf.ID(c.Network().ID)

	signature, err := c.Network().Keys.Sign(raw.Value)
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

// Asynchronously emit a message to a given peer.
func (c *PeerClient) Tell(message proto.Message) error {
	msg, err := c.prepareMessage(message)
	if err != nil {
		return err
	}
	err = c.Stream.Send(msg)
	if err != nil {
		return err
	}

	return nil
}

// Used within message processors to reply to a given request message.
func (c *PeerClient) Reply(nonce uint64, message proto.Message) error {
	msg, err := c.prepareMessage(message)

	msg.Nonce = nonce
	msg.IsResponse = true

	if err != nil {
		return err
	}

	err = c.Stream.Send(msg)
	if err != nil {
		return err
	}

	return nil
}

// Provide a response to a request. Internal use only.
func (c *PeerClient) handleResponse(nonce uint64, response proto.Message) {
	// Check if the request is currently looking to be received.
	if channel, exists := c.requests.Load(nonce); exists {
		channel.(chan proto.Message) <- response
	}
}

// Initiate a request/response-style RPC call to the given peer.
func (c *PeerClient) Request(request *rpc.Request) (proto.Message, error) {
	msg, err := c.prepareMessage(request.Message)
	if err != nil {
		return nil, err
	}

	// Set the request nonce.
	msg.Nonce = atomic.AddUint64(&c.requestNonce, 1)

	// Send the client the request.
	err = c.Stream.Send(msg)
	if err != nil {
		return nil, err
	}

	// Start tracking the request.
	channel := make(chan proto.Message, 1)
	c.requests.Store(msg.Nonce, channel)

	// Stop tracking the request.
	defer close(channel)
	defer c.requests.Delete(msg.Nonce)

	select {
	case response := <-channel:
		return response, nil
	case <-time.After(request.Timeout): // TODO: Make delay customizable.
		return nil, errors.New("request timed out")
	}
}

// Fails if c.refCount == 0; otherwise increases c.refCount by one
func (c *PeerClient) open() error {
	c.refMutex.Lock()
	defer c.refMutex.Unlock()

	if c.refCount == 0 {
		return errors.New("attempting to open a closed PeerClient")
	}

	c.refCount++
	return nil
}

// Clean up mailbox for peer client.
func (c *PeerClient) close() {
	c.refMutex.Lock()

	old := c.refCount
	if old > 0 {
		c.refCount--
	}

	c.refMutex.Unlock()

	if old == 1 {
		if c.Conn != nil {
			c.Network().Peers.Delete(c.Id.Address)
			c.Conn.Close()
		}

		close(c.mailbox)
	} else if old <= 0 {
		glog.Fatal("BUG: old <= 0")
	}
}
