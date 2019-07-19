package ecdh

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/transport"
	"github.com/stretchr/testify/assert"
)

type dummyBlock struct {
	reachable uint32
}

func (b *dummyBlock) OnRegister(p *protocol.Protocol, node *noise.Node) {
}

func (b *dummyBlock) OnBegin(p *protocol.Protocol, peer *noise.Peer) error {
	atomic.StoreUint32(&b.reachable, 1)
	return nil
}

func (b *dummyBlock) OnEnd(p *protocol.Protocol, peer *noise.Peer) error {
	return nil
}

func TestECDH(t *testing.T) {
	log.Disable()
	defer log.Enable()

	params := noise.DefaultParams()
	params.Transport = transport.NewBuffered()

	alice, err := noise.NewNode(params)
	assert.NoError(t, err)

	bob, err := noise.NewNode(params)
	assert.NoError(t, err)

	go alice.Listen()
	go bob.Listen()

	defer alice.Kill()
	defer bob.Kill()

	blockAlice, blockBob := new(dummyBlock), new(dummyBlock)

	p := protocol.New()
	p.Register(New())
	p.Register(blockAlice)
	p.Enforce(alice)

	p = protocol.New()
	p.Register(New())
	p.Register(blockBob)
	p.Enforce(bob)

	_, err = alice.Dial(bob.ExternalAddress())
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	assert.True(t, atomic.LoadUint32(&blockAlice.reachable) == 1)
	assert.True(t, atomic.LoadUint32(&blockBob.reachable) == 1)
}
