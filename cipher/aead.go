package cipher

import (
	"crypto/cipher"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/handshake"
	"github.com/pkg/errors"
	"sync/atomic"
	"time"
)

const (
	SignalReadyAEAD = "cipher.aead.authenticated"
	OpcodeAckAEAD   = "cipher.aead.ack"
)

type AEAD struct {
	timeout time.Duration
}

func NewAEAD() *AEAD {
	return &AEAD{timeout: 3 * time.Second}
}

func (b *AEAD) TimeoutAfter(timeout time.Duration) *AEAD {
	b.timeout = timeout
	return b
}

func (b *AEAD) RegisterOpcodes(n *noise.Node) {
	n.RegisterOpcode(OpcodeAckAEAD, n.NextAvailableOpcode())
}

func (b *AEAD) Protocol() noise.ProtocolBlock {
	return func(ctx noise.Context) error {
		ephemeral := ctx.Get(handshake.KeyEphemeral)

		if ephemeral == nil {
			return errors.New("aead: expected peer to have ephemeral key set")
		}

		if _, ok := ephemeral.([]byte); !ok {
			return errors.New("aead: ephemeral key must be a byte slice")
		}

		symmetric, err := b.Setup(ephemeral.([]byte), ctx)

		if err != nil {
			return err
		}

		ctx.Set(KeySuite, symmetric)

		return nil
	}
}

func (b *AEAD) Setup(ephemeralSharedKey []byte, ctx noise.Context) (cipher.AEAD, error) {
	node, peer := ctx.Node(), ctx.Peer()

	suite, symmetric, err := deriveCipherSuite(Aes256Gcm, sha256.New, ephemeralSharedKey, nil)
	if err != nil {
		return nil, errors.Wrap(err, "aead: failed to derive suite given ephemeral shared key")
	}

	signal := peer.RegisterSignal(SignalReadyAEAD)
	defer signal()

	locker := peer.LockOnRecv(node.Opcode(OpcodeAckAEAD))
	defer locker.Unlock()

	if err = peer.Send(node.Opcode(OpcodeAckAEAD), nil); err != nil {
		return suite, errors.Wrap(err, "aead: failed to send ACK")
	}

	select {
	case <-ctx.Done():
		return suite, noise.ErrDisconnect
	case <-time.After(b.timeout):
		return suite, errors.Wrap(noise.ErrTimeout, "aead: timed out waiting for ACK")
	case <-peer.Recv(node.Opcode(OpcodeAckAEAD)):
	}

	codec := peer.WireCodec()

	var ourNonce, theirNonce uint64
	ourNonceBuf := make([]byte, suite.NonceSize())
	theirNonceBuf := make([]byte, suite.NonceSize())

	codec.InterceptSend(func(buf []byte) ([]byte, error) {
		binary.LittleEndian.PutUint64(ourNonceBuf, atomic.AddUint64(&ourNonce, 1))
		return suite.Seal(buf[:0], ourNonceBuf, buf, nil), nil
	})

	codec.InterceptRecv(func(buf []byte) ([]byte, error) {
		binary.LittleEndian.PutUint64(theirNonceBuf, atomic.AddUint64(&theirNonce, 1))
		return suite.Open(buf[:0], theirNonceBuf, buf, nil)
	})

	fmt.Printf("Performed AEAD: %x\n", symmetric)

	return suite, nil
}
