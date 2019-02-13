package noise

import (
	"bytes"
	"fmt"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/payload"
	"github.com/pkg/errors"
	"reflect"
	"sync"
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

func init() {
	resetOpcodes()
}

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

// To have Noise send/receive messages of a given type, said type must implement the
// following Message interface.
//
// Noise by default encodes messages as bytes in little-endian order, and provides
// utility classes to assist with serializing/deserializing arbitrary Go types into
// bytes efficiently.
//
// By exposing raw network packets as bytes to users, any additional form of serialization
// or message packing or compression scheme or cipher scheme may be bootstrapped on top of
// any particular message type registered to Noise.
type Message interface {
	Read(reader payload.Reader) (Message, error)
	Write() []byte
}

// EncodeMessage serializes a message body into its byte representation, and prefixes
// said byte representation with the messages opcode for the purpose of sending said
// bytes over the wire.
//
// Additional header/footer bytes is prepended/appended accordingly.
//
// Refer to the functions `OnEncodeHeader` and `OnEncodeFooter` available in `noise.Peer`
// to prepend/append additional information on every single message sent over the wire.
func (p *Peer) EncodeMessage(opcode Opcode, message Message) ([]byte, error) {
	var buf bytes.Buffer

	_, err := buf.Write(payload.NewWriter(nil).WriteByte(byte(opcode)).Bytes())
	if err != nil {
		return nil, errors.Wrap(err, "failed to serialize message opcode")
	}

	_, err = buf.Write(message.Write())
	if err != nil {
		return nil, errors.Wrap(err, "failed to serialize and write message contents")
	}

	header, errs := p.onEncodeHeaderCallbacks.RunCallbacks([]byte{}, p.node, buf.Bytes())

	if len(errs) > 0 {
		err := errs[0]

		for _, e := range errs[1:] {
			err = errors.Wrap(e, e.Error())
		}

		return nil, errors.Wrap(err, "failed to serialize custom footer")
	}

	footer, errs := p.onEncodeFooterCallbacks.RunCallbacks([]byte{}, p.node, buf.Bytes())

	if len(errs) > 0 {
		err := errs[0]

		for _, e := range errs[1:] {
			err = errors.Wrap(e, e.Error())
		}

		return nil, errors.Wrap(err, "failed to serialize custom footer")
	}

	return append(header.([]byte), append(buf.Bytes(), footer.([]byte)...)...), nil
}

func (p *Peer) DecodeMessage(buf []byte) (Opcode, Message, error) {
	reader := payload.NewReader(buf)

	// Read custom header from network packet.
	errs := p.onDecodeHeaderCallbacks.RunCallbacks(p.node, reader)

	if len(errs) > 0 {
		err := errs[0]

		for _, e := range errs[1:] {
			err = errors.Wrap(e, e.Error())
		}

		return OpcodeNil, nil, errors.Wrap(err, "failed to decode custom headers")
	}

	afterHeaderSize := len(buf) - reader.Len()

	opcode, err := reader.ReadByte()
	if err != nil {
		return OpcodeNil, nil, errors.Wrap(err, "failed to read opcode")
	}

	message, err := MessageFromOpcode(Opcode(opcode))
	if err != nil {
		return Opcode(opcode), nil, errors.Wrap(err, "opcode<->message pairing not registered")
	}

	message, err = message.Read(reader)
	if err != nil {
		return Opcode(opcode), nil, errors.Wrap(err, "failed to read message contents")
	}

	afterMessageSize := len(buf) - reader.Len()

	// Read custom footer from network packet.
	errs = p.onDecodeFooterCallbacks.RunCallbacks(p.node, buf[afterHeaderSize:afterMessageSize], reader)

	if len(errs) > 0 {
		err := errs[0]

		for _, e := range errs[1:] {
			err = errors.Wrap(e, e.Error())
		}

		return OpcodeNil, nil, errors.Wrap(err, "failed to decode custom footer")
	}

	return Opcode(opcode), message, nil
}

var _ Message = (*EmptyMessage)(nil)

type EmptyMessage struct{}

func (EmptyMessage) Read(reader payload.Reader) (Message, error) {
	return EmptyMessage{}, nil
}

func (EmptyMessage) Write() []byte {
	return nil
}
