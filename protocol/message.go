package protocol

import (
	"encoding/binary"
	"io"
	"io/ioutil"
)

const MaxPayloadLen = 1048576

type Message struct {
	Sender    []byte
	Recipient []byte
	Body      *MessageBody
}

type MessageBody struct {
	Service      uint16
	RequestNonce uint64
	Payload      []byte
}

func (b *MessageBody) Serialize() []byte {
	buf := make([]byte, 0)
	writeUint16(&buf, b.Service)
	writeUint64(&buf, b.RequestNonce)
	buf = append(buf, b.Payload...)
	return buf
}

func DeserializeMessageBody(reader io.Reader) (*MessageBody, error) {
	ret := &MessageBody{}
	serviceBuf := make([]byte, 2)
	requestNonceBuf := make([]byte, 4)

	if _, err := io.ReadFull(reader, serviceBuf); err != nil {
		return nil, err
	}
	ret.Service = binary.LittleEndian.Uint16(serviceBuf)

	if _, err := io.ReadFull(reader, requestNonceBuf); err != nil {
		return nil, err
	}
	ret.RequestNonce = binary.LittleEndian.Uint64(requestNonceBuf)

	payloadReader := io.LimitReader(reader, MaxPayloadLen)
	var err error
	ret.Payload, err = ioutil.ReadAll(payloadReader)
	if err != nil {
		return nil, err
	}

	return ret, nil
}
