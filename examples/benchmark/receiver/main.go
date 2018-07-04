package main

import _ "net/http/pprof"

import (
	"flag"
	"fmt"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/examples/benchmark/messages"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/builders"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"time"
)

type BenchmarkPlugin struct {
	*network.Plugin
	counter int
}

func (state *BenchmarkPlugin) Receive(ctx *network.MessageContext) error {
	switch ctx.Message().(type) {
	case *messages.BasicMessage:
		state.counter++
	}

	return nil
}

var profile = flag.String("profile", "", "write cpu profile to file")

func main() {
	cryptoProvider := crypto.NewEd25519()

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	flag.Parse()

	runtime.GOMAXPROCS(runtime.NumCPU())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		pprof.StopCPUProfile()
		os.Exit(0)
	}()

	if *profile != "" {
		f, err := os.Create(*profile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
	}

	builder := builders.NewNetworkBuilder()
	builder.SetAddress("kcp://localhost:3001")
	builder.SetKeys(crypto.RandomKeyPair(cryptoProvider))
	builder.SetCryptoProvider(cryptoProvider)

	state := new(BenchmarkPlugin)
	builder.AddPlugin(state)

	net, err := builder.Build()
	if err != nil {
		panic(err)
	}

	go net.Listen()

	fmt.Println("Waiting for sender on kcp://localhost:3001.")

	// Run loop every 1 second.
	for range time.Tick(1 * time.Second) {
		fmt.Printf("Got %d messages.\n", state.counter)

		state.counter = 0
	}
}
