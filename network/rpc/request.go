package rpc

import (
	"github.com/golang/protobuf/proto"
	"time"
)

type Request struct {
	Message proto.Message
	Timeout time.Duration
}

func (r *Request) SetMessage(message proto.Message) {
	r.Message = message
}

func (r *Request) SetTimeout(timeout time.Duration) {
	r.Timeout = timeout
}
