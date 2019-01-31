package payload

import (
	"bytes"
	"encoding/binary"
	"io"
)

var _ io.Writer = (*Writer)(nil)

type Writer struct {
	buffer *bytes.Buffer
}

func NewWriter(buf []byte) Writer {
	return Writer{
		buffer: bytes.NewBuffer(buf),
	}
}

// Len returns the number of bytes written so far.
func (b Writer) Len() int {
	return b.buffer.Len()
}

func (b Writer) Bytes() []byte {
	return b.buffer.Bytes()
}

func (b Writer) Write(buf []byte) (n int, err error) {
	return b.buffer.Write(buf)
}

func (b Writer) WriteBytes(buf []byte) Writer {
	b.WriteUint32(uint32(len(buf)))
	b.Write(buf)

	return b
}

func (b Writer) WriteString(x string) Writer {
	b.WriteBytes([]byte(x))

	return b
}

func (b Writer) WriteByte(x byte) Writer {
	b.buffer.WriteByte(x)

	return b
}

func (b Writer) WriteUint16(x uint16) Writer {
	var buf [2]byte
	binary.LittleEndian.PutUint16(buf[:], x)
	b.Write(buf[:])

	return b
}

func (b Writer) WriteUint32(x uint32) Writer {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], x)
	b.Write(buf[:])

	return b
}

func (b Writer) WriteUint64(x uint64) Writer {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], x)
	b.Write(buf[:])

	return b
}
