package network

import (
	"sync"

	"github.com/pkg/errors"
)

// RecvWindow represents a window that buffers and cuts off messages based on their priority.
type RecvWindow struct {
	sync.Mutex

	size       int
	buffer     *RingBuffer
	localNonce uint64
}

// NewRecvWindow creates a new receive buffer window with a specific buffer size.
func NewRecvWindow(size int) *RecvWindow {
	return &RecvWindow{
		size:       size,
		buffer:     NewRingBuffer(size),
		localNonce: 1,
	}
}

func (w *RecvWindow) SetLocalNonce(nonce uint64) {
	w.Lock()
	w.localNonce = nonce
	w.Unlock()
}

// Update pushes messages from the networks receive queue into the buffer.
func (w *RecvWindow) Update() []interface{} {
	ready := make([]interface{}, 0)

	w.Lock()
	i := 0
	for ; i < w.size; i++ {
		cursor := w.buffer.Index(i)
		if *cursor == nil {
			break
		}
		ready = append(ready, *cursor)
		*cursor = nil
	}
	if i > 0 && i < w.size {
		w.buffer.MoveForward(i)
	}
	w.localNonce += uint64(i)
	w.Unlock()

	return ready
}

// Input places a new received message into the receive buffer.
func (w *RecvWindow) Input(nonce uint64, msg interface{}) error {
	w.Lock()
	defer w.Unlock()

	offset := int(nonce - w.localNonce)

	if offset < 0 || offset >= w.size {
		return errors.Errorf("Local nonce is %d while received %d", w.localNonce, nonce)
	}

	*w.buffer.Index(offset) = msg
	return nil
}
