package noise

import (
	"io"
	"sync"
	"time"
)

type Conn interface {
	io.Closer

	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

type Dialer func(n *Node, address string) (*Peer, error)

type evt struct {
	opcode byte
	nonce  uint32
	msg    []byte

	oneway bool
	done   chan error
}

var evtPool sync.Pool

func acquireEvt() *evt {
	v := evtPool.Get()
	if v == nil {
		v = &evt{done: make(chan error, 1)}
	}
	evt := v.(*evt)
	if len(evt.done) != 0 {
		panic("BUG: evt.done must be empty")
	}
	return evt
}

func releaseEvt(e *evt) {
	if len(e.done) != 0 {
		panic("BUG: evt.done must be empty")
	}
	e.msg = e.msg[:0]
	evtPool.Put(e)
}
