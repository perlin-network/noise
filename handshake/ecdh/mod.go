package ecdh

import (
	"crypto"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/internal/edwards25519"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
	"time"
)

const DefaultHandshakeMessage = ".noise_handshake"

var (
	_ protocol.Block = (*block)(nil)
)

type block struct {
	opcodeHandshake noise.Opcode
	timeoutDuration time.Duration

	handshakeMessage string
}

// New returns an ECDH policy with sensible defaults.
//
// By default, should a peer not complete the handshake protocol in 10 seconds, they will be disconnected.
// All handshake-related messages are appended with ECDSA signatures that are automatically verified.
func New() *block {
	return &block{
		timeoutDuration:  10 * time.Second,
		handshakeMessage: DefaultHandshakeMessage,
	}
}

func (b *block) TimeoutAfter(timeoutDuration time.Duration) *block {
	b.timeoutDuration = timeoutDuration
	return b
}

func (b *block) WithHandshakeMessage(handshakeMessage string) *block {
	b.handshakeMessage = handshakeMessage
	return b
}

func (b *block) OnRegister(p *protocol.Protocol, node *noise.Node) {
	b.opcodeHandshake = noise.RegisterMessage(noise.NextAvailableOpcode(), (*Handshake)(nil))
}

func (b *block) OnBegin(p *protocol.Protocol, peer *noise.Peer) error {
	// Send a handshake request with a generated ephemeral keypair.
	ephemeralPublicKey, ephemeralPrivateKey, err := edwards25519.GenerateKey(nil)
	if err != nil {
		return errors.Wrap(errors.Wrap(protocol.DisconnectPeer, err.Error()), "failed to generate ephemeral keypair")
	}

	req := Handshake{publicKey: ephemeralPublicKey}
	req.signature, err = ephemeralPrivateKey.Sign(nil, []byte(b.handshakeMessage), crypto.Hash(0))
	if err != nil {
		return errors.Wrap(errors.Wrap(protocol.DisconnectPeer, err.Error()), "failed to sign handshake message using ECDSA")
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

	peersPublicKey := edwards25519.PublicKey(res.publicKey)

	if !isEd25519GroupElement(peersPublicKey) {
		return errors.Wrap(protocol.DisconnectPeer, "failed to unmarshal our peers ephemeral public key")
	}

	if !edwards25519.Verify(peersPublicKey, []byte(b.handshakeMessage), res.signature) {
		return errors.Wrap(protocol.DisconnectPeer, "failed to verify signature in handshake request")
	}

	ephemeralSharedKey := computeSharedKey(ephemeralPrivateKey, peersPublicKey)
	protocol.SetSharedKey(peer, ephemeralSharedKey[:])

	log.Debug().
		Hex("ephemeral_shared_key", ephemeralSharedKey[:]).
		Msg("Successfully performed ECDH with our peer.")

	return nil
}

func (b *block) OnEnd(p *protocol.Protocol, peer *noise.Peer) error {
	return nil
}
