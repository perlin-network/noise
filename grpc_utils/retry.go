package grpc_utils

import (
	"context"
	"fmt"
	"time"

	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protobuf"
	"google.golang.org/grpc"
)

func BlockUntilServerReady(host string, port int, timeout time.Duration) error {
	address := fmt.Sprintf("%s:%d", host, port)
	startTime := time.Now()
	//time.Sleep(3 * time.Second)
	//return nil

	for i := 0; i < 100; i++ {
		if time.Now().Sub(startTime) > timeout {
			break
		}

		time.Sleep(time.Duration(100*i) * time.Millisecond)
		conn, err := grpc.Dial(address, grpc.WithInsecure())
		if err != nil {
			continue
		}
		defer conn.Close()

		client := protobuf.NewNoiseClient(conn)
		if stream, err := client.Stream(context.Background()); err != nil {
			continue
		} else {
			if err := stream.CloseSend(); err != nil {
				log.Debug(fmt.Sprintf("Close error: %+v", err))
			}
			conn.Close()
		}
		time.Sleep(3 * time.Second)
		log.Debug(fmt.Sprintf("Server ready after %d ms\n", time.Now().Sub(startTime).Nanoseconds()/1000000))
		return nil
	}
	return fmt.Errorf("Unable to connect locally after %f seconds", time.Now().Sub(startTime).Seconds())
}
