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
	"github.com/perlin-network/noise/xnoise"
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

	node, err := xnoise.ListenTCP(0)
	if err != nil {
		panic(err)
	}

	node.RegisterOpcode(OpcodeChat, node.NextAvailableOpcode())

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

	node.FollowProtocol(protocol(node, ecdh, aead, network))

	fmt.Println("Listening for connections on port:", node.Addr().(*net.TCPAddr).Port)

	defer node.Shutdown()

	if addresses := flag.Args(); len(addresses) > 0 {
		for _, address := range addresses {
			peer, err := xnoise.DialTCP(node, address)

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
