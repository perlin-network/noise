package wire

import (
	"bytes"
	"github.com/valyala/bytebufferpool"
	"sync"
)

var statePool sync.Pool
var parserPool sync.Pool
var writerPool sync.Pool

func NewState() *State {
	return &State{
		strings: make(map[byte]string),
		slices:  make(map[byte][]byte),
		bytes:   make(map[byte]byte),
		bools:   make(map[byte]bool),
		uint16s: make(map[byte]uint16),
		uint32s: make(map[byte]uint32),
		uint64s: make(map[byte]uint64),
	}
}

func AcquireState() *State {
	state := statePool.Get()

	if state == nil {
		state = NewState()
	}

	return state.(*State)
}

func ReleaseState(state *State) {
	for key := range state.strings {
		delete(state.strings, key)
	}

	for key := range state.slices {
		delete(state.slices, key)
	}

	for key := range state.bytes {
		delete(state.bytes, key)
	}

	for key := range state.bools {
		delete(state.bools, key)
	}

	for key := range state.uint16s {
		delete(state.uint16s, key)
	}

	for key := range state.uint32s {
		delete(state.uint32s, key)
	}

	for key := range state.uint64s {
		delete(state.uint64s, key)
	}

	statePool.Put(state)
}

func AcquireReader(buf []byte) *Reader {
	p := parserPool.Get()

	if p == nil {
		p = &Reader{buf: bytes.NewReader(nil)}
	}

	parser := p.(*Reader)
	parser.buf.Reset(buf)

	return parser
}

func ReleaseReader(p *Reader) {
	p.buf.Reset(nil)
	p.err = nil
	p.len = 0
	parserPool.Put(p)
}

func AcquireWriter() *Writer {
	ww := writerPool.Get()

	if ww == nil {
		ww = new(Writer)
	}

	writer := ww.(*Writer)
	writer.buf = bytebufferpool.Get()

	return writer
}

func ReleaseWriter(w *Writer) {
	bytebufferpool.Put(w.buf)

	w.buf = nil
	w.err = nil

	writerPool.Put(w)
}
