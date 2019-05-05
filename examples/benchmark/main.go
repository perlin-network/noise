package main

import (
	"github.com/perlin-network/noise/xnoise"
	"github.com/pkg/errors"
	"net/http"
	_ "net/http/pprof"
)

import (
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

const (
	OpcodeBenchmark = "examples.benchmark"
	C1              = 8
	C2              = 8
)

func protocol(node *noise.Node) noise.Protocol {
	ecdh := handshake.NewECDH()
	ecdh.RegisterOpcodes(node)

	aead := cipher.NewAEAD()
	aead.RegisterOpcodes(node)

	keys, err := skademlia.NewKeys(C1, C2)
	if err != nil {
		panic(err)
	}

	overlay := skademlia.New(net.JoinHostPort("127.0.0.1", strconv.Itoa(node.Addr().(*net.TCPAddr).Port)), keys, xnoise.DialTCP)
	overlay.RegisterOpcodes(node)
	overlay.WithC1(C1)
	overlay.WithC2(C2)

	node.RegisterOpcode(OpcodeBenchmark, node.NextAvailableOpcode())

	return noise.NewProtocol(ecdh.Protocol(), aead.Protocol(), overlay.Protocol())
}

func launch() *noise.Node {
	node, err := xnoise.ListenTCP(0)
	if err != nil {
		panic(err)
	}

	node.RegisterOpcode(OpcodeBenchmark, node.NextAvailableOpcode())
	node.FollowProtocol(protocol(node))

	fmt.Println("Listening for connections on port:", node.Addr().(*net.TCPAddr).Port)

	return node
}

func main() {
	go func() {
		panic(http.ListenAndServe("localhost:6060", nil))
	}()

	alice := launch()
	defer alice.Shutdown()

	bob := launch()
	defer bob.Shutdown()

	var sendCount uint64
	var recvCount uint64

	aliceToBob, err := xnoise.DialTCP(alice, bob.Addr().String())

	if err != nil {
		panic(err)
	}

	aliceToBob.WaitFor(skademlia.SignalAuthenticated)

	bobOpcode := bob.Opcode(OpcodeBenchmark)
	aliceOpcode := alice.Opcode(OpcodeBenchmark)

	// Notifier.
	go func() {
		for range time.Tick(1 * time.Second) {
			fmt.Printf("Sent %d messages, and received %d messages.\n", atomic.SwapUint64(&sendCount, 0), atomic.SwapUint64(&recvCount, 0))
		}
	}()

	// Receiver.
	go func() {
		bobToAlice := bob.Peers()[0]

		for {
			select {
			case <-bobToAlice.Ctx().Done():
				return
			case <-bobToAlice.Recv(bobOpcode):
				atomic.AddUint64(&recvCount, 1)
			}
		}
	}()

	var buf [600]byte

	// Sender.
	for {
		if _, err := rand.Read(buf[:]); err != nil {
			panic(err)
		}

		if err := aliceToBob.Send(aliceOpcode, buf[:]); err != nil {
			if errors.Cause(err) == noise.ErrSendQueueFull {
				continue
			}

			panic(err)
		}

		atomic.AddUint64(&sendCount, 1)
	}
}
