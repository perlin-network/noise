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
	C1         = 8
	C2         = 8
)

func protocol(node *noise.Node) (*skademlia.Protocol, noise.Protocol) {
	ecdh := handshake.NewECDH()
	ecdh.Logger().SetOutput(os.Stdout)
	ecdh.RegisterOpcodes(node)

	aead := cipher.NewAEAD()
	aead.Logger().SetOutput(os.Stdout)
	aead.RegisterOpcodes(node)

	keys, err := skademlia.NewKeys(C1, C2)
	if err != nil {
		panic(err)
	}

	overlay := skademlia.New(net.JoinHostPort("127.0.0.1", strconv.Itoa(node.Addr().(*net.TCPAddr).Port)), keys, xnoise.DialTCP)
	overlay.Logger().SetOutput(os.Stdout)
	overlay.RegisterOpcodes(node)
	overlay.WithC1(C1)
	overlay.WithC2(C2)

	node.RegisterOpcode(OpcodeChat, node.NextAvailableOpcode())

	chatProtocol := func(ctx noise.Context) error {
		id := ctx.Get(skademlia.KeyID).(*skademlia.ID)

		for {
			select {
			case <-ctx.Done():
				return nil
			case ctx := <-ctx.Peer().Recv(node.Opcode(OpcodeChat)):
				fmt.Printf("%s> %s\n", id.Address(), ctx.Bytes())
			}
		}
	}

	return overlay, noise.NewProtocol(xnoise.LogErrors, ecdh.Protocol(), aead.Protocol(), overlay.Protocol(), chatProtocol)
}

func main() {
	flag.Parse()

	node, err := xnoise.ListenTCP(0)
	if err != nil {
		panic(err)
	}

	network, protocol := protocol(node)
	node.FollowProtocol(protocol)

	fmt.Println("Listening for connections on port:", node.Addr().(*net.TCPAddr).Port)

	defer node.Shutdown()

	if addresses := flag.Args(); len(addresses) > 0 {
		for _, address := range addresses {
			peer, err := xnoise.DialTCP(node, address)

			if err != nil {
				panic(err)
			}

			peer.WaitFor(skademlia.SignalAuthenticated)
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
