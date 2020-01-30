package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/kademlia"
	"github.com/spf13/pflag"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"
)

type chatMessage struct {
	contents string
}

func (m chatMessage) Marshal() []byte {
	return []byte(m.contents)
}

func unmarshalChatMessage(buf []byte) (chatMessage, error) {
	return chatMessage{contents: strings.ToValidUTF8(string(buf), "")}, nil
}

var (
	hostFlag    = pflag.IPP("host", "h", nil, "binding host")
	portFlag    = pflag.Uint16P("port", "p", 0, "binding port")
	addressFlag = pflag.StringP("address", "a", "", "publicly reachable network address")
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

const printedLength = 8

func main() {
	pflag.Parse()

	node, err := noise.NewNode(
		noise.WithNodeBindHost(*hostFlag),
		noise.WithNodeBindPort(*portFlag),
		noise.WithNodeAddress(*addressFlag),
	)
	check(err)

	defer node.Close()

	node.RegisterMessage(chatMessage{}, unmarshalChatMessage)

	node.Handle(func(ctx noise.HandlerContext) error {
		if ctx.IsRequest() {
			return nil
		}

		obj, err := ctx.DecodeMessage()
		if err != nil {
			return nil
		}

		msg, ok := obj.(chatMessage)
		if !ok {
			return nil
		}

		fmt.Printf("%s(%s)> %s\n", ctx.ID().Address, ctx.ID().ID.String()[:printedLength], msg.contents)

		return nil
	})

	overlay := kademlia.NewProtocol()
	node.Bind(overlay)

	check(node.Listen())

	fmt.Printf("Your ID is %s(%s). Type '/discover' to attempt to discover new peers, or '/peers' to list"+
		" out all peers you are connected to.\n", node.ID().Address, node.ID().ID.String()[:printedLength])

	peers := make([]*noise.Client, 0, pflag.NArg())

	for _, peerAddr := range pflag.Args() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		peer, err := node.Ping(ctx, peerAddr)
		cancel()

		if err != nil {
			fmt.Printf("Failed to ping bootstrap node (%s). Skipping... [error: %s]\n", peerAddr, err)
			continue
		}

		peers = append(peers, peer)
	}

	ids := overlay.Discover()

	var str []string
	for _, id := range ids {
		str = append(str, fmt.Sprintf("%s(%s)", id.Address, id.ID.String()[:printedLength]))
	}

	if len(ids) > 0 {
		fmt.Printf("Discovered %d peer(s): [%v]\n", len(ids), strings.Join(str, ", "))
	} else {
		fmt.Printf("Did not discover any peers.\n")
	}

	go func() {
		r := bufio.NewReader(os.Stdin)

		for {
			buf, _, err := r.ReadLine()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}

				check(err)
			}

			line := string(buf)

			if len(line) == 0 {
				continue
			}

			switch line {
			case "/discover":
				ids := overlay.Discover()

				var str []string
				for _, id := range ids {
					str = append(str, fmt.Sprintf("%s(%s)", id.Address, id.ID.String()[:printedLength]))
				}

				if len(ids) > 0 {
					fmt.Printf("Discovered %d peer(s): [%v]\n", len(ids), strings.Join(str, ", "))
				} else {
					fmt.Printf("Did not discover any peers.\n")
				}

				continue
			case "/peers":
				ids := overlay.Table().Peers()

				var str []string
				for _, id := range ids {
					str = append(str, fmt.Sprintf("%s(%s)", id.Address, id.ID.String()[:printedLength]))
				}

				fmt.Printf("You know %d peer(s): [%v]\n", len(ids), strings.Join(str, ", "))

				continue
			default:
			}

			if strings.HasPrefix(line, "/") {
				fmt.Printf("Your ID is %s(%s). Type '/discover' to attempt to discover new "+
					"peers, or '/peers' to list out all peers you are connected to.\n",
					node.ID().Address,
					node.ID().ID.String()[:printedLength],
				)

				continue
			}

			for _, id := range overlay.Table().Peers() {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				err := node.SendMessage(ctx, id.Address, chatMessage{contents: line})
				cancel()

				if err != nil {
					fmt.Printf("Failed to send message to %s(%s). Skipping... [error: %s]\n",
						id.Address,
						id.ID.String()[:printedLength],
						err,
					)
					continue
				}
			}
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	check(os.Stdin.Close())
	println()
}
