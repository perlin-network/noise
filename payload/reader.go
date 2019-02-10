package payload

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
	"io"
)

var _ io.Reader = (*Reader)(nil)

type Reader struct {
	reader *bytes.Reader
}

func NewReader(payload []byte) Reader {
	return Reader{
		reader: bytes.NewReader(payload),
	}
}

// Len returns the number of bytes that have not yet been read so far.
func (r Reader) Len() int {
	return r.reader.Len()
}

func (r Reader) Read(b []byte) (n int, err error) {
	return r.reader.Read(b)
}

func (r Reader) ReadBytes() ([]byte, error) {
	raw, err := r.ReadUint32()
	if err != nil {
		return nil, err
	}

	size := int(raw)

	if size < 0 || size > r.reader.Len() {
		return nil, errors.New("bytes out of bounds")
	}

	buf := make([]byte, size)
	r.Read(buf)

	return buf, nil
}

func (r Reader) ReadString() (string, error) {
	bytes, err := r.ReadBytes()
	return string(bytes), err
}

func (r Reader) ReadByte() (byte, error) {
	return r.reader.ReadByte()
}

func (r Reader) ReadUint16() (uint16, error) {
	var buf [2]byte
	_, err := r.reader.Read(buf[:])
	return binary.LittleEndian.Uint16(buf[:]), err
}

func (r Reader) ReadUint32() (uint32, error) {
	var buf [4]byte
	_, err := r.reader.Read(buf[:])
	return binary.LittleEndian.Uint32(buf[:]), err
}

func (r Reader) ReadUint64() (uint64, error) {
	var buf [8]byte
	_, err := r.reader.Read(buf[:])
	return binary.LittleEndian.Uint64(buf[:]), err
}
