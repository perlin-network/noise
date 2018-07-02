package main

import _ "net/http/pprof"

import (
	"github.com/perlin-network/noise/network/builders"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/examples/benchmark/messages"
	"flag"
	"fmt"
	"runtime"
	"os"
	"log"
	"runtime/pprof"
	"os/signal"
	"net/http"
	"time"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var receiver = "kcp://127.0.0.1:3001"

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:7070", nil))
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

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
	}

	builder := builders.NewNetworkBuilder()
	builder.SetAddress("kcp://localhost:3003")
	builder.SetKeys(crypto.RandomKeyPair())

	net, err := builder.Build()
	if err != nil {
		panic(err)
	}

	go net.Listen()
	net.Bootstrap(receiver)

	time.Sleep(500 * time.Millisecond)

	fmt.Println("Spamming messages...")

	msg := &messages.BasicMessage{}
	for {
		err := net.Tell(receiver, msg)
		if err != nil {
			panic(err)
		}
	}
}
