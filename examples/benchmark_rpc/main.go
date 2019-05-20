package main

import (
	"fmt"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/cipher"
	"github.com/perlin-network/noise/handshake"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const payloadSize = 600
const concurrency = 4

var sendCount, recvCount uint64

type server struct{}

func (server) GetMessage(context.Context, *Message) (*Message, error) {
	atomic.AddUint64(&recvCount, 1)

	m := &Message{Contents: make([]byte, payloadSize)}

	if _, err := rand.Read(m.Contents); err != nil {
		return nil, err
	}

	return m, nil
}

func main() {
	protocol := noise.NewCredentials("127.0.0.1", handshake.NewECDH(), cipher.NewAEAD())

	go func() {
		for range time.Tick(1 * time.Second) {
			fmt.Printf("Sent %d messages, and received %d messages.\n", atomic.SwapUint64(&sendCount, 0), atomic.SwapUint64(&recvCount, 0))
		}
	}()

	go func() {
		l, err := net.Listen("tcp", ":3000")
		if err != nil {
			panic(err)
		}

		s := grpc.NewServer(grpc.Creds(protocol))
		RegisterRouteMessageServer(s, &server{})

		if err := s.Serve(l); err != nil {
			panic(err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	conn, err := grpc.Dial(net.JoinHostPort("127.0.0.1", strconv.Itoa(3000)), grpc.WithTransportCredentials(protocol))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	client := NewRouteMessageClient(conn)

	for {
		var wg sync.WaitGroup
		wg.Add(concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				defer wg.Done()

				msg := &Message{Contents: make([]byte, payloadSize)}
				if _, err := rand.Read(msg.Contents); err != nil {
					panic(err)
				}

				res, err := client.GetMessage(context.Background(), msg)
				if err != nil {
					panic(err)
				}

				if len(res.Contents) != payloadSize {
					panic("something wrong happened")
				}

				atomic.AddUint64(&sendCount, 1)
			}()
		}

		wg.Wait()
	}
}
