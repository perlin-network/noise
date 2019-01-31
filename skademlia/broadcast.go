package skademlia

import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/protocol"
)

func Broadcast(node *noise.Node, opcode noise.Opcode, message noise.Message) error {
	for _, peerID := range FindClosestPeers(Table(node), protocol.NodeID(node).Hash(), DefaultBucketSize) {
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
