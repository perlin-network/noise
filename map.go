package noise

import (
	"errors"
	"math"
	"sync"
)

type clientMap struct {
	sync.Mutex

	order   []string
	entries map[string]*Client
}

func newClientMap(cap int) *clientMap {
	return &clientMap{
		order:   make([]string, 0, cap),
		entries: make(map[string]*Client, cap),
	}
}

func (c *clientMap) get(n *Node, addr string) (*Client, bool) {
	c.Lock()
	defer c.Unlock()

	client, exists := c.entries[addr]
	if !exists {
		if len(c.entries) == n.maxInboundConnections {
			closing := c.order[0]
			c.order = c.order[1:]

			client := c.entries[closing]
			delete(c.entries, closing)

			client.close()
			client.waitUntilClosed()
		}

		client = newClient(n)
		c.entries[addr] = client
		c.order = append(c.order, addr)
	}

	return client, exists
}

func (c *clientMap) remove(addr string) {
	c.Lock()
	defer c.Unlock()

	var order []string

	for _, entry := range c.order {
		if entry != addr {
			order = append(order, entry)
		}
	}

	c.order = order
	delete(c.entries, addr)
}

func (c *clientMap) release() {
	c.Lock()

	entries := c.entries
	c.entries = make(map[string]*Client, cap(c.order))
	c.order = make([]string, 0, cap(c.order))

	c.Unlock()

	for _, client := range entries {
		client.close()
		client.waitUntilClosed()
	}
}

func (c *clientMap) slice() []*Client {
	c.Lock()
	defer c.Unlock()

	clients := make([]*Client, len(c.entries))
	for i := 0; i < len(c.entries); i++ {
		clients[i] = c.entries[c.order[i]]
	}

	return clients
}

type requestMap struct {
	sync.Mutex
	entries map[uint64]chan message
	nonce   uint64
}

func newRequestMap() *requestMap {
	return &requestMap{entries: make(map[uint64]chan message)}
}

func (r *requestMap) nextNonce() (<-chan message, uint64, error) {
	r.Lock()
	defer r.Unlock()

	if r.nonce == math.MaxUint64 {
		r.nonce = 0
	}

	r.nonce++
	nonce := r.nonce

	if _, exists := r.entries[nonce]; exists {
		return nil, 0, errors.New("ran out of available nonce to use for making a new request")
	}

	ch := make(chan message, 1)
	r.entries[nonce] = ch

	return ch, nonce, nil
}

func (r *requestMap) markRequestFailed(nonce uint64) {
	r.Lock()
	defer r.Unlock()

	close(r.entries[nonce])
	delete(r.entries, nonce)
}

func (r *requestMap) findRequest(nonce uint64) chan<- message {
	r.Lock()
	defer r.Unlock()

	ch, exists := r.entries[nonce]
	if exists {
		delete(r.entries, nonce)
	}

	return ch
}
