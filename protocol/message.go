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
	Service uint16
	Payload []byte
}

func (b *MessageBody) Serialize() []byte {
	buf := make([]byte, 0)
	writeUint16(&buf, b.Service)
	buf = append(buf, b.Payload...)
	return buf
}

func DeserializeMessageBody(reader io.Reader) (*MessageBody, error) {
	ret := &MessageBody{}
	shortBuf := make([]byte, 2)

	_, err := io.ReadFull(reader, shortBuf)
	if err != nil {
		return nil, err
	}

	ret.Service = binary.LittleEndian.Uint16(shortBuf)

	payloadReader := io.LimitReader(reader, MaxPayloadLen)
	ret.Payload, err = ioutil.ReadAll(payloadReader)

	if err != nil {
		return nil, err
	}

	return ret, nil
}
