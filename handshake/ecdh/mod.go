package ecdh

import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
	"go.dedis.ch/kyber/v3/group/edwards25519"
	"go.dedis.ch/kyber/v3/sign/schnorr"
	"time"
)

const msgEphemeralHandshake = ".noise_handshake"

var (
	_ protocol.Block = (*block)(nil)
)

type block struct {
	opcodeHandshake noise.Opcode

	suite           crypto.EllipticSuite
	timeoutDuration time.Duration
}

// New returns an ECDH policy with sensible defaults.
//
// By default, should a peer not complete the handshake protocol in 10 seconds, they will be disconnected.
// All handshake-related messages are appended with Schnorr signatures that are automatically verified.
func New() *block {
	return &block{
		suite:           edwards25519.NewBlakeSHA256Ed25519(),
		timeoutDuration: 10 * time.Second,
	}
}

func (b *block) WithSuite(suite crypto.EllipticSuite) *block {
	b.suite = suite
	return b
}

func (b *block) TimeoutAfter(timeoutDuration time.Duration) *block {
	b.timeoutDuration = timeoutDuration
	return b
}

func (b *block) OnRegister(p *protocol.Protocol, node *noise.Node) {
	b.opcodeHandshake = noise.RegisterMessage(noise.NextAvailableOpcode(), (*Handshake)(nil))
}

func (b *block) OnBegin(p *protocol.Protocol, peer *noise.Peer) error {
	// Send a handshake request with a generated ephemeral keypair.
	ephemeralPrivateKey := b.suite.Scalar().Pick(b.suite.RandomStream())
	ephemeralPublicKey := b.suite.Point().Mul(ephemeralPrivateKey, b.suite.Point().Base())

	var err error
	var req Handshake

	req.publicKey, err = ephemeralPublicKey.MarshalBinary()
	if err != nil {
		return errors.Wrap(errors.Wrap(protocol.DisconnectPeer, err.Error()), "failed to marshal ephemeral public key into binary")
	}

	req.signature, err = schnorr.Sign(b.suite, ephemeralPrivateKey, []byte(msgEphemeralHandshake))
	if err != nil {
		return errors.Wrap(errors.Wrap(protocol.DisconnectPeer, err.Error()), "failed to sign handshake message using Schnorr signature scheme")
	}

	err = peer.SendMessage(req)
	if err != nil {
		return errors.Wrap(errors.Wrap(protocol.DisconnectPeer, err.Error()), "failed to send our ephemeral public key to our peer")
	}

	// Wait for handshake response.
	var res Handshake
	var ok bool

	select {
	case <-time.After(b.timeoutDuration):
		return errors.Wrap(protocol.DisconnectPeer, "timed out receiving handshake request")
	case msg := <-peer.Receive(b.opcodeHandshake):
		res, ok = msg.(Handshake)
		if !ok {
			return errors.Wrap(protocol.DisconnectPeer, "did not get a handshake response back")
		}
	}

	peersPublicKey := b.suite.Point()

	err = peersPublicKey.UnmarshalBinary(res.publicKey)
	if err != nil {
		return errors.Wrap(errors.Wrap(protocol.DisconnectPeer, err.Error()), "failed to unmarshal our peers ephemeral public key")
	}

	err = schnorr.Verify(b.suite, peersPublicKey, []byte(msgEphemeralHandshake), res.signature)
	if err != nil {
		return errors.Wrap(errors.Wrap(protocol.DisconnectPeer, err.Error()), "failed to verify signature in handshake request")
	}

	ephemeralSharedKey := computeSharedKey(b.suite, ephemeralPrivateKey, peersPublicKey)

	sharedKeyBuf, err := ephemeralSharedKey.MarshalBinary()
	if err != nil {
		peer.Disconnect()
		return errors.Wrap(err, "failed to marshal post-handshake shared key")
	}

	protocol.SetSharedKey(peer, sharedKeyBuf)

	log.Debug().
		Str("ephemeral_shared_key", ephemeralSharedKey.String()).
		Msg("Successfully performed ECDH with our peer.")

	return nil
}

func (b *block) OnEnd(p *protocol.Protocol, peer *noise.Peer) error {
	return nil
}
