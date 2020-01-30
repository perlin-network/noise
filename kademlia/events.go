package kademlia

import "github.com/perlin-network/noise"

// Events comprise of callbacks that are to be called upon the encountering of various events as a node follows
// the Kademlia protocol. An Events declaration may be registered to a Protocol upon instantiation through calling
// New with the WithProtocolEvents functional option.
type Events struct {
	// OnPeerAdmitted is called when a peer is admitted to being inserted into your nodes' routing table.
	OnPeerAdmitted func(id noise.ID)

	// OnPeerActivity is called when your node interacts with a peer, causing the peer's entry in your nodes' routing
	// table to be bumped to the head of its respective bucket.
	OnPeerActivity func(id noise.ID)

	// OnPeerEvicted is called when your node fails to ping/dial a peer that was previously admitted into your nodes'
	// routing table, which leads to an eviction of the peers ID from your nodes' routing table.
	OnPeerEvicted func(id noise.ID)
}
