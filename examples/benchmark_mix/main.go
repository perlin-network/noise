package main

import (
	"fmt"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/cipher"
	"github.com/perlin-network/noise/handshake"
	"github.com/perlin-network/noise/skademlia"
	"github.com/perlin-network/noise/xnoise"
	"math/rand"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	C1 = 8
	C2 = 8
)

func protocol(node *noise.Node) noise.Protocol {
	ecdh := handshake.NewECDH()
	ecdh.RegisterOpcodes(node)
	ecdh.Logger().SetOutput(os.Stdout)

	aead := cipher.NewAEAD()
	aead.RegisterOpcodes(node)
	aead.Logger().SetOutput(os.Stdout)

	keys, err := skademlia.NewKeys(C1, C2)
	if err != nil {
		panic(err)
	}

	overlay := skademlia.New(net.JoinHostPort("127.0.0.1", strconv.Itoa(node.Addr().(*net.TCPAddr).Port)), keys, xnoise.DialTCP)
	overlay.RegisterOpcodes(node)
	overlay.WithC1(C1)
	overlay.WithC2(C2)
	overlay.Logger().SetOutput(os.Stdout)

	return noise.NewProtocol(xnoise.LogErrors, ecdh.Protocol(), aead.Protocol(), overlay.Protocol())
}

func launch() *noise.Node {
	node, err := xnoise.ListenTCP(0)
	if err != nil {
		panic(err)
	}

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

	var sendSendCount uint64
	var recvSendCount uint64

	var sendRPCCount uint64
	var recvRPCCount uint64

	aliceToBob, err := xnoise.DialTCP(alice, bob.Addr().String())

	if err != nil {
		panic(err)
	}

	aliceToBob.WaitFor(skademlia.SignalAuthenticated)

	opcodeBenchmarkSend := bob.Handle(bob.NextAvailableOpcode(), func(ctx noise.Context, buf []byte) ([]byte, error) {
		atomic.AddUint64(&recvSendCount, 1)
		return nil, nil
	})

	opcodeBenchmarkRPC := bob.Handle(bob.NextAvailableOpcode(), func(ctx noise.Context, buf []byte) ([]byte, error) {
		atomic.AddUint64(&recvRPCCount, 1)
		return nil, nil
	})

	alice.Handle(opcodeBenchmarkSend, nil)
	alice.Handle(opcodeBenchmarkRPC, nil)

	// Notifier.
	go func() {
		for range time.Tick(1 * time.Second) {
			fmt.Printf("Sent %d messages and %d requests, and received %d messages and %d responses.\n", atomic.SwapUint64(&sendSendCount, 0), atomic.SwapUint64(&sendRPCCount, 0), atomic.SwapUint64(&recvSendCount, 0), atomic.SwapUint64(&recvRPCCount, 1))
		}
	}()

	var buf [600]byte

	// Sender.
	for {
		if _, err := rand.Read(buf[:]); err != nil {
			panic(err)
		}

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()

			if err := aliceToBob.Send(opcodeBenchmarkSend, buf[:]); err != nil {
				panic(err)
			}

			atomic.AddUint64(&sendSendCount, 1)
		}()

		go func() {
			defer wg.Done()

			if _, err := aliceToBob.Request(opcodeBenchmarkRPC, buf[:]); err != nil {
				panic(err)
			}

			atomic.AddUint64(&sendRPCCount, 1)
		}()

		wg.Wait()
	}
}
