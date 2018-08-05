package network

import (
	"sync"
)

// RecvWindow represents a window that buffers and cuts off messages based on their priority.
type RecvWindow struct {
	sync.Mutex
	size      int
	lastNonce uint64
	data      map[uint64]interface{}
}

// NewRecvWindow creates a new receive buffer window with a specific buffer size.
func NewRecvWindow(size int) *RecvWindow {
	return &RecvWindow{
		size:      size,
		lastNonce: 1,
		data:      make(map[uint64]interface{}, 0),
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
	w.data[nonce] = value
	w.Unlock()
}

// Pop returns a slice of values from last till not yet received nonce.
func (w *RecvWindow) Pop() []interface{} {
	res := make([]interface{}, 0)

	w.Lock()
	defer w.Unlock()

	id := w.lastNonce
	for {
		val, ok := w.data[id]
		if !ok {
			w.lastNonce = id
			break
		}
		res = append(res, val)
		delete(w.data, id)
		id++
	}
	return res
}
