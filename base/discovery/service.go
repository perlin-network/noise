package discovery

import (
	"github.com/gogo/protobuf/proto"
	"github.com/perlin-network/noise/base/discovery/messages"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"math/rand"
	"time"
)

const (
	serviceID            = 5
	pollFrequencyInSec   = 20
	samplerSeed          = 123
	opCodeLookupRequest  = 1
	opCodeLookupResponse = 2
)

// SendHandler is a callback when this service needs to send a message to a peer
type SendHandler func(peerID []byte, body *protocol.MessageBody) error

// Service is a service that handles periodic lookups of remote peers
type Service struct {
	active        bool
	DisableLookup bool
	sendHandler   SendHandler
	connAdapter   protocol.ConnectionAdapter
	sampler       *rand.Rand
}

// NewService creates a new instance of the Discovery Service
func NewService(sendHandler SendHandler, connAdapter protocol.ConnectionAdapter) *Service {
	return &Service{
		active:        false,
		DisableLookup: false,
		sendHandler:   sendHandler,
		connAdapter:   connAdapter,
		sampler:       rand.New(rand.NewSource(samplerSeed)),
	}
}

// StartLookups starts the goroutine to periodically sample peers for their connection info
func (s *Service) StartLookups() {
	s.active = true
	ticker := time.NewTicker(time.Second * pollFrequencyInSec)

	go func() {
		for range ticker.C {
			if !s.active {
				break
			}
			if err := s.SampledLookup(); err != nil {
				log.Warn().Err(err).Msg("Error looking up peers")
			}
		}
	}()
}

// StopLookups stops sampling the peers connection info
func (s *Service) StopLookups() {
	s.active = false
}

// SampledLookup gets a random peer from the connection adapter and makes a lookup request
func (s *Service) SampledLookup() error {
	peerIDs := s.connAdapter.GetConnectionIDs()
	randPeer := peerIDs[s.sampler.Intn(len(peerIDs))]

	body, err := makeRequestMessageBody(randPeer)
	if err != nil {
		return err
	}

	if err := s.sendHandler(randPeer, body); err != nil {
		return err
	}
	return nil
}

// Receive is the handler when a message is received
func (s *Service) Receive(message *protocol.Message) {
	if message == nil || message.Body == nil || message.Body.Service != serviceID {
		// corrupt message so ignore
		return
	}
	if len(message.Body.Payload) == 0 {
		// corrupt payload so ignore
		return
	}

	if s.DisableLookup {
		// disabled so ignore
		return
	}

	var msg messages.Message
	if err := proto.Unmarshal(message.Body.Payload, &msg); err != nil {
		// not a lookup request so ignore
		return
	}

	switch msg.Opcode {
	case opCodeLookupRequest:
		// Prepare response
		peerIDs := s.connAdapter.GetConnectionIDs()
		var IDs []*messages.ID
		for _, peer := range peerIDs {
			addr, err := s.connAdapter.GetAddressByID(peer)
			if err != nil {
				continue
			}
			id := &messages.ID{
				Id:      peer,
				Address: addr,
			}
			IDs = append(IDs, id)
		}

		body, err := makeResponseMessageBody(IDs)
		if err != nil {
			log.Warn().Err(err).Msg("Unable to marshal response")
			break
		}

		// reply
		if err := s.sendHandler(message.Sender, body); err != nil {
			log.Warn().Err(err).Msg("Unable to send response")
		}
	case opCodeLookupResponse:
		m := messages.LookupNodeResponse{}
		if err := proto.Unmarshal(msg.Message, &m); err != nil {
			log.Warn().Err(err).Msg("Unable to read response")
			break
		}
		// load up all the new connections into the connection adapter
		for _, peer := range m.Peers {
			s.connAdapter.AddConnection(peer.Id, peer.Address)
		}
	default:
		log.Warn().Msgf("Unknown message opcode type: %d", msg.Opcode)
	}
}

func makeResponseMessageBody(ids []*messages.ID) (*protocol.MessageBody, error) {
	content := &messages.LookupNodeResponse{
		Peers: ids,
	}
	contentBytes, err := proto.Marshal(content)
	if err != nil {
		return nil, err
	}
	msg := &messages.Message{
		Opcode:  opCodeLookupResponse,
		Message: contentBytes,
	}
	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	body := &protocol.MessageBody{
		Service: serviceID,
		Payload: msgBytes,
	}
	return body, nil
}

func makeRequestMessageBody(id []byte) (*protocol.MessageBody, error) {
	content := &messages.LookupNodeRequest{
		Target: &messages.ID{
			Id: id,
		},
	}
	contentBytes, err := proto.Marshal(content)
	if err != nil {
		return nil, err
	}
	msg := &messages.Message{
		Opcode:  opCodeLookupRequest,
		Message: contentBytes,
	}
	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	body := &protocol.MessageBody{
		Service: serviceID,
		Payload: msgBytes,
	}
	return body, nil
}
