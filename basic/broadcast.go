package basic

import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/protocol"
)

const (
	defaultBroadcastSize = 16
)

func Broadcast(node *noise.Node, opcode noise.Opcode, message noise.Message) error {
	for _, peerID := range GetPeers(defaultBroadcastSize) {
		peer := protocol.Peer(node, peerID)

		if peer == nil {
			continue
		}

		if err := peer.SendMessage(opcode, message); err != nil {
			return err
		}
	}

	return nil
}
