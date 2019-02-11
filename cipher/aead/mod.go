package aead

import (
	"crypto/sha256"
	"encoding/binary"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/payload"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
	"go.dedis.ch/kyber/v3/group/edwards25519"
	"hash"
)

var (
	curve crypto.EllipticSuite = edwards25519.NewBlakeSHA256Ed25519()

	_ protocol.Block = (*block)(nil)
)

type block struct{ hash func() hash.Hash }

var OpcodeAeadFence noise.Opcode

type messageAeadFence struct{}

func (messageAeadFence) Read(reader payload.Reader) (noise.Message, error) {
	return &messageAeadFence{}, nil
}

func (messageAeadFence) Write() []byte {
	return nil
}

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

func (b block) OnRegister(p *protocol.Protocol, node *noise.Node) {
	OpcodeAeadFence = noise.RegisterMessage(noise.NextAvailableOpcode(), (*messageAeadFence)(nil))
}

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

	peer.SendMessage(OpcodeAeadFence, &messageAeadFence{})
	peer.Receive(OpcodeAeadFence, func() {
		peer.BeforeMessageSent(func(node *noise.Node, peer *noise.Peer, msg []byte) (bytes []byte, e error) {
			ourNonce++
			binary.LittleEndian.PutUint64(ourNonceBuf, ourNonce)
			return suite.Seal(msg[:0], ourNonceBuf, msg, nil), nil
		})

		peer.BeforeMessageReceived(func(node *noise.Node, peer *noise.Peer, msg []byte) (bytes []byte, e error) {
			theirNonce++
			binary.LittleEndian.PutUint64(theirNonceBuf, theirNonce)
			return suite.Open(msg[:0], theirNonceBuf, msg, nil)
		})
	})

	log.Debug().Hex("derived_shared_key", sharedKey).Msg("Derived HMAC, and successfully initialized session w/ AEAD cipher suite.")

	return nil
}

func (b block) OnEnd(p *protocol.Protocol, peer *noise.Peer) error {

	return nil
}
