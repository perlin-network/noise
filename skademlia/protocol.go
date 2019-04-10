package skademlia

import (
	"bytes"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/edwards25519"
	"github.com/phf/go-queue/queue"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"io"
	"io/ioutil"
	"log"
	"net"
	"sort"
	"sync"
	"time"
)

const (
	DefaultPrefixDiffLen = 128
	DefaultPrefixDiffMin = 32

	DefaultC1 = 16
	DefaultC2 = 16

	SignalAuthenticated = "skademlia.authenticated"

	OpcodePing   = "skademlia.ping"
	OpcodeLookup = "skademlia.lookup"

	KeyID = "skademlia.id"
)

type Protocol struct {
	logger *log.Logger

	table *Table
	keys  *Keypair

	dialer noise.Dialer

	prefixDiffLen int
	prefixDiffMin int

	c1, c2 int

	handshakeTimeout time.Duration
	lookupTimeout    time.Duration

	peers     map[[blake2b.Size256]byte]*noise.Peer
	peersLock sync.Mutex
}

func New(address string, keys *Keypair, dialer noise.Dialer) *Protocol {
	return &Protocol{
		logger: log.New(ioutil.Discard, "", 0),

		table: NewTable(keys.ID(address)),
		keys:  keys,

		dialer: dialer,

		prefixDiffLen: DefaultPrefixDiffLen,
		prefixDiffMin: DefaultPrefixDiffMin,

		c1: DefaultC1,
		c2: DefaultC2,

		handshakeTimeout: 3 * time.Second,
		lookupTimeout:    3 * time.Second,

		peers: make(map[[blake2b.Size256]byte]*noise.Peer),
	}
}

func (b *Protocol) Logger() *log.Logger {
	return b.logger
}

func (b *Protocol) WithC1(c1 int) *Protocol {
	b.c1 = c1
	return b
}

func (b *Protocol) WithC2(c2 int) *Protocol {
	b.c2 = c2
	return b
}

func (b *Protocol) WithPrefixDiffLen(prefixDiffLen int) *Protocol {
	b.prefixDiffLen = prefixDiffLen
	return b
}

func (b *Protocol) WithPrefixDiffMin(prefixDiffMin int) *Protocol {
	b.prefixDiffMin = prefixDiffMin
	return b
}

func (b *Protocol) WithHandshakeTimeout(handshakeTimeout time.Duration) *Protocol {
	b.handshakeTimeout = handshakeTimeout
	return b
}

func (b *Protocol) Peers(node *noise.Node) (peers []*noise.Peer) {
	ids := b.table.FindClosest(b.table.self, b.table.getBucketSize())

	for _, id := range ids {
		if peer := b.PeerByID(node, id); peer != nil {
			peers = append(peers, peer)
		}
	}

	return
}

func (b *Protocol) PeerByID(node *noise.Node, id *ID) *noise.Peer {
	if id.address == b.table.self.address {
		return nil
	}

	b.peersLock.Lock()
	peer, recorded := b.peers[id.checksum]
	b.peersLock.Unlock()

	if recorded {
		return peer
	}

	peer = node.PeerByAddr(id.address)

	if peer != nil {
		return peer
	}

	peer, err := b.dialer(node, id.address)

	if err != nil {
		b.evict(id)
		return nil
	}

	return peer
}

func wrap(f func() error) {
	_ = f()
}

func (b *Protocol) RegisterOpcodes(n *noise.Node) {
	n.RegisterOpcode(OpcodePing, n.NextAvailableOpcode())
	n.RegisterOpcode(OpcodeLookup, n.NextAvailableOpcode())
}

func (b *Protocol) Protocol() noise.ProtocolBlock {
	return func(ctx noise.Context) error {
		id, err := b.Handshake(ctx)

		if err != nil {
			return err
		}

		ctx.Set(KeyID, id)

		return nil
	}
}

