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

func setup(node *noise.Node) (*protocol.Protocol, *block) {
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

func TestBlock_OnBegin(t *testing.T) {
	log.Disable()
	defer log.Enable()

	params := noise.DefaultParams()
	params.Port = 3000
	params.Transport = transport.NewBuffered()

	alice, err := noise.NewNode(params)
	assert.NoError(t, err)
	aliceProtocol, aliceBlock := setup(alice)

	params.Port = 3001

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
	assert.True(t, strings.Contains(aliceBlock.OnBegin(aliceProtocol, peerBob).Error(), "failed to send AEAD ACK"))
}
