package noise

import "github.com/pkg/errors"

var (
	ErrTimeout = errors.New("timed out")
)

type ErrorInterceptor func(err error)
