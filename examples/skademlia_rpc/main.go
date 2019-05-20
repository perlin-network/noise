package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/cipher"
	"github.com/perlin-network/noise/handshake"
	"github.com/perlin-network/noise/skademlia"
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

type handler struct{}

func (handler) GetMessage(context.Context, *Message) (*Message, error) {
	atomic.AddUint64(&recvCount, 1)

	m := &Message{Contents: make([]byte, payloadSize)}

	if _, err := rand.Read(m.Contents); err != nil {
		return nil, err
	}

	return m, nil
}

const (
	C1 = 1
	C2 = 1
)

func main() {
	flag.Parse()

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}

	fmt.Println("Listening for peers on port:", listener.Addr().(*net.TCPAddr).Port)

	keys, err := skademlia.NewKeys(C1, C2)
	if err != nil {
		panic(err)
	}

	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(listener.Addr().(*net.TCPAddr).Port))

	client := skademlia.NewClient(addr, keys, skademlia.WithC1(C1), skademlia.WithC2(C2))
	client.SetCredentials(noise.NewCredentials(addr, handshake.NewECDH(), cipher.NewAEAD(), client.Protocol()))

	go func() {
		for range time.Tick(1 * time.Second) {
			fmt.Printf("Sent %d messages, and received %d messages.\n", atomic.SwapUint64(&sendCount, 0), atomic.SwapUint64(&recvCount, 0))
		}
	}()

	go func() {
		server := client.Listen()
		RegisterRouteMessageServer(server, &handler{})

		if err := server.Serve(listener); err != nil {
			panic(err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	for _, addr := range flag.Args() {
		conn, err := client.Dial(addr)
		if err != nil {
			panic(err)
		}

		go func() {
			rpc := NewRouteMessageClient(conn)

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

						res, err := rpc.GetMessage(context.Background(), msg)
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
		}()
	}

	fmt.Println(client.Bootstrap())

	select {}
}
