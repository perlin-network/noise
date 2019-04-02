package xnoise

import (
	"github.com/perlin-network/noise"
	"github.com/pkg/errors"
	"io"
	"log"
)

func LogErrors(ctx noise.Context) error {
	ctx.Peer().InterceptErrors(func(err error) {
		switch errors.Cause(err) {
		case noise.ErrDisconnect:
		case io.ErrClosedPipe:
		case io.EOF:
		default:
			log.Printf("%+v\n", err)
		}
	})

	return nil
}
