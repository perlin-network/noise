package noise

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/perlin-network/noise/log"
	"github.com/pkg/errors"
)

type Opcode byte

const (
	OpcodeNil Opcode = 0
)

var (
	opcodes  map[Opcode]Message
	messages map[reflect.Type]Opcode

	opcodesMutex sync.Mutex
)

// Bytes returns this opcodes' byte representation.
func (o Opcode) Bytes() (buf [1]byte) {
	buf[0] = byte(o)
	return
}

// NextAvailableOpcode returns the next available unregistered message opcode
// registered to Noise.
func NextAvailableOpcode() Opcode {
	opcodesMutex.Lock()
	defer opcodesMutex.Unlock()

	return Opcode(len(opcodes))
}

// DebugOpcodes prints out all opcodes registered to Noise thus far.
func DebugOpcodes() {
	opcodesMutex.Lock()
	defer opcodesMutex.Unlock()

	log.Debug().Msg("Here are all opcodes registered so far.")

	for i := 0; i < len(opcodes); i++ {
		fmt.Printf("Opcode %d is registered to: %s\n", i, reflect.TypeOf(opcodes[Opcode(i)]).String())
	}
}

// MessageFromOpcode returns an empty message representation associated to a registered
// message opcode.
//
// It errors if the specified message opcode is not registered to Noise.
func MessageFromOpcode(opcode Opcode) (Message, error) {
	opcodesMutex.Lock()
	defer opcodesMutex.Unlock()

	typ, exists := opcodes[Opcode(opcode)]
	if !exists {
		return nil, errors.Errorf("there is no message type registered to opcode %d", opcode)
	}

	message, ok := reflect.New(reflect.TypeOf(typ)).Elem().Interface().(Message)
	if !ok {
		return nil, errors.Errorf("invalid message type associated to opcode %d", opcode)
	}

	return message, nil
}

// OpcodeFromMessage uses reflection to extract and return the opcode associated to a message
// value type.
//
// It errors if the specified message value type is not registered to Noise.
func OpcodeFromMessage(msg Message) (Opcode, error) {
	opcodesMutex.Lock()
	defer opcodesMutex.Unlock()

	typ := reflect.TypeOf(msg)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	opcode, exists := messages[typ]
	if !exists {
		return OpcodeNil, errors.Errorf("there is no opcode registered for message type %v", typ)
	}

	return opcode, nil
}

func RegisterMessage(o Opcode, m interface{}) Opcode {
	typ := reflect.TypeOf(m).Elem()

	opcodesMutex.Lock()
	defer opcodesMutex.Unlock()

	if opcode, registered := messages[typ]; registered {
		return opcode
	}

	opcodes[o] = reflect.New(typ).Elem().Interface().(Message)
	messages[typ] = o

	return o
}

func resetOpcodes() {
	opcodesMutex.Lock()
	defer opcodesMutex.Unlock()

	opcodes = map[Opcode]Message{
		OpcodeNil: reflect.New(reflect.TypeOf((*EmptyMessage)(nil)).Elem()).Elem().Interface().(Message),
	}

	messages = map[reflect.Type]Opcode{
		reflect.TypeOf((*EmptyMessage)(nil)).Elem(): OpcodeNil,
	}
}
