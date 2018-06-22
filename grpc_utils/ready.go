package grpc_utils

import (
	"context"
	"fmt"
	"time"

	"github.com/perlin-network/noise/log"
	"google.golang.org/grpc"
)

func BlockUntilConnectionReady(host string, port int, dialTimeout time.Duration) error {
	address := fmt.Sprintf("%s:%d", host, port)
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBlock(),
	}
	conn, err := grpc.DialContext(ctx, address, opts...)
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Debug(fmt.Sprintf("Server ready after %fs\n", time.Now().Sub(startTime).Seconds()))
	return nil
}
