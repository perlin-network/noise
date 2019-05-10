package noise

import (
	"bufio"
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fastrand"
	"io"
	"math"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type Peer struct {
	n *Node

	addr net.Addr

	w io.Writer
	r io.Reader

	bw *bufio.Writer
	br *bufio.Reader

	c   Conn
	ctx Context

	pendingSend     chan *evt
	pendingRecv     map[byte]chan []byte
	pendingRecvLock sync.Mutex

	pendingRPC     map[uint32]*evt
	pendingRPCLock sync.Mutex

	queue    chan evtRPC
	queueRPC chan evtRPC

	interceptSend, interceptRecv         []func([]byte) ([]byte, error)
	interceptSendLock, interceptRecvLock sync.RWMutex

	afterSend, afterRecv         []func()
	afterSendLock, afterRecvLock sync.RWMutex

	signals     map[string]chan struct{}
	signalsLock sync.Mutex

	interceptErrors     []ErrorInterceptor
	interceptErrorsLock sync.RWMutex

	startOnce uint32
	stopOnce  uint32

	recvLockOpcode uint32
	recvLock       sync.Mutex
}

func (p *Peer) Start() {
	if !atomic.CompareAndSwapUint32(&p.startOnce, 0, 1) {
		return
	}

	var g []func(<-chan struct{}) error

	g = append(g, continuously(p.sendMessages()))
	g = append(g, continuously(p.receiveMessages()))

	for i := 0; i < runtime.NumCPU(); i++ {
		g = append(g, continuously(p.processRecv()))
		g = append(g, continuously(p.processRPC()))
	}

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
		p.interceptErrorsLock.RLock()
		for _, i := range p.interceptErrors {
			i(err)
		}
		p.interceptErrorsLock.RUnlock()
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

func (p *Peer) SetWriteDeadline(t time.Time) error {
	return p.c.SetWriteDeadline(t)
}

func (p *Peer) SetReadDeadline(t time.Time) error {
	return p.c.SetReadDeadline(t)
}

func (p *Peer) Node() *Node {
	return p.n
}

func (p *Peer) Ctx() Context {
	return p.ctx
}

func (p *Peer) Addr() net.Addr {
	return p.addr
}

func (p *Peer) Send(opcode byte, msg []byte) error {
	e := acquireEvt()
	e.oneway = true
	e.nonce = 0
	e.opcode = opcode
	e.msg = msg

	if err := p.queueSend(e); err != nil {
		releaseEvt(e)
		return errors.Wrap(err, "failed to queue message to send")
	}

	return nil
}

func (p *Peer) SendAwait(opcode byte, msg []byte) error {
	e := acquireEvt()
	e.done = make(chan error, 1)
	e.oneway = true
	e.nonce = 0
	e.opcode = opcode
	e.msg = msg

	if err := p.queueSend(e); err != nil {
		releaseEvt(e)
		return errors.Wrap(err, "failed to queue message to send")
	}

	var err error

	timeout := acquireTimer(3 * time.Second)
	defer releaseTimer(timeout)

	select {
	case err = <-e.done:
	case <-timeout.C:
		releaseEvt(e)
		return ErrTimeout
	case <-p.ctx.stop:
		releaseEvt(e)
		return ErrDisconnect
	}

	releaseEvt(e)

	if err != nil {
		return errors.Wrap(err, "got an error sending message")
	}

	return nil
}

func (p *Peer) Request(opcode byte, msg []byte) ([]byte, error) {
	e := acquireEvt()
	e.done = make(chan error, 1)
	e.oneway = false
	e.opcode = opcode
	e.msg = msg

	if err := p.queueSend(e); err != nil {
		releaseEvt(e)
		return nil, errors.Wrap(err, "failed to queue request")
	}

	var err error

	timeout := acquireTimer(3 * time.Second)
	defer releaseTimer(timeout)

	select {
	case err = <-e.done:
	case <-timeout.C:
		releaseEvt(e)
		return nil, ErrTimeout
	case <-p.ctx.stop:
		releaseEvt(e)
		return nil, ErrDisconnect
	}

	res := e.msg
	releaseEvt(e)

	if err != nil {
		return nil, errors.Wrap(err, "got an error sending request")
	}

	return res, nil
}

func (p *Peer) Recv(opcode byte) <-chan []byte {
	p.pendingRecvLock.Lock()
	if _, exists := p.pendingRecv[opcode]; !exists {
		p.pendingRecv[opcode] = make(chan []byte, 128)
	}
	ch := p.pendingRecv[opcode]
	p.pendingRecvLock.Unlock()

	return ch
}

func (p *Peer) LockOnRecv(opcode byte) func() {
	p.recvLock.Lock()
	atomic.StoreUint32(&p.recvLockOpcode, uint32(opcode))

	return func() {
		p.recvLock.Unlock()
	}
}

func (p *Peer) InterceptSend(f func(buf []byte) ([]byte, error)) {
	p.interceptSendLock.Lock()
	p.interceptSend = append(p.interceptSend, f)
	p.interceptSendLock.Unlock()
}

func (p *Peer) InterceptRecv(f func(buf []byte) ([]byte, error)) {
	p.interceptRecvLock.Lock()
	p.interceptRecv = append(p.interceptRecv, f)
	p.interceptRecvLock.Unlock()
}

func (p *Peer) InterceptErrors(i ErrorInterceptor) {
	p.interceptErrorsLock.Lock()
	p.interceptErrors = append(p.interceptErrors, i)
	p.interceptErrorsLock.Unlock()
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

func newPeer(n *Node, addr net.Addr, w io.Writer, r io.Reader, c Conn) *Peer {
	p := &Peer{
		n:    n,
		addr: addr,

		w: w,
		r: r,

		bw: bufio.NewWriterSize(w, 4096),
		br: bufio.NewReaderSize(r, 4096),

		c: c,

		pendingSend: make(chan *evt, 1024),
		pendingRecv: make(map[byte]chan []byte),
		pendingRPC:  make(map[uint32]*evt),

		queue:    make(chan evtRPC, 1024),
		queueRPC: make(chan evtRPC, 1024),

		signals: make(map[string]chan struct{}),

		recvLockOpcode: math.MaxUint32,
	}

	// The channel buffer size of '3 + 2 * runtime.NumCPU()' is selected on purpose. It is the number of
	// goroutines expected to be spawned per-peer.

	p.ctx = Context{
		n:      n,
		p:      p,
		result: make(chan error, 3+2*runtime.NumCPU()),
		stop:   make(chan struct{}),
		v:      make(map[string]interface{}),
		vm:     new(sync.RWMutex),
	}

	return p
}

func (p *Peer) queueSend(e *evt) error {
	select {
	case p.pendingSend <- e:
		return nil
	default:
		timeout := acquireTimer(3 * time.Second)
		defer releaseTimer(timeout)

		select {
		case p.pendingSend <- e:
			return nil
		case <-p.ctx.stop:
			return ErrDisconnect
		case <-timeout.C:
			return ErrSendQueueFull
		}
	}
}

func (p *Peer) queueRecv(ch chan []byte, buf []byte) error {
	select {
	case ch <- buf:
		return nil
	default:
		timeout := acquireTimer(3 * time.Second)
		defer releaseTimer(timeout)

		select {
		case ch <- buf:
			return nil
		case <-p.ctx.stop:
			return ErrDisconnect
		case <-timeout.C:
			return ErrRecvQueueFull
		}
	}
}

func (p *Peer) deregister() {
	if p.n != nil && p.addr != nil {
		p.n.peersLock.Lock()
		delete(p.n.peers, p.addr.String())
		p.n.peersLock.Unlock()
	}
}

func continuously(fn func(stop <-chan struct{}) error) func(stop <-chan struct{}) error {
	return func(stop <-chan struct{}) error {
		for {
			if err := fn(stop); err != nil {
				return err
			}
		}
	}
}

func (p *Peer) sendMessages() func(stop <-chan struct{}) error {
	flushDelay := 100 * time.Nanosecond

	var (
		e   *evt
		err error

		flush       <-chan time.Time
		flushTimer  = acquireTimer(flushDelay)
		flushAlways = make(chan time.Time)

		uint32Buf [4]byte
	)

	close(flushAlways)

	return func(stop <-chan struct{}) error {
		select {
		case e = <-p.pendingSend:
		default:
			select {
			case <-stop:
				return ErrDisconnect
			case <-flush:
				if err := p.bw.Flush(); err != nil {
					return errors.Wrap(err, "could not flush messages")
				}

				flush = nil
				return nil
			case e = <-p.pendingSend:
			}
		}

		buf := bytebufferpool.Get()
		defer bytebufferpool.Put(buf)

		if !e.oneway {
			e.nonce = fastrand.Uint32()

			p.pendingRPCLock.Lock()
			for _, exists := p.pendingRPC[e.nonce]; e.nonce == 0 || exists; {
				e.nonce = fastrand.Uint32()
			}
			p.pendingRPC[e.nonce] = e
			p.pendingRPCLock.Unlock()
		}

		binary.BigEndian.PutUint32(uint32Buf[:], e.nonce)
		buf.B = append(buf.B, uint32Buf[:]...)
		buf.B = append(buf.B, e.opcode)
		buf.B = append(buf.B, e.msg...)

		p.interceptSendLock.RLock()
		for _, f := range p.interceptSend {
			if buf.B, err = f(buf.B); err != nil {
				p.interceptSendLock.RUnlock()

				err = errors.Wrap(err, "failed to apply send interceptor")

				if e.done != nil {
					e.done <- err
				} else {
					releaseEvt(e)
				}

				return err
			}
		}
		p.interceptSendLock.RUnlock()

		binary.BigEndian.PutUint32(uint32Buf[:], uint32(buf.Len()))

		if _, err := p.bw.Write(uint32Buf[:]); err != nil {
			err = errors.Wrap(err, "failed to write size")

			if e.done != nil {
				e.done <- err
			} else {
				releaseEvt(e)
			}

			return err
		}

		if _, err := p.bw.Write(buf.B); err != nil {
			err = errors.Wrap(err, "failed to write message")

			if e.done != nil {
				e.done <- err
			} else {
				releaseEvt(e)
			}

			return err
		}

		p.afterSendLock.RLock()
		for _, f := range p.afterSend {
			f()
		}
		p.afterSendLock.RUnlock()

		if e.oneway {
			if e.done != nil {
				e.done <- nil
			} else {
				releaseEvt(e)
			}
		}

		if flush == nil && len(p.pendingSend) == 0 {
			if flushDelay > 0 {
				releaseTimer(flushTimer)
				flushTimer = acquireTimer(flushDelay)

				flush = flushTimer.C
			} else {
				flush = flushAlways
			}
		}

		return nil
	}
}

func (p *Peer) receiveMessages() func(stop <-chan struct{}) error {
	var (
		uint32Buf [4]byte
		err       error
	)

	return func(stop <-chan struct{}) error {
		if _, err := io.ReadFull(p.br, uint32Buf[:]); err != nil {
			return errors.Wrap(err, "couldn't read size")
		}

		size := binary.BigEndian.Uint32(uint32Buf[:])

		buf := bytebufferpool.Get()
		defer bytebufferpool.Put(buf)

		buf.B = append(buf.B, make([]byte, size)...)

		if _, err := io.ReadFull(p.br, buf.B); err != nil {
			return errors.Wrap(err, "couldn't read message")
		}

		p.interceptRecvLock.RLock()
		for _, f := range p.interceptRecv {
			if buf.B, err = f(buf.B); err != nil {
				p.interceptRecvLock.RUnlock()
				return errors.Wrap(err, "failed to apply recv interceptor")
			}
		}
		p.interceptRecvLock.RUnlock()

		nonce := binary.BigEndian.Uint32(buf.B[:4])
		buf.B = buf.B[4:]

		opcode := buf.B[0]
		buf.B = buf.B[1:]

		p.pendingRPCLock.Lock()
		req := p.pendingRPC[nonce]
		delete(p.pendingRPC, nonce)
		p.pendingRPCLock.Unlock()

		var handler Handler

		if p.n != nil {
			var registered bool

			p.n.opcodesLock.RLock()
			handler, registered = p.n.opcodes[opcode]
			p.n.opcodesLock.RUnlock()

			if !registered {
				return nil
			}
		}

		if req != nil {
			req.msg = make([]byte, len(buf.B))
			copy(req.msg, buf.B)

			req.done <- nil
		} else if nonce > 0 {
			msg := make([]byte, len(buf.B))
			copy(msg, buf.B)

			p.queueRPC <- evtRPC{nonce: nonce, opcode: opcode, msg: msg, handler: handler}
		} else if nonce == 0 {
			msg := make([]byte, len(buf.B))
			copy(msg, buf.B)

			p.queue <- evtRPC{nonce: nonce, opcode: opcode, msg: msg, handler: handler}
		}

		if lockOpcode := atomic.LoadUint32(&p.recvLockOpcode); lockOpcode != math.MaxUint32 {
			if opcode == byte(lockOpcode) {
				p.recvLock.Lock()
				p.recvLock.Unlock()

				atomic.StoreUint32(&p.recvLockOpcode, math.MaxUint32)
				return nil
			}
		}

		p.afterRecvLock.RLock()
		for _, f := range p.afterRecv {
			f()
		}
		p.afterRecvLock.RUnlock()

		return nil
	}
}

func (p *Peer) processRecv() func(stop <-chan struct{}) error {
	return func(stop <-chan struct{}) error {
		var erpc evtRPC

		select {
		case erpc = <-p.queue:
		default:
			select {
			case <-stop:
				return ErrDisconnect
			case erpc = <-p.queue:
			}
		}

		var err error

		if erpc.handler != nil {
			_, err = erpc.handler(p.ctx, erpc.msg)

			if err != nil {
				return nil
			}
		} else {
			p.pendingRecvLock.Lock()
			if _, exists := p.pendingRecv[erpc.opcode]; !exists {
				p.pendingRecv[erpc.opcode] = make(chan []byte, 128)
			}
			ch := p.pendingRecv[erpc.opcode]
			p.pendingRecvLock.Unlock()

			if err := p.queueRecv(ch, erpc.msg); err != nil {
				return err
			}
		}

		return nil
	}
}

func (p *Peer) processRPC() func(stop <-chan struct{}) error {
	return func(stop <-chan struct{}) error {
		var erpc evtRPC

		select {
		case erpc = <-p.queueRPC:
		default:
			select {
			case <-stop:
				return ErrDisconnect
			case erpc = <-p.queueRPC:
			}
		}

		var res []byte
		var err error

		if erpc.handler != nil {
			res, err = erpc.handler(p.ctx, erpc.msg)

			if err != nil {
				return nil
			}
		}

		e := acquireEvt()
		e.oneway = true
		e.nonce = erpc.nonce
		e.opcode = erpc.opcode
		e.msg = res

		if err := p.queueSend(e); err != nil {
			releaseEvt(e)
		}

		return nil
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
