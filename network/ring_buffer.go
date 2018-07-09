package network

// RingBuffer is a circular list.
type RingBuffer struct {
	data []interface{}
	pos  int
}

// NewRingBuffer returns a new ring buffer with a fixed length.
func NewRingBuffer(len int) *RingBuffer {
	return &RingBuffer{
		data: make([]interface{}, len),
		pos:  0,
	}
}

// Index returns an item at pos % len(Ringbuffer) in O(1) time.
func (b *RingBuffer) Index(pos int) *interface{} {
	if pos < 0 {
		panic("index out of bounds")
	}

	target := b.pos + pos

	if target >= len(b.data) {
		target -= len(b.data)
		if target >= b.pos {
			panic("index out of bounds")
		}
	}

	return &b.data[target]
}

// MoveForward shifts all items in the ring buffer N steps.
func (b *RingBuffer) MoveForward(n int) {
	if n >= len(b.data) || n < 0 {
		panic("n out of range")
	}
	b.pos += n
	if b.pos >= len(b.data) {
		b.pos -= len(b.data)
	}
}
