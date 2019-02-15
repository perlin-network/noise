package protocol

import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/transport"
	"github.com/stretchr/testify/assert"
	"sync/atomic"
	"testing"
	"time"
)

type dummyBlock struct {
	earlyStop        bool
	stopByDisconnect bool
	blockCount       *uint32
}

func (b *dummyBlock) OnRegister(p *Protocol, node *noise.Node) {}

func (b *dummyBlock) OnBegin(p *Protocol, peer *noise.Peer) error {
	atomic.AddUint32(b.blockCount, 1)

	if b.earlyStop {
		if b.stopByDisconnect {
			return DisconnectPeer
		} else {
			return CompletedAllBlocks
		}
	} else {
		return nil
	}
}

func (b *dummyBlock) OnEnd(p *Protocol, peer *noise.Peer) error {
	return nil
}

func setupNodeForTest(node *noise.Node, totalBlocks, earlyStopIdx int, stopByDisconnect bool) (*Protocol, *uint32) {
	p := New()
	count := new(uint32)

	for i := 0; i < totalBlocks; i++ {
		blk := &dummyBlock{
			blockCount: count,
		}
		if i == earlyStopIdx {
			blk.earlyStop = true
			blk.stopByDisconnect = stopByDisconnect
		}
		p.Register(blk)
	}
	p.Enforce(node)
	go node.Listen()

	return p, count
}

func runTestProtocol(t *testing.T, stopByDisconnect bool) {
	log.Disable()
	defer log.Enable()

	params := noise.DefaultParams()
	params.Port = 3000
	params.Transport = transport.NewBuffered()

	alice, err := noise.NewNode(params)
	assert.NoError(t, err)
	_, aliceCount := setupNodeForTest(alice, 10, -1, false)

	params.Port++
	bob, err := noise.NewNode(params)
	assert.NoError(t, err)
	_, bobCount := setupNodeForTest(bob, 10, 5, stopByDisconnect)

	alicePeer, err := alice.Dial(bob.ExternalAddress())
	assert.NoError(t, err)

	aliceDisconnected := uint32(0)

	alicePeer.OnDisconnect(func(node *noise.Node, peer *noise.Peer) error {
		atomic.StoreUint32(&aliceDisconnected, 1)
		return nil
	})

	time.Sleep(100 * time.Millisecond) // Race condition!

	assert.Equal(t, atomic.LoadUint32(aliceCount), uint32(10))
	assert.Equal(t, atomic.LoadUint32(bobCount), uint32(6))

	if stopByDisconnect {
		assert.Equal(t, atomic.LoadUint32(&aliceDisconnected), uint32(1))
	} else {
		assert.Equal(t, atomic.LoadUint32(&aliceDisconnected), uint32(0))
	}
}

func TestProtocol_NoDisconnect(t *testing.T) {
	runTestProtocol(t, false)
}

func TestProtocol_Disconnect(t *testing.T) {
	runTestProtocol(t, true)
}
