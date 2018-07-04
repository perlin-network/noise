package main

import _ "net/http/pprof"

import (
	"flag"
	"fmt"
	"github.com/perlin-network/noise/crypto/signing/ed25519"
	"github.com/perlin-network/noise/examples/benchmark/messages"
	"github.com/perlin-network/noise/network/builders"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strconv"
	"time"
)

var profile = flag.String("profile", "", "write cpu profile to file")
var port = flag.Uint("port", 0, "port to listen on")
var receiver = "kcp://localhost:3001"

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

	if *profile != "" {
		f, err := os.Create(*profile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
	}

	builder := builders.NewNetworkBuilder()
	builder.SetAddress("kcp://localhost:" + strconv.Itoa(int(*port)))
	builder.SetKeys(ed25519.RandomKeyPair())

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
