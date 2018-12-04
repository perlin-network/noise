package discovery_test

import (
	"context"
	"testing"

	"github.com/perlin-network/noise/internal/protobuf"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/skademlia/discovery"
	"github.com/perlin-network/noise/skademlia/peer"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type MockSendAdapter struct {
	RequestCallback func(ctx context.Context, recipient []byte, body *protocol.MessageBody) (*protocol.MessageBody, error)
}

func (m *MockSendAdapter) Send(ctx context.Context, recipient []byte, body *protocol.MessageBody) error {
	return errors.New("Not implemented")
}

func (m *MockSendAdapter) Request(ctx context.Context, recipient []byte, body *protocol.MessageBody) (*protocol.MessageBody, error) {
	return m.RequestCallback(ctx, recipient, body)
}

func (m *MockSendAdapter) Broadcast(ctx context.Context, body *protocol.MessageBody) error {
	return errors.New("Not implemented")
}

func TestDiscoveryPing(t *testing.T) {
	s := discovery.NewService(nil, peer.CreateID("selfAddr", ([]byte)("self")))
	assert.NotNil(t, s)
	s.Routes.Update(peer.CreateID("senderAddr", ([]byte)("sender")))
	s.Routes.Update(peer.CreateID("recipientAddr", ([]byte)("recipient")))

	body, err := discovery.ToMessageBody(discovery.ServiceID, discovery.OpCodePing, &protobuf.Ping{})
	assert.Nil(t, err)
	reply, err := s.Receive(context.Background(), &protocol.Message{
		Sender:    ([]byte)("sender"),
		Recipient: ([]byte)("recipient"),
		Body:      body,
	})
	assert.Nil(t, err)

	var respMsg protobuf.Pong
	opCode, err := discovery.ParseMessageBody(reply, &respMsg)
	assert.Nil(t, err)
	assert.Equal(t, discovery.OpCodePong, opCode)
}

func TestDiscoveryPong(t *testing.T) {
	msh := &MockSendAdapter{
		RequestCallback: func(ctx context.Context, target []byte, reqBody *protocol.MessageBody) (*protocol.MessageBody, error) {
			var respMsg protobuf.LookupNodeRequest
			opCode, err := discovery.ParseMessageBody(reqBody, &respMsg)
			assert.Nil(t, err)
			assert.Equal(t, discovery.OpCodeLookupRequest, opCode)
			respBody, err := discovery.ToMessageBody(discovery.ServiceID, discovery.OpCodeLookupResponse, &protobuf.LookupNodeResponse{})
			assert.Nil(t, err)
			return respBody, nil
		},
	}
	s := discovery.NewService(msh, peer.CreateID("selfAddr", ([]byte)("self")))
	assert.NotNil(t, s)
	s.Routes.Update(peer.CreateID("senderAddr", ([]byte)("sender")))
	s.Routes.Update(peer.CreateID("recipientAddr", ([]byte)("recipient")))

	content := &protobuf.Pong{}
	body, err := discovery.ToMessageBody(discovery.ServiceID, discovery.OpCodePong, content)
	assert.Nil(t, err)
	reply, err := s.Receive(context.Background(), &protocol.Message{
		Sender:    ([]byte)("sender"),
		Recipient: ([]byte)("recipient"),
		Body:      body,
	})
	assert.Nil(t, err)
	assert.Nil(t, reply)
}

func TestDiscoveryLookupRequest(t *testing.T) {
	s := discovery.NewService(nil, peer.CreateID("selfAddr", ([]byte)("self")))
	assert.NotNil(t, s)
	s.Routes.Update(peer.CreateID("senderAddr", ([]byte)("sender")))
	s.Routes.Update(peer.CreateID("recipientAddr", ([]byte)("recipient")))

	reqTargetID := protobuf.ID(peer.CreateID("senderAddr", ([]byte)("sender")))
	content := &protobuf.LookupNodeRequest{Target: &reqTargetID}
	body, err := discovery.ToMessageBody(discovery.ServiceID, discovery.OpCodeLookupRequest, content)
	assert.Nil(t, err)
	reply, err := s.Receive(context.Background(), &protocol.Message{
		Sender:    ([]byte)("sender"),
		Recipient: ([]byte)("recipient"),
		Body:      body,
	})
	assert.Nil(t, err)

	var respMsg protobuf.LookupNodeResponse
	opCode, err := discovery.ParseMessageBody(reply, &respMsg)
	assert.Nil(t, err)
	assert.Equal(t, discovery.OpCodeLookupResponse, opCode)

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
