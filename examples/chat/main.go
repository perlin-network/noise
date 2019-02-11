package main

import (
	"bufio"
	"flag"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/cipher/aead"
	"github.com/perlin-network/noise/handshake/ecdh"
	"github.com/perlin-network/noise/identity/ed25519"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/payload"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/skademlia"
	"github.com/pkg/errors"
	"os"
	"strconv"
	"strings"
)

/** DEFINE MESSAGES **/
var (
	opcodeChat noise.Opcode
	_          noise.Message = (*chatMessage)(nil)
)

type chatMessage struct {
	text string
}

func (chatMessage) Read(reader payload.Reader) (noise.Message, error) {
	text, err := reader.ReadString()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read chat msg")
	}

	return chatMessage{text: text}, nil
}

func (m chatMessage) Write() []byte {
	return payload.NewWriter(nil).WriteString(m.text).Bytes()
}

/** ENTRY POINT **/
func setup(node *noise.Node) {
	opcodeChat = noise.RegisterMessage(noise.NextAvailableOpcode(), (*chatMessage)(nil))

	node.OnPeerInit(func(node *noise.Node, peer *noise.Peer) error {
		peer.OnConnError(func(node *noise.Node, peer *noise.Peer, err error) error {
			log.Info().Msgf("Got an error: %v", err)

			return nil
		})

		peer.OnDisconnect(func(node *noise.Node, peer *noise.Peer) error {
			log.Info().Msgf("Peer %v has disconnected.", peer.RemoteIP().String()+":"+strconv.Itoa(int(peer.RemotePort())))

			return nil
		})

		go func() {
			for {
				msg := <-peer.Receive(opcodeChat)
				log.Info().Msgf("[%s]: %s", protocol.PeerID(peer), msg.(chatMessage).text)
			}
		}()

		return nil
	})
}

func main() {
	portFlag := flag.Uint("p", 3000, "port to listen for peers on")
	flag.Parse()

	params := noise.DefaultParams()
	params.ID = ed25519.Random()
	params.Port = uint16(*portFlag)

	node, err := noise.NewNode(params)
	if err != nil {
		panic(err)
	}

	p := protocol.New()
	p.Register(ecdh.New())
	p.Register(aead.New())
	p.Register(skademlia.New())
	p.Enforce(node)

	noise.DebugOpcodes()

	setup(node)
	go node.Listen()

	log.Info().Msgf("Listening for peers on port %d.", node.Port())

	if len(flag.Args()) > 0 {
		for _, address := range flag.Args() {
			peer, err := node.Dial(address)
			if err != nil {
				panic(err)
			}

			skademlia.WaitUntilAuthenticated(peer)
		}

		peers := skademlia.FindNode(node, protocol.NodeID(node).(skademlia.ID), skademlia.BucketSize(), 8)
		log.Info().Msgf("Bootstrapped with peers: %+v", peers)
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		txt, err := reader.ReadString('\n')

		if err != nil {
			panic(err)
		}

		err = skademlia.Broadcast(node, opcodeChat, chatMessage{text: strings.TrimSpace(txt)})
		if err != nil {
			panic(err)
		}
	}
}
