package discovery

import (
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/skademlia/protobuf"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
)

const (
	ServiceID            = 5
	OpCodePing           = 1
	OpCodePong           = 2
	OpCodeLookupRequest  = 3
	OpCodeLookupResponse = 4
)

func ToMessageBody(serviceID int, opcode int, content proto.Message) (*protocol.MessageBody, error) {
	raw, err := proto.Marshal(content)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to marshal content")
	}
	msg := &protobuf.Message{
		Message: raw,
		Opcode:  uint32(opcode),
	}
	msgBytes, err := msg.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "Unable to marshal message")
	}
	body := &protocol.MessageBody{
		Service: uint16(serviceID),
		Payload: msgBytes,
	}
	return body, nil
}

func ParseMessageBody(body *protocol.MessageBody, dest proto.Message) (int, error) {
	if body == nil || len(body.Payload) == 0 {
		return -1, errors.New("body is empty")
	}
	var msg protobuf.Message
	if err := proto.Unmarshal(body.Payload, &msg); err != nil {
		return -1, errors.Wrap(err, "unable to unmarshal payload")
	}
	if err := proto.Unmarshal(msg.Message, dest); err != nil {
		return -1, errors.Wrap(err, "unable to unmarshal message")
	}
	return int(msg.Opcode), nil
}
