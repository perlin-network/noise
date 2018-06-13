package network

import (
	"github.com/golang/protobuf/ptypes"
	"github.com/perlin-network/noise/actor"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"io"
)

type Server struct {
	network *Network
	actors  []actor.Actor
}

func createServer(network *Network, actors ...actor.Actor) *Server {
	return &Server{
		network: network,
		actors:  actors,
	}
}

func (s Server) Stream(server protobuf.Noise_StreamServer) error {
	var id *peer.ID
	var client protobuf.Noise_StreamClient

	for {
		raw, err := server.Recv()

		if err == io.EOF || err != nil {
			if id != nil {
				// TODO: Tell routing actor that the peer has been disconnected.
				//s.routes.RemovePeer(*id)
				log.Info("Peer " + id.Address + " has disconnected.")
			}
			return nil
		}

		if raw.Message == nil || raw.Sender == nil || raw.Sender.PublicKey == nil || len(raw.Sender.Address) == 0 || raw.Signature == nil {
			log.Debug("Received an invalid message (either no message, no sender, or no signature) from a peer.")
			continue
		}

		if !crypto.Verify(raw.Sender.PublicKey, raw.Message.Value, raw.Signature) {
			continue
		}

		val := peer.ID(*raw.Sender)
		id = &val

		var ptr ptypes.DynamicAny
		if err := ptypes.UnmarshalAny(raw.Message, &ptr); err != nil {
			continue
		}

		msg := ptr.Message

		if client != nil {
			for _, actor := range s.actors {
				actor.Receive(client, *id, msg)
			}
		}
	}
	return nil
}
