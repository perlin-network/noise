package noise

import "github.com/pkg/errors"

var (
	ErrTimeout       = errors.New("timed out")
	ErrDisconnect    = errors.New("disconnect requested")
	ErrSendQueueFull = errors.New("send queue is full")
	ErrRecvQueueFull = errors.New("recv queue is full")
)

type ErrorInterceptor func(err error)
