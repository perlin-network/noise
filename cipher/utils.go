package cipher

import (
	"encoding/binary"
	"github.com/pkg/errors"
)

// sliceForAppend takes a slice and a requested number of bytes. It returns a
// slice with the contents of the given slice followed by that many bytes and a
// second slice that aliases into it and contains only the extra bytes. If the
// original slice has sufficient capacity then no allocation is performed.
func sliceForAppend(in []byte, n int) (head, tail []byte) {
	if total := len(in) + n; cap(in) >= total {
		head = in[:total]
	} else {
		head = make([]byte, total)
		copy(head, in)
	}
	tail = head[len(in):]
	return head, tail
}

func parseFrame(b []byte, maxLength uint32) ([]byte, []byte, error) {
	if len(b) < MsgLenFieldSize {
		return nil, b, nil
	}

	lengthField := b[:MsgLenFieldSize]
	length := binary.BigEndian.Uint32(lengthField)

	if length > maxLength {
		return nil, nil, errors.Errorf("received frame length %d which is larger than the limit %d", length, maxLength)
	}

	if len(b) < int(length) { // Frame not complete yet.
		return nil, b, nil
	}

	return b[:length], b[length:], nil
}
