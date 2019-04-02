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

const (
	SignalAuthenticated = "cipher.aead.authenticated"
	OpcodeAckAEAD       = "cipher.aead.ack"
)

type AEAD struct{}

func NewAEAD() *AEAD {
	return new(AEAD)
}

func (b *AEAD) RegisterOpcodes(n *noise.Node) {
	n.RegisterOpcode(OpcodeAckAEAD, n.NextAvailableOpcode())
}

func (b *AEAD) Setup(ephemeralSharedKey []byte, ctx noise.Context) error {
	node, peer := ctx.Node(), ctx.Peer()

	suite, sharedKey, err := deriveCipherSuite(Aes256Gcm, sha256.New, ephemeralSharedKey, nil)
	if err != nil {
		return errors.Wrap(err, "failed to derive AEAD cipher suite given ephemeral shared key")
	}

	signal := peer.RegisterSignal(SignalAuthenticated)
	defer signal()

	locker := peer.LockOnRecv(node.Opcode(OpcodeAckAEAD))
	defer locker.Unlock()

	if err = peer.Send(node.Opcode(OpcodeAckAEAD), nil); err != nil {
		return errors.Wrap(err, "failed to send AEAD ACK")
	}

	select {
	case <-ctx.Done():
		return noise.ErrDisconnect
	case <-time.After(3 * time.Second):
		return errors.Wrap(noise.ErrTimeout, "timed out waiting for AEAD ACK")
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

	fmt.Printf("Performed AEAD: %x\n", sharedKey)

	return nil
}
