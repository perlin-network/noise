package network

import (
	"github.com/golang/protobuf/proto"
	"github.com/perlin-network/noise/peer"
)

// MessageContext provides parameters and helper functions to a MessageProcessor
// for interacting with/analyzing incoming messages from a select peer.
type MessageContext struct {
	client  *PeerClient
	message proto.Message
	nonce   uint64
}

// Reply sends back a message to an incoming message's incoming stream.
func (ctx *MessageContext) Reply(message proto.Message) error {
	return ctx.client.Reply(ctx.nonce, message)
}

// Message returns the decoded protobuf message.
func (ctx *MessageContext) Message() proto.Message {
	return ctx.message
}

// Client returns the peer client.
func (ctx *MessageContext) Client() *PeerClient {
	return ctx.client
}

// Network returns the entire node's network.
func (ctx *MessageContext) Network() *Network {
	return ctx.client.Network
}

// Self returns the node's ID.
func (ctx *MessageContext) Self() peer.ID {
	return ctx.Network().ID
}

// Sender returns the peer's ID.
func (ctx *MessageContext) Sender() peer.ID {
	return *ctx.client.GetID()
}
