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
	"strings"
	"testing"
	"time"
)

var transportLayer = transport.NewBuffered()
var port = 3000

func node(t *testing.T) *noise.Node {
	params := noise.DefaultParams()
	params.Transport = transportLayer
	params.Port = uint16(port)

	port++

	node, err := noise.NewNode(params)
	assert.NoError(t, err)

	go node.Listen()

	return node
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
	err := peer.SendMessage(msg{})
	if err != nil {
		return err
	}

	b.receiver <- <-peer.Receive(b.opcodeMsg)
	return nil
}

func (b *receiverBlock) OnEnd(p *protocol.Protocol, peer *noise.Peer) error {
	return nil
}

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

		return p, block
	}

	// Setup Alice and Bob.
	alice, bob := node(t), node(t)

	defer alice.Kill()
	defer bob.Kill()

	// Register protocols.
	aliceProtocol, aliceBlock := setup(alice)
	bobProtocol, bobBlock := setup(bob)

	aliceProtocol.Enforce(alice)
	bobProtocol.Enforce(bob)

	// Have Alice dial Bob.
	peerBob1, err := alice.Dial(bob.ExternalAddress())
	assert.NoError(t, err)

	// Check opcode is registered.
	assert.True(t, aliceBlock.opcodeACK != noise.OpcodeNil)
	assert.True(t, bobBlock.opcodeACK != noise.OpcodeNil)

	// Expect a disconnect calling OnBegin without Bob yet having an ephemeral shared key.
	assert.True(t, errors.Cause(aliceBlock.OnBegin(aliceProtocol, peerBob1)) == protocol.DisconnectPeer)

	// Now set an invalid ephemeral shared key to Bob, and check OnBegin fails
	peerBob2, err := alice.Dial(bob.ExternalAddress())
	assert.NoError(t, err)

	protocol.SetSharedKey(peerBob2, []byte("test ephemeral key"))
	assert.True(t, strings.Contains(aliceBlock.OnBegin(aliceProtocol, peerBob2).Error(), "failed to unmarshal ephemeral shared key buf"))

	// Now restart connections, and set a proper ephemeral shared key to Bob.
	ephemeralSharedKey, err := hex.DecodeString("d8747263b4d54588c2c8f17862d827dee6d3893a02fb7a84800b001ad4f1cee8")
	assert.NoError(t, err)

	peerBob3, err := alice.Dial(bob.ExternalAddress())
	assert.NoError(t, err)

	protocol.SetSharedKey(peerBob3, ephemeralSharedKey)
	assert.True(t, errors.Cause(aliceBlock.OnBegin(aliceProtocol, peerBob3)) == protocol.DisconnectPeer)
}

func TestBlock_OnBeginSuccessful(t *testing.T) {
	log.Disable()
	defer log.Enable()

	ephemeralSharedKey, err := hex.DecodeString("d8747263b4d54588c2c8f17862d827dee6d3893a02fb7a84800b001ad4f1cee8")
	assert.NoError(t, err)

	alice, bob := node(t), node(t)

	defer alice.Kill()
	defer bob.Kill()

	alice.OnPeerInit(func(node *noise.Node, peer *noise.Peer) error {
		peer.Set(protocol.KeySharedKey, ephemeralSharedKey)
		return nil
	})

	bob.OnPeerInit(func(node *noise.Node, peer *noise.Peer) error {
		peer.Set(protocol.KeySharedKey, ephemeralSharedKey)
		return nil
	})

	aliceReceiver, bobReceiver := new(receiverBlock), new(receiverBlock)

	p := protocol.New()
	p.Register(New())
	p.Register(aliceReceiver)
	p.Enforce(alice)

	p = protocol.New()
	p.Register(New())
	p.Register(bobReceiver)
	p.Enforce(bob)

	_, err = alice.Dial(bob.ExternalAddress())
	assert.NoError(t, err)

	<-aliceReceiver.receiver
	<-bobReceiver.receiver
}
