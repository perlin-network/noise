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

const (
	OpcodeChat = "examples.chat"
)

func protocol(node *noise.Node, ecdh *handshake.ECDH, aead *cipher.AEAD, skad *skademlia.Protocol) func() noise.Protocol {
	return func() noise.Protocol {
		var ephemeralSharedKey []byte
		var err error

		var id *skademlia.ID

		var p1, p2, p3, p4 noise.Protocol

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
			if id, err = skad.Handshake(ctx); err != nil {
				return nil, err
			}

			return p4, nil
		}

		p4 = func(ctx noise.Context) (noise.Protocol, error) {
			var msg []byte

			for {
				select {
				case <-ctx.Done():
					return nil, nil
				case ctx := <-ctx.Peer().Recv(node.Opcode(OpcodeChat)):
					msg = ctx.Bytes()
				}

				fmt.Printf("%s> %s\n", id.Address(), msg)
			}
		}

		return p1
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

	node := noise.NewNode(listener)
	node.RegisterOpcode(OpcodeChat, node.NextAvailableOpcode())

	ecdh := handshake.NewECDH()
	ecdh.RegisterOpcodes(node)

	aead := cipher.NewAEAD()
	aead.RegisterOpcodes(node)

	network := skademlia.New(keys, net.JoinHostPort("127.0.0.1", strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)))
	network.RegisterOpcodes(node)
	network.WithC1(8)
	network.WithC2(8)

	node.FollowProtocol(protocol(node, ecdh, aead, network))

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
			err = peer.Send(node.Opcode(OpcodeChat), bytes.TrimSpace(line))

			if err != nil {
				panic(err)
			}
		}
	}
}
