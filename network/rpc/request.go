package rpc

import (
	"github.com/golang/protobuf/proto"
	"time"
)

// Request represents a single message which, once sent, expects
// a response designated by a timeout.
type Request struct {
	Message proto.Message
	Timeout time.Duration
}

// SetMessage sets the message body contents of the request.
func (r *Request) SetMessage(message proto.Message) {
	r.Message = message
}

// SetTimeout sets the expected deadline for a response to come w.r.t. the request.
func (r *Request) SetTimeout(timeout time.Duration) {
	r.Timeout = timeout
}
