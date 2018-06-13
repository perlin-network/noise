package network

import (
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/perlin-network/noise/actor"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"google.golang.org/grpc"
	"net"
	"strconv"
	"v/github.com/golang/protobuf@v1.1.0/ptypes"
)

type Network struct {
	Keys    *crypto.KeyPair
	Address string
	Port    int

	ID peer.ID

	Actors []actor.Actor

	listener net.Listener
	server   *Server
}

func CreateNetwork(keys *crypto.KeyPair, address string, port int, actors ...actor.Actor) *Network {
	id := peer.CreateID(address+":"+strconv.Itoa(port), keys.PublicKey)
	return &Network{Keys: keys, Address: address, Port: port, ID: id, Actors: actors}
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

	return err
}

type Sendable interface {
	Send(*protobuf.Message) error
}
