package skademlia

import (
	"bytes"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/callbacks"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/timeout"
	"github.com/pkg/errors"
	"sort"
	"time"
)

func Table(node *noise.Node) *table {
	t := node.Get(KeyKademliaTable)

	if t == nil {
		panic("kademlia: node has not enforced identity policy, and thus has no table associated to it")
	}

	if t, ok := t.(*table); ok {
		return t
	}

	panic("kademlia: table associated to node is not an instance of a kademlia table")
}

// FindClosestPeers returns a list of K peers with in order of ascending XOR distance.
func FindClosestPeers(t *table, target []byte, K int) (peers []protocol.ID) {
	bucketID := t.bucketID(xor(target, t.self.Hash()))
	bucket := t.bucket(bucketID)

	bucket.RLock()

	for e := bucket.Front(); e != nil; e = e.Next() {
		if !e.Value.(protocol.ID).Equals(t.self) {
			peers = append(peers, e.Value.(protocol.ID))
		}
	}

	bucket.RUnlock()

	for i := 1; len(peers) < K && (bucketID-i >= 0 || bucketID+i < len(t.self.Hash())*8); i++ {
		if bucketID-i >= 0 {
			other := t.bucket(bucketID - i)
			other.RLock()
			for e := other.Front(); e != nil; e = e.Next() {
				if !e.Value.(protocol.ID).Equals(t.self) {
					peers = append(peers, e.Value.(protocol.ID))
				}
			}
			other.RUnlock()
		}

		if bucketID+i < len(t.self.Hash())*8 {
			other := t.bucket(bucketID + i)
			other.RLock()
			for e := other.Front(); e != nil; e = e.Next() {
				if !e.Value.(protocol.ID).Equals(t.self) {
					peers = append(peers, e.Value.(protocol.ID))
				}
			}
			other.RUnlock()
		}
	}

	// Sort peers by XOR distance.
	sort.Slice(peers, func(i, j int) bool {
		return bytes.Compare(xor(peers[i].Hash(), target), xor(peers[j].Hash(), target)) == -1
	})

	if len(peers) > K {
		peers = peers[:K]
	}

	return peers
}

func UpdateTable(node *noise.Node, target protocol.ID) (err error) {
	table := Table(node)

	if err = table.Update(target); err != nil {
		switch err {
		case ErrBucketFull:
			bucket := table.bucket(table.bucketID(target.Hash()))

			last := bucket.Back()
			lastPeer := protocol.Peer(node, last.Value.(protocol.ID))

			if lastPeer == nil {
				return errors.New("kademlia: last peer in bucket was not actually connected to our node")
			}

			// If the candidate peer to-be-evicted responds with a pong, move him to the front of the bucket
			// and do not push the target ID into the bucket.
			//
			// Else, evict the candidate peer and push the target ID to the front of the bucket.
			lastPeer.OnMessageReceived(OpcodePong, func(node *noise.Node, opcode noise.Opcode, peer *noise.Peer, message noise.Message) error {
				bucket.MoveToFront(last)

				if err = timeout.Clear(lastPeer, keyPingTimeoutDispatcher); err != nil {
					lastPeer.Disconnect()
					return errors.Wrap(err, "error enforcing ping timeout policy")
				}

				return callbacks.DeregisterCallback
			})

			// Send a ping to the candidate peer to-be-evicted.
			err := lastPeer.SendMessage(OpcodePing, Ping{})

			evictLastPeer := func() {
				lastPeer.Disconnect()

				bucket.Remove(last)
				bucket.PushFront(target)
			}

			if err != nil {
				evictLastPeer()
				return nil
			}

			timeout.Enforce(lastPeer, keyPingTimeoutDispatcher, 3*time.Second, evictLastPeer)
		default:
			return err
		}
	}

	return nil
}
