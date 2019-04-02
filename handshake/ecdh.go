package handshake

import (
	"crypto"
	"fmt"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/edwards25519"
	"github.com/pkg/errors"
	"time"
)

const (
	SignalCompletedECDH = "handshake.ecdh.completed"
	OpcodeHandshakeECDH = "handshake.ecdh"
)

type ECDH struct {
	message []byte
	timeout time.Duration
}

func NewECDH() *ECDH {
	return &ECDH{
		message: []byte(".noise_handshake_"),
		timeout: 3 * time.Second,
	}
}

func (b *ECDH) WithMessage(message []byte) *ECDH {
	b.message = message
	return b
}

func (b *ECDH) TimeoutAfter(timeout time.Duration) *ECDH {
	b.timeout = timeout
	return b
}

func (b *ECDH) RegisterOpcodes(n *noise.Node) {
	n.RegisterOpcode(OpcodeHandshakeECDH, n.NextAvailableOpcode())
}

func (b *ECDH) Protocol() noise.ProtocolBlock {
	return func(ctx noise.Context) error {
		ephemeral, err := b.Handshake(ctx)

		if err != nil {
			return err
		}

		ctx.Set(KeyEphemeral, ephemeral)

		return nil
	}
}

func (b *ECDH) Handshake(ctx noise.Context) (ephemeral []byte, err error) {
	node, peer := ctx.Node(), ctx.Peer()

	signal := peer.RegisterSignal(SignalCompletedECDH)
	defer signal()

	ephemeralPublicKey, ephemeralPrivateKey, err := edwards25519.GenerateKey(nil)

	if err != nil {
		return nil, errors.New("ecdh: failed to generate ephemeral keypair")
	}

	req := Handshake{publicKey: ephemeralPublicKey}

	if req.signature, err = ephemeralPrivateKey.Sign(b.message, crypto.Hash(0)); err != nil {
		return nil, errors.Wrap(err, "ecdh: failed to sign handshake message")
	}

	if err = peer.SendWithTimeout(node.Opcode(OpcodeHandshakeECDH), req.Marshal(), b.timeout); err != nil {
		return nil, errors.Wrap(err, "ecdh: failed to send our ephemeral public key to our peer")
	}

	var buf []byte

	select {
	case <-ctx.Done():
		return nil, noise.ErrDisconnect
	case <-time.After(b.timeout):
		return nil, errors.Wrap(noise.ErrTimeout, "ecdh: timed out receiving handshake response")
	case ctx := <-peer.Recv(node.Opcode(OpcodeHandshakeECDH)):
		buf = ctx.Bytes()
	}

	res, err := UnmarshalHandshake(buf)

	if err != nil {
		return nil, errors.Wrap(err, "ecdh: failed to unmarshal handshake response")
	}

	if !isEd25519GroupElement(res.publicKey) {
		return nil, errors.New("ecdh: failed to unmarshal our peers ephemeral public key")
	}

	if !edwards25519.Verify(res.publicKey, []byte(b.message), res.signature) {
		return nil, errors.New("ecdh: failed to verify signature in handshake request")
	}

	ephemeral = computeSharedKey(ephemeralPrivateKey, res.publicKey)
	fmt.Printf("Performed ECDH: %x\n", ephemeral)

	return
}
