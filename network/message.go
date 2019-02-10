package network

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/perlin-network/noise/peer"
)

// PluginContext provides parameters and helper functions to a Plugin
// for interacting with/analyzing incoming messages from a select peer.
type PluginContext struct {
	client  *PeerClient
	message proto.Message
	nonce   uint64
}

// Reply sends back a message to an incoming message's incoming stream.
func (pctx *PluginContext) Reply(ctx context.Context, message proto.Message) error {
	return pctx.client.Reply(ctx, pctx.nonce, message)
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

// Self returns the node's ID.
func (pctx *PluginContext) Self() peer.ID {
	return pctx.Network().ID
}

// Sender returns the peer's ID.
func (pctx *PluginContext) Sender() peer.ID {
	return *pctx.client.ID
}
