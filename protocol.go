package noise

import "sync"

type Protocol func(ctx Context) (Protocol, error)
type ProtocolBlock func(ctx Context) error

func NewProtocol(blocks ...ProtocolBlock) Protocol {
	result := make([]Protocol, len(blocks))

	for i := len(blocks) - 1; i >= 0; i-- {
		i := i

		var next Protocol

		if i != len(blocks)-1 {
			next = result[i+1]
		}

		result[i] = func(ctx Context) (Protocol, error) {
			if err := blocks[i](ctx); err != nil {
				return nil, err
			}

			return next, nil
		}
	}

	return result[0]
}

type Context struct {
	n *Node
	p *Peer
	d chan struct{}

	v  map[string]interface{}
	vm *sync.RWMutex
}

func (c Context) Done() <-chan struct{} {
	return c.d
}

func (c Context) Node() *Node {
	return c.n
}

func (c Context) Peer() *Peer {
	return c.p
}

func (c Context) Get(key string) interface{} {
	c.vm.RLock()
	v := c.v[key]
	c.vm.RUnlock()

	return v
}

func (c Context) Set(key string, val interface{}) {
	c.vm.Lock()
	c.v[key] = val
	c.vm.Unlock()
}
