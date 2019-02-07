package basic

import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/callbacks"
	"github.com/perlin-network/noise/protocol"
)

var (
	_ protocol.NetworkPolicy = (*networkPolicy)(nil)
)

type networkPolicy struct {
}

func NewNetworkPolicy() *networkPolicy {
	return &networkPolicy{}
}

func (p *networkPolicy) EnforceNetworkPolicy(node *noise.Node) {

}

func (p *networkPolicy) OnSessionEstablished(node *noise.Node, peer *noise.Peer) error {
	return callbacks.DeregisterCallback
}
