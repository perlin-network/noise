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
	"github.com/perlin-network/noise/rpc"
	"github.com/perlin-network/noise/skademlia"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
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
func registerLogCallbacks(node *noise.Node) {
	node.OnPeerInit(func(node *noise.Node, peer *noise.Peer) error {
		peer.OnConnError(func(node *noise.Node, peer *noise.Peer, err error) error {
			log.Info().Msgf("Got an error: %v", err)

			return nil
		})

		peer.OnDisconnect(func(node *noise.Node, peer *noise.Peer) error {
			log.Info().Msgf("Peer %v has disconnected.", peer.RemoteIP().String()+":"+strconv.Itoa(int(peer.RemotePort())))

			return nil
		})

		return nil
	})
}

func registerMessageCallbacks(node *noise.Node) {
	opcodeChat = noise.RegisterMessage(noise.NextAvailableOpcode(), (*chatMessage)(nil))

	node.OnMessageReceived(opcodeChat, func(node *noise.Node, opcode noise.Opcode, peer *noise.Peer, message noise.Message) error {
		peer.OnMessageReceived(opcodeChat, func(node *noise.Node, opcode noise.Opcode, peer *noise.Peer, message noise.Message) error {
			log.Info().Msgf("[%s]: %s", protocol.PeerID(peer).String(), message.(chatMessage).text)

			return nil
		})

		return nil
	})
}

func main() {
	portFlag := flag.Uint("p", 3000, "port to listen for peers on")
	flag.Parse()

	log.Level(zerolog.InfoLevel)

	params := noise.DefaultParams()
	params.ID = ed25519.Random()
	params.Port = uint16(*portFlag)

	node, err := noise.NewNode(params)
	if err != nil {
		panic(err)
	}

	rpc.Register(node)

	protocol.EnforceHandshakePolicy(node, ecdh.New())
	protocol.EnforceCipherPolicy(node, aead.New())

	protocol.EnforceIdentityPolicy(node, skademlia.NewIdentityPolicy())
	protocol.EnforceNetworkPolicy(node, skademlia.NewNetworkPolicy())

	registerLogCallbacks(node)
	registerMessageCallbacks(node)

	log.Info().Msgf("Listening for peers on port %d.", node.Port())

	go node.Listen()

	if len(flag.Args()) > 0 {
		for _, address := range flag.Args() {
			peer, err := node.Dial(address)
			if err != nil {
				panic(err)
			}

			protocol.BlockUntilAuthenticated(peer)
		}

		peers := skademlia.FindNode(node, protocol.NodeID(node).(skademlia.ID), skademlia.DefaultBucketSize, 8)
		log.Info().Msgf("Bootstrapped with peers: %+v", peers)
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		text, err := reader.ReadString('\n')

		if err != nil {
			panic(err)
		}

		err = skademlia.Broadcast(node, opcodeChat, chatMessage{text: strings.TrimSpace(text)})

		if err != nil {
			panic(err)
		}
	}
}
