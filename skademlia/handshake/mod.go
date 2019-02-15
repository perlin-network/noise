package handshake

import (
	"encoding/json"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/skademlia"
	"github.com/pkg/errors"
)

var (
	_ protocol.Block = (*block)(nil)
)

type block struct {
	opcodeHandshake noise.Opcode
	nodeID          *skademlia.IdentityManager
}

type HandshakeState struct {
	passive bool
}

func New() *block {
	return &block{}
}

func (b *block) OnRegister(p *protocol.Protocol, node *noise.Node) {
	b.opcodeHandshake = noise.RegisterMessage(noise.NextAvailableOpcode(), (*Handshake)(nil))
	b.nodeID = node.ID.(*skademlia.IdentityManager)
}

func (b *block) OnBegin(p *protocol.Protocol, peer *noise.Peer) error {
	if b.nodeID == nil {
		return errors.New("node not setup with skademlia properly")
	}

	// TODO: need to fill this in

	return nil
}

func (b *block) OnEnd(p *protocol.Protocol, peer *noise.Peer) error {
	return nil
}

// ActivelyInitHandshake sends the current node's public key, id, and nonce to be verified by a peer
func (b *block) ActivelyInitHandshake() ([]byte, *HandshakeState, error) {
	msg := &Handshake{
		Msg:       "init",
		PublicKey: b.nodeID.PublicID(),
		ID:        b.nodeID.NodeID,
		Nonce:     b.nodeID.Nonce,
		C1:        b.nodeID.C1,
		C2:        b.nodeID.C2,
	}

	return msg.Write(), &HandshakeState{passive: false}, nil
}

// PassivelyInitHandshake initiates a passive handshake to a peer
func (b *block) PassivelyInitHandshake() (*HandshakeState, error) {
	return &HandshakeState{passive: true}, nil
}

// ProcessHandshakeMessage takes a handshake state and payload and sends a handshake message to a peer
func (b *block) ProcessHandshakeMessage(handshakeState *HandshakeState, payload []byte) ([]byte, error) {
	var msg Handshake
	if err := json.Unmarshal(payload, &msg); err != nil {
		return nil, errors.New("skademlia: failed to unmarshal handshake payload")
	}

	if msg.C1 < b.nodeID.C1 || msg.C2 < b.nodeID.C2 {
		return nil, errors.Errorf("skademlia: S/Kademlia constants (%d, %d) for (c1, c2) do not satisfy local constants (%d, %d)",
			msg.C1, msg.C2, b.nodeID.C1, b.nodeID.C2)
	}

	// Verify that the remote peer ID is valid for the current node's c1 and c2 settings
	if ok := skademlia.VerifyPuzzle(msg.PublicKey, msg.ID, msg.Nonce, b.nodeID.C1, b.nodeID.C2); !ok {
		return nil, errors.New("skademlia: keypair failed skademlia verification")
	}

	if handshakeState.passive {
		if msg.Msg == "init" {
			// If the handshake state is passive, construct a message to send the peer with this node's public key,
			// id and nonce for verification
			msg := &Handshake{
				Msg:       "ack",
				PublicKey: b.nodeID.PublicID(),
				ID:        b.nodeID.NodeID,
				Nonce:     b.nodeID.Nonce,
				C1:        b.nodeID.C1,
				C2:        b.nodeID.C2,
			}
			return msg.Write(), nil
		}
		return nil, errors.New("skademlia: invalid handshake (passive)")
	}

	if msg.Msg == "ack" {
		return nil, nil
	}
	return nil, errors.New("skademlia: invalid handshake (active)")
}
