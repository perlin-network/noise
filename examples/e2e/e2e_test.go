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
	var errs []error
	var mutex sync.Mutex

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

	var wg sync.WaitGroup
	wg.Add(numNodes - 1)

	for i := 1; i < numNodes; i++ {
		i := i

		go func() {
			defer wg.Done()

			peer, err := nodes[i].Dial(nodes[0].ExternalAddress())
			if err != nil {
				mutex.Lock()
				errs = append(errs, err)
				mutex.Unlock()
				return
			}

			skademlia.WaitUntilAuthenticated(peer)

			_ = skademlia.FindNode(nodes[i], protocol.NodeID(nodes[i]).(skademlia.ID), skademlia.BucketSize(), 8)
		}()
	}

	wg.Wait()

	if len(errs) > 0 {
		return errs[0]
	}

	wg.Add(numNodes)

	for i := 0; i < numNodes; i++ {
		go func(i int) {
			defer wg.Done()

			for j := 0; j < numTxEach; j++ {
				txt := fmt.Sprintf("sending from %d tx %d", i, j)

				broadcastErrors := skademlia.Broadcast(nodes[i], testMessage{text: strings.TrimSpace(txt)})
				errs = append(errs, broadcastErrors...)
			}
		}(i)
	}

	wg.Wait()

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

func TestRun(t *testing.T) {
	log.Disable()
	defer log.Enable()

	assert.Nil(t, Run(numNodes, numMessagesEach))
}
