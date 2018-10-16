package main

import (
	"context"
	"flag"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strconv"
	"time"

	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/examples/local_benchmark/messages"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/types/opcode"
)

var profile = flag.String("profile", "", "write cpu profile to file")
var port = flag.Uint("port", 3002, "port to listen on")
var receiver = "tcp://localhost:3001"

func main() {
	flag.Set("logtostderr", "true")

	go func() {
		log.Info().Err(http.ListenAndServe("localhost:7070", nil))
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
	builder.SetAddress("tcp://localhost:" + strconv.Itoa(int(*port)))
	builder.SetKeys(ed25519.RandomKeyPair())

	net, err := builder.Build()
	if err != nil {
		panic(err)
	}

	go net.Listen()
	net.Bootstrap(receiver)

	time.Sleep(500 * time.Millisecond)

	log.Info().Msgf("Spamming messages to %s...", receiver)

	client, err := net.Client(receiver)
	if err != nil {
		panic(err)
	}

	for {
		err = client.Tell(context.Background(), &messages.BasicMessage{})
		if err != nil {
			panic(err)
		}
	}
}
