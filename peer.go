package noise

import (
	"github.com/heptio/workgroup"
	"github.com/perlin-network/noise/wire"
	"github.com/pkg/errors"
	"io"
	"log"
	"math/rand"
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

	c Conn

	send chan evtSend

	recv     map[uint64]map[byte]evtRecv
	recvLock sync.RWMutex

	newMuxLock sync.Mutex

	afterSend, afterRecv         []func()
	afterSendLock, afterRecvLock sync.RWMutex

	signals     map[string]chan struct{}
	signalsLock sync.Mutex

	errs     []ErrorInterceptor
	errsLock sync.RWMutex

	killed chan struct{}

	stop     chan error
	stopOnce sync.Once

	codec atomic.Value // *wire.Codec
}

func (p *Peer) Start() {
	var g workgroup.Group

	g.Add(continuously(p.sendMessages()))
	g.Add(continuously(p.receiveMessages()))

	if p.n != nil && p.n.p != nil {
		g.Add(p.followProtocol)
	}

	g.Add(p.cleanup)

	if err := g.Run(); err != nil {
		p.reportError(err)

		switch errors.Cause(err) {
		case ErrDisconnect:
		case io.ErrClosedPipe:
		case io.EOF:
		default:
			log.Printf("%+v\n", err)
		}
	}

	p.deregisterFromNode()
	p.killed <- struct{}{}
}

func (p *Peer) Disconnect(err error) {
	p.stopOnce.Do(func() {
		p.reportError(err)

		p.deregisterFromNode()
		p.stop <- err

		if len(p.killed) == 1 {
			<-p.killed
		}
	})
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
	rand.Seed(time.Now().UnixNano())

	var id uint64

	for {
		id = rand.Uint64()

		if id == 0 {
			continue
		}

		p.recvLock.RLock()
		_, exists := p.recv[id]
		p.recvLock.RUnlock()

		if !exists {
			break
		}
	}

	p.initMuxQueue(id)

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
	signal, exists := p.signals[name]
	if !exists {
		signal = make(chan struct{})
		p.signals[name] = signal
	}
	p.signalsLock.Unlock()

	return func() {
		var closed bool

		select {
		case _, ok := <-signal:
			if !ok {
				closed = true
			}
		default:
		}

		if !closed {
			close(signal)
		}
	}
}

func (p *Peer) WaitFor(name string) {
	p.signalsLock.Lock()
	signal, exists := p.signals[name]
	if !exists {
		signal = make(chan struct{})
		p.signals[name] = signal
	}
	p.signalsLock.Unlock()

	<-signal
}

func (p *Peer) InterceptErrors(i ErrorInterceptor) {
	p.errsLock.Lock()
	p.errs = append(p.errs, i)
	p.errsLock.Unlock()
}

func newPeer(n *Node, addr net.Addr, w io.Writer, r io.Reader, c Conn) *Peer {
	p := &Peer{
		n:       n,
		addr:    addr,
		w:       w,
		r:       r,
		c:       c,
		send:    make(chan evtSend, 1024),
		recv:    make(map[uint64]map[byte]evtRecv),
		signals: make(map[string]chan struct{}),
		stop:    make(chan error, 1),
		killed:  make(chan struct{}, 1),
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
		queue = evtRecv{ch: make(chan Wire, 1024), lock: make(chan struct{}, 1)}
		queues[opcode] = queue
	}

	p.recvLock.Unlock()
	p.newMuxLock.Unlock()

	return queue
}

func (p *Peer) initMuxQueue(mux uint64) {
	p.newMuxLock.Lock()
	p.recvLock.Lock()

	if _, exists := p.recv[mux]; !exists {
		p.recv[mux] = make(map[byte]evtRecv)

		// Move all messages from our zero mux, originally intended for our
		// mux channel, to our newly generated mux.

		for opcode, queue := range p.recv[0] {
			if _, exists := p.recv[mux][opcode]; !exists {
				p.recv[mux][opcode] = evtRecv{ch: make(chan Wire, 1024), lock: make(chan struct{}, 1)}
			}

			for n := len(queue.ch); n > 0; n-- {
				e := <-queue.ch

				if e.m.id == mux {
					p.recv[mux][opcode].ch <- e
				} else {
					queue.ch <- e
				}
			}
		}
	}

	p.recvLock.Unlock()
	p.newMuxLock.Unlock()
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

func (p *Peer) deregisterFromNode() {
	if p.n != nil && p.addr != nil {
		p.n.peersLock.Lock()
		delete(p.n.peers, p.addr.String())
		p.n.peersLock.Unlock()
	}
}

func (p *Peer) cleanup(stop <-chan struct{}) error {
	err := ErrDisconnect

	select {
	case err = <-p.stop:
	case <-stop:
	}

	if p.c != nil {
		if err := p.c.Close(); err != nil {
			return err
		}
	}

	return err
}

func (p *Peer) followProtocol(stop <-chan struct{}) (err error) {
	initial := p.n.p()

	for state := initial; state != nil; state, err = state(newContext(p, stop)) {
		if err != nil {
			return err
		}
	}

	<-stop
	return
}

func (p *Peer) sendMessages() func(stop <-chan struct{}) error {
	return func(stop <-chan struct{}) error {
		var evt evtSend

		select {
		case <-stop:
			return nil
		case evt = <-p.send:
		}

		state := wire.AcquireState()
		defer wire.ReleaseState(state)

		state.SetByte(WireKeyOpcode, evt.opcode)
		state.SetUint64(WireKeyMuxID, evt.mux)
		state.SetMessage(evt.msg)

		err := p.WireCodec().DoWrite(p.w, state)
		evt.done <- err

		if err != nil {
			return nil
		}

		p.afterSendLock.RLock()
		for _, f := range p.afterSend {
			f()
		}
		p.afterSendLock.RUnlock()

		return nil
	}
}

func (p *Peer) receiveMessages() func(stop <-chan struct{}) error {
	return func(stop <-chan struct{}) error {
		select {
		case <-stop:
			return nil
		default:
		}

		state := wire.AcquireState()
		defer wire.ReleaseState(state)

		if err := p.WireCodec().DoRead(p.r, state); err != nil {
			return err
		}

		opcode := state.Byte(WireKeyOpcode)
		mux := state.Uint64(WireKeyMuxID)

		if opcode == 0x00 {
			return nil
		}

		hub := p.getMuxQueue(mux, opcode)

		if len(hub.ch) == cap(hub.ch) { // If the queue is full, pop from the front and push the new message to the back.
			select {
			case <-hub.ch:
			default:
			}
		}

		hub.ch <- Wire{m: Mux{id: mux, peer: p}, o: opcode, b: state.Message()}

		hub.lock <- struct{}{}
		<-hub.lock

		p.afterRecvLock.RLock()
		for _, f := range p.afterRecv {
			f()
		}
		p.afterRecvLock.RUnlock()

		return nil
	}
}

func (p *Peer) Ctx() Context {
	return Context{n: p.n, p: p, d: p.killed}
}
