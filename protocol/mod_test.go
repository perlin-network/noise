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
	earlyStop  bool
	blockCount *uint32
}

func (b *dummyBlock) OnRegister(p *Protocol, node *noise.Node) {}

func (b *dummyBlock) OnBegin(p *Protocol, peer *noise.Peer) error {
	atomic.AddUint32(b.blockCount, 1)

	if b.earlyStop {
		return DisconnectPeer
	} else {
		return nil
	}
}

func (b *dummyBlock) OnEnd(p *Protocol, peer *noise.Peer) error {
	return nil
}

func setupNodeForTest(node *noise.Node, totalBlocks, earlyStopIdx int) (*Protocol, *uint32) {
	p := New()
	count := new(uint32)

	for i := 0; i < totalBlocks; i++ {
		blk := &dummyBlock{
			blockCount: count,
		}
		if i == earlyStopIdx {
			blk.earlyStop = true
		}
		p.Register(blk)
	}
	p.Enforce(node)
	go node.Listen()

	return p, count
}

func TestProtocol(t *testing.T) {
	log.Disable()
	defer log.Enable()

	params := noise.DefaultParams()
	params.Transport = transport.NewBuffered()

	alice, err := noise.NewNode(params)
	assert.NoError(t, err)
	_, aliceCount := setupNodeForTest(alice, 10, -1)

	bob, err := noise.NewNode(params)
	assert.NoError(t, err)
	_, bobCount := setupNodeForTest(bob, 10, 5)

	alicePeer, err := alice.Dial(bob.ExternalAddress())
	assert.NoError(t, err)

	aliceDisconnected := uint32(0)

	alicePeer.OnDisconnect(func(node *noise.Node, peer *noise.Peer) error {
		atomic.StoreUint32(&aliceDisconnected, 1)
		return nil
	})

	time.Sleep(1 * time.Millisecond) // Race condition!

	assert.Equal(t, atomic.LoadUint32(aliceCount), uint32(10))
	assert.Equal(t, atomic.LoadUint32(bobCount), uint32(6))

	assert.Equal(t, atomic.LoadUint32(&aliceDisconnected), uint32(1))
}
