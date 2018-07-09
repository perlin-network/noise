package network

// RingBuffer is a circular list.
type RingBuffer struct {
	Data     []interface{}
	Position int
}

// NewRingBuffer returns a new ring buffer with a fixed length.
func NewRingBuffer(len int) *RingBuffer {
	return &RingBuffer{
		Data:     make([]interface{}, len),
		Position: 0,
	}
}

// Index returns an item at Position % len(Ringbuffer) in O(1) time.
func (b *RingBuffer) Index(pos int) *interface{} {
	if pos < 0 {
		panic("index out of bounds")
	}

	target := b.Position + pos

	if target >= len(b.Data) {
		target -= len(b.Data)
		if target >= b.Position {
			panic("index out of bounds")
		}
	}

	return &b.Data[target]
}

// MoveForward shifts all items in the ring buffer N steps.
func (b *RingBuffer) MoveForward(n int) {
	if n >= len(b.Data) || n < 0 {
		panic("n out of range")
	}
	b.Position += n
	if b.Position >= len(b.Data) {
		b.Position -= len(b.Data)
	}
}
