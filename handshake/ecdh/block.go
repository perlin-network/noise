package ecdh

import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/callbacks"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/timeout"
	"github.com/pkg/errors"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/group/edwards25519"
	"go.dedis.ch/kyber/v3/sign/schnorr"
	"time"
)

var (
	_ protocol.Block = (*Block)(nil)
)

type Block struct {
	OpcodeHandshake noise.Opcode

	suite           crypto.EllipticSuite
	timeoutDuration time.Duration

	handshakeCompleted bool
}

// New returns an ECDH policy with sensible defaults.
//
// By default, should a peer not complete the handshake protocol in 10 seconds, they will be disconnected.
// All handshake-related messages are appended with Schnorr signatures that are automatically verified.
func NewECDH() *Block {
	return &Block{
		OpcodeHandshake:    noise.RegisterMessage(noise.NextAvailableOpcode(), (*messageHandshake)(nil)),
		suite:              edwards25519.NewBlakeSHA256Ed25519(),
		timeoutDuration:    10 * time.Second,
		handshakeCompleted: false,
	}
}

func (p *Block) WithSuite(suite crypto.EllipticSuite) *Block {
	p.suite = suite
	return p
}

func (p *Block) TimeoutAfter(timeoutDuration time.Duration) *Block {
	p.timeoutDuration = timeoutDuration
	return p
}

func (p *Block) OnBegin(node *noise.Node, peer *noise.Peer) error {
	if !p.handshakeCompleted {
		// Register event listeners to our node.
		node.OnPeerInit(p.onPeerInit)
		node.OnPeerDisconnected(p.onPeerDisconnected)

		node.OnMessageReceived(p.OpcodeHandshake, func(node *noise.Node, opcode noise.Opcode, peer *noise.Peer, message noise.Message) error {
			return p.DoHandshake(peer, opcode, message)
		})
	} else {
		if protocol, ok := peer.Get(protocol.KeyProtocolBlocks).(*protocol.Protocol); ok {
			return protocol.Next(node, peer)
		}
		return errors.Errorf("protocol not setup")
	}
	return nil
}

func (b *Block) OnEnd(node *noise.Node, peer *noise.Peer) error {
	// TODO: remove all the handlers that were registered in OnBegin
	return nil
}

func (p *Block) DoHandshake(peer *noise.Peer, opcode noise.Opcode, message noise.Message) error {
	if !peer.Has(keyEphemeralPrivateKey) {
		peer.Disconnect()
		return errors.New("peer attempted to perform ECDH with us even though we never have instantiated a handshake")
	}

	msg := message.(messageHandshake)

	peersPublicKey := p.suite.Point()
	err := peersPublicKey.UnmarshalBinary(msg.publicKey)

	if err != nil {
		peer.Disconnect()
		return errors.Wrap(err, "failed to unmarshal our peers ephemeral public key")
	}

	err = schnorr.Verify(p.suite, peersPublicKey, []byte(msgEphemeralHandshake), msg.signature)

	if err != nil {
		peer.Disconnect()
		return errors.Wrap(err, "failed to verify signature in handshake request")
	}

	ourPrivateKey := peer.Get(keyEphemeralPrivateKey).(kyber.Scalar)
	ephemeralSharedKey := computeSharedKey(p.suite, ourPrivateKey, peersPublicKey)

	log.Debug().Str("ephemeral_shared_key", ephemeralSharedKey.String()).Msg("Successfully performed ECDH with our peer.")

	sharedKeyBuf, err := ephemeralSharedKey.MarshalBinary()
	if err != nil {
		peer.Disconnect()
		return errors.Wrap(err, "failed to marshal post-handshake shared key")
	}

	peer.Delete(keyEphemeralPrivateKey)
	protocol.SetSharedKey(peer, sharedKeyBuf)

	p.handshakeCompleted = true

	if err = timeout.Clear(peer, keyTimeoutDispatcher); err != nil {
		peer.Disconnect()
		return errors.Wrap(err, "error enforcing handshake timeout policy")
	}

	return callbacks.DeregisterCallback
}

func (p *Block) onPeerInit(node *noise.Node, peer *noise.Peer) (err error) {
	if peer.Has(keyEphemeralPrivateKey) {
		peer.Disconnect()
		return errors.New("peer attempted to have us instantiate a 2nd handshake")
	}

	// Generate an ephemeral keypair to perform ECDH with our peer.
	ephemeralPrivateKey := p.suite.Scalar().Pick(p.suite.RandomStream())
	ephemeralPublicKey := p.suite.Point().Mul(ephemeralPrivateKey, p.suite.Point().Base())

	peer.Set(keyEphemeralPrivateKey, ephemeralPrivateKey)

	msg := messageHandshake{}
	msg.publicKey, err = ephemeralPublicKey.MarshalBinary()

	if err != nil {
		peer.Disconnect()
		return errors.Wrap(err, "failed to marshal ephemeral public key into binary")
	}

	msg.signature, err = schnorr.Sign(p.suite, ephemeralPrivateKey, []byte(msgEphemeralHandshake))

	if err != nil {
		peer.Disconnect()
		return errors.Wrap(err, "failed to sign handshake message using Schnorr signature scheme")
	}

	err = peer.SendMessage(OpcodeHandshake, msg)
	if err != nil {
		peer.Disconnect()
		return errors.Wrap(err, "failed to send our ephemeral public key to our peer")
	}

	timeout.Enforce(peer, keyTimeoutDispatcher, p.timeoutDuration, peer.Disconnect)

	return nil
}

func (p *Block) onPeerDisconnected(node *noise.Node, peer *noise.Peer) error {
	peer.Delete(keyEphemeralPrivateKey)
	protocol.DeleteSharedKey(peer)

	return callbacks.DeregisterCallback
}
