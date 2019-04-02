package main

import (
	"github.com/perlin-network/noise/xnoise"
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
	OpcodeBenchmark = "examples.chat"
)

func protocol(ecdh *handshake.ECDH, aead *cipher.AEAD, skad *skademlia.Protocol) func() noise.Protocol {
	return func() noise.Protocol {
		var ephemeralSharedKey []byte
		var err error

		var p1, p2, p3 noise.Protocol

		p1 = func(ctx noise.Context) (noise.Protocol, error) {
			if ephemeralSharedKey, err = ecdh.Handshake(ctx); err != nil {
				return nil, err
			}

			return p2, nil
		}

		p2 = func(ctx noise.Context) (noise.Protocol, error) {
			if err := aead.Setup(ephemeralSharedKey, ctx); err != nil {
				return nil, err
			}

			return p3, nil
		}

		p3 = func(ctx noise.Context) (noise.Protocol, error) {
			if _, err := skad.Handshake(ctx); err != nil {
				return nil, err
			}

			return nil, nil
		}

		return p1
	}
}

func launch() *noise.Node {
	node, err := xnoise.ListenTCP(0)
	if err != nil {
		panic(err)
	}

	node.RegisterOpcode(OpcodeBenchmark, node.NextAvailableOpcode())

	ecdh := handshake.NewECDH()
	ecdh.RegisterOpcodes(node)

	aead := cipher.NewAEAD()
	aead.RegisterOpcodes(node)

	keys, err := skademlia.NewKeys(net.JoinHostPort("127.0.0.1", strconv.Itoa(node.Addr().(*net.TCPAddr).Port)), 8, 8)
	if err != nil {
		panic(err)
	}

	network := skademlia.New(keys, xnoise.DialTCP)
	network.RegisterOpcodes(node)
	network.WithC1(8)
	network.WithC2(8)

	node.FollowProtocol(protocol(ecdh, aead, network))

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

	aliceToBob.WaitFor(skademlia.SignalHandshakeComplete)

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
			case <-bobToAlice.Recv(bob.Opcode(OpcodeBenchmark)):
				atomic.AddUint64(&recvCount, 1)
			}
		}
	}()

	// Sender.
	for {
		var buf [600]byte
		if _, err := rand.Read(buf[:]); err != nil {
			panic(err)
		}

		if err := aliceToBob.Send(alice.Opcode(OpcodeBenchmark), buf[:]); err != nil {
			panic(err)
		}

		atomic.AddUint64(&sendCount, 1)
	}
}
