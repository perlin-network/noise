// Package kademlia is a noise implementation of the routing and discovery portion of the Kademlia protocol, with
// minor improvements suggested by the S/Kademlia paper.
package kademlia

import (
	"context"
	"errors"
	"fmt"
	"github.com/perlin-network/noise"
	"time"
)

// BucketSize returns the capacity, or the total number of peer ID entries a single routing table bucket may hold.
const BucketSize int = 16

// ErrBucketFull is returned when a routing table bucket is at max capacity.
var ErrBucketFull = errors.New("bucket is full")

// Protocol implements noise.Protocol.
var _ noise.Protocol = (*Protocol)(nil)

// Protocol implements routing/discovery portion of the Kademlia protocol with improvements suggested by the
// S/Kademlia paper. It is expected that Protocol is bound to a noise.Node via (*noise.Node).Bind before the node
// starts listening for incoming peers.
type Protocol struct {
	node  *noise.Node
	table *Table
}

// NewProtocol returns a new instance of Kademlia.
func NewProtocol() *Protocol {
	return &Protocol{}
}

// Find executes the FIND_NODE S/Kademlia RPC call to find the closest peers to some given target public key. It
// returns the IDs of the closest peers it finds.
func (p *Protocol) Find(target noise.PublicKey, opts ...IteratorOption) []noise.ID {
	return NewIterator(p.node, p.table, opts...).Find(target)
}

// Discover attempts to discover new peers to your node through peers your node  already knows about by calling
// the FIND_NODE S/Kademlia RPC call with your nodes ID.
func (p *Protocol) Discover(opts ...IteratorOption) []noise.ID {
	return p.Find(p.node.ID().ID, opts...)
}

// Ping sends a ping request to addr, and returns no error if a pong is received back before ctx has expired/was
// cancelled. It also throws an error if the connection to addr intermittently drops, or if handshaking with addr
// should there be no live connection to addr yet fails.
func (p *Protocol) Ping(ctx context.Context, addr string) error {
	msg, err := p.node.RequestMessage(ctx, addr, Ping{})
	if err != nil {
		return fmt.Errorf("failed to ping: %w", err)
	}

	if _, ok := msg.(Pong); !ok {
		return errors.New("did not get a pong back")
	}

	return nil
}

// Bind implements noise.Protocol and registers messages Ping, Pong, FindNodeRequest, FindNodeResponse, and
// handles them by register the (*Protocol).Handle Handler.
func (p *Protocol) Bind(node *noise.Node) error {
	p.node = node
	p.table = NewTable(p.node.ID())

	node.RegisterMessage(Ping{}, UnmarshalPing)
	node.RegisterMessage(Pong{}, UnmarshalPong)
	node.RegisterMessage(FindNodeRequest{}, UnmarshalFindNodeRequest)
	node.RegisterMessage(FindNodeResponse{}, UnmarshalFindNodeResponse)

	node.Handle(p.Handle)

	return nil
}

// Table returns this Kademlia overlay's routing table from your nodes perspective.
func (p *Protocol) Table() *Table {
	return p.table
}

// Ack attempts to insert a peer ID into your nodes routing table. If the routing table bucket in which your peer ID
// was expected to be inserted on is full, the peer ID at the tail of the bucket is pinged. If the ping fails, the
// peer ID at the tail of the bucket is evicted and your peer ID is inserted to the head of the bucket.
func (p *Protocol) Ack(id noise.ID) {
	for {
		if err := p.table.Update(id); err == nil {
			return
		}

		bucket := p.table.Bucket(id.ID)
		last := bucket[len(bucket)-1]

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		pong, err := p.node.RequestMessage(ctx, last.Address, Ping{})
		cancel()

		if err != nil {
			p.table.Delete(last.ID)
			continue
		}

		if _, ok := pong.(Pong); !ok {
			p.table.Delete(last.ID)
			continue
		}

		return
	}
}

// OnPeerJoin implements noise.Protocol and attempts to acknowledge the new peers existence by placing its
// entry into your nodes' routing table via (*Protocol).Ack.
func (p *Protocol) OnPeerJoin(client *noise.Client) {
	p.Ack(client.ID())
}

// OnPeerLeave implements noise.Protocol and does nothing.
func (p *Protocol) OnPeerLeave(client *noise.Client) {
}

// OnMessageSent implements noise.Protocol and attempts to push the position in which the clients ID resides in
// your nodes' routing table's to the head of the bucket it reside within.
func (p *Protocol) OnMessageSent(client *noise.Client) {
	p.Ack(client.ID())
}

// OnMessageRecv implements noise.Protocol and attempts to push the position in which the clients ID resides in
// your nodes' routing table's to the head of the bucket it reside within.
func (p *Protocol) OnMessageRecv(client *noise.Client) {
	p.Ack(client.ID())
}

// Handle implements noise.Protocol and handles Ping and FindNodeRequest messages.
func (p *Protocol) Handle(ctx noise.HandlerContext) error {
	msg, err := ctx.DecodeMessage()
	if err != nil {
		return nil
	}

	switch msg := msg.(type) {
	case Ping:
		if !ctx.IsRequest() {
			return errors.New("got a ping that was not sent as a request")
		}
		return ctx.SendMessage(Pong{})
	case FindNodeRequest:
		if !ctx.IsRequest() {
			return errors.New("got a find node request that was not sent as a request")
		}
		return ctx.SendMessage(FindNodeResponse{Results: p.table.FindClosest(msg.Target, BucketSize)})
	}

	return nil
}
