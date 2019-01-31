package skademlia

import (
	"bytes"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/callbacks"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/rpc"
	"github.com/perlin-network/noise/timeout"
	"github.com/pkg/errors"
	"sort"
	"sync"
	"time"
)

var (
	OpcodePing           noise.Opcode
	OpcodePong           noise.Opcode
	OpcodeLookupRequest  noise.Opcode
	OpcodeLookupResponse noise.Opcode

	registerOpcodesOnce sync.Once

	_ protocol.NetworkPolicy = (*networkPolicy)(nil)
)

const (
	keyPingTimeoutDispatcher = "kademlia.timeout.ping"
)

type networkPolicy struct{}

func NewNetworkPolicy() *networkPolicy {
	return new(networkPolicy)
}

func (p networkPolicy) EnforceNetworkPolicy(node *noise.Node) {
	protocol.MustIdentityPolicy(node)

	registerOpcodesOnce.Do(func() {
		OpcodePing = noise.RegisterMessage(noise.NextAvailableOpcode(), (*Ping)(nil))
		OpcodePong = noise.RegisterMessage(noise.NextAvailableOpcode(), (*Pong)(nil))
		OpcodeLookupRequest = noise.RegisterMessage(noise.NextAvailableOpcode(), (*LookupRequest)(nil))
		OpcodeLookupResponse = noise.RegisterMessage(noise.NextAvailableOpcode(), (*LookupResponse)(nil))
	})
}

func (p networkPolicy) OnSessionEstablished(node *noise.Node, peer *noise.Peer) error {
	peer.OnMessageReceived(OpcodePing, onReceivePing)

	peer.OnMessageReceived(OpcodePong, func(node *noise.Node, opcode noise.Opcode, peer *noise.Peer, message noise.Message) error {
		if err := timeout.Clear(peer, keyPingTimeoutDispatcher); err != nil {
			peer.Disconnect()
			return errors.Wrap(err, "error enforcing ping timeout policy")
		}

		return callbacks.DeregisterCallback
	})

	rpc.OnRequestReceived(peer, OpcodeLookupRequest, onLookupRequest)

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

func onLookupRequest(peer *noise.Peer, message noise.Message) (noise.Message, error) {
	targetID, res := message.(LookupRequest), LookupResponse{}

	for _, peerID := range FindClosestPeers(Table(peer.Node()), targetID.Hash(), DefaultBucketSize) {
		res.peers = append(res.peers, peerID.(ID))
	}

	log.Info().Strs("addrs", Table(peer.Node()).GetPeers()).Msg("Connected to peer(s).")

	return res, nil
}

func queryPeerByID(node *noise.Node, peerID, targetID ID, responses chan []ID) {
	var err error

	if peerID.Equals(protocol.NodeID(node)) {
		responses <- []ID{}
		return
	}

	peer := protocol.Peer(node, peerID)

	if peer == nil {
		peer, err = node.Dial(peerID.address)

		if err != nil {
			responses <- []ID{}
			return
		}

		protocol.BlockUntilAuthenticated(peer)
	}

	res, err := rpc.Request(peer, 3*time.Second, OpcodeLookupRequest, LookupRequest{targetID})

	if err != nil {
		responses <- []ID{}
		return
	}

	responses <- res.(LookupResponse).peers
}

type lookupBucket struct {
	pending int
	queue   []ID
}

func (lookup *lookupBucket) performLookup(node *noise.Node, table *table, targetID ID, alpha int, visited *sync.Map) (results []ID) {
	responses := make(chan []ID)

	// Go through every peer in the entire queue and queue up what peers believe
	// is closest to a target ID.

	for ; lookup.pending < alpha && len(lookup.queue) > 0; lookup.pending++ {
		go queryPeerByID(node, lookup.queue[0], targetID, responses)

		lookup.queue = lookup.queue[1:]
	}

	// Empty queue.
	lookup.queue = lookup.queue[:0]

	// Asynchronous breadth-first search.
	for lookup.pending > 0 {
		response := <-responses

		lookup.pending--

		// Expand responses containing a peer's belief on the closest peers to target ID.
		for _, id := range response {
			if _, seen := visited.LoadOrStore(string(id.Hash()), struct{}{}); !seen {
				// Append new peer to be queued by the routing table.
				results = append(results, id)
				lookup.queue = append(lookup.queue, id)
			}
		}

		// Queue and request for #ALPHA closest peers to target ID from expanded results.
		for ; lookup.pending < alpha && len(lookup.queue) > 0; lookup.pending++ {
			go queryPeerByID(node, lookup.queue[0], targetID, responses)
			lookup.queue = lookup.queue[1:]
		}

		// Empty queue.
		lookup.queue = lookup.queue[:0]
	}

	return
}

// FindNode queries all peers this current node acknowledges for the closest peers
// to a specified target ID.
//
// All lookups are done under a number of disjoint lookups in parallel.
//
// Queries at most #ALPHA nodes at a time per lookup, and returns all peer IDs closest to a target peer ID.
func FindNode(node *noise.Node, targetID ID, alpha int, disjointPaths int) (results []ID) {
	table, visited := Table(node), new(sync.Map)

	visited.Store(string(protocol.NodeID(node).Hash()), struct{}{})
	visited.Store(string(targetID.Hash()), struct{}{})

	var lookups []*lookupBucket

	// Start searching for target from #ALPHA peers closest to target by queuing
	// them up and marking them as visited.
	for i, peerID := range FindClosestPeers(table, targetID.Hash(), alpha) {
		visited.Store(string(peerID.Hash()), struct{}{})

		if len(lookups) < disjointPaths {
			lookups = append(lookups, new(lookupBucket))
		}

		lookup := lookups[i%disjointPaths]
		lookup.queue = append(lookup.queue, peerID.(ID))

		results = append(results, peerID.(ID))
	}

	wait, mutex := new(sync.WaitGroup), new(sync.Mutex)

	for _, lookup := range lookups {
		go func(lookup *lookupBucket) {
			mutex.Lock()
			results = append(results, lookup.performLookup(node, table, targetID, alpha, visited)...)
			mutex.Unlock()

			wait.Done()
		}(lookup)

		wait.Add(1)
	}

	// Wait until all #D parallel lookups have been completed.
	wait.Wait()

	// Sort resulting peers by XOR distance.
	sort.Slice(results, func(i, j int) bool {
		return bytes.Compare(xor(results[i].Hash(), targetID.Hash()), xor(results[j].Hash(), targetID.Hash())) == -1
	})

	// Cut off list of results to only have the routing table focus on the
	// #BucketSize closest peers to the current node.
	if len(results) > DefaultBucketSize {
		results = results[:DefaultBucketSize]
	}

	return
}
