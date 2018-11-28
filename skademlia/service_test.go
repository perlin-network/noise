package skademlia_test

import (
	"context"
	"testing"

	"github.com/perlin-network/noise/internal/protobuf"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/skademlia"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type MockSendHandler struct {
	RequestCallback func(target []byte, body *protocol.MessageBody) (*protocol.MessageBody, error)
}

func (m *MockSendHandler) Request(ctx context.Context, target []byte, body *protocol.MessageBody) (*protocol.MessageBody, error) {
	return m.RequestCallback(target, body)
}

func (m *MockSendHandler) Broadcast(body *protocol.MessageBody) error {
	return errors.New("Not implemented")
}

func TestDiscoveryPing(t *testing.T) {
	s := skademlia.NewService(nil, peer.CreateID("selfAddr", ([]byte)("self")))
	assert.NotNil(t, s)
	s.Routes.Update(peer.CreateID("senderAddr", ([]byte)("sender")))
	s.Routes.Update(peer.CreateID("recipientAddr", ([]byte)("recipient")))

	body, err := skademlia.ToMessageBody(skademlia.ServiceID, skademlia.OpCodePing, &protobuf.Ping{})
	assert.Nil(t, err)
	reply, err := s.Receive(&protocol.Message{
		Sender:    ([]byte)("sender"),
		Recipient: ([]byte)("recipient"),
		Body:      body,
	})
	assert.Nil(t, err)

	var respMsg protobuf.Pong
	opCode, err := skademlia.ParseMessageBody(reply, &respMsg)
	assert.Nil(t, err)
	assert.Equal(t, skademlia.OpCodePong, opCode)
}

func TestDiscoveryPong(t *testing.T) {
	msh := &MockSendHandler{
		RequestCallback: func(target []byte, reqBody *protocol.MessageBody) (*protocol.MessageBody, error) {
			var respMsg protobuf.LookupNodeRequest
			opCode, err := skademlia.ParseMessageBody(reqBody, &respMsg)
			assert.Nil(t, err)
			assert.Equal(t, skademlia.OpCodeLookupRequest, opCode)
			respBody, err := skademlia.ToMessageBody(skademlia.ServiceID, skademlia.OpCodeLookupResponse, &protobuf.LookupNodeResponse{})
			assert.Nil(t, err)
			return respBody, nil
		},
	}
	s := skademlia.NewService(msh, peer.CreateID("selfAddr", ([]byte)("self")))
	assert.NotNil(t, s)
	s.Routes.Update(peer.CreateID("senderAddr", ([]byte)("sender")))
	s.Routes.Update(peer.CreateID("recipientAddr", ([]byte)("recipient")))

	content := &protobuf.Pong{}
	body, err := skademlia.ToMessageBody(skademlia.ServiceID, skademlia.OpCodePong, content)
	assert.Nil(t, err)
	reply, err := s.Receive(&protocol.Message{
		Sender:    ([]byte)("sender"),
		Recipient: ([]byte)("recipient"),
		Body:      body,
	})
	assert.Nil(t, err)
	assert.Nil(t, reply)
}

func TestDiscoveryLookupRequest(t *testing.T) {
	s := skademlia.NewService(nil, peer.CreateID("selfAddr", ([]byte)("self")))
	assert.NotNil(t, s)
	s.Routes.Update(peer.CreateID("senderAddr", ([]byte)("sender")))
	s.Routes.Update(peer.CreateID("recipientAddr", ([]byte)("recipient")))

	reqTargetID := protobuf.ID(peer.CreateID("senderAddr", ([]byte)("sender")))
	content := &protobuf.LookupNodeRequest{Target: &reqTargetID}
	body, err := skademlia.ToMessageBody(skademlia.ServiceID, skademlia.OpCodeLookupRequest, content)
	assert.Nil(t, err)
	reply, err := s.Receive(&protocol.Message{
		Sender:    ([]byte)("sender"),
		Recipient: ([]byte)("recipient"),
		Body:      body,
	})
	assert.Nil(t, err)

	var respMsg protobuf.LookupNodeResponse
	opCode, err := skademlia.ParseMessageBody(reply, &respMsg)
	assert.Nil(t, err)
	assert.Equal(t, skademlia.OpCodeLookupResponse, opCode)

	assert.Equal(t, 3, len(respMsg.Peers))
	for _, addr := range []string{"selfAddr", "recipientAddr", "senderAddr"} {
		found := false
		for _, peer := range respMsg.Peers {
			if peer.Address == addr {
				found = true
				break
			}
		}
		assert.Truef(t, found, "Unable to find address in list: %s", addr)
	}
}