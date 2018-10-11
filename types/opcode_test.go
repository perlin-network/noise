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

	msgOpcode := Opcode(1)
	msg := protobuf.TestMessage{}
	err := RegisterMessageType(msgOpcode, &msg)
	assert.NotEqual(t, nil, err, "expecting an error")

	err = RegisterMessageType(Opcode(999), &msg)
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
		msgType, err := GetMessageType(tt.opcode)
		assert.Equal(t, nil, err, "opcode should be found")
		assert.Equal(t, reflect.TypeOf(tt.msg), reflect.TypeOf(msgType), "message types should be equal")
	}

	msg := &pb.Ping{}
	msgType, err := GetMessageType(Opcode(9999))
	assert.NotEqual(t, nil, err, "there should be an error, opcode does not exist")
	assert.NotEqual(t, reflect.TypeOf(msg), reflect.TypeOf(msgType), "message types should not be equal")

	msgType, err = GetMessageType(PongCode)
	assert.Equal(t, nil, err, "opcode should be found")
	assert.NotEqual(t, reflect.TypeOf(msg), reflect.TypeOf(msgType), "message types should not be equal")
}

func TestGetOpcode(t *testing.T) {
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
		opcode, err := GetOpcode(tt.msg)
		assert.Equal(t, nil, err, "message type should be found")
		assert.Equal(t, tt.opcode, opcode, "opcodes should be equal")
	}
}
