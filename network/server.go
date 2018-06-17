package network

import (
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
	"io"
)

type Server struct {
	network *Network
}

type responseMessage struct {
	nonce uint64
	msg proto.Message
}

type commonMessage struct {
	id *peer.ID
	raw *protobuf.Message
	msg proto.Message
}

func createServer(network *Network) *Server {
	return &Server{
		network: network,
	}
}

func (s Server) handleResponse(messages chan responseMessage) {
	for item := range messages {
		s.network.HandleResponse(item.nonce, item.msg)
	}
}

func (s Server) handleCommon(messages chan commonMessage) {
	for item := range messages {
		id := item.id
		raw := item.raw
		msg := item.msg

		switch msg.(type) {
		case *protobuf.HandshakeRequest:
			// Update routing table w/ peer's ID.
			s.network.Routes.Update(*id)

			// Send handshake response to peer.
			client, err := s.network.Client(peer.ID(*raw.Sender))

			if err != nil {
				continue
			}
			err = s.network.Tell(client, &protobuf.HandshakeResponse{})
			if err != nil {
				continue
			}

			log.Info("Peer " + raw.Sender.Address + " has connected to you.")

			continue
		case *protobuf.HandshakeResponse:
			// Update routing table w/ peer's ID.
			s.network.Routes.Update(*id)

			BootstrapPeers(s.network, *id, dht.BucketSize)

			log.Info("Successfully bootstrapped w/ peer " + raw.Sender.Address + ".")

			continue
		case *protobuf.LookupNodeRequest:
			response := &protobuf.LookupNodeResponse{Peers: []*protobuf.ID{}}
			msg := msg.(*protobuf.LookupNodeRequest)

			// Update routing table w/ peer's ID.
			s.network.Routes.Update(*id)

			// Respond back with closest peers to a provided target.
			for _, id := range s.network.Routes.FindClosestPeers(peer.ID(*msg.Target), dht.BucketSize) {
				id := protobuf.ID(id)
				response.Peers = append(response.Peers, &id)
			}

			client, err := s.network.Client(peer.ID(*raw.Sender))
			if err != nil {
				continue
			}
			err = s.network.Reply(client, raw.Nonce, response)
			if err != nil {
				continue
			}
		}
	}
}

func (s Server) Stream(server protobuf.Noise_StreamServer) error {
	var id *peer.ID

	responseHandler := make(chan responseMessage)
	commonHandler := make(chan commonMessage)

	defer close(responseHandler)
	defer close(commonHandler)

	go s.handleResponse(responseHandler)
	go s.handleCommon(commonHandler)

	for {
		raw, err := server.Recv()

		if err == io.EOF || err != nil {
			if id != nil {
				s.network.Routes.RemovePeer(*id)
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

		// Handle request/response.
		if raw.IsResponse {
			select {
				case responseHandler <- responseMessage {
					nonce: raw.Nonce,
					msg: msg,
				}:
				default:
			}
		} else {
			select {
				case commonHandler <- commonMessage {
					id: id,
					raw: raw,
					msg: msg,
				}:
				default:
			}
		}
	}
	return nil
}
