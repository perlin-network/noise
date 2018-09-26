package types

import (
	"reflect"
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

	msgTable = map[reflect.Type]Opcode{
		reflect.TypeOf(protobuf.Ping{}):               PingCode,
		reflect.TypeOf(protobuf.Pong{}):               PongCode,
		reflect.TypeOf(protobuf.LookupNodeRequest{}):  LookupNodeRequestCode,
		reflect.TypeOf(protobuf.LookupNodeResponse{}): LookupNodeResponseCode,
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
		msgTable[reflect.TypeOf(msg)] = opcode
	}
	return nil
}

// GetMessageType returns the corresponding proto message type given an opcode
func GetMessageType(code Opcode) (proto.Message, error) {
	if i, ok := opcodeTable[code]; ok {
		return i, nil
	}
	return nil, errors.New("opcode not found, did you register it?")
}

// GetMessageType returns the corresponding proto message type given an opcode
func GetOpcode(t reflect.Type) (*Opcode, error) {
	if i, ok := msgTable[t]; ok {
		return &i, nil
	}
	return nil, errors.New("message type not found, did you register it?")
}
