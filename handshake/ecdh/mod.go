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
	"sync"
	"time"
)

var (
	OpcodeHandshake     noise.Opcode
	registerOpcodesOnce sync.Once
)

const (
	keyTimeoutDispatcher = "ecdh.timeout"

	keyEphemeralPrivateKey = "ecdh.ephemeralPrivateKey"
	msgEphemeralHandshake  = ".noise_handshake"
)

var (
	_ protocol.Block = (*Block)(nil)
)

type Block struct {
	suite           crypto.EllipticSuite
	timeoutDuration time.Duration

	handshakeCompleted bool
}

func init() {
	OpcodeHandshake = noise.RegisterMessage(noise.NextAvailableOpcode(), (*messageHandshake)(nil))
}

// New returns an ECDH policy with sensible defaults.
//
// By default, should a peer not complete the handshake protocol in 10 seconds, they will be disconnected.
// All handshake-related messages are appended with Schnorr signatures that are automatically verified.
func New() *Block {
	return &Block{
		suite:              edwards25519.NewBlakeSHA256Ed25519(),
		timeoutDuration:    10 * time.Second,
		handshakeCompleted: false,
	}
}

func (b *Block) WithSuite(suite crypto.EllipticSuite) *Block {
	b.suite = suite
	return b
}

func (b *Block) TimeoutAfter(timeoutDuration time.Duration) *Block {
	b.timeoutDuration = timeoutDuration
	return b
}

func (b *Block) OnBegin(p *protocol.Protocol, peer *noise.Peer) error {
	var err error

	if peer.Has(keyEphemeralPrivateKey) {
		peer.Disconnect()
		return errors.New("peer attempted to have us instantiate a 2nd handshake")
	}

	// Generate an ephemeral keypair to perform ECDH with our peer.
	ephemeralPrivateKey := b.suite.Scalar().Pick(b.suite.RandomStream())
	ephemeralPublicKey := b.suite.Point().Mul(ephemeralPrivateKey, b.suite.Point().Base())

	peer.OnMessageReceived(OpcodeHandshake, b.DoHandshake)
	peer.Set(keyEphemeralPrivateKey, ephemeralPrivateKey)

	msg := messageHandshake{}
	msg.publicKey, err = ephemeralPublicKey.MarshalBinary()

	if err != nil {
		peer.Disconnect()
		return errors.Wrap(err, "failed to marshal ephemeral public key into binary")
	}

	msg.signature, err = schnorr.Sign(b.suite, ephemeralPrivateKey, []byte(msgEphemeralHandshake))

	if err != nil {
		peer.Disconnect()
		return errors.Wrap(err, "failed to sign handshake message using Schnorr signature scheme")
	}

	err = peer.SendMessage(OpcodeHandshake, msg)
	if err != nil {
		peer.Disconnect()
		return errors.Wrap(err, "failed to send our ephemeral public key to our peer")
	}

	timeout.Enforce(peer, keyTimeoutDispatcher, b.timeoutDuration, peer.Disconnect)

	return nil
}

func (b *Block) OnEnd(p *protocol.Protocol, peer *noise.Peer) error {
	// TODO: remove all the handlers that were registered in OnBegin
	peer.Delete(keyEphemeralPrivateKey)
	protocol.DeleteSharedKey(peer)

	return nil
}

func (b *Block) DoHandshake(node *noise.Node, opcode noise.Opcode, peer *noise.Peer, message noise.Message) error {
	if !peer.Has(keyEphemeralPrivateKey) {
		peer.Disconnect()
		return errors.New("peer attempted to perform ECDH with us even though we never have instantiated a handshake")
	}

	msg := message.(messageHandshake)

	peersPublicKey := b.suite.Point()
	err := peersPublicKey.UnmarshalBinary(msg.publicKey)

	if err != nil {
		peer.Disconnect()
		return errors.Wrap(err, "failed to unmarshal our peers ephemeral public key")
	}

	err = schnorr.Verify(b.suite, peersPublicKey, []byte(msgEphemeralHandshake), msg.signature)

	if err != nil {
		peer.Disconnect()
		return errors.Wrap(err, "failed to verify signature in handshake request")
	}

	ourPrivateKey := peer.Get(keyEphemeralPrivateKey).(kyber.Scalar)
	ephemeralSharedKey := computeSharedKey(b.suite, ourPrivateKey, peersPublicKey)

	log.Debug().Str("ephemeral_shared_key", ephemeralSharedKey.String()).Msg("Successfully performed ECDH with our peer.")

	sharedKeyBuf, err := ephemeralSharedKey.MarshalBinary()
	if err != nil {
		peer.Disconnect()
		return errors.Wrap(err, "failed to marshal post-handshake shared key")
	}

	peer.Delete(keyEphemeralPrivateKey)
	protocol.SetSharedKey(peer, sharedKeyBuf)

	b.handshakeCompleted = true

	if err = timeout.Clear(peer, keyTimeoutDispatcher); err != nil {
		peer.Disconnect()
		return errors.Wrap(err, "error enforcing handshake timeout policy")
	}

	return callbacks.DeregisterCallback
}
