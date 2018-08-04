package test

import (
	"github.com/perlin-network/noise/internal/test/protobuf"
	"github.com/perlin-network/noise/network"
)

var (
	mailboxPluginID = (*MailBoxPlugin)(nil)
)

// MailBoxPlugin buffers all messages into a mailbox for test validation.
type MailBoxPlugin struct {
	*network.Plugin
	RecvMailbox chan *protobuf.TestMessage
	SendMailbox chan *protobuf.TestMessage
}

// Startup creates a mailbox channel
func (state *MailBoxPlugin) Startup(net *network.Network) {
	state.RecvMailbox = make(chan *protobuf.TestMessage)
	state.SendMailbox = make(chan *protobuf.TestMessage)
}

// Send puts a sent message into the SendMailbox channel
func (state *MailBoxPlugin) Send(ctx *network.PluginContext) error {
	switch msg := ctx.Message().(type) {
	case *protobuf.TestMessage:
		state.SendMailbox <- msg
	}
	return nil
}

// Receive puts a received message into the RecvMailbox channel
func (state *MailBoxPlugin) Receive(ctx *network.PluginContext) error {
	switch msg := ctx.Message().(type) {
	case *protobuf.TestMessage:
		state.RecvMailbox <- msg
	}
	return nil
}
