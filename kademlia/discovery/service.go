package discovery

import (
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
	protocol.Service

	DisablePing   bool
	DisablePong   bool
	DisableLookup bool

	Routes      *dht.RoutingTable
	sendHandler SendHandler
}

// NewService creates a new instance of the Discovery Service
func NewService(sendHandler SendHandler, selfID peer.ID) *Service {
	return &Service{
		Routes:      dht.CreateRoutingTable(selfID),
		sendHandler: sendHandler,
	}
}

// Receive is the handler when a message is received
func (s *Service) Receive(message *protocol.Message) (*protocol.MessageBody, error) {
	if message.Body.Service != ServiceID {
		return nil, nil
	}

	if message == nil || message.Body == nil || len(message.Body.Payload) == 0 {
		// corrupt payload so ignore
		return nil, errors.New("Message body is corrupt")
	}

	sender := peer.CreateID(message.Metadata["remoteAddr"], message.Sender)
	target := s.Routes.Self()

	var msg protobuf.Message
	if err := proto.Unmarshal(message.Body.Payload, &msg); err != nil {
		// unknown type so ignore
		return nil, errors.Wrap(err, "Unable to parse message")
	}

	reply, err := s.processMsg(sender, target, msg)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (s *Service) processMsg(sender peer.ID, target peer.ID, msg protobuf.Message) (*protocol.MessageBody, error) {
	s.Routes.Update(sender)

	switch msg.Opcode {
	case OpCodePing:
		if s.DisablePing {
			break
		}
		// send the pong to the peer
		return ToMessageBody(ServiceID, OpCodePong, &protobuf.Pong{})
	case OpCodePong:
		if s.DisablePong {
			break
		}
		peers := FindNode(s.Routes, s.sendHandler, sender, dht.BucketSize, 8)

		var debugPeers []string

		// Update routing table w/ closest peers to self.
		for _, peerID := range peers {
			s.Routes.Update(peerID)
			debugPeers = append(debugPeers, peerID.Address)
		}

		log.Info().
			Str("self", s.Routes.Self().Address).
			Strs("peers", s.Routes.GetPeerAddresses()).
			Strs("debugPeers", debugPeers).
			Msg("Bootstrapped w/ peer(s).")
	case OpCodeLookupRequest:
		if s.DisableLookup {
			break
		}

		var reqMsg protobuf.LookupNodeRequest
		if err := proto.Unmarshal(msg.Message, &reqMsg); err != nil {
			return nil, errors.Wrap(err, "Unable to marse lookup request")
		}
		reqTargetID := peer.ID(*reqMsg.Target)

		// Prepare response
		response := &protobuf.LookupNodeResponse{}
		var respAddr []string

		// Respond back with closest peers to a provided target.
		for _, peerID := range s.Routes.FindClosestPeers(reqTargetID, dht.BucketSize) {
			id := protobuf.ID(peerID)
			response.Peers = append(response.Peers, &id)
			respAddr = append(respAddr, peerID.Address)
		}

		log.Debug().
			Str("self", s.Routes.Self().Address).
			Strs("peers", respAddr).
			Msg("Replying LookupNodeResponse")

		return ToMessageBody(ServiceID, OpCodeLookupResponse, response)
	default:
		// ignore
	}
	return nil, nil
}

// PeerDisconnect handles updating the routing table on disconnect
func (s *Service) PeerDisconnect(target []byte) {
	t := peer.CreateID("", target)
	// Delete peer if in routing table.
	if s.Routes.PeerExists(t) {
		s.Routes.RemovePeer(t)

		log.Debug().
			Str("address", s.Routes.Self().Address).
			Str("peer_pub_key", t.PublicKeyHex()).
			Msg("Peer has disconnected.")
	}
}

func (s *Service) Bootstrap() error {
	body, err := ToMessageBody(ServiceID, OpCodePing, &protobuf.Ping{})
	if err != nil {
		return err
	}
	return s.sendHandler.Broadcast(body)
}
