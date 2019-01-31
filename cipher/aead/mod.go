package aead

import (
	"crypto/cipher"
	"crypto/sha256"
	"encoding/binary"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/callbacks"
	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/pkg/errors"
	"go.dedis.ch/kyber/v3/group/edwards25519"
	"hash"
)

const (
	keyCipherSuite = "aead.suite"
	keyOurNonce    = "aead.localNonce"
	keyTheirNonce  = "aead.remoteNonce"
)

var (
	curve crypto.EllipticSuite = edwards25519.NewBlakeSHA256Ed25519()

	_ protocol.CipherPolicy = (*policy)(nil)
)

type policy struct{ hash func() hash.Hash }

func New() policy {
	return policy{hash: sha256.New}
}

func (policy) WithHash(hash func() hash.Hash) policy {
	return policy{hash: hash}
}

func (p policy) WithSuite(suite crypto.EllipticSuite) policy {
	curve = suite
	return p
}

func (p policy) EnforceCipherPolicy(node *noise.Node) {
	// AEAD requires a handshake policy to yield an ephemeral shared key.
	protocol.MustHandshakePolicy(node)

	protocol.OnEachSessionEstablished(node, p.onSessionEstablished)
}

func (policy) Encrypt(peer *noise.Peer, buf []byte) ([]byte, error) {
	if !peer.Has(keyCipherSuite) {
		panic("noise: attempted to seal message via AEAD but no cipher suite registered")
	}

	ourNonce, ourNonceBuf := peer.Get(keyOurNonce).(uint64), make([]byte, 12)
	binary.LittleEndian.PutUint64(ourNonceBuf, ourNonce)

	// Increment our nonce by 1.
	peer.Set(keyOurNonce, ourNonce+1)

	suite := peer.Get(keyCipherSuite).(cipher.AEAD)

	return suite.Seal(buf[:0], ourNonceBuf, buf, nil), nil
}

func (policy) Decrypt(peer *noise.Peer, buf []byte) ([]byte, error) {
	if !peer.Has(keyCipherSuite) {
		panic("noise: attempted to seal message via AEAD but no cipher suite registered")
	}

	theirNonce, theirNonceBuf := peer.Get(keyTheirNonce).(uint64), make([]byte, 12)
	binary.LittleEndian.PutUint64(theirNonceBuf, theirNonce)

	// Increment their nonce by 1.
	peer.Set(keyTheirNonce, theirNonce+1)

	suite := peer.Get(keyCipherSuite).(cipher.AEAD)

	return suite.Open(buf[:0], theirNonceBuf, buf, nil)
}

func (p policy) onSessionEstablished(node *noise.Node, peer *noise.Peer) error {
	ephemeralSharedKeyBuf := protocol.LoadSharedKey(peer)

	if ephemeralSharedKeyBuf == nil {
		peer.Disconnect()
		return errors.New("session was established, but no ephemeral shared key found")
	}

	ephemeralSharedKey := curve.Point()

	err := ephemeralSharedKey.UnmarshalBinary(ephemeralSharedKeyBuf)
	if err != nil {
		peer.Disconnect()
		return errors.Wrap(err, "failed to unmarshal ephemeral shared key buf")
	}

	suite, sharedKey, err := deriveCipherSuite(p.hash, ephemeralSharedKey, nil)
	if err != nil {
		peer.Disconnect()
		return errors.Wrap(err, "failed to derive AEAD cipher suite given ephemeral shared key")
	}

	peer.Set(keyOurNonce, uint64(0))
	peer.Set(keyTheirNonce, uint64(0))
	peer.Set(keyCipherSuite, suite)

	protocol.SetSharedKey(peer, sharedKey)

	log.Debug().Hex("derived_shared_key", sharedKey).Msg("Derived HMAC, and successfully initialized session w/ AEAD cipher suite.")

	return callbacks.DeregisterCallback
}