func (b *Protocol) Handshake(ctx noise.Context) (*ID, error) {
	signal := ctx.Peer().RegisterSignal(SignalAuthenticated)
	defer signal()

	go func() {
		node, peer := ctx.Node(), ctx.Peer()

		for {
			select {
			case <-ctx.Done():
				return
			case wire := <-peer.Recv(node.Opcode(OpcodePing)):
				id := b.table.self.Marshal()
				signature := edwards25519.Sign(b.keys.privateKey, id)

				if err := wire.Send(node.Opcode(OpcodePing), append(id, signature[:]...)); err != nil {
					ctx.Peer().Disconnect(errors.Wrap(err, "skademlia: failed to send ping"))
				}
			case wire := <-peer.Recv(node.Opcode(OpcodeLookup)):
				target, err := UnmarshalID(bytes.NewReader(wire.Bytes()))

				if err != nil {
					ctx.Peer().Disconnect(errors.Wrap(err, "skademlia: received invalid lookup request"))
				}

				if err := wire.Send(node.Opcode(OpcodeLookup), b.table.FindClosest(&target, b.table.getBucketSize()).Marshal()); err != nil {
					ctx.Peer().Disconnect(errors.Wrap(err, "skademlia: failed to send lookup response"))
				}
			}
		}
	}()

	/*
		From here on out:

		1. Check if our current connection has the same address recorded in the ID.

		3. If not, establish a connection to the address recorded in the ID and
			see if its reachable (in other words, ping the address).
		4. If not reachable, disconnect the peer.

		5. Else, deregister the ping by disconnecting the connection created by the ping.

		6. Attempt to register the ID into our routing table.

		7. Should any errors occur on the connection lead to a disconnection,
			dissociate this connection from the ID.
	*/

	id, err := b.Ping(ctx)

	if err != nil {
		return nil, err
	}

	err = func() error {
		b.peersLock.Lock()
		_, existed := b.peers[id.checksum]
		 b.peersLock.Unlock()

		if !existed && ctx.Peer().Addr().String() != id.address {
			reachable := b.PeerByID(ctx.Node(), id)

			if reachable == nil {
				return noise.ErrTimeout
			}

			reachable.Disconnect(nil)
		}

		b.peers[id.checksum] = ctx.Peer()
		return nil
	}()

	if err != nil {
		return nil, err
	}

	if err := b.Update(id); err != nil {
		return nil, err
	}

	ctx.Peer().InterceptErrors(func(err error) {
		b.peersLock.Lock()
		delete(b.peers, id.checksum)
		b.peersLock.Unlock()

		if err, ok := err.(net.Error); ok && err.Timeout() {
			b.evict(id)
			return
		}

		if errors.Cause(err) == noise.ErrTimeout {
			b.evict(id)
			return
		}
	})

	ctx.Peer().AfterRecv(func() {
		if err := b.Update(id); err != nil {
			ctx.Peer().Disconnect(err)
		}
	})

	b.logger.Printf("Registered to S/Kademlia: %s\n", id)

	return id, nil
}

func (b *Protocol) Ping(ctx noise.Context) (*ID, error) {
	node, peer := ctx.Node(), ctx.Peer()

	mux := peer.Mux()
	defer wrap(mux.Close)

	if err := mux.Send(node.Opcode(OpcodePing), nil); err != nil {
		return nil, errors.Wrap(err, "skademlia: failed to send ping")
	}

	r := bytes.NewReader(nil)

	select {
	case <-ctx.Done():
		return nil, noise.ErrDisconnect
	case <-time.After(b.handshakeTimeout):
		return nil, errors.Wrap(noise.ErrTimeout, "skademlia: timed out receiving pong")
	case ctx := <-mux.Recv(node.Opcode(OpcodePing)):
		r.Reset(ctx.Bytes())
	}

	id, err := UnmarshalID(r)

	if err != nil {
		return nil, errors.Wrap(err, "skademlia: failed to unmarshal pong")
	}

	var signature edwards25519.Signature

	n, err := io.ReadFull(r, signature[:])

	if err != nil {
		return nil, errors.Wrap(err, "skademlia: failed to read signature")
	}

	if n != edwards25519.SizeSignature {
		return nil, errors.New("skademlia: did not read enough bytes for signature")
	}

	if !edwards25519.Verify(id.publicKey, id.Marshal(), signature) {
		return nil, errors.New("skademlia: got invalid signature for pong")
	}

	if err := verifyPuzzle(id.checksum, id.nonce, b.c1, b.c2); err != nil {
		return nil, errors.Wrap(err, "skademlia: peer connected with invalid id")
	}

	if prefixDiff(b.table.self.checksum[:], id.checksum[:], b.prefixDiffLen) < b.prefixDiffMin {
		return nil, errors.New("skademlia: peer id is too similar to ours")
	}

	return &id, err
}

