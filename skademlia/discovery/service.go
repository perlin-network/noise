package discovery

import (
	"context"
	"time"

	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/skademlia/dht"
	"github.com/perlin-network/noise/skademlia/peer"
	"github.com/perlin-network/noise/skademlia/protobuf"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
)

const (
	pingTimeout = 4 * time.Second

	// prefix needs to differ by a certain number of bytes before adding to table to prevent
	// attacks which attempt to flood the routing table
	prefixDiffLength = 128
	prefixDiffMin    = 32
)

var (
	errRemovePeerFailed = errors.New("skademlia: failed to remove last seen peer")
)

// Service is a service that handles periodic lookups of remote peers
type Service struct {
	protocol.Service

	DisablePing   bool
	DisablePong   bool
	DisableLookup bool

	Routes      *dht.RoutingTable
	sendAdapter protocol.SendAdapter
}

// NewService creates a new instance of the Discovery Service
func NewService(sendAdapter protocol.SendAdapter, selfID peer.ID) *Service {
	return &Service{
		Routes:      dht.NewRoutingTable(selfID),
		sendAdapter: sendAdapter,
	}
}

// Receive is the handler when a message is received
func (s *Service) Receive(ctx context.Context, message *protocol.Message) (*protocol.MessageBody, error) {
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
	if peer.PrefixDiff(sender.Id, target.Id, prefixDiffLength) > prefixDiffMin {
		err := s.Routes.Update(sender)
		if err == dht.ErrBucketFull {
			// TODO: don't block the code path in every call
			if ok, _ := s.EvictLastSeenPeer(sender.Id); ok {
				s.Routes.Update(sender)
			}
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
		peers := FindNode(s.Routes, s.sendAdapter, sender, s.Routes.Opts().BucketSize, 8)

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
		for _, peerID := range s.Routes.FindClosestPeers(reqTargetID, s.Routes.Opts().BucketSize) {
			id := protobuf.ID(peerID)
			response.Peers = append(response.Peers, &id)
		}

		log.Info().
			Str("self", s.Routes.Self().Address).
			Strs("peers", s.Routes.GetPeerAddresses()).
			Msg("Connected to peer(s).")

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
	if other, ok := s.Routes.GetPeer(t.Id); ok {
		s.Routes.RemovePeer(t.Id)

		log.Debug().
			Str("peer", other.Address).
			Str("self", s.Routes.Self().Address).
			Msg("Peer has disconnected.")
	}
}

func (s *Service) EvictLastSeenPeer(id []byte) (bool, error) {
	// bucket is full, ping the least-seen node
	bucketID := s.Routes.GetBucketID(id)
	bucket := s.Routes.Bucket(bucketID)
	element := bucket.Back()
	lastSeen := element.Value.(peer.ID)
	body, err := ToMessageBody(ServiceID, OpCodePing, &protobuf.Ping{})
	if err != nil {
		return false, errRemovePeerFailed
	}
	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	reply, err := s.sendAdapter.Request(ctx, lastSeen.PublicKey, body)
	if err != nil || reply == nil {
		bucket.Remove(element)
		return true, nil
	}
	var respMsg protobuf.Pong
	opCode, err := ParseMessageBody(reply, &respMsg)
	if opCode != OpCodePong || err != nil {
		bucket.Remove(element)
		return true, nil
	}
	// last-seen has replied, move to the front
	bucket.MoveToFront(element)
	return false, errRemovePeerFailed
}
