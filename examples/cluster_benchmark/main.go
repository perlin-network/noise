package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/pprof"
	"strings"
	"sync/atomic"
	"time"

	"github.com/perlin-network/noise/crypto/ed25519"
	"github.com/perlin-network/noise/examples/cluster_benchmark/messages"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/backoff"
	"github.com/perlin-network/noise/network/discovery"
	"github.com/rs/zerolog/log"
)

const MESSAGE_THRESHOLD uint64 = 2000

var numPeers int64
var numMessages uint64

type BenchPlugin struct{ network.Plugin }

func (state *BenchPlugin) PeerConnect(client *network.PeerClient) {
	atomic.AddInt64(&numPeers, 1)
}

func (state *BenchPlugin) PeerDisconnect(client *network.PeerClient) {
	atomic.AddInt64(&numPeers, -1)
}

func (state *BenchPlugin) Receive(ctx *network.PluginContext) error {
	atomic.AddUint64(&numMessages, 1)
	sendBroadcast(ctx.Network())
	return nil
}

func sendBroadcast(n *network.Network) {
	if atomic.LoadUint64(&numMessages) > MESSAGE_THRESHOLD {
		return
	}

	targetNumPeers := atomic.LoadInt64(&numPeers)/2 + 1
	n.BroadcastRandomly(&messages.Empty{}, int(targetNumPeers))
}

func setupPPROF(port int) {
	// Usage:
	// terminal_1$ vgo build && ./cluster_benchmark -port 3000
	// terminal_2$ ./cluster_benchmark -port 3001 -peers tcp://localhost:3000
	// terminal_3:
	//  go tool pprof cluster_benchmark http://127.0.0.1:3500/debug/pprof/profile
	//  go tool pprof cluster_benchmark http://127.0.0.1:3500/debug/pprof/heap
	//  go tool pprof cluster_benchmark http://127.0.0.1:3500/debug/pprof/goroutine
	//  go tool pprof cluster_benchmark http://127.0.0.1:3500/debug/pprof/block

	r := http.NewServeMux()

	// Register pprof handlers
	r.HandleFunc("/debug/pprof/", pprof.Index)
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/debug/pprof/trace", pprof.Trace)
	r.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	r.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	r.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	r.Handle("/debug/pprof/block", pprof.Handler("block"))

	log.Info().Msgf("Pprof listening on port %d.\n", port+500)
	http.ListenAndServe(fmt.Sprintf(":%d", port+500), r)
}

func main() {
	// process flags
	portFlag := flag.Int("port", 3000, "port to listen to")
	hostFlag := flag.String("host", "localhost", "host to listen to")
	protocolFlag := flag.String("protocol", "tcp", "protocol to use (kcp/tcp)")
	peersFlag := flag.String("peers", "", "peers to connect to")
	flag.Parse()

	port := uint16(*portFlag)
	host := *hostFlag
	protocol := *protocolFlag
	peers := strings.Split(*peersFlag, ",")

	keys := ed25519.RandomKeyPair()

	go setupPPROF(*portFlag)

	log.Info().Msgf("Private Key: %s", keys.PrivateKeyHex())
	log.Info().Msgf("Public Key: %s", keys.PublicKeyHex())

	builder := network.NewBuilder()
	builder.SetKeys(keys)
	builder.SetAddress(network.FormatAddress(protocol, host, port))

	// Register peer discovery plugin.
	builder.AddPlugin(new(discovery.Plugin))

	// Add backoff plugin.
	builder.AddPlugin(new(backoff.Plugin))

	// Add benchmark plugin.
	builder.AddPlugin(new(BenchPlugin))

	net, err := builder.Build()
	if err != nil {
		log.Fatal().Err(err)
		return
	}

	go net.Listen()

	net.BlockUntilListening()

	if len(peers) > 0 {
		net.Bootstrap(peers...)
	}

	go func() {
		for range time.Tick(1 * time.Second) {
			currentNumMessages := atomic.SwapUint64(&numMessages, 0)
			log.Info().Msgf("Got %d messages, %d peers", currentNumMessages, atomic.LoadInt64(&numPeers))
		}
	}()

	for range time.Tick(300 * time.Millisecond) {
		sendBroadcast(net)
	}
}
