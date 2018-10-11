package types

import (
	"reflect"
	"sync"

	"github.com/perlin-network/noise/internal/protobuf"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
)

func init() {
	msgOpcodePairs := []struct {
		msg    proto.Message
		opcode Opcode
	}{
		{&protobuf.Ping{}, PingCode},
		{&protobuf.Pong{}, PongCode},
		{&protobuf.LookupNodeRequest{}, LookupNodeRequestCode},
		{&protobuf.LookupNodeResponse{}, LookupNodeResponseCode},
	}

	mu.Lock()
	defer mu.Unlock()
	for _, pair := range msgOpcodePairs {
		opcodeTable[pair.opcode] = pair.msg
		t := reflect.TypeOf(pair.msg)
		msgTable[t] = pair.opcode
	}
}

type Opcode uint16

const (
	PingCode               Opcode = 0x0000 // 0
	PongCode               Opcode = 0x0001 // 1
	LookupNodeRequestCode  Opcode = 0x0002 // 2
	LookupNodeResponseCode Opcode = 0x0003 // 3
)

var (
	opcodeTable = make(map[Opcode]proto.Message, 0)
	msgTable    = make(map[reflect.Type]Opcode, 0)

	mu = sync.RWMutex{}
)

// RegisterMessageType registers a new proto message to the given opcode
func RegisterMessageType(opcode Opcode, msg proto.Message) error {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := opcodeTable[opcode]; ok {
		return errors.New("types: opcode already registered, choose a different opcode")
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
	return nil, errors.New("types: opcode not found, did you register it?")
}

// GetOpcode returns the corresponding opcode given a proto message
func GetOpcode(msg proto.Message) (*Opcode, error) {
	t := reflect.TypeOf(msg)
	if i, ok := msgTable[t]; ok {
		return &i, nil
	}
	return nil, errors.New("types: message type not found, did you register it?")
}
