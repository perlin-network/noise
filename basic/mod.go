package basic

import (
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/protocol"
	"sync"
)

var (
	// protocol.ID -> struct{}
	peers sync.Map
)

// GetPeers returns a list of K peers with in order of ascending XOR distance.
func GetPeers(K int) (results []protocol.ID) {
	peers.Range(func(k, _ interface{}) bool {
		if len(results) >= K {
			return false
		}
		results = append(results, k.(protocol.ID))
		return true
	})

	return results
}

func AddPeer(node *noise.Node, target protocol.ID) (err error) {
	peers.Store(target, struct{}{})
	return nil
}

func DeletePeer(node *noise.Node, target protocol.ID) (err error) {
	peers.Delete(target)
	return nil
}
