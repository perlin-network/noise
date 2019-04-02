package xnoise

import (
	"github.com/perlin-network/noise"
	"github.com/pkg/errors"
	"log"
)

func LogErrors(ctx noise.Context) error {
	ctx.Peer().InterceptErrors(func(err error) {
		switch errors.Cause(err) {
		default:
			log.Printf("%+v\n", err)
		}
	})

	return nil
}
