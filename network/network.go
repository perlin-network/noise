package network

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"google.golang.org/grpc"
)

type Network struct {
	// Routing table.
	Routes *dht.RoutingTable

	// Node's keypair.
	Keys *crypto.KeyPair

	// Node's Network information.
	Address string
	Port    int

	// To do with handling request/responses.
	RequestNonce uint64
	// map[uint64]*proto.Message
	Requests *sync.Map

	// Map of incoming message processors for the Network.
	// map[string]MessageProcessor
	Processors *sync.Map

	// Node's cryptographic ID.
	ID peer.ID

	listener net.Listener
	server   *Server

	// Map of connection addresses (string) <-> *grpc.Conn
	// so that the network doesn't dial multiple times to the same ip
	SocketPool *sync.Map
}

var (
	dialTimeout = 3 * time.Second
)

func (n *Network) Host() string {
	return n.Address + ":" + strconv.Itoa(n.Port)
}

func (n *Network) Listen() {
	go n.listen()
}

// Listen for peers on a port specified on instantation of Network{}.
func (n *Network) listen() {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(n.Port))
	if err != nil {
		log.Debug(err)
		return
	}

	client := grpc.NewServer()
	server := createServer(n)

	protobuf.RegisterNoiseServer(client, server)

	n.listener = listener
	n.server = server

	log.Debug("Listening for peers on port " + strconv.Itoa(n.Port) + ".")

	err = client.Serve(listener)
	if err != nil {
		log.Debug(err)
		return
	}
}

// Bootstrap with a number of peers and send a handshake to them.
func (n *Network) Bootstrap(addresses ...string) {
	for _, address := range addresses {
		conn, err := n.dial(address)
		if err != nil {
			continue
		}

		// Create a temporary client for now and send a handshake request.
		client, err := protobuf.NewNoiseClient(conn).Stream(context.Background())
		if err != nil {
			continue
		}

		err = n.Tell(client, &protobuf.HandshakeRequest{})
		if err != nil {
			continue
		}
	}
}

// Dial a peer.
func (n *Network) Dial(address string) (*grpc.ClientConn, error) {
	return n.dial(address)
}

// Dials a peer via. gRPC.
func (n *Network) dial(address string) (*grpc.ClientConn, error) {
	if len(strings.Trim(address, " ")) == 0 {
		return nil, fmt.Errorf("Cannot dial, address was empty")
	}

	// load a cached connection
	if conn, ok := n.SocketPool.Load(address); ok && conn != nil {
		return conn.(*grpc.ClientConn), nil
	}

	// block in case the server on the other side isn't ready right away
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()

	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBlock(),
	}
	conn, err := grpc.DialContext(ctx, address, opts...)

	if err != nil {
		return nil, err
	}

	// cache the connection
	n.SocketPool.Store(address, conn)

	return conn, nil
}

// Marshals message into proto.Message and signs it with this node's private key.
func (n *Network) prepareMessage(message proto.Message) (*protobuf.Message, error) {
	raw, err := ptypes.MarshalAny(message)
	if err != nil {
		return nil, err
	}

	id := protobuf.ID(n.ID)

	signature, err := n.Keys.Sign(raw.Value)
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

func (n *Network) Tell(client Sendable, message proto.Message) error {
	msg, err := n.prepareMessage(message)
	if err != nil {
		return err
	}
	err = client.Send(msg)
	if err != nil {
		return err
	}

	return nil
}

func (n *Network) Reply(client Sendable, nonce uint64, message proto.Message) error {
	msg, err := n.prepareMessage(message)

	msg.Nonce = nonce
	msg.IsResponse = true

	if err != nil {
		return err
	}

	err = client.Send(msg)
	if err != nil {
		return err
	}

	return nil
}

// Provide a response to a request.
func (n *Network) HandleResponse(nonce uint64, response proto.Message) {
	// Check if the request is currently looking to be received.
	if channel, exists := n.Requests.Load(nonce); exists {
		channel.(chan proto.Message) <- response
	}
}

func (n *Network) Request(client Sendable, message proto.Message) (proto.Message, error) {
	msg, err := n.prepareMessage(message)
	if err != nil {
		return nil, err
	}

	// Set the request nonce.
	msg.Nonce = atomic.AddUint64(&n.RequestNonce, 1)

	// Send the client the request.
	err = client.Send(msg)
	if err != nil {
		return nil, err
	}

	// Start tracking the request.
	channel := make(chan proto.Message, 1)
	n.Requests.Store(msg.Nonce, channel)

	// Stop tracking the request.
	defer close(channel)
	defer n.Requests.Delete(msg.Nonce)

	select {
	case response := <-channel:
		return response, nil
	case <-time.After(10 * time.Second): // TODO: Make delay customizable.
		return nil, errors.New("request timed out")
	}
}

type Sendable interface {
	Send(*protobuf.Message) error
}
