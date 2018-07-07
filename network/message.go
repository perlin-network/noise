package network

import (
	"github.com/golang/protobuf/proto"
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
func (ctx *PluginContext) Reply(message proto.Message) error {
	return ctx.client.Reply(ctx.nonce, message)
}

// Message returns the decoded protobuf message.
func (ctx *PluginContext) Message() proto.Message {
	return ctx.message
}

// Client returns the peer client.
func (ctx *PluginContext) Client() *PeerClient {
	return ctx.client
}

// Network returns the entire node's network.
func (ctx *PluginContext) Network() *Network {
	return ctx.client.Network
}

// Self returns the node's ID.
func (ctx *PluginContext) Self() peer.ID {
	return ctx.Network().ID
}

// Sender returns the peer's ID.
func (ctx *PluginContext) Sender() peer.ID {
	return *ctx.client.ID
}
