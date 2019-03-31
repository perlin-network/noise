package cipher

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/perlin-network/noise"
	"github.com/pkg/errors"
	"sync/atomic"
	"time"
)

type BuilderAEAD struct {
	ephemeralSharedKey []byte
}

func NewAEAD(ephemeralSharedKey []byte) *BuilderAEAD {
	return &BuilderAEAD{ephemeralSharedKey: ephemeralSharedKey}
}

func (b *BuilderAEAD) Setup(ctx noise.Context) error {
	suite, sharedKey, err := deriveCipherSuite(Aes256Gcm, sha256.New, b.ephemeralSharedKey, nil)
	if err != nil {
		return errors.Wrap(err, "failed to derive AEAD cipher suite given ephemeral shared key")
	}

	locker := ctx.Peer().LockOnRecv(0x02)
	defer locker.Unlock()

	if err = ctx.Peer().Send(0x02, nil); err != nil {
		return errors.Wrap(err, "failed to send AEAD ACK")
	}

	select {
	case <-ctx.Done():
		return noise.ErrDisconnect
	case <-time.After(3 * time.Second):
		return errors.Wrap(noise.ErrTimeout, "timed out waiting for AEAD ACK")
	case <-ctx.Peer().Recv(0x02):
	}

	codec := ctx.Peer().WireCodec()

	var ourNonce, theirNonce uint64

	codec.InterceptSend(func(buf []byte) ([]byte, error) {
		ourNonceBuf := make([]byte, suite.NonceSize())
		binary.LittleEndian.PutUint64(ourNonceBuf, atomic.AddUint64(&ourNonce, 1))

		return suite.Seal(nil, ourNonceBuf, buf, nil), nil
	})

	codec.InterceptRecv(func(buf []byte) ([]byte, error) {
		theirNonceBuf := make([]byte, suite.NonceSize())
		binary.LittleEndian.PutUint64(theirNonceBuf, atomic.AddUint64(&theirNonce, 1))

		return suite.Open(nil, theirNonceBuf, buf, nil)
	})

	fmt.Printf("Performed AEAD: %x\n", sharedKey)

	return nil
}
