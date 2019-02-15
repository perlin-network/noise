package handshake

import (
	"encoding/hex"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/skademlia"
	"github.com/perlin-network/noise/transport"
	"github.com/stretchr/testify/assert"
	"testing"
)

var transportLayer = transport.NewBuffered()
var port = 3000

func node(t *testing.T) *noise.Node {
	params := noise.DefaultParams()
	params.Transport = transportLayer
	params.Port = uint16(port)
	params.ID = skademlia.NewIdentityRandom()

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

func TestBlock_OnBeginSuccessful(t *testing.T) {
	log.Disable()
	defer log.Enable()

	alice, bob := node(t), node(t)

	defer alice.Kill()
	defer bob.Kill()

	aliceReceiver, bobReceiver := new(receiverBlock), new(receiverBlock)

	p := protocol.New()
	p.Register(New())
	p.Register(aliceReceiver)
	p.Enforce(alice)

	p = protocol.New()
	p.Register(New())
	p.Register(bobReceiver)
	p.Enforce(bob)

	peer, err := alice.Dial(bob.ExternalAddress())
	assert.NoError(t, err)

	<-aliceReceiver.receiver
	<-bobReceiver.receiver

	assert.Equal(t, hex.EncodeToString(bob.ID.PublicID()), peer.Get(KeySkademliaHandshake))
}
