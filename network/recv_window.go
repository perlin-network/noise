package network

import (
	"sync"
)

// RecvWindow represents a window that buffers and cuts off messages based on their priority.
type RecvWindow struct {
	sync.Mutex
	size      int
	lastNonce uint64
	buf       []interface{}
}

// NewRecvWindow creates a new receive buffer window with a specific buffer size.
func NewRecvWindow(size int) *RecvWindow {
	return &RecvWindow{
		size:      size,
		lastNonce: 1,
		buf:       make([]interface{}, size),
	}
}

// SetLocalNonce sets a expected nonce.
func (w *RecvWindow) SetLocalNonce(nonce uint64) {
	w.Lock()
	w.lastNonce = nonce
	w.Unlock()
}

// Push adds value with a given nonce to the window.
func (w *RecvWindow) Push(nonce uint64, value interface{}) {
	w.Lock()
	w.buf[nonce%uint64(w.size)] = value
	w.Unlock()
}

// Pop returns a slice of values from last till not yet received nonce.
func (w *RecvWindow) Pop() []interface{} {
	return w.Range(func(nonce uint64, v interface{}) bool {
		return v != nil
	})
}

// Range will return items from the queue while `fn` returns true.
// If `fn` never return false, the result will be a full buffer.
func (w *RecvWindow) Range(fn func(uint64, interface{}) bool) []interface{} {
	res := make([]interface{}, 0)

	w.Lock()

	i := 0
	id := w.lastNonce
	for {
		idx := w.idx(id)
		val := w.buf[idx]
		if i == w.size || !fn(id, val) {
			w.lastNonce = idx
			break
		}
		res = append(res, val)
		w.buf[idx] = nil
		id++
		i++
	}

	w.Unlock()
	return res
}

func (w *RecvWindow) idx(id uint64) uint64 {
	return id % uint64(w.size)
}
