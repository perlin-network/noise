package protocol

import (
	"bufio"
	"encoding/binary"
	"github.com/pkg/errors"
	"io"
)

const MaxPayloadLen = 1048576

type Message struct {
	Signature []byte
	Body      *MessageBody
}

type MessageBody struct {
	Sender    []byte
	Recipient []byte
	Service   uint16
	Payload   []byte
}

func (b *MessageBody) Serialize() []byte {
	if len(b.Sender) > 255 || len(b.Recipient) > 255 {
		panic("invalid sender or recipient")
	}

	buf := make([]byte, 0)
	buf = append(buf, byte(len(b.Sender)))
	buf = append(buf, b.Sender...)
	buf = append(buf, byte(len(b.Recipient)))
	buf = append(buf, b.Recipient...)
	writeUint16(&buf, b.Service)
	writeUvarint(&buf, uint64(len(b.Payload)))
	buf = append(buf, b.Payload...)
	return buf
}

func (m *Message) Serialize() []byte {
	if len(m.Signature) > 255 {
		panic("invalid signature")
	}

	buf := make([]byte, 0)
	buf = append(buf, byte(len(m.Signature)))
	buf = append(buf, m.Signature...)
	buf = append(buf, m.Body.Serialize()...)
	return buf
}

func DeserializeMessageBody(reader *bufio.Reader) (*MessageBody, error) {
	ret := &MessageBody{}
	byteBuf := make([]byte, 1)
	shortBuf := make([]byte, 2)

	_, err := io.ReadFull(reader, byteBuf)
	if err != nil {
		return nil, err
	}

	ret.Sender = make([]byte, int(byteBuf[0]))
	_, err = io.ReadFull(reader, ret.Sender)
	if err != nil {
		return nil, err
	}

	_, err = io.ReadFull(reader, byteBuf)
	if err != nil {
		return nil, err
	}

	ret.Recipient = make([]byte, int(byteBuf[0]))
	_, err = io.ReadFull(reader, ret.Recipient)
	if err != nil {
		return nil, err
	}

	_, err = io.ReadFull(reader, shortBuf)
	if err != nil {
		return nil, err
	}

	ret.Service = binary.LittleEndian.Uint16(shortBuf)

	payloadLen, err := binary.ReadUvarint(reader)
	if err != nil {
		return nil, err
	}
	if payloadLen > MaxPayloadLen {
		return nil, errors.Errorf("payload too long")
	}
	ret.Payload = make([]byte, int(payloadLen))
	_, err = io.ReadFull(reader, ret.Payload)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func DeserializeMessage(reader *bufio.Reader) (*Message, error) {
	ret := &Message{}
	byteBuf := make([]byte, 1)

	_, err := io.ReadFull(reader, byteBuf)
	if err != nil {
		return nil, err
	}
	ret.Signature = make([]byte, int(byteBuf[0]))
	_, err = io.ReadFull(reader, ret.Signature)
	if err != nil {
		return nil, err
	}
	ret.Body, err = DeserializeMessageBody(reader)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
