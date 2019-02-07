package basic

import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/callbacks"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/timeout"
	"github.com/pkg/errors"
	"sync"
	"time"
)

var (
	OpcodePing noise.Opcode
	OpcodePong noise.Opcode

	registerOpcodesOnce sync.Once

	_ protocol.NetworkPolicy = (*networkPolicy)(nil)
)

const (
	keyPingTimeoutDispatcher = "kademlia.timeout.ping"
)

type networkPolicy struct{}

func NewNetworkPolicy() *networkPolicy {
	return &networkPolicy{}
}

func (p *networkPolicy) EnforceNetworkPolicy(node *noise.Node) {
	registerOpcodesOnce.Do(func() {
		OpcodePing = noise.RegisterMessage(noise.NextAvailableOpcode(), (*Ping)(nil))
		OpcodePong = noise.RegisterMessage(noise.NextAvailableOpcode(), (*Pong)(nil))
	})
}

func (p *networkPolicy) OnSessionEstablished(node *noise.Node, peer *noise.Peer) error {
	peer.OnMessageReceived(OpcodePing, onReceivePing)

	peer.OnMessageReceived(OpcodePong, func(node *noise.Node, opcode noise.Opcode, peer *noise.Peer, message noise.Message) error {
		if err := timeout.Clear(peer, keyPingTimeoutDispatcher); err != nil {
			peer.Disconnect()
			return errors.Wrap(err, "error enforcing ping timeout policy")
		}

		return callbacks.DeregisterCallback
	})

	// Send a ping.
	err := peer.SendMessage(OpcodePing, Ping{})

	if err != nil {
		peer.Disconnect()
		return errors.Wrap(err, "failed to ping peer")
	}

	timeout.Enforce(peer, keyPingTimeoutDispatcher, 3*time.Second, peer.Disconnect)

	return callbacks.DeregisterCallback
}

// Send a pong.
func onReceivePing(node *noise.Node, opcode noise.Opcode, peer *noise.Peer, message noise.Message) error {
	err := peer.SendMessage(OpcodePong, Pong{})

	if err != nil {
		peer.Disconnect()
		return errors.Wrap(err, "failed to pong peer")
	}

	// Never de-register accepting pings.
	return nil
}
