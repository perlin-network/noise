package noise

import (
	"encoding/binary"
	"errors"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"io"
)

type message struct {
	nonce uint64
	data  []byte
}

func (m message) marshal(dst []byte) []byte {
	dst = append(dst, make([]byte, 8)...)
	binary.BigEndian.PutUint64(dst[:8], m.nonce)
	dst = append(dst, m.data...)

	return dst
}

func unmarshalMessage(data []byte) (message, error) {
	if len(data) < 8 {
		return message{}, io.ErrUnexpectedEOF
	}

	nonce := binary.BigEndian.Uint64(data[:8])
	data = data[8:]

	return message{nonce: nonce, data: data}, nil
}

// HandlerContext provides contextual information upon the recipient of data from an inbound/outbound connection. It
// provides the option of responding to a request should the data received be of a request.
type HandlerContext struct {
	client *Client
	msg    message
	sent   atomic.Bool
}

// ID returns the ID of the inbound/outbound peer that sent you the data that is currently being handled.
func (ctx *HandlerContext) ID() ID {
	return ctx.client.ID()
}

// Logger returns the logger instance associated to the inbound/outbound peer being handled.
func (ctx *HandlerContext) Logger() *zap.Logger {
	return ctx.client.Logger()
}

// Data returns the raw bytes that some peer has sent to you.
//
// Data may be called concurrently.
func (ctx *HandlerContext) Data() []byte {
	return ctx.msg.data
}

// IsRequest marks whether or not the data received was intended to be of a request.
//
// IsRequest may be called concurrently.
func (ctx *HandlerContext) IsRequest() bool {
	return ctx.msg.nonce > 0
}

// Send sends data back to the peer that has sent you data. Should the data the peer send you be of a request, Send
// will send data back as a response. It returns an error if multiple responses attempt to be sent to a single request,
// or if an error occurred while attempting to send the peer a message.
//
// Send may be called concurrently.
func (ctx *HandlerContext) Send(data []byte) error {
	if ctx.IsRequest() && !ctx.sent.CAS(false, true) {
		return errors.New("server-side may only send back a single response to a request")
	}

	return ctx.client.send(ctx.msg.nonce, data)
}

// DecodeMessage decodes the raw bytes that some peer has sent you into a Go type. The Go type must have previously
// been registered to the node to which the handler this context is under was registered on. An error is thrown
// otherwise.
//
// It is highly recommended that should you choose to have your application utilize noise's serialization/
// deserialization framework for data over-the-wire, that all handlers use them by default.
//
// DecodeMessage may be called concurrently.
func (ctx *HandlerContext) DecodeMessage() (Serializable, error) {
	return ctx.client.node.DecodeMessage(ctx.Data())
}

// SendMessage encodes and serializes a Go type into a byte slice, and sends data back to the peer that has sent you
// data as either a response or message. Refer to (*HandlerContext).Send for more details. An error is thrown if
// the Go type passed in has not been registered to the node to which the handler this context is under was registered
// on.
//
// It is highly recommended that should you choose to have your application utilize noise's
// serialization/deserialization framework for data over-the-wire, that all handlers use them by default.
//
// SendMessage may be called concurrently.
func (ctx *HandlerContext) SendMessage(msg Serializable) error {
	data, err := ctx.client.node.EncodeMessage(msg)
	if err != nil {
		return err
	}

	return ctx.Send(data)
}
