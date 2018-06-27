package network

import (
	"github.com/golang/protobuf/proto"
	"github.com/perlin-network/noise/peer"
	"github.com/xtaci/smux"
)

type MessageProcessor interface {
	Handle(ctx *MessageContext) error
}

type MessageContext struct {
	client  *PeerClient
	stream  *smux.Stream
	message proto.Message
}

// Send opens a new stream and send a message to a client.
func (ctx *MessageContext) Send(message proto.Message) error {
	return ctx.client.Tell(message)
}

// Reply sends back a message to an incoming message's incoming stream.
func (ctx *MessageContext) Reply(message proto.Message) error {
	return ctx.client.sendMessage(ctx.stream, message)
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
	return *ctx.client.Id
}
