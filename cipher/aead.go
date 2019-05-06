package cipher

import (
	"crypto/cipher"
	"crypto/sha256"
	"encoding/binary"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/handshake"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"sync/atomic"
	"time"
)

const (
	SignalReadyAEAD = "cipher.aead.authenticated"
)

type AEAD struct {
	opcodeACK byte

	logger  *log.Logger
	timeout time.Duration
}

func NewAEAD() *AEAD {
	return &AEAD{logger: log.New(ioutil.Discard, "", 0), timeout: 3 * time.Second}
}

func (b *AEAD) Logger() *log.Logger {
	return b.logger
}

func (b *AEAD) TimeoutAfter(timeout time.Duration) *AEAD {
	b.timeout = timeout
	return b
}

func (b *AEAD) RegisterOpcodes(n *noise.Node) {
	b.opcodeACK = n.Handle(n.NextAvailableOpcode(), nil)
}

func (b *AEAD) Protocol() noise.ProtocolBlock {
	return func(ctx noise.Context) error {
		ephemeral := ctx.Get(handshake.KeyEphemeralSharedKey)

		if ephemeral == nil {
			return errors.New("aead: expected peer to have ephemeral key set")
		}

		if _, ok := ephemeral.([]byte); !ok {
			return errors.New("aead: ephemeral key must be a byte slice")
		}

		suite, err := b.Setup(ephemeral.([]byte), ctx)

		if err != nil {
			return err
		}

		ctx.Set(KeySuite, suite)

		return nil
	}
}

func (b *AEAD) Setup(ephemeralSharedKey []byte, ctx noise.Context) (cipher.AEAD, error) {
	peer := ctx.Peer()

	suite, symmetric, err := deriveCipherSuite(Aes256Gcm, sha256.New, ephemeralSharedKey, nil)
	if err != nil {
		return nil, errors.Wrap(err, "aead: failed to derive suite given ephemeral shared key")
	}

	signal := peer.RegisterSignal(SignalReadyAEAD)
	defer signal()

	unlock := peer.LockOnRecv(b.opcodeACK)
	defer unlock()

	if err = peer.SendAwait(b.opcodeACK, nil); err != nil {
		return nil, errors.Wrap(err, "aead: failed to send ACK")
	}

	select {
	case <-ctx.Done():
		return nil, noise.ErrDisconnect
	case <-peer.Recv(b.opcodeACK):
	}

	var ourNonce, theirNonce uint64

	ourNonceBuf := make([]byte, suite.NonceSize())
	theirNonceBuf := make([]byte, suite.NonceSize())

	peer.InterceptSend(func(buf []byte) ([]byte, error) {
		binary.LittleEndian.PutUint64(ourNonceBuf, atomic.AddUint64(&ourNonce, 1))
		return suite.Seal(buf[:0], ourNonceBuf, buf, nil), nil
	})

	peer.InterceptRecv(func(buf []byte) ([]byte, error) {
		binary.LittleEndian.PutUint64(theirNonceBuf, atomic.AddUint64(&theirNonce, 1))
		return suite.Open(buf[:0], theirNonceBuf, buf, nil)
	})

	b.logger.Printf("Performed AEAD: %x\n", symmetric)

	return suite, nil
}
