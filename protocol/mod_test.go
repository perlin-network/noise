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
		return CompletedAllBlocks
	} else {
		return nil
	}
}

func (b *dummyBlock) OnEnd(p *Protocol, peer *noise.Peer) error {
	return nil
}

func TestProtocol(t *testing.T) {
	log.Disable()
	defer log.Enable()

	setup := func(node *noise.Node, totalBlocks, earlyStopIdx int) (*Protocol, *uint32) {
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

	params := noise.DefaultParams()
	params.Port = 3000
	params.Transport = transport.NewBuffered()

	alice, err := noise.NewNode(params)
	assert.NoError(t, err)
	_, aliceCount := setup(alice, 10, -1)

	params.Port++
	bob, err := noise.NewNode(params)
	assert.NoError(t, err)
	_, bobCount := setup(bob, 10, 5)

	_, err = alice.Dial(bob.ExternalAddress())
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond) // Race condition!

	assert.Equal(t, atomic.LoadUint32(aliceCount), uint32(10))
	assert.Equal(t, atomic.LoadUint32(bobCount), uint32(6))
}
