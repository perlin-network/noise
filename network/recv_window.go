package network

import (
	"sync"
)

// RecvWindow represents a window that buffers and cuts off messages based on their priority.
type RecvWindow struct {
	sync.Mutex
	lastNonce uint64
	size      int
	buf       []interface{}
	once      sync.Once
}

// NewRecvWindow creates a new receive buffer window with a specific buffer size.
func NewRecvWindow(size int) *RecvWindow {
	return &RecvWindow{
		size: size,
		buf:  make([]interface{}, size),
	}
}

// SetLocalNonce sets a expected nonce.
func (w *RecvWindow) SetLocalNonce(nonce uint64) {
	w.Lock()
	w.lastNonce = nonce
	w.Unlock()
}

// LocalNonce gets last nonce.
func (w *RecvWindow) LocalNonce() uint64 {
	w.Lock()
	nonce := w.lastNonce
	w.Unlock()
	return nonce
}

// Push adds value with a given nonce to the window.
func (w *RecvWindow) Push(nonce uint64, value interface{}) {
	w.Lock()
	w.once.Do(func() {
		w.lastNonce = nonce
	})
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

	id := w.lastNonce
	for i := 0; i < w.size; i++ {
		idx := w.idx(id)
		val := w.buf[idx]
		if !fn(id, val) {
			w.lastNonce = idx
			break
		}
		res = append(res, val)
		w.buf[idx] = nil
		id++
	}

	w.Unlock()
	return res
}

func (w *RecvWindow) idx(id uint64) uint64 {
	return id % uint64(w.size)
}
