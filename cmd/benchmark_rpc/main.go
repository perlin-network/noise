package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/perlin-network/noise"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	a, err := noise.NewNode()
	check(err)

	sent, received := uint32(0), uint32(0)

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			fmt.Printf("Sent %d request(s), and received %d response(s) in the last second.\n",
				atomic.SwapUint32(&sent, 0),
				atomic.SwapUint32(&received, 0),
			)
		}
	}()

	a.Handle(func(ctx noise.HandlerContext) error {
		check(ctx.Send([]byte("hello from alice!")))
		atomic.AddUint32(&received, 1)
		return nil
	})

	b, err := noise.NewNode()
	check(err)

	check(a.Listen())
	check(b.Listen())

	defer a.Close()
	defer b.Close()

	for {
		var wg sync.WaitGroup
		wg.Add(runtime.NumCPU())

		for i := 0; i < runtime.NumCPU(); i++ {
			go func() {
				res, err := b.Request(context.TODO(), a.Addr(), []byte("hello from bob!"))
				check(err)
				if !bytes.Equal(res, []byte("hello from alice!")) {
					check(fmt.Errorf("got unexpected response '%s'", string(res)))
				}
				atomic.AddUint32(&sent, 1)
				wg.Done()
			}()
		}

		wg.Wait()
	}
}
