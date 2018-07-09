package network

type RingBuffer struct {
	data []interface{}
	pos  int
}

func NewRingBuffer(len int) *RingBuffer {
	return &RingBuffer{
		data: make([]interface{}, len),
		pos:  0,
	}
}

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

func (b *RingBuffer) MoveForward(n int) {
	if n >= len(b.data) || n < 0 {
		panic("n out of range")
	}
	b.pos += n
	if b.pos >= len(b.data) {
		b.pos -= len(b.data)
	}
}
