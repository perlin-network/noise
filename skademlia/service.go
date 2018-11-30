package skademlia

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/perlin-network/noise/internal/protobuf"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protocol"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
)

const (
	pingTimeout = 4 * time.Second
)

var (
	ErrRemovePeerFailed = errors.New("skademlia: failed to remove last seen peer")
)

// Service is a service that handles periodic lookups of remote peers
type Service struct {
	protocol.Service

	DisablePing   bool
	DisablePong   bool
	DisableLookup bool

	Routes      *RoutingTable
	sendAdapter protocol.SendAdapter
}

// NewService creates a new instance of the Discovery Service
func NewService(sendAdapter protocol.SendAdapter, selfID peer.ID) *Service {
	return &Service{
		Routes:      CreateRoutingTable(selfID),
		sendAdapter: sendAdapter,
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
	err := s.Routes.Update(sender)
	if err == ErrBucketFull {
		// TODO: don't block the code path in every call
		if ok, _ := s.EvictLastSeenPeer(sender.Id); ok {
			s.Routes.Update(sender)
		}
	}

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
		peers := FindNode(s.Routes, s.sendAdapter, sender, BucketSize, 8)

		// Update routing table w/ closest peers to self.
		for _, peerID := range peers {
			s.Routes.Update(peerID)
		}

		log.Info().
			Str("self", s.Routes.Self().Address).
			Strs("peers", s.Routes.GetPeerAddresses()).
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

		// Respond back with closest peers to a provided target.
		for _, peerID := range s.Routes.FindClosestPeers(reqTargetID, BucketSize) {
			id := protobuf.ID(peerID)
			response.Peers = append(response.Peers, &id)
		}

		return ToMessageBody(ServiceID, OpCodeLookupResponse, response)
	default:
		// ignore
	}
	return nil, nil
}

func (s *Service) PeerConnect(id []byte) {
	log.Debug().
		Str("peer", hex.EncodeToString(id)).
		Str("self", s.Routes.Self().Address).
		Msg("Peer has connected.")
}

// PeerDisconnect handles updating the routing table on disconnect
func (s *Service) PeerDisconnect(target []byte) {
	t := peer.CreateID("", target)
	// Delete peer if in routing table.
	if other, ok := s.Routes.GetPeer(t.Id); ok {
		s.Routes.RemovePeer(t.Id)

		log.Debug().
			Str("peer", other.Address).
			Str("self", s.Routes.Self().Address).
			Msg("Peer has disconnected.")
	}
}

func (s *Service) Bootstrap() error {
	if s.sendAdapter == nil {
		return errors.New("SendAdapter not set")
	}
	body, err := ToMessageBody(ServiceID, OpCodePing, &protobuf.Ping{})
	if err != nil {
		return err
	}
	return s.sendAdapter.Broadcast(body)
}

func (s *Service) EvictLastSeenPeer(id []byte) (bool, error) {
	// bucket is full, ping the least-seen node
	bucketID := s.Routes.GetBucketID(id)
	bucket := s.Routes.Bucket(bucketID)
	element := bucket.Back()
	lastSeen := element.Value.(peer.ID)
	body, err := ToMessageBody(ServiceID, OpCodePing, &protobuf.Ping{})
	if err != nil {
		return false, ErrRemovePeerFailed
	}
	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	reply, err := s.sendAdapter.Request(ctx, lastSeen.Id, body)
	if err != nil || reply == nil {
		bucket.Remove(element)
		return true, nil
	}
	// last-seen has replied, move to the front
	bucket.MoveToFront(element)
	return false, ErrRemovePeerFailed
}
