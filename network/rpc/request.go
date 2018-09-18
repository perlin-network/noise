package rpc

import (
	"github.com/gogo/protobuf/proto"
)

// Request represents a single message
type Request struct {
	Message proto.Message
}

// SetMessage sets the message body contents of the request.
func (r *Request) SetMessage(message proto.Message) {
	r.Message = message
}
