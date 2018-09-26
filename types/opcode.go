package types

import (
	"sync"

	"github.com/perlin-network/noise/internal/protobuf"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
)

type Opcode uint16

const (
	PingCode               Opcode = 0
	PongCode               Opcode = 1
	LookupNodeRequestCode  Opcode = 2
	LookupNodeResponseCode Opcode = 3
)

var (
	opcodeTable = map[Opcode]proto.Message{
		PingCode:               &protobuf.Ping{},
		PongCode:               &protobuf.Pong{},
		LookupNodeRequestCode:  &protobuf.LookupNodeRequest{},
		LookupNodeResponseCode: &protobuf.LookupNodeResponse{},
	}

	mu = sync.RWMutex{}
)

// RegisterMessageType adds a new proto message with a corresponding opcode
func RegisterMessageType(opcode Opcode, msg proto.Message) error {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := opcodeTable[opcode]; ok {
		return errors.New("types: opcode already exists, choose a different opcode")
	} else {
		opcodeTable[opcode] = msg
	}
	return nil
}

// GetMessageType returns the corresponding proto message type given an opcode
func GetMessageType(code Opcode) proto.Message {
	if i, ok := opcodeTable[code]; ok {
		return i
	}
	return nil
}
