package aead

import (
	"crypto/sha256"
	"encoding/binary"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
	"hash"
	"sync/atomic"
	"time"
)

const keyAuthChannel = "aead.auth.ch"

var (
	_ protocol.Block = (*block)(nil)
)

type block struct {
	opcodeACK noise.Opcode

	ackTimeout time.Duration
	hash       func() hash.Hash
}

func New() *block {
	return &block{hash: sha256.New, ackTimeout: 3 * time.Second}
}

func (b *block) WithHash(hash func() hash.Hash) *block {
	b.hash = hash
	return b
}

func (b *block) WithACKTimeout(ackTimeout time.Duration) *block {
	b.ackTimeout = ackTimeout
	return b
}

func (b *block) OnRegister(p *protocol.Protocol, node *noise.Node) {
	b.opcodeACK = noise.RegisterMessage(noise.NextAvailableOpcode(), (*ACK)(nil))
}

func (b *block) OnBegin(p *protocol.Protocol, peer *noise.Peer) error {
	ephemeralSharedKey := protocol.LoadSharedKey(peer)

	if ephemeralSharedKey == nil {
		return errors.Wrap(protocol.DisconnectPeer, "session was established, but no ephemeral shared key found")
	}

	suite, sharedKey, err := deriveCipherSuite(b.hash, ephemeralSharedKey, nil)
	if err != nil {
		return errors.Wrap(errors.Wrap(protocol.DisconnectPeer, err.Error()), "failed to derive AEAD cipher suite given ephemeral shared key")
	}

	locker := peer.ReceiveAndLock(b.opcodeACK)
	defer locker.Unlock()

	err = peer.SendMessage(ACK{})
	if err != nil {
		return errors.Wrap(errors.Wrap(protocol.DisconnectPeer, err.Error()), "failed to send AEAD ACK")
	}

	select {
	case <-time.After(b.ackTimeout):
		return errors.Wrap(protocol.DisconnectPeer, "timed out waiting for AEAD ACK")
	case <-peer.Receive(b.opcodeACK):
	}

	var ourNonce uint64
	var theirNonce uint64

	peer.BeforeMessageReceived(func(node *noise.Node, peer *noise.Peer, msg []byte) (buf []byte, err error) {
		theirNonceBuf := make([]byte, suite.NonceSize())
		binary.LittleEndian.PutUint64(theirNonceBuf, atomic.AddUint64(&theirNonce, 1))

		buf, err = suite.Open(msg[:0], theirNonceBuf, msg, nil)

		if err != nil {
			panic(err)
		}

		return
	})

	peer.BeforeMessageSent(func(node *noise.Node, peer *noise.Peer, msg []byte) (buf []byte, err error) {
		ourNonceBuf := make([]byte, suite.NonceSize())
		binary.LittleEndian.PutUint64(ourNonceBuf, atomic.AddUint64(&ourNonce, 1))

		buf = suite.Seal(msg[:0], ourNonceBuf, msg, nil)

		return
	})

	log.Debug().Hex("derived_shared_key", sharedKey).Msg("Derived HMAC, and successfully initialized session w/ AEAD cipher suite.")

	close(peer.LoadOrStore(keyAuthChannel, make(chan struct{})).(chan struct{}))
	return nil
}

func (b *block) OnEnd(p *protocol.Protocol, peer *noise.Peer) error {
	return nil
}

func WaitUntilAuthenticated(peer *noise.Peer) {
	<-peer.LoadOrStore(keyAuthChannel, make(chan struct{})).(chan struct{})
}
