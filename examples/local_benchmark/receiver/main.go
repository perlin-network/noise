package main

import (
	"flag"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/examples/local_benchmark/messages"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/types/opcode"
)

type BenchmarkPlugin struct {
	*network.Plugin
	counter int
}

func (state *BenchmarkPlugin) Receive(ctx *network.PluginContext) error {
	switch ctx.Message().(type) {
	case *messages.BasicMessage:
		state.counter++
	}

	return nil
}

var profile = flag.String("profile", "", "write cpu profile to file")

func main() {
	flag.Set("logtostderr", "true")

	go func() {
		log.Info().Err(http.ListenAndServe("localhost:6060", nil))
	}()

	flag.Parse()

	runtime.GOMAXPROCS(runtime.NumCPU())
	opcode.RegisterMessageType(opcode.Opcode(1000), &messages.BasicMessage{})

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
			log.Fatal().Err(err).Msg("")
		}
		pprof.StartCPUProfile(f)
	}

	builder := network.NewBuilder()
	builder.SetAddress("tcp://localhost:3001")
	builder.SetKeys(ed25519.RandomKeyPair())

	state := new(BenchmarkPlugin)
	builder.AddPlugin(state)

	net, err := builder.Build()
	if err != nil {
		panic(err)
	}

	go net.Listen()

	log.Info().Msg("waiting for sender on tcp://localhost:3001.")

	// Run loop every 1 second.
	for range time.Tick(1 * time.Second) {
		log.Info().Int("num_messages", state.counter).Msg("received messages")

		state.counter = 0
	}
}
