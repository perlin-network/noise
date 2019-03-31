package noise

type Protocol func(ctx Context) (Protocol, error)

type Context struct {
	n *Node
	p *Peer
	d <-chan struct{}
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

func newContext(peer *Peer, done <-chan struct{}) Context {
	return Context{n: peer.Node(), p: peer, d: done}
}
