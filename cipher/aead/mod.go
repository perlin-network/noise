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
)

var (
	curve crypto.EllipticSuite = edwards25519.NewBlakeSHA256Ed25519()

	_ protocol.Block = (*block)(nil)
)

type block struct{ hash func() hash.Hash }

func New() block {
	return block{hash: sha256.New}
}

func (block) WithHash(hash func() hash.Hash) block {
	return block{hash: hash}
}

func (b block) WithSuite(suite crypto.EllipticSuite) block {
	curve = suite
	return b
}

func (b block) OnRegister(p *protocol.Protocol, node *noise.Node) {}

func (b block) OnBegin(p *protocol.Protocol, peer *noise.Peer) error {
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

	var ourNonce uint64
	var theirNonce uint64

	ourNonceBuf, theirNonceBuf := make([]byte, suite.NonceSize()), make([]byte, suite.NonceSize())

	// TODO(kenta): these callbacks cause a 'race condition' with any future messages sent. Using the `chat` example, try uncomment "aead.New()".
	peer.BeforeMessageSent(func(node *noise.Node, peer *noise.Peer, msg []byte) (bytes []byte, e error) {
		binary.LittleEndian.PutUint64(ourNonceBuf, atomic.AddUint64(&ourNonce, 1))
		return suite.Seal(msg[:0], ourNonceBuf, msg, nil), nil
	})

	peer.BeforeMessageReceived(func(node *noise.Node, peer *noise.Peer, msg []byte) (bytes []byte, e error) {
		binary.LittleEndian.PutUint64(theirNonceBuf, atomic.AddUint64(&theirNonce, 1))
		return suite.Open(msg[:0], theirNonceBuf, msg, nil)
	})

	log.Debug().Hex("derived_shared_key", sharedKey).Msg("Derived HMAC, and successfully initialized session w/ AEAD cipher suite.")

	return nil
}

func (b block) OnEnd(p *protocol.Protocol, peer *noise.Peer) error {

	return nil
}
