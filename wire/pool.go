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
	state.strings = make(map[byte]string)
	state.slices = make(map[byte][]byte)
	state.bytes = make(map[byte]byte)
	state.bools = make(map[byte]bool)
	state.uint16s = make(map[byte]uint16)
	state.uint32s = make(map[byte]uint32)
	state.uint64s = make(map[byte]uint64)

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