func (b *Protocol) Lookup(ctx noise.Context, target *ID) (IDs, error) {
	node, peer := ctx.Node(), ctx.Peer()

	mux := peer.Mux()
	defer wrap(mux.Close)

	if err := mux.Send(node.Opcode(OpcodeLookup), target.Marshal()); err != nil {
		return nil, errors.Wrap(err, "skademlia: failed to send find node request")
	}

	var buf []byte

	select {
	case <-ctx.Done():
		return nil, noise.ErrDisconnect
	case <-time.After(b.lookupTimeout):
		return nil, errors.Wrap(noise.ErrTimeout, "skademlia: timed out receiving find node response")
	case ctx := <-mux.Recv(node.Opcode(OpcodeLookup)):
		buf = ctx.Bytes()
	}

	return UnmarshalIDs(bytes.NewReader(buf))
}

func (b *Protocol) Update(id *ID) error {
	for b.table.Update(id) == ErrBucketFull {
		bucket := b.table.buckets[getBucketID(b.table.self.checksum, id.checksum)]

		bucket.Lock()
		b.peersLock.Lock()

		last := bucket.Back()
		lastid := last.Value.(*ID)
		lastp, exists := b.peers[lastid.checksum]

		b.peersLock.Unlock()
		bucket.Unlock()

		if !exists {
			b.table.Delete(bucket, lastid)
			continue
		}

		pid, err := b.Ping(lastp.Ctx())

		if err != nil { // Failed to ping peer at back of bucket.
			lastp.Disconnect(errors.Wrap(noise.ErrTimeout, "skademlia: failed to ping last peer in bucket"))
			continue
		}

		if pid.checksum != lastid.checksum || pid.nonce != lastid.nonce || pid.address != lastid.address { // Failed to authenticate peer at back of bucket.
			lastp.Disconnect(errors.Wrap(noise.ErrTimeout, "skademlia: got invalid id pinging last peer in bucket"))
			continue
		}

		b.logger.Printf("Routing table is full; evicting peer %s.\n", id)

		return errors.Wrap(noise.ErrDisconnect, "skademlia: cannot evict any peers to make room for new peer")
	}

	return nil
}

func (b *Protocol) Bootstrap(node *noise.Node) (results []*ID) {
	return b.FindNode(node, b.table.self, b.table.getBucketSize(), 3, 8)
}

func (b *Protocol) FindNode(node *noise.Node, target *ID, k int, a int, d int) (results []*ID) {
	type request ID

	type response struct {
		requestee *request
		ids       []*ID
	}

	var mu sync.Mutex

	visited := map[[blake2b.Size256]byte]struct{}{
		b.table.self.checksum: {},
		target.checksum:       {},
	}

	lookups := make([]queue.Queue, d)

	for i, id := range b.table.FindClosest(target, k) {
		visited[id.checksum] = struct{}{}
		lookups[i%d].PushBack(id)
	}

	var wg sync.WaitGroup
	wg.Add(d)

	for _, lookup := range lookups { // Perform d parallel disjoint lookups.
		go func(lookup queue.Queue) {
			requests := make(chan *request, a)
			responses := make(chan *response, a)

			for i := 0; i < a; i++ { // Perform α queries in parallel per disjoint lookup.
				go func() {
					for id := range requests {
						peer := b.PeerByID(node, (*ID)(id))

						if peer == nil {
							responses <- nil
							continue
						}

						ids, err := b.Lookup(peer.Ctx(), (*ID)(id))

						if err != nil {
							responses <- nil
							peer.Disconnect(err)
							continue
						}

						responses <- &response{requestee: id, ids: ids}
					}
				}()
			}

			pending := 0

			for lookup.Len() > 0 || pending > 0 {
				for lookup.Len() > 0 && len(requests) < cap(requests) {
					requests <- (*request)(lookup.PopFront().(*ID))
					pending++
				}

				if pending > 0 {
					res := <-responses

					if res != nil {
						for _, id := range res.ids {
							mu.Lock()
							if _, seen := visited[id.checksum]; !seen {
								visited[id.checksum] = struct{}{}
								lookup.PushBack(id)
							}
							mu.Unlock()
						}

						mu.Lock()
						results = append(results, (*ID)(res.requestee))
						mu.Unlock()
					}

					pending--
				}
			}

			close(requests)

			wg.Done()
		}(lookup)
	}

	wg.Wait() // Wait until all d parallel disjoint lookups are complete.

	sort.Slice(results, func(i, j int) bool {
		return bytes.Compare(xor(results[i].checksum[:], target.checksum[:]), xor(results[j].checksum[:], target.checksum[:])) == -1
	})

	if len(results) > k {
		results = results[:k]
	}

	return
}

func (b *Protocol) evict(id *ID) {
	b.logger.Printf("Peer %s could not be reached, and has been evicted.\n", id)

	bucket := b.table.buckets[getBucketID(b.table.self.checksum, id.checksum)]
	b.table.Delete(bucket, id)
}
