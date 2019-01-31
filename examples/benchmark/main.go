package main

import (
	"crypto/rand"
	"fmt"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/callbacks"
	"github.com/perlin-network/noise/cipher/aead"
	"github.com/perlin-network/noise/handshake/ecdh"
	"github.com/perlin-network/noise/identity/ed25519"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/payload"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/skademlia"
	"github.com/pkg/errors"
	"strconv"
	"sync/atomic"
	"time"
)

var (
	opcodeBenchmark noise.Opcode  = noise.RegisterMessage(noise.NextAvailableOpcode(), (*benchmarkMessage)(nil))
	_               noise.Message = (*benchmarkMessage)(nil)

	messagesSentPerSecond     uint64
	messagesReceivedPerSecond uint64
)

type benchmarkMessage struct {
	text string
}

func (benchmarkMessage) Read(reader payload.Reader) (noise.Message, error) {
	text, err := reader.ReadString()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read msg")
	}

	return benchmarkMessage{text: text}, nil
}

func (m benchmarkMessage) Write() []byte {
	return payload.NewWriter(nil).WriteString(m.text).Bytes()
}

func spawnNode(port uint16, server bool) *noise.Node {
	params := noise.DefaultParams()
	params.ID = ed25519.Random()

	node, err := noise.NewNode(params)
	if err != nil {
		panic(err)
	}

	protocol.EnforceHandshakePolicy(node, ecdh.New())
	protocol.EnforceCipherPolicy(node, aead.New())

	protocol.EnforceIdentityPolicy(node, skademlia.NewIdentityPolicy())
	protocol.EnforceNetworkPolicy(node, skademlia.NewNetworkPolicy())

	if server {
		protocol.OnEachSessionEstablished(node, func(node *noise.Node, peer *noise.Peer) error {
			peer.OnMessageReceived(opcodeBenchmark, func(node *noise.Node, opcode noise.Opcode, peer *noise.Peer, message noise.Message) error {
				atomic.AddUint64(&messagesReceivedPerSecond, 1)
				return nil
			})

			return nil
		})
	} else {
		protocol.OnEachSessionEstablished(node, func(node *noise.Node, peer *noise.Peer) error {
			peer.BeforeMessageSent(func(node *noise.Node, peer *noise.Peer, msg []byte) (bytes []byte, e error) {
				atomic.AddUint64(&messagesSentPerSecond, 1)
				return msg, nil
			})

			return nil
		})
	}

	go node.Listen()

	log.Info().Msgf("Listening for peers on port %d.", node.Port())

	return node

}

func main() {
	server, client := spawnNode(0, true), spawnNode(0, false)

	_, err := client.Dial("127.0.0.1:" + strconv.Itoa(int(server.Port())))
	if err != nil {
		panic(err)
	}

	go func() {
		for range time.Tick(3 * time.Second) {
			sent, received := atomic.SwapUint64(&messagesSentPerSecond, 0), atomic.SwapUint64(&messagesReceivedPerSecond, 0)

			fmt.Printf("Sent %d, and received %d messages per second.\n", sent, received)
		}
	}()

	protocol.OnEachSessionEstablished(server, func(node *noise.Node, peer *noise.Peer) error {
		go func() {
			for {
				payload := make([]byte, 600)
				rand.Read(payload)

				peer.SendMessage(opcodeBenchmark, benchmarkMessage{string(payload)})
			}
		}()

		return callbacks.DeregisterCallback
	})

	select {}
}
