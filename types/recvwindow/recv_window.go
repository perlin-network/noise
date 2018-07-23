package recvwindow

import (
	"sync"

	"github.com/pkg/errors"
)

// RecvWindow represents a window that buffers and cuts off messages based on their priority.
type RecvWindow struct {
	sync.Mutex
	size     int
	data     []interface{}
	currSize int
}

// NewRecvWindow creates a new receive buffer window with a specific buffer size.
func NewRecvWindow(size int) *RecvWindow {
	return &RecvWindow{
		size:     size,
		data:     make([]interface{}, size),
		currSize: 0,
	}
}

// PopWindow pops all the messages out of the receive window
func (w *RecvWindow) PopWindow() []interface{} {
	ready := make([]interface{}, 0)

	w.Lock()
	for i := 0; i < w.currSize; i++ {
		ready = append(ready, w.data[i])
		w.data[i] = nil
	}

	w.currSize = 0
	w.Unlock()

	return ready
}

// Insert places a new received message into the receive window
func (w *RecvWindow) Insert(msg interface{}) error {
	w.Lock()
	defer w.Unlock()

	if w.currSize >= w.size {
		return errors.Errorf("recv_window has reached maximum capacity of %s", w.size)
	}

	w.data[w.currSize] = msg
	w.currSize++

	return nil
}
