package network

import (
	"context"
	"errors"
	"github.com/golang/protobuf/proto"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"google.golang.org/grpc"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"v/github.com/golang/protobuf@v1.1.0/ptypes"
)

type Network struct {
	Routes  *dht.RoutingTable
	Keys    *crypto.KeyPair
	Address string
	Port    int

	RequestNonce uint64
	Requests     map[uint64]chan proto.Message
	RequestMutex *sync.RWMutex

	ID peer.ID

	listener net.Listener
	server   *Server
}

func CreateNetwork(keys *crypto.KeyPair, address string, port int) *Network {
	id := peer.CreateID(address+":"+strconv.Itoa(port), keys.PublicKey)
	return &Network{
		Keys:    keys,
		Address: address,
		Port:    port,
		ID:      id,

		// Set initial nonce to 1 as 0 is null value.
		RequestNonce: 1,
		Requests:     make(map[uint64]chan proto.Message),
		RequestMutex: &sync.RWMutex{},

		Routes: dht.CreateRoutingTable(peer.CreateID(id.Address, keys.PublicKey)),
	}
}

func (n *Network) Host() string {
	return n.Address + ":" + strconv.Itoa(n.Port)
}

func (n *Network) Listen() {
	go n.listen()
}

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
		client, err := n.dial(address)
		if err != nil {
			continue
		}

		err = n.Tell(client, &protobuf.HandshakeRequest{})
		if err != nil {
			continue
		}
	}
}

// Dial a peer w/o a handshake request.
func (n *Network) Dial(address string) (protobuf.Noise_StreamClient, error) {
	return n.dial(address)
}

func (n *Network) dial(address string) (protobuf.Noise_StreamClient, error) {
	conn, err := grpc.Dial(address, grpc.WithInsecure())

	if err != nil {
		return nil, err
	}

	client, err := protobuf.NewNoiseClient(conn).Stream(context.Background())
	if err != nil {
		return nil, err
	}

	return client, nil
}

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

// Provide a response to a request.
func (n *Network) HandleRequest(nonce uint64, response proto.Message) {
	n.RequestMutex.RLock()
	if channel, exists := n.Requests[nonce]; exists {
		channel <- response
	}
	n.RequestMutex.RUnlock()
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
	n.RequestMutex.Lock()
	n.Requests[msg.Nonce] = make(chan proto.Message, 1)
	n.RequestMutex.Unlock()

	select {
	case response := <-n.Requests[msg.Nonce]:
		// Stop tracking the request.
		n.RequestMutex.Lock()
		delete(n.Requests, msg.Nonce)
		n.RequestMutex.Unlock()

		return response, nil
	case <-time.After(1 * time.Second): // TODO: Make delay customizable.
	}

	// Stop tracking the request.
	n.RequestMutex.Lock()
	delete(n.Requests, msg.Nonce)
	n.RequestMutex.Unlock()

	return nil, errors.New("request timed out")
}

type Sendable interface {
	Send(*protobuf.Message) error
}
