// Copyright (c) 2019 Perlin
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

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
	"sync/atomic"
	"time"
)

const payloadSize = 600

var sendCount, recvCount uint64

type handler struct{}

func (handler) GetMessage(stream RouteMessage_GetMessageServer) error {
	for {
		msg, err := stream.Recv()

		if err != nil {
			return err
		}

		if len(msg.Contents) != payloadSize {
			panic("something went wrong")
		}

		atomic.AddUint64(&recvCount, 1)
	}
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
			fmt.Printf("Error dialing %s: %v\n", addr, err)
			continue
		}

		go func() {
			rpc := NewRouteMessageClient(conn)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			client, err := rpc.GetMessage(ctx)
			if err != nil {
				panic(err)
			}

			for {
				msg := &Message{Contents: make([]byte, payloadSize)}
				if _, err := rand.Read(msg.Contents); err != nil {
					panic(err)
				}

				if err := client.Send(msg); err != nil {
					return
				}

				atomic.AddUint64(&sendCount, 1)
			}
		}()
	}

	fmt.Println(client.Bootstrap())

	select {}
}
