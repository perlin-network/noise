package ecdh

import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/transport"
	"github.com/stretchr/testify/assert"
	"sync/atomic"
	"testing"
	"time"
)

type DummyBlock struct {
	reachable uint32
}

func (b *DummyBlock) OnRegister(p *protocol.Protocol, node *noise.Node) {
}

func (b *DummyBlock) OnBegin(p *protocol.Protocol, peer *noise.Peer) error {
	atomic.StoreUint32(&b.reachable, 1)
	return nil
}

func (b *DummyBlock) OnEnd(p *protocol.Protocol, peer *noise.Peer) error {
	return nil
}

func TestECDH(t *testing.T) {
	log.Disable()
	defer log.Enable()

	params := noise.DefaultParams()
	params.Port = 3000
	params.Transport = transport.NewBuffered()

	alice, err := noise.NewNode(params)
	assert.NoError(t, err)

	params.Port++
	bob, err := noise.NewNode(params)
	assert.NoError(t, err)

	go alice.Listen()
	go bob.Listen()

	blockAlice, blockBob := &DummyBlock{}, &DummyBlock{}

	p := protocol.New()
	p.Register(New())
	p.Register(blockAlice)
	p.Enforce(alice)

	p = protocol.New()
	p.Register(New())
	p.Register(blockBob)
	p.Enforce(bob)

	alice.Dial(bob.ExternalAddress())

	time.Sleep(100 * time.Millisecond)

	assert.True(t, atomic.LoadUint32(&blockAlice.reachable) == 1)
	assert.True(t, atomic.LoadUint32(&blockBob.reachable) == 1)
}
