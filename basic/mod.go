package basic

import (
	"encoding/hex"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/protocol"
	"sync"
)

var (
	// string --> protocol.ID
	peers sync.Map
)

// GetPeers returns a list of K peers with in order of ascending XOR distance.
func GetPeers(K int) (results []protocol.ID) {
	peers.Range(func(_, v interface{}) bool {
		if len(results) >= K {
			return false
		}
		results = append(results, v.(protocol.ID))
		return true
	})

	return results
}

func AddPeer(node *noise.Node, target protocol.ID) (err error) {
	peers.Store(hex.EncodeToString(target.Hash()), target)
	return nil
}

func DeletePeer(node *noise.Node, target protocol.ID) (err error) {
	peers.Delete(hex.EncodeToString(target.Hash()))
	return nil
}
