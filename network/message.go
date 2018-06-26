package network

import (
	"errors"
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

func (ctx *MessageContext) Send(message proto.Message) error {
	return ctx.client.Tell(message)
}

func (ctx *MessageContext) Reply(message proto.Message) error {
	msg, err := ctx.client.prepareMessage(message)
	if err != nil {
		return err
	}

	bytes, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	n, err := ctx.stream.Write(bytes)
	if n != len(bytes) {
		return errors.New("failed to write all bytes to stream")
	}

	if err != nil {
		return err
	}

	return nil
}

func (ctx *MessageContext) Message() proto.Message {
	return ctx.message
}

func (ctx *MessageContext) Client() *PeerClient {
	return ctx.client
}

func (ctx *MessageContext) Network() *Network {
	return ctx.client.Network
}

func (ctx *MessageContext) Self() peer.ID {
	return ctx.Network().ID
}

func (ctx *MessageContext) Sender() peer.ID {
	return *ctx.client.Id
}
