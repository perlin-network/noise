package kademlia

import (
	"context"
	"errors"
	"fmt"
	"github.com/perlin-network/noise"
	"time"
)

const BucketSize int = 16

var ErrBucketFull = errors.New("bucket is full")

var _ noise.Binder = (*Protocol)(nil)

type Protocol struct {
	node  *noise.Node
	table *Table
}

func NewProtocol() *Protocol {
	return &Protocol{}
}

func (p *Protocol) Find(target noise.ID, opts ...IteratorOption) []noise.ID {
	return NewIterator(p.node, p.table, opts...).Find(target)
}

func (p *Protocol) Discover(opts ...IteratorOption) []noise.ID {
	return p.Find(p.node.ID(), opts...)
}

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

func (p *Protocol) Table() *Table {
	return p.table
}

func (p *Protocol) Ack(id noise.ID) {
	for {
		if err := p.table.Update(id); err == nil {
			return
		}

		bucket := p.table.Bucket(id)
		last := bucket[len(bucket)-1]

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		pong, err := p.node.RequestMessage(ctx, last.Address, Ping{})
		cancel()

		if err != nil {
			p.table.Delete(last)
			continue
		}

		if _, ok := pong.(Pong); !ok {
			p.table.Delete(last)
			continue
		}

		return
	}
}

func (p *Protocol) OnPeerJoin(client *noise.Client) {
	p.Ack(client.ID())
}

func (p *Protocol) OnPeerLeave(client *noise.Client) {
}

func (p *Protocol) OnMessageSent(client *noise.Client) {
	p.Ack(client.ID())
}

func (p *Protocol) OnMessageRecv(client *noise.Client) {
	p.Ack(client.ID())
}

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
