// Package kademlia is a noise implementation of the routing and discovery portion of the Kademlia protocol, with
// minor improvements suggested by the S/Kademlia paper.
package kademlia

import (
	"context"
	"errors"
	"fmt"
	"github.com/perlin-network/noise"
	"go.uber.org/zap"
	"time"
)

// BucketSize returns the capacity, or the total number of peer ID entries a single routing table bucket may hold.
const BucketSize int = 16

// ErrBucketFull is returned when a routing table bucket is at max capacity.
var ErrBucketFull = errors.New("bucket is full")

// Protocol implements routing/discovery portion of the Kademlia protocol with improvements suggested by the
// S/Kademlia paper. It is expected that Protocol is bound to a noise.Node via (*noise.Node).Bind before the node
// starts listening for incoming peers.
type Protocol struct {
	node   *noise.Node
	logger *zap.Logger
	table  *Table

	events Events

	pingTimeout time.Duration
}

// New returns a new instance of the Kademlia protocol.
func New(opts ...ProtocolOption) *Protocol {
	p := &Protocol{
		pingTimeout: 3 * time.Second,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
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

// Table returns this Kademlia overlay's routing table from your nodes perspective.
func (p *Protocol) Table() *Table {
	return p.table
}

// Ack attempts to insert a peer ID into your nodes routing table. If the routing table bucket in which your peer ID
// was expected to be inserted on is full, the peer ID at the tail of the bucket is pinged. If the ping fails, the
// peer ID at the tail of the bucket is evicted and your peer ID is inserted to the head of the bucket.
func (p *Protocol) Ack(id noise.ID) {
	for {
		inserted, err := p.table.Update(id)
		if err == nil {
			if inserted {
				p.logger.Debug("Peer was inserted into routing table.",
					zap.String("peer_id", id.String()),
					zap.String("peer_addr", id.Address),
				)
			}

			if inserted {
				if p.events.OnPeerAdmitted != nil {
					p.events.OnPeerAdmitted(id)
				}
			} else {
				if p.events.OnPeerActivity != nil {
					p.events.OnPeerActivity(id)
				}
			}

			return
		}

		last := p.table.Last(id.ID)

		ctx, cancel := context.WithTimeout(context.Background(), p.pingTimeout)
		pong, err := p.node.RequestMessage(ctx, last.Address, Ping{})
		cancel()

		if err != nil {
			if id, deleted := p.table.Delete(last.ID); deleted {
				p.logger.Debug("Peer was evicted from routing table by failing to be pinged.",
					zap.String("peer_id", id.String()),
					zap.String("peer_addr", id.Address),
					zap.Error(err),
				)

				if p.events.OnPeerEvicted != nil {
					p.events.OnPeerEvicted(id)
				}
			}
			continue
		}

		if _, ok := pong.(Pong); !ok {
			if id, deleted := p.table.Delete(last.ID); deleted {
				p.logger.Debug("Peer was evicted from routing table by failing to be pinged.",
					zap.String("peer_id", id.String()),
					zap.String("peer_addr", id.Address),
					zap.Error(err),
				)

				if p.events.OnPeerEvicted != nil {
					p.events.OnPeerEvicted(id)
				}
			}
			continue
		}

		p.logger.Debug("Peer failed to be inserted into routing table as it's intended bucket is full.",
			zap.String("peer_id", id.String()),
			zap.String("peer_addr", id.Address),
		)

		if p.events.OnPeerEvicted != nil {
			p.events.OnPeerEvicted(id)
		}

		return
	}
}

// Protocol returns a noise.Protocol that may registered to a node via (*noise.Node).Bind.
func (p *Protocol) Protocol() noise.Protocol {
	return noise.Protocol{
		Bind:            p.Bind,
		OnPeerConnected: p.OnPeerConnected,
		OnPingFailed:    p.OnPingFailed,
		OnMessageSent:   p.OnMessageSent,
		OnMessageRecv:   p.OnMessageRecv,
	}
}

// Bind registers messages Ping, Pong, FindNodeRequest, FindNodeResponse, and handles them by registering the
// (*Protocol).Handle Handler.
func (p *Protocol) Bind(node *noise.Node) error {
	p.node = node
	p.table = NewTable(p.node.ID())

	if p.logger == nil {
		p.logger = p.node.Logger()
	}

	node.RegisterMessage(Ping{}, UnmarshalPing)
	node.RegisterMessage(Pong{}, UnmarshalPong)
	node.RegisterMessage(FindNodeRequest{}, UnmarshalFindNodeRequest)
	node.RegisterMessage(FindNodeResponse{}, UnmarshalFindNodeResponse)

	node.Handle(p.Handle)

	return nil
}

// OnPeerConnected attempts to acknowledge the new peers existence by placing its entry into your nodes' routing table
// via (*Protocol).Ack.
func (p *Protocol) OnPeerConnected(client *noise.Client) {
	p.Ack(client.ID())
}

// OnPingFailed evicts peers that your node has failed to dial.
func (p *Protocol) OnPingFailed(addr string, err error) {
	if id, deleted := p.table.DeleteByAddress(addr); deleted {
		p.logger.Debug("Peer was evicted from routing table by failing to be dialed.", zap.Error(err))

		if p.events.OnPeerEvicted != nil {
			p.events.OnPeerEvicted(id)
		}
	}
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
