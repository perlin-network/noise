package network

import (
	"context"

	"github.com/perlin-network/noise/internal/protobuf"
	"github.com/perlin-network/noise/peer"

	"github.com/gogo/protobuf/proto"
)

// PluginContext provides parameters and helper functions to a Plugin
// for interacting with/analyzing incoming messages from a select peer.
type PluginContext struct {
	client     *PeerClient
	message    proto.Message
	nonce      uint64
	rawMessage protobuf.Message
	signature  []byte
}

// Reply sends back a message to an incoming message's incoming stream.
func (pctx *PluginContext) Reply(ctx context.Context, message proto.Message, opts ...ReplyOption) error {
	opts = append(opts, WithRequestNonce(pctx.nonce))
	return pctx.client.Reply(ctx, message, opts...)
}

// Message returns the decoded protobuf message.
func (pctx *PluginContext) Message() proto.Message {
	return pctx.message
}

// Client returns the peer client.
func (pctx *PluginContext) Client() *PeerClient {
	return pctx.client
}

// Network returns the entire node's network.
func (pctx *PluginContext) Network() *Network {
	return pctx.client.Network
}

// RawMessage returns the raw protobuf message
func (pctx *PluginContext) RawMessage() protobuf.Message {
	return pctx.rawMessage
}

// Self returns the node's ID.
func (pctx *PluginContext) Self() peer.ID {
	return pctx.Network().ID
}

// Sender returns the peer's ID.
func (pctx *PluginContext) Sender() peer.ID {
	return *pctx.client.ID
}
