package discovery

import (
	"context"
	"github.com/gogo/protobuf/proto"
	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/internal/protobuf"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
)

// Service is a service that handles periodic lookups of remote peers
type Service struct {
	DisablePing   bool
	DisablePong   bool
	DisableLookup bool

	Routes         *dht.RoutingTable
	requestAdapter RequestAdapter
}

// NewService creates a new instance of the Discovery Service
func NewService(requestAdapter RequestAdapter, selfID peer.ID) *Service {
	return &Service{
		Routes:         dht.CreateRoutingTable(selfID),
		requestAdapter: requestAdapter,
	}
}

// ReceiveHandler is the handler when a message is received
func (s *Service) ReceiveHandler(message *protocol.Message) {

	if message == nil || message.Body == nil || message.Body.Service != serviceID {
		// corrupt message so ignore
		return
	}
	if len(message.Body.Payload) == 0 {
		// corrupt payload so ignore
		return
	}

	sender, ok := s.Routes.LookupRemoteAddress(message.Sender)
	if !ok {
		// TODO: handle known peer
		return
	}
	target, ok := s.Routes.LookupRemoteAddress(message.Recipient)
	if !ok {
		// TODO: handle known peer
		return
	}

	var msg protobuf.Message
	if err := proto.Unmarshal(message.Body.Payload, &msg); err != nil {
		// unknown type so ignore
		return
	}

	if err := s.receive(*sender, *target, msg); err != nil {
		log.Warn().Err(err).Msg("")
	}
}

func (s *Service) receive(sender peer.ID, target peer.ID, msg protobuf.Message) error {
	// update the routes on every message
	s.Routes.Update(sender)

	switch msg.Opcode {
	case opCodePing:
		if s.DisablePing {
			break
		}
		// send the pong to the peer
		if err := s.reply(sender, opCodePong, &protobuf.Pong{}); err != nil {
			return err
		}
	case opCodePong:
		if s.DisablePong {
			break
		}
		peers := FindNode(s.Routes, s.requestAdapter, sender, dht.BucketSize, 8)

		// Update routing table w/ closest peers to self.
		for _, peerID := range peers {
			s.Routes.Update(peerID)
		}

		log.Info().
			Strs("peers", s.Routes.GetPeerAddresses()).
			Msg("Bootstrapped w/ peer(s).")
	case opCodeLookupRequest:
		if s.DisableLookup {
			break
		}

		// Prepare response
		response := &protobuf.LookupNodeResponse{}

		// Respond back with closest peers to a provided target.
		for _, peerID := range s.Routes.FindClosestPeers(target, dht.BucketSize) {
			id := protobuf.ID(peerID)
			response.Peers = append(response.Peers, &id)
		}

		if err := s.reply(sender, opCodeLookupResponse, response); err != nil {
			return err
		}
	default:
		return errors.Errorf("Unknown message opcode type: %d", msg.Opcode)
	}
	return nil
}

// PeerDisconnect handles updating the routing table on disconnect
func (s *Service) PeerDisconnect(target peer.ID) {
	// Delete peer if in routing table.
	if s.Routes.PeerExists(target) {
		s.Routes.RemovePeer(target)

		log.Debug().
			Str("address", s.Routes.Self().Address).
			Str("peer_address", target.Address).
			Msg("Peer has disconnected.")
	}
}

func (s *Service) reply(target peer.ID, opcode int, content proto.Message) error {
	msg, err := toProtobufMessage(opcode, content)
	if err != nil {
		return err
	}
	msg.ReplyFlag = true
	return s.requestAdapter.Reply(context.Background(), target, msg)
}
