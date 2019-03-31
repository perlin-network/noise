package noise

import "github.com/pkg/errors"

var (
	ErrSendQueueFull = errors.New("send queue is full")
	ErrTimeout       = errors.New("timed out")
)

type ErrorInterceptor func(err error)
