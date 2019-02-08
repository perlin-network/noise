package basic_test

import (
	"fmt"
	"github.com/perlin-network/noise"
	bbasic "github.com/perlin-network/noise/basic"
	cbasic "github.com/perlin-network/noise/cipher/basic"
	hbasic "github.com/perlin-network/noise/handshake/basic"
	"github.com/perlin-network/noise/identity/ed25519"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/payload"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/rpc"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"strconv"
	"strings"
	"testing"
	"time"
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
			log.Info().Msgf("Got an error: %+v", err)

			return nil
		})

		peer.OnDisconnect(func(node *noise.Node, peer *noise.Peer) error {
			log.Info().Msgf("Peer %v has disconnected.", peer.RemoteIP().String()+":"+strconv.Itoa(int(peer.RemotePort())))

			return nil
		})

		return nil
	})
}

func registerMessageCallbacks(node *noise.Node, opcodeChat noise.Opcode) {
	protocol.OnEachSessionEstablished(node, func(node *noise.Node, peer *noise.Peer) error {
		peer.OnMessageReceived(opcodeChat, func(node *noise.Node, opcode noise.Opcode, peer *noise.Peer, message noise.Message) error {
			log.Info().Msgf("[%s]: %s", protocol.PeerID(peer).String(), message.(chatMessage).text)

			return nil
		})

		return nil
	})
}

func makeNode(port int, opcodeChat noise.Opcode) (*noise.Node, error) {
	params := noise.DefaultParams()
	params.ID = ed25519.Random()
	params.Port = uint16(port)

	node, err := noise.NewNode(params)
	if err != nil {
		return nil, err
	}

	rpc.Register(node)

	protocol.EnforceHandshakePolicy(node, hbasic.New())
	protocol.EnforceCipherPolicy(node, cbasic.New())

	protocol.EnforceIdentityPolicy(node, bbasic.NewIdentityPolicy())
	protocol.EnforceNetworkPolicy(node, bbasic.NewNetworkPolicy())

	registerLogCallbacks(node)
	registerMessageCallbacks(node, opcodeChat)

	log.Info().Msgf("Listening for peers on port %d.", node.Port())

	go node.Listen()

	return node, nil
}

func Run(numNodes int, numMessages int) error {
	startPort := 6000
	opcodeChat := noise.RegisterMessage(noise.NextAvailableOpcode(), (*chatMessage)(nil))
	var nodes []*noise.Node
	for i := 0; i < numNodes; i++ {
		n, err := makeNode(startPort+i, opcodeChat)
		if err != nil {
			return err
		}
		nodes = append(nodes, n)
	}

	noise.DebugOpcodes()

	address := fmt.Sprintf("0.0.0.0:%d", startPort)

	time.Sleep(time.Duration(numNodes*25) * time.Millisecond)

	for i := 1; i < numNodes; i++ {
		peer, err := nodes[i].Dial(address)
		if err != nil {
			return err
		}

		protocol.BlockUntilAuthenticated(peer)
	}

	time.Sleep(time.Duration(numNodes*25) * time.Millisecond)

	for i := 0; i < numMessages; i++ {
		text := fmt.Sprintf("%d", i)

		if err := bbasic.Broadcast(nodes[i%numNodes], opcodeChat, chatMessage{text: strings.TrimSpace(text)}); err != nil {
			return err
		}
		time.Sleep(time.Duration(numNodes*25) * time.Millisecond)
	}

	return nil
}

func TestMultiple(t *testing.T) {
	assert.Nil(t, Run(3, 10))
	t.Log("done")
}
