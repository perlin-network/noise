package noise

import (
	"github.com/perlin-network/noise/wire"
	"github.com/pkg/errors"
	"github.com/valyala/fastrand"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Peer struct {
	m Mux

	n *Node

	addr net.Addr

	w io.Writer
	r io.Reader

	c   Conn
	ctx Context

	codec atomic.Value // *wire.Codec

	flush sync.Mutex

	recv     map[uint64]map[byte]evtRecv
	recvLock sync.RWMutex

	newMuxLock sync.Mutex

	afterSend, afterRecv         []func()
	afterSendLock, afterRecvLock sync.RWMutex

	signals     map[string]chan struct{}
	signalsLock sync.Mutex

	errs     []ErrorInterceptor
	errsLock sync.RWMutex

	startOnce uint32
	stopOnce  uint32
}

func (p *Peer) Start() {
	if !atomic.CompareAndSwapUint32(&p.startOnce, 0, 1) {
		return
	}

	var g []func(<-chan struct{}) error

	g = append(g, continuously(p.receiveMessages()))

	if p.n != nil {
		if protocol := p.n.p.Load(); protocol != nil {
			g = append(g, p.followProtocol(protocol.(Protocol)))
		}
	}

	var wg sync.WaitGroup
	wg.Add(len(g))

	for _, fn := range g {
		go func(fn func(<-chan struct{}) error) {
			defer wg.Done()
			p.ctx.result <- fn(p.ctx.stop)
		}(fn)
	}

	err := <-p.ctx.result

	if p.c != nil {
		if e := p.c.Close(); e != nil {
			err = errors.Wrap(err, e.Error())
		}
	}

	if err != nil {
		p.reportError(err)
	}

	close(p.ctx.stop)

	wg.Wait()

	p.deregister()
}

func (p *Peer) Disconnect(err error) {
	if !atomic.CompareAndSwapUint32(&p.stopOnce, 0, 1) {
		return
	}

	p.deregister()

	p.ctx.result <- err
	<-p.ctx.stop
}

// UpdateWireCodec atomically updates the message codec a peer utilizes in
// amidst sending/receiving messages.
func (p *Peer) UpdateWireCodec(codec *wire.Codec) {
	p.codec.Store(codec)
}

func (p *Peer) WireCodec() *wire.Codec {
	return p.codec.Load().(*wire.Codec)
}

func (p *Peer) SetWriteDeadline(t time.Time) error {
	return p.c.SetWriteDeadline(t)
}

func (p *Peer) SetReadDeadline(t time.Time) error {
	return p.c.SetReadDeadline(t)
}

func (p *Peer) Node() *Node {
	return p.n
}

func (p *Peer) Addr() net.Addr {
	return p.addr
}

// Mux establishes a new multiplexed session with a peer for the
// purpose of sending/receiving messages concurrently.
func (p *Peer) Mux() Mux {
	var id uint64

	p.newMuxLock.Lock()
	p.recvLock.Lock()

	for {
		id = uint64(fastrand.Uint32())<<32 + uint64(fastrand.Uint32())

		if id == 0 {
			continue
		}

		if _, exists := p.recv[id]; !exists {
			break
		}
	}

	p.initMuxQueue(id, false)

	p.recvLock.Unlock()
	p.newMuxLock.Unlock()

	return Mux{id: id, peer: p}
}

// Send invokes Mux.Send on a peers default mux session.
func (p *Peer) Send(opcode byte, msg []byte) error {
	return p.m.Send(opcode, msg)
}

// SendWithTimeout invokes Mux.SendWithTimeout on a peers default mux session.
func (p *Peer) SendWithTimeout(opcode byte, msg []byte, timeout time.Duration) error {
	return p.m.SendWithTimeout(opcode, msg, timeout)
}

// Recv invokes Mux.Recv on a peers default mux session.
func (p *Peer) Recv(opcode byte) <-chan Wire {
	return p.m.Recv(opcode)
}

// LockOnRecv invokes Mux.LockOnRecv on a peers default mux session.
func (p *Peer) LockOnRecv(opcode byte) Locker {
	return p.m.LockOnRecv(opcode)
}

func (p *Peer) AfterSend(f func()) {
	p.afterSendLock.Lock()
	p.afterSend = append(p.afterSend, f)
	p.afterSendLock.Unlock()
}

func (p *Peer) AfterRecv(f func()) {
	p.afterRecvLock.Lock()
	p.afterRecv = append(p.afterRecv, f)
	p.afterRecvLock.Unlock()
}

func (p *Peer) RegisterSignal(name string) func() {
	p.signalsLock.Lock()

	s, exists := p.signals[name]
	if !exists {
		s = make(chan struct{})
		p.signals[name] = s
	}

	p.signalsLock.Unlock()

	return func() {
		opened := true

		select {
		case _, opened = <-s:
		default:
		}

		if opened {
			close(s)
		}
	}
}

func (p *Peer) WaitFor(name string) {
	p.signalsLock.Lock()

	s, exists := p.signals[name]
	if !exists {
		s = make(chan struct{})
		p.signals[name] = s
	}

	p.signalsLock.Unlock()

	<-s
}

func (p *Peer) InterceptErrors(i ErrorInterceptor) {
	p.errsLock.Lock()
	p.errs = append(p.errs, i)
	p.errsLock.Unlock()
}

func (p *Peer) Ctx() Context {
	return p.ctx
}

func newPeer(n *Node, addr net.Addr, w io.Writer, r io.Reader, c Conn) *Peer {
	p := &Peer{
		n:       n,
		addr:    addr,
		w:       w,
		r:       r,
		c:       c,
		recv:    make(map[uint64]map[byte]evtRecv),
		signals: make(map[string]chan struct{}),
	}

	// The channel buffer size of '4' is selected on purpose. It is the number of
	// goroutines expected to be spawned per-peer.

	p.ctx = Context{
		n:      n,
		p:      p,
		result: make(chan error, 4),
		stop:   make(chan struct{}),
		v:      make(map[string]interface{}),
		vm:     new(sync.RWMutex),
	}

	p.m = Mux{peer: p}

	codec := DefaultProtocol.Clone()
	p.UpdateWireCodec(&codec)

	return p
}

// getMuxQueue returns a map of message queues pertaining to a specified mux ID, if the
// mux ID is registered beforehand.
//
// If the mux was not registered beforehand, by default, it returns a map of message queues
// pertaining to the zero mux (mux ID zero).
//
// If the zero mux was not yet initialized, it additionally atomically registers a map of
// message queues to mux ID zero.
//
// It additionally registers a new buffered channel for an opcode under a specified
// mux ID if there does not exist one beforehand.
func (p *Peer) getMuxQueue(mux uint64, opcode byte) evtRecv {
	p.newMuxLock.Lock()
	p.recvLock.Lock()

	queues, registered := p.recv[mux]

	if !registered {
		if _, init := p.recv[0]; !init {
			p.recv[0] = make(map[byte]evtRecv)
		}

		queues = p.recv[0]
	}

	queue, exists := queues[opcode]

	if !exists {
		queue = evtRecv{ch: make(chan Wire, 64), lock: make(chan struct{}, 1)}
		queues[opcode] = queue
	}

	p.recvLock.Unlock()
	p.newMuxLock.Unlock()

	return queue
}

func (p *Peer) initMuxQueue(mux uint64, lock bool) {
	if lock {
		p.newMuxLock.Lock()
		p.recvLock.Lock()
	}

	if _, exists := p.recv[mux]; !exists {
		p.recv[mux] = make(map[byte]evtRecv)

		// Move all messages from our zero mux, originally intended for our
		// mux channel, to our newly generated mux.

		for opcode, queue := range p.recv[0] {
			if _, exists := p.recv[mux][opcode]; !exists {
				p.recv[mux][opcode] = evtRecv{ch: make(chan Wire, 64), lock: make(chan struct{}, 1)}
			}

		L:
			for n := len(queue.ch); n > 0; n-- {
				select {
				case e := <-queue.ch:
					if e.m.id == mux {
						p.recv[mux][opcode].ch <- e
					} else {
						queue.ch <- e
					}
				default:
					break L
				}
			}
		}
	}

	if lock {
		p.recvLock.Unlock()
		p.newMuxLock.Unlock()
	}
}

func (p *Peer) reportError(err error) {
	if err == nil {
		return
	}

	p.errsLock.RLock()
	for _, i := range p.errs {
		i(err)
	}
	p.errsLock.RUnlock()
}

func (p *Peer) deregister() {
	if p.n != nil && p.addr != nil {
		p.n.peersLock.Lock()
		delete(p.n.peers, p.addr.String())
		p.n.peersLock.Unlock()
	}
}

func (p *Peer) followProtocol(init Protocol) func(stop <-chan struct{}) error {
	return func(stop <-chan struct{}) error {
		for state, err := init(p.ctx); err != nil || state != nil; state, err = state(p.ctx) {
			if err != nil {
				return err
			}
		}

		<-stop
		return nil
	}
}

func (p *Peer) receiveMessages() func(stop <-chan struct{}) error {
	return func(stop <-chan struct{}) error {
		state := wire.AcquireState()
		defer wire.ReleaseState(state)

		if err := p.WireCodec().DoRead(p.r, state); err != nil {
			return err
		}

		opcode := state.Byte(WireKeyOpcode)
		mux := state.Uint64(WireKeyMuxID)

		if opcode == 0 {
			return nil
		}

		if p.n != nil {
			p.n.opcodesLock.RLock()
			_, registered := p.n.opcodesIndex[opcode]
			p.n.opcodesLock.RUnlock()

			if !registered {
				return nil
			}
		}

		hub := p.getMuxQueue(mux, opcode)

		select {
		case hub.ch <- Wire{m: Mux{id: mux, peer: p}, o: opcode, b: state.Message()}:
			hub.lock <- struct{}{}
			<-hub.lock

			p.afterRecvLock.RLock()
			for _, f := range p.afterRecv {
				f()
			}
			p.afterRecvLock.RUnlock()
		default:
		}

		return nil
	}
}
