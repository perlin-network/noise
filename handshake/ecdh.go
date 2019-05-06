package handshake

import (
	"crypto"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/edwards25519"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"time"
)

const (
	SignalCompletedECDH = "handshake.ecdh.completed"
)

type ECDH struct {
	opcodeHandshake byte

	logger  *log.Logger
	message []byte
	timeout time.Duration
}

func NewECDH() *ECDH {
	return &ECDH{
		logger:  log.New(ioutil.Discard, "", 0),
		message: []byte(".noise_handshake_"),
		timeout: 3 * time.Second,
	}
}

func (b *ECDH) Logger() *log.Logger {
	return b.logger
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
	b.opcodeHandshake = n.Handle(n.NextAvailableOpcode(), nil)
}

func (b *ECDH) Protocol() noise.ProtocolBlock {
	return func(ctx noise.Context) error {
		ephemeral, err := b.Handshake(ctx)

		if err != nil {
			return err
		}

		ctx.Set(KeyEphemeralSharedKey, ephemeral)

		return nil
	}
}

func (b *ECDH) Handshake(ctx noise.Context) (ephemeral []byte, err error) {
	peer := ctx.Peer()

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

	if err := peer.Send(b.opcodeHandshake, req.Marshal()); err != nil {
		return nil, errors.Wrap(err, "ecdh: failed to request ephemeral public key to our peer")
	}

	var buf []byte

	select {
	case <-ctx.Done():
		return nil, noise.ErrDisconnect
	case buf = <-peer.Recv(b.opcodeHandshake):
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

	b.logger.Printf("Performed ECDH: %x\n", ephemeral)

	return
}
