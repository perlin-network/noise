package types

import (
	"reflect"
	"testing"

	pb "github.com/perlin-network/noise/internal/protobuf"
	"github.com/perlin-network/noise/internal/test/protobuf"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestRegisterMessageType(t *testing.T) {
	t.Parallel()

	msgOpcode := Opcode(0)
	msg := protobuf.TestMessage{}
	err := RegisterMessageType(msgOpcode, &msg)
	assert.NotEqual(t, nil, err, "expecting an error")
	msgOpcode = Opcode(1000)
	err = RegisterMessageType(msgOpcode, &msg)
	assert.Equal(t, nil, err, "not expecting an error")
}

func TestGetMessageType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		msg    proto.Message
		opcode Opcode
	}{
		{&pb.Ping{}, PingCode},
		{&pb.Pong{}, PongCode},
		{&pb.LookupNodeRequest{}, LookupNodeRequestCode},
		{&pb.LookupNodeResponse{}, LookupNodeResponseCode},
	}

	for _, tt := range testCases {
		assert.Equal(t, reflect.TypeOf(tt.msg), reflect.TypeOf(GetMessageType(tt.opcode)), "message types should be equal")
	}

	msg := &pb.Ping{}
	assert.NotEqual(t, reflect.TypeOf(msg), reflect.TypeOf(GetMessageType(PongCode)), "message types should not be equal")
}
