package discovery

import (
	"context"
	"github.com/gogo/protobuf/proto"
	"github.com/perlin-network/noise/internal/protobuf"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
)

const (
	DiscoveryServiceID   = 5
	opCodePing           = 1
	opCodePong           = 2
	opCodeLookupRequest  = 3
	opCodeLookupResponse = 4
)

type SendHandler interface {
	Request(ctx context.Context, target []byte, body *protocol.MessageBody) (*protocol.MessageBody, error)
}

func toProtobufMessage(opcode int, content proto.Message) (*protobuf.Message, error) {
	raw, err := proto.Marshal(content)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to unmarshal reply")
	}
	msg := &protobuf.Message{
		Message: raw,
		Opcode:  uint32(opcode),
	}
	return msg, nil
}
