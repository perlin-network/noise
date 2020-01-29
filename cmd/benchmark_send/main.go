package main

import (
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
			fmt.Printf("Sent %d message(s), and received %d message(s) in the last second.\n",
				atomic.SwapUint32(&sent, 0),
				atomic.SwapUint32(&received, 0),
			)
		}
	}()

	a.Handle(func(ctx noise.HandlerContext) error {
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
		wg.Add(4 * runtime.NumCPU())

		for i := 0; i < 4*runtime.NumCPU(); i++ {
			go func() {
				check(b.Send(context.TODO(), a.Addr(), []byte("hello world!")))
				atomic.AddUint32(&sent, 1)
				wg.Done()
			}()
		}

		wg.Wait()
	}
}
