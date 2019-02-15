package skademlia

import (
	"encoding/json"
	"github.com/perlin-network/noise"
	"github.com/pkg/errors"
)

type HandshakeProcessor struct {
	nodeID *IdentityManager
}

type HandshakeState struct {
	passive bool
}

type HandshakeMessage struct {
	Msg       string
	ID        []byte
	PublicKey []byte
	Nonce     []byte
	C1        int
	C2        int
}

// ActivelyInitHandshake sends the current node's public key, id, and nonce to be verified by a peer
func (p *HandshakeProcessor) ActivelyInitHandshake(node *noise.Node) ([]byte, interface{}, error) {
	id, ok := node.ID.(*IdentityManager)
	if !ok {
		return nil, nil, errors.New("skademlia: identity manager not compatible")
	}

	msg := &HandshakeMessage{
		Msg:       "init",
		PublicKey: id.PublicID(),
		ID:        id.nodeID,
		Nonce:     id.nonce,
		C1:        id.c1,
		C2:        id.c2,
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
func (p *HandshakeProcessor) ProcessHandshakeMessage(handshakeState *HandshakeState, payload []byte) ([]byte, error) {
	var msg HandshakeMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return nil, errors.New("skademlia: failed to unmarshal handshake payload")
	}

	if msg.C1 < p.nodeID.c1 || msg.C2 < p.nodeID.c2 {
		return nil, errors.Errorf("skademlia: S/Kademlia constants (%d, %d) for (c1, c2) do not satisfy local constants (%d, %d)",
			msg.C1, msg.C2, p.nodeID.c1, p.nodeID.c2)
	}

	// Verify that the remote peer ID is valid for the current node's c1 and c2 settings
	if ok := VerifyPuzzle(msg.PublicKey, msg.ID, msg.Nonce, p.nodeID.c1, p.nodeID.c2); !ok {
		return nil, errors.New("skademlia: keypair failed skademlia verification")
	}

	if handshakeState.passive {
		if msg.Msg == "init" {
			// If the handshake state is passive, construct a message to send the peer with this node's public key,
			// id and nonce for verification
			msg := &HandshakeMessage{
				Msg:       "ack",
				PublicKey: p.nodeID.PublicID(),
				ID:        p.nodeID.nodeID,
				Nonce:     p.nodeID.nonce,
				C1:        p.nodeID.c1,
				C2:        p.nodeID.c2,
			}
			b, err := json.Marshal(msg)
			if err != nil {
				return nil, errors.New("skademlia: failed to marshal handshake message")
			}
			return b, nil
		}
		return nil, errors.New("skademlia: invalid handshake (passive)")
	}

	if msg.Msg == "ack" {
		return nil, nil
	}
	return nil, errors.New("skademlia: invalid handshake (active)")
}
