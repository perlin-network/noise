package basic

import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/callbacks"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/timeout"
	"github.com/pkg/errors"
	"sync"
	"time"
)

const (
	keyTimeoutDispatcher = "basic.timeout"

	keyEphemeralPrivateKey = "basic.ephemeralPrivateKey"
	msgEphemeralHandshake  = ".noise_handshake"
)

var (
	OpcodeHandshake     noise.Opcode
	registerOpcodesOnce sync.Once

	_ protocol.HandshakePolicy = (*policy)(nil)
)

type policy struct {
	timeoutDuration time.Duration
}

func New() *policy {
	return &policy{}
}

func (p *policy) EnforceHandshakePolicy(node *noise.Node) {
	node.OnPeerInit(p.onPeerInit)
	node.OnPeerDisconnected(p.onPeerDisconnected)

}
func (p *policy) Opcodes() []noise.Opcode {
	// Register messages to Noise.
	registerOpcodesOnce.Do(func() {
		OpcodeHandshake = noise.RegisterMessage(noise.NextAvailableOpcode(), (*messageHandshake)(nil))
	})

	return []noise.Opcode{OpcodeHandshake}
}

func (p *policy) DoHandshake(peer *noise.Peer, opcode noise.Opcode, message noise.Message) error {
	if !peer.Has(keyEphemeralPrivateKey) {
		peer.Disconnect()
		return errors.New("peer attempted to perform handshake")
	}

	log.Debug().Msg("Successfully performed basic handshake with our peer.")

	//sharedKeyBuf := []byte("dummySharedKeyBuf")

	peer.Delete(keyEphemeralPrivateKey)
	//protocol.SetSharedKey(peer, sharedKeyBuf)

	if err := timeout.Clear(peer, keyTimeoutDispatcher); err != nil {
		peer.Disconnect()
		return errors.Wrap(err, "error enforcing handshake timeout policy")
	}

	return callbacks.DeregisterCallback
}

func (p *policy) onPeerInit(node *noise.Node, peer *noise.Peer) (err error) {
	if peer.Has(keyEphemeralPrivateKey) {
		peer.Disconnect()
		return errors.New("peer attempted to have us instantiate a 2nd handshake")
	}

	peer.Set(keyEphemeralPrivateKey, "dummyPrivateKey")

	msg := messageHandshake{
		publicKey: node.ID.PublicID(),
	}

	err = peer.SendMessage(OpcodeHandshake, msg)
	if err != nil {
		peer.Disconnect()
		return errors.Wrap(err, "failed to send our ephemeral public key to our peer")
	}

	timeout.Enforce(peer, keyTimeoutDispatcher, p.timeoutDuration, func() {
		log.Warn().Msg("PeerInit timed out")
		peer.Disconnect()
	})

	return nil
}

func (p *policy) onPeerDisconnected(node *noise.Node, peer *noise.Peer) error {
	peer.Delete(keyEphemeralPrivateKey)
	//protocol.DeleteSharedKey(peer)

	return callbacks.DeregisterCallback
}
