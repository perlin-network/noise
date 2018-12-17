package noise

import (
	"context"
	"github.com/perlin-network/noise/protocol"
)

// Startup is called only once when the service is loaded
func (n *Noise) Startup(node *protocol.Node) {
	for _, cb := range n.onStartup {
		cb(n.node.GetIdentityAdapter().MyIdentity())
	}
}

// Receive is called every time when messages are received
func (n *Noise) Receive(ctx context.Context, request *protocol.Message) (*protocol.MessageBody, error) {
	opcode := OpCode(request.Body.Service)
	if onReceive, ok := n.onReceive[opcode]; ok {
		for _, cb := range onReceive {
			reply, err := cb(ctx, (*Message)(request))
			if err != nil {
				return nil, err
			}
			if reply != nil {
				return (*protocol.MessageBody)(reply), nil
			}
		}
	}
	return nil, nil
}

// Cleanup is called only once after network stops listening
func (n *Noise) Cleanup(node *protocol.Node) {
	for _, cb := range n.onCleanup {
		cb(n.node.GetIdentityAdapter().MyIdentity())
	}
}

// PeerConnect is called every time a PeerClient is initialized and connected
func (n *Noise) PeerConnect(id []byte) {
	for _, cb := range n.onPeerConnect {
		cb(n.node.GetIdentityAdapter().MyIdentity())
	}
}

// PeerDisconnect is called every time a PeerClient connection is closed
func (n *Noise) PeerDisconnect(id []byte) {
	for _, cb := range n.onPeerDisconnect {
		cb(n.node.GetIdentityAdapter().MyIdentity())
	}
}
