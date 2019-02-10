package noise

import (
	"bytes"
	"fmt"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/payload"
	"github.com/pkg/errors"
	"reflect"
)

type Opcode byte

const (
	OpcodeNil Opcode = 0
)

var (
	opcodes  map[Opcode]Message
	messages map[reflect.Type]Opcode
)

func init() {
	resetOpcodes()
}

func (o Opcode) Bytes() (buf [1]byte) {
	buf[0] = byte(o)
	return
}

func NextAvailableOpcode() Opcode {
	return Opcode(len(opcodes))
}

func DebugOpcodes() {
	log.Debug().Msg("Here are all opcodes registered so far.")

	for i := 0; i < len(opcodes); i++ {
		fmt.Printf("Opcode %d is registered to: %s\n", i, reflect.TypeOf(opcodes[Opcode(i)]).String())
	}
}

func MessageFromOpcode(opcode Opcode) (Message, error) {
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

func OpcodeFromMessage(msg Message) (Opcode, error) {
	typ := reflect.TypeOf(msg)

	opcode, exists := messages[typ]
	if !exists {
		return OpcodeNil, errors.Errorf("there is no opcode registered for message type %v", typ)
	}

	return opcode, nil
}

type Message interface {
	Read(reader payload.Reader) (Message, error)
	Write() []byte
}

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

func RegisterMessage(o Opcode, m interface{}) Opcode {
	if t, registered := opcodes[o]; registered {
		panic(errors.Errorf("noise: opcode %v was already registered with type %T; tried registering it with type %T", o, m, t))
	}

	typ := reflect.TypeOf(m).Elem()

	opcodes[o] = reflect.New(typ).Elem().Interface().(Message)
	messages[typ] = o

	return o
}

func resetOpcodes() {
	opcodes = map[Opcode]Message{
		OpcodeNil: reflect.New(reflect.TypeOf((*EmptyMessage)(nil)).Elem()).Elem().Interface().(Message),
	}

	messages = map[reflect.Type]Opcode{
		reflect.TypeOf((*EmptyMessage)(nil)).Elem(): OpcodeNil,
	}
}
