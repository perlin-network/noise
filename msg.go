package noise

import (
	"encoding/binary"
	"errors"
	"go.uber.org/atomic"
	"io"
)

type message struct {
	nonce uint64
	data  []byte
}

func (m message) Marshal() []byte {
	header := make([]byte, 8)
	binary.BigEndian.PutUint64(header[:8], m.nonce)

	return append(header, m.data...)
}

func unmarshalMessage(data []byte) (message, error) {
	if len(data) < 8 {
		return message{}, io.ErrUnexpectedEOF
	}

	nonce := binary.BigEndian.Uint64(data[:8])
	data = data[8:]

	return message{nonce: nonce, data: data}, nil
}

type HandlerContext struct {
	client *Client
	msg    message
	sent   atomic.Bool
}

func (ctx *HandlerContext) ID() ID {
	return ctx.client.ID()
}

func (ctx *HandlerContext) Data() []byte {
	return ctx.msg.data
}

func (ctx *HandlerContext) IsRequest() bool {
	return ctx.msg.nonce > 0
}

func (ctx *HandlerContext) Send(data []byte) error {
	if ctx.IsRequest() && !ctx.sent.CAS(false, true) {
		return errors.New("server-side may only send back a single response to a request")
	}

	return ctx.client.send(ctx.msg.nonce, data)
}

func (ctx *HandlerContext) DecodeMessage() (Serializable, error) {
	return ctx.client.node.DecodeMessage(ctx.Data())
}

func (ctx *HandlerContext) SendMessage(msg Serializable) error {
	data, err := ctx.client.node.EncodeMessage(msg)
	if err != nil {
		return err
	}

	return ctx.Send(data)
}
