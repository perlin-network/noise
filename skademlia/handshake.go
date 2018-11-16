package skademlia

import (
	"bytes"
	"fmt"

	"github.com/perlin-network/noise/protocol"

	"github.com/pkg/errors"
)

var _ protocol.HandshakeProcessor = (*HandshakeProcessor)(nil)

type HandshakeProcessor struct {
	nodeID *IdentityAdapter
}

type HandshakeMessage struct {
	passive bool
	nodeID  []byte
	nonce   []byte
}

// NewHandshakeProcessor returns a new S/Kademlia handshake processor
func NewHandshakeProcessor(id *IdentityAdapter) *HandshakeProcessor {
	return &HandshakeProcessor{id}
}

func (p *HandshakeProcessor) ActivelyInitHandshake() ([]byte, interface{}, error) {
	fmt.Printf("active sending\n")
	return []byte("init"), &HandshakeMessage{passive: false}, nil
}

func (p *HandshakeProcessor) PassivelyInitHandshake() (interface{}, error) {
	fmt.Printf("passive sending\n")
	return &HandshakeMessage{passive: true}, nil
}

func (p *HandshakeProcessor) ProcessHandshakeMessage(state interface{}, payload []byte) ([]byte, protocol.DoneAction, error) {
	fmt.Printf("processing handshake %+v %+v\n", state, payload)
	if state.(*HandshakeMessage).passive {
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
