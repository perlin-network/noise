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
	SignalHandshakeComplete = "handshake.ecdh"
	OpcodeHandshakeECDH     = "handshake.ecdh"
)

type ECDH struct {
	header  []byte
	timeout time.Duration
}

func NewECDH() *ECDH {
	return &ECDH{
		header:  []byte(".noise_handshake_"),
		timeout: 3 * time.Second,
	}
}

func (b *ECDH) TimeoutAfter(timeout time.Duration) *ECDH {
	b.timeout = timeout
	return b
}

func (b *ECDH) RegisterOpcodes(n *noise.Node) {
	n.RegisterOpcode(OpcodeHandshakeECDH, n.NextAvailableOpcode())
}

func (b *ECDH) Handshake(ctx noise.Context) (ephemeralSharedKey []byte, err error) {
	node, peer := ctx.Node(), ctx.Peer()

	signal := peer.RegisterSignal(SignalHandshakeComplete)
	defer signal()

	ephemeralPublicKey, ephemeralPrivateKey, err := edwards25519.GenerateKey(nil)

	if err != nil {
		return nil, errors.New("failed to generate ephemeral keypair")
	}

	req := Handshake{publicKey: ephemeralPublicKey}

	if req.signature, err = ephemeralPrivateKey.Sign(b.header, crypto.Hash(0)); err != nil {
		return nil, errors.Wrap(err, "failed to sign handshake message")
	}

	if err = peer.SendWithTimeout(node.Opcode(OpcodeHandshakeECDH), req.Marshal(), b.timeout); err != nil {
		return nil, errors.Wrap(err, "failed to send our ephemeral public key to our peer")
	}

	var buf []byte

	select {
	case <-ctx.Done():
		return nil, noise.ErrDisconnect
	case <-time.After(b.timeout):
		return nil, errors.Wrap(noise.ErrTimeout, "timed out receiving handshake response")
	case ctx := <-peer.Recv(node.Opcode(OpcodeHandshakeECDH)):
		buf = ctx.Bytes()
	}

	res, err := UnmarshalHandshake(buf)

	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal handshake response")
	}

	if !isEd25519GroupElement(res.publicKey) {
		return nil, errors.New("failed to unmarshal our peers ephemeral public key")
	}

	if !edwards25519.Verify(res.publicKey, []byte(b.header), res.signature) {
		return nil, errors.New("failed to verify signature in handshake request")
	}

	ephemeralSharedKey = computeSharedKey(ephemeralPrivateKey, res.publicKey)
	fmt.Printf("Performed ECDH: %x\n", ephemeralSharedKey)

	return
}
