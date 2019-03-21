package noise

import (
	"bytes"
	"github.com/perlin-network/noise/payload"
	"github.com/pkg/errors"
)

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
func (p *Peer) EncodeMessage(channelID ChannelID, message Message) ([]byte, error) {
	opcode, err := OpcodeFromMessage(message)
	if err != nil {
		return nil, errors.Wrap(err, "could not find opcode registered for message")
	}

	var buf bytes.Buffer

	_, err = buf.Write(channelID[:])
	if err != nil {
		return nil, errors.Wrap(err, "failed to serialize channel id")
	}

	_, err = buf.Write(payload.NewWriter(nil).WriteByte(byte(opcode)).Bytes())
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

func (p *Peer) DecodeMessage(buf []byte) (ChannelID, Opcode, Message, error) {
	var channelID ChannelID
	if len(buf) < ChannelIDSize {
		return [ChannelIDSize]byte{}, OpcodeNil, nil, errors.New("unable to read channel id")
	}
	copy(channelID[:], buf)
	buf = buf[ChannelIDSize:]

	reader := payload.NewReader(buf)

	// Read custom header from network packet.
	errs := p.onDecodeHeaderCallbacks.RunCallbacks(p.node, reader)

	if len(errs) > 0 {
		err := errs[0]

		for _, e := range errs[1:] {
			err = errors.Wrap(e, e.Error())
		}

		return [ChannelIDSize]byte{}, OpcodeNil, nil, errors.Wrap(err, "failed to decode custom headers")
	}

	afterHeaderSize := len(buf) - reader.Len()

	opcode, err := reader.ReadByte()
	if err != nil {
		return [ChannelIDSize]byte{}, OpcodeNil, nil, errors.Wrap(err, "failed to read opcode")
	}

	message, err := MessageFromOpcode(Opcode(opcode))
	if err != nil {
		return [ChannelIDSize]byte{}, Opcode(opcode), nil, errors.Wrap(err, "opcode<->message pairing not registered")
	}

	message, err = message.Read(reader)
	if err != nil {
		return [ChannelIDSize]byte{}, Opcode(opcode), nil, errors.Wrap(err, "failed to read message contents")
	}

	afterMessageSize := len(buf) - reader.Len()

	// Read custom footer from network packet.
	errs = p.onDecodeFooterCallbacks.RunCallbacks(p.node, buf[afterHeaderSize:afterMessageSize], reader)

	if len(errs) > 0 {
		err := errs[0]

		for _, e := range errs[1:] {
			err = errors.Wrap(e, e.Error())
		}

		return [ChannelIDSize]byte{}, OpcodeNil, nil, errors.Wrap(err, "failed to decode custom footer")
	}

	return channelID, Opcode(opcode), message, nil
}
