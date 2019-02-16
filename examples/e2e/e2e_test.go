package e2e

import (
	"fmt"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/cipher/aead"
	"github.com/perlin-network/noise/handshake/ecdh"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/payload"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/skademlia"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"strconv"
	"strings"
	"sync"
	"testing"
)

var (
	_               noise.Message = (*testMessage)(nil)
	startPort                     = 4000
	numNodes                      = 10
	numMessagesEach               = 100
)

type testMessage struct {
	text string
}

func (testMessage) Read(reader payload.Reader) (noise.Message, error) {
	text, err := reader.ReadString()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read test message")
	}

	return testMessage{text: text}, nil
}

func (m testMessage) Write() []byte {
	return payload.NewWriter(nil).WriteString(m.text).Bytes()
}

func setup(node *noise.Node, opcodeTest noise.Opcode) {
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
				<-peer.Receive(opcodeTest)
			}
		}()

		return nil
	})
}

func Run(numNodes int, numTxEach int) error {
	opcodeTest := noise.RegisterMessage(noise.NextAvailableOpcode(), (*testMessage)(nil))
	var nodes []*noise.Node
	var errors []error

	for i := 0; i < numNodes; i++ {
		params := noise.DefaultParams()
		params.Keys = skademlia.RandomKeys()

		node, err := noise.NewNode(params)
		if err != nil {
			return err
		}

		p := protocol.New()
		p.Register(ecdh.New())
		p.Register(aead.New())
		p.Register(skademlia.New())
		p.Enforce(node)

		setup(node, opcodeTest)
		go node.Listen()

		nodes = append(nodes, node)
	}

	defer func() {
		for _, node := range nodes {
			node.Kill()
		}
	}()

	for i := 1; i < numNodes; i++ {
		peer, err := nodes[i].Dial(nodes[0].ExternalAddress())
		if err != nil {
			errors = append(errors, err)
		}

		skademlia.WaitUntilAuthenticated(peer)
		_ = skademlia.FindNode(nodes[i], protocol.NodeID(nodes[i]).(skademlia.ID), skademlia.BucketSize(), 8)
	}

	if len(errors) > 0 {
		return errors[0]
	}

	var wg sync.WaitGroup

	for i := 0; i < numNodes; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			for j := 0; j < numTxEach; j++ {
				txt := fmt.Sprintf("sending from %d tx %d", i, j)

				errs := skademlia.Broadcast(nodes[i], testMessage{text: strings.TrimSpace(txt)})
				if len(errs) > 0 {
					errors = append(errors, errs...)
				}
			}
		}(i)
	}
	wg.Wait()

	if len(errors) > 0 {
		return errors[0]
	}

	return nil
}

func TestRun(t *testing.T) {
	//log.Disable()
	//defer log.Enable()

	assert.Nil(t, Run(numNodes, numMessagesEach))

	noise.DebugOpcodes()
}
