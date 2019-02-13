package aead

import (
	"crypto/sha256"
	"encoding/binary"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
	"go.dedis.ch/kyber/v3/group/edwards25519"
	"hash"
	"sync/atomic"
	"time"
)

var (
	curve crypto.EllipticSuite = edwards25519.NewBlakeSHA256Ed25519()

	_ protocol.Block = (*block)(nil)
)

type block struct {
	opcodeACK noise.Opcode

	hash func() hash.Hash
}

func New() *block {
	return &block{hash: sha256.New}
}

func (b *block) WithHash(hash func() hash.Hash) *block {
	b.hash = hash
	return b
}

func (b *block) WithSuite(suite crypto.EllipticSuite) *block {
	curve = suite
	return b
}

func (b *block) OnRegister(p *protocol.Protocol, node *noise.Node) {
	b.opcodeACK = noise.RegisterMessage(noise.NextAvailableOpcode(), (*ACK)(nil))
}

func (b *block) OnBegin(p *protocol.Protocol, peer *noise.Peer) error {
	ephemeralSharedKeyBuf := protocol.LoadSharedKey(peer)

	if ephemeralSharedKeyBuf == nil {
		return errors.Wrap(protocol.DisconnectPeer, "session was established, but no ephemeral shared key found")
	}

	ephemeralSharedKey := curve.Point()

	err := ephemeralSharedKey.UnmarshalBinary(ephemeralSharedKeyBuf)
	if err != nil {
		return errors.Wrap(errors.Wrap(protocol.DisconnectPeer, err.Error()), "failed to unmarshal ephemeral shared key buf")
	}

	suite, sharedKey, err := deriveCipherSuite(b.hash, ephemeralSharedKey, nil)
	if err != nil {
		return errors.Wrap(errors.Wrap(protocol.DisconnectPeer, err.Error()), "failed to derive AEAD cipher suite given ephemeral shared key")
	}

	err = peer.SendMessage(ACK{})
	if err != nil {
		return errors.Wrap(errors.Wrap(protocol.DisconnectPeer, err.Error()), "failed to send AEAD ACK")
	}

	peer.EnterCriticalReadMode()
	defer peer.LeaveCriticalReadMode()

	select {
	case <-time.After(3 * time.Second):
		return errors.Wrap(protocol.DisconnectPeer, "timed out waiting for AEAD ACK")
	case <-peer.Receive(b.opcodeACK):
	}

	var ourNonce uint64
	var theirNonce uint64

	peer.BeforeMessageSent(func(node *noise.Node, peer *noise.Peer, msg []byte) (bytes []byte, e error) {
		ourNonceBuf := make([]byte, suite.NonceSize())
		binary.LittleEndian.PutUint64(ourNonceBuf, atomic.AddUint64(&ourNonce, 1))
		return suite.Seal(msg[:0], ourNonceBuf, msg, nil), nil
	})

	peer.BeforeMessageReceived(func(node *noise.Node, peer *noise.Peer, msg []byte) (bytes []byte, e error) {
		theirNonceBuf := make([]byte, suite.NonceSize())
		binary.LittleEndian.PutUint64(theirNonceBuf, atomic.AddUint64(&theirNonce, 1))
		return suite.Open(msg[:0], theirNonceBuf, msg, nil)
	})

	log.Debug().Hex("derived_shared_key", sharedKey).Msg("Derived HMAC, and successfully initialized session w/ AEAD cipher suite.")
	return nil
}

func (b *block) OnEnd(p *protocol.Protocol, peer *noise.Peer) error {

	return nil
}
