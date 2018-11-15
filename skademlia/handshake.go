package skademlia

import (
	"bytes"

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

func (p *HandshakeProcessor) ActivelyInitHandshake() ([]byte, interface{}, error) {
	return []byte("init"), &HandshakeState{passive: false}, nil
}

func (p *HandshakeProcessor) PassivelyInitHandshake() (interface{}, error) {
	return &HandshakeState{passive: true}, nil
}

func (p *HandshakeProcessor) ProcessHandshakeMessage(state interface{}, payload []byte) ([]byte, protocol.DoneAction, error) {
	if state.(*HandshakeState).passive {
		if bytes.Equal(payload, []byte("init")) {
			return []byte("ack"), protocol.DoneAction_SendMessage, nil
		} else {
			return nil, protocol.DoneAction_Invalid, errors.New("invalid handshake (passive)")
		}
	} else {
		if bytes.Equal(payload, []byte("ack")) {
			return nil, protocol.DoneAction_DoNothing, nil
		} else {
			return nil, protocol.DoneAction_Invalid, errors.New("invalid handshake (active)")
		}
	}
}
