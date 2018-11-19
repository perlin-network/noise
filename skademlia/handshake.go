package skademlia

import (
	"encoding/json"

	"github.com/perlin-network/noise/protocol"

	"github.com/pkg/errors"
)

var _ protocol.HandshakeProcessor = (*HandshakeProcessor)(nil)

type HandshakeProcessor struct {
	nodeID *IdentityAdapter
}

type HandshakeState struct {
	passive bool
}

type HandshakeMessage struct {
	Msg       string
	ID        []byte
	PublicKey []byte
	Nonce     []byte
}

// NewHandshakeProcessor returns a new S/Kademlia handshake processor
func NewHandshakeProcessor(id *IdentityAdapter) *HandshakeProcessor {
	return &HandshakeProcessor{id}
}

// ActivelyInitHandshake sends the current node's public key, id, and nonce to be verified by a peer
func (p *HandshakeProcessor) ActivelyInitHandshake() ([]byte, interface{}, error) {
	msg := &HandshakeMessage{
		Msg:       "init",
		PublicKey: p.nodeID.MyIdentity(),
		ID:        p.nodeID.id(),
		Nonce:     p.nodeID.Nonce,
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return nil, nil, errors.New("skademlia: failed to marshal handshake message")
	}

	return b, &HandshakeState{passive: false}, nil
}

// PassivelyInitHandshake initiates a passive handshake to a peer
func (p *HandshakeProcessor) PassivelyInitHandshake() (interface{}, error) {
	return &HandshakeState{passive: true}, nil
}

// ProcessHandshakeMessage takes a handshake state and payload and sends a handshake message to a peer
func (p *HandshakeProcessor) ProcessHandshakeMessage(state interface{}, payload []byte) ([]byte, protocol.DoneAction, error) {
	handshakeState := state.(*HandshakeState)
	var msg HandshakeMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return nil, protocol.DoneAction_Invalid, errors.New("skademlia: failed to unmarshal handshake payload")
	}
	// Verify that the remote peer ID is valid for the current node's c1 and c2 settings
	if ok := VerifyPuzzle(msg.PublicKey, msg.ID, msg.Nonce, p.nodeID.c1, p.nodeID.c2); !ok {
		return nil, protocol.DoneAction_Invalid, errors.New("skademlia: keypair failed skademlia verification")
	}
	if handshakeState.passive {
		if msg.Msg == "init" {
			// If the handshake state is passive, construct a message to send the peer with this node's public key,
			// id and nonce for verification
			msg := &HandshakeMessage{
				Msg:       "ack",
				PublicKey: p.nodeID.MyIdentity(),
				ID:        p.nodeID.id(),
				Nonce:     p.nodeID.Nonce}
			b, err := json.Marshal(msg)
			if err != nil {
				return nil, protocol.DoneAction_Invalid, errors.New("skademlia: failed to marshal handshake message")
			}
			return b, protocol.DoneAction_SendMessage, nil
		}
		return nil, protocol.DoneAction_Invalid, errors.New("skademlia: invalid handshake (passive)")
	}
	if msg.Msg == "ack" {
		return nil, protocol.DoneAction_DoNothing, nil
	}
	return nil, protocol.DoneAction_Invalid, errors.New("skademlia: invalid handshake (active)")
}
