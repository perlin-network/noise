package grpc_utils

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
)

// block until a connection to the host and port is successful
func BlockUntilConnectionReady(host string, port int, dialTimeout time.Duration) error {
	address := fmt.Sprintf("%s:%d", host, port)

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
	conn.Close()

	return nil
}
