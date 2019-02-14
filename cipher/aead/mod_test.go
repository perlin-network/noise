package aead

import (
	"crypto/sha512"
	"encoding/hex"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/transport"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"go.dedis.ch/kyber/v3/group/edwards25519"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestBlock_OnBeginEdgeCases(t *testing.T) {
	log.Disable()
	defer log.Enable()

	setup := func(node *noise.Node) (*protocol.Protocol, *block) {
		block := New()
		block.WithHash(sha512.New)
		block.WithCurve(edwards25519.NewBlakeSHA256Ed25519())
		block.WithACKTimeout(1 * time.Millisecond)

		p := protocol.New()
		p.Register(block)
		p.Enforce(node)

		go node.Listen()

		return p, block
	}

	params := noise.DefaultParams()
	params.Port = 3000
	params.Transport = transport.NewBuffered()

	alice, err := noise.NewNode(params)
	assert.NoError(t, err)
	aliceProtocol, aliceBlock := setup(alice)

	params.Port++
	bob, err := noise.NewNode(params)
	assert.NoError(t, err)
	_, bobBlock := setup(bob)

	peerBob, err := alice.Dial(bob.ExternalAddress())
	assert.NoError(t, err)

	// Check opcode is registered.
	assert.True(t, aliceBlock.opcodeACK != noise.OpcodeNil)
	assert.True(t, bobBlock.opcodeACK != noise.OpcodeNil)

	// Expect a disconnect calling OnBegin without Bob yet having an ephemeral shared key.
	assert.True(t, errors.Cause(aliceBlock.OnBegin(aliceProtocol, peerBob)) == protocol.DisconnectPeer)

	// Now set an invalid ephemeral shared key to Bob, and check OnBegin fails
	protocol.SetSharedKey(peerBob, []byte("test ephemeral key"))
	assert.True(t, strings.Contains(aliceBlock.OnBegin(aliceProtocol, peerBob).Error(), "failed to unmarshal ephemeral shared key buf"))

	// Now restart connections, and set a proper ephemeral shared key to Bob.
	ephemeralSharedKey, err := hex.DecodeString("d8747263b4d54588c2c8f17862d827dee6d3893a02fb7a84800b001ad4f1cee8")
	assert.NoError(t, err)

	peerBob, err = alice.Dial(bob.ExternalAddress())
	assert.NoError(t, err)

	protocol.SetSharedKey(peerBob, ephemeralSharedKey)
	assert.True(t, errors.Cause(aliceBlock.OnBegin(aliceProtocol, peerBob)) == protocol.DisconnectPeer)
}

type msg struct{ noise.EmptyMessage }

var _ protocol.Block = (*receiverBlock)(nil)

type receiverBlock struct {
	opcodeMsg noise.Opcode
	receiver  chan interface{}
}

func (b *receiverBlock) OnRegister(p *protocol.Protocol, node *noise.Node) {
	b.opcodeMsg = noise.RegisterMessage(noise.NextAvailableOpcode(), (*msg)(nil))
	b.receiver = make(chan interface{}, 1)
}

func (b *receiverBlock) OnBegin(p *protocol.Protocol, peer *noise.Peer) error {
	return peer.SendMessage(msg{})
}

func (b *receiverBlock) OnEnd(p *protocol.Protocol, peer *noise.Peer) error {
	b.receiver <- <-peer.Receive(b.opcodeMsg)
	return nil
}

func TestBlock_OnBeginSuccessful(t *testing.T) {
	log.Disable()
	defer log.Enable()

	ephemeralSharedKey, err := hex.DecodeString("d8747263b4d54588c2c8f17862d827dee6d3893a02fb7a84800b001ad4f1cee8")
	assert.NoError(t, err)

	params := noise.DefaultParams()
	params.Port = 3000
	params.Transport = transport.NewBuffered()

	alice, err := noise.NewNode(params)
	assert.NoError(t, err)

	alice.OnPeerInit(func(node *noise.Node, peer *noise.Peer) error {
		peer.Set(protocol.KeySharedKey, ephemeralSharedKey)
		return nil
	})

	params.Port++
	bob, err := noise.NewNode(params)
	assert.NoError(t, err)

	bob.OnPeerInit(func(node *noise.Node, peer *noise.Peer) error {
		peer.Set(protocol.KeySharedKey, ephemeralSharedKey)
		return nil
	})

	go alice.Listen()
	go bob.Listen()

	aliceReceiver, bobReceiver := new(receiverBlock), new(receiverBlock)

	p := protocol.New()
	p.Register(New())
	p.Register(aliceReceiver)
	p.Enforce(alice)

	p = protocol.New()
	p.Register(New())
	p.Register(bobReceiver)
	p.Enforce(bob)

	_, err = alice.Dial(":" + strconv.Itoa(int(bob.Port())))
	assert.NoError(t, err)

	<-aliceReceiver.receiver
	<-bobReceiver.receiver
}
