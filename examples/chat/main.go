package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/cipher"
	"github.com/perlin-network/noise/handshake"
	"github.com/perlin-network/noise/skademlia"
	"net"
	"os"
	"strconv"
	"strings"
)

func protocol(network *skademlia.Protocol) func() noise.Protocol {
	return func() noise.Protocol {
		var ephemeralSharedKey []byte
		var err error

		var id *skademlia.ID

		var ecdh, aead, skad, chat noise.Protocol

		ecdh = func(ctx noise.Context) (noise.Protocol, error) {
			if ephemeralSharedKey, err = handshake.NewECDH().Handshake(ctx); err != nil {
				return nil, err
			}

			return aead, nil
		}

		aead = func(ctx noise.Context) (noise.Protocol, error) {
			if err := cipher.NewAEAD(ephemeralSharedKey).Setup(ctx); err != nil {
				return nil, err
			}

			return skad, nil
		}

		skad = func(ctx noise.Context) (noise.Protocol, error) {
			if id, err = network.Handshake(ctx); err != nil {
				return nil, err
			}

			return chat, nil
		}

		chat = func(ctx noise.Context) (noise.Protocol, error) {
			var msg []byte

			for {
				select {
				case <-ctx.Done():
					return nil, nil
				case ctx := <-ctx.Peer().Recv(0x16):
					msg = ctx.Bytes()
				}

				fmt.Printf("%s> %s\n", id.Address(), msg)
			}
		}

		return ecdh
	}
}

func main() {
	flag.Parse()

	keys, err := skademlia.NewKeys(8, 8)
	if err != nil {
		panic(err)
	}

	// Hooking Noise onto a net.Listener.
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}

	network := skademlia.New(keys, net.JoinHostPort("127.0.0.1", strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)))
	network.WithC1(8)
	network.WithC2(8)

	node := noise.NewNode(listener)
	node.FollowProtocol(protocol(network))
	defer node.Shutdown()

	go func() {
		fmt.Println("Listening for connections on port:", listener.Addr().(*net.TCPAddr).Port)

		for {
			conn, err := listener.Accept()

			if err != nil {
				break
			}

			peer := node.Wrap(conn)
			go peer.Start()
		}
	}()

	if addresses := flag.Args(); len(addresses) > 0 {
		for _, address := range addresses {
			peer, err := node.Dial(address)

			if err != nil {
				panic(err)
			}

			peer.WaitFor(skademlia.SignalHandshakeComplete)
		}
	}

	if peers := network.Bootstrap(node); len(peers) > 0 {
		var ids []string

		for _, id := range peers {
			ids = append(ids, id.String())
		}

		fmt.Println("Bootstrapped to:", strings.Join(ids, ", "))
	}

	// Read input and broadcast out to peers.
	reader := bufio.NewReader(os.Stdin)

	for {
		line, err := reader.ReadBytes('\n')

		if err != nil {
			panic(err)
		}

		for _, peer := range network.Peers(node) {
			err = peer.Send(0x16, bytes.TrimSpace(line))

			if err != nil {
				panic(err)
			}
		}
	}
}
