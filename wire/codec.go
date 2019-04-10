package wire

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/valyala/bytebufferpool"
	"io"
	"sync"
)

type Interceptor func(buf []byte) ([]byte, error)

type Codec struct {
	PrefixSize bool
	Read       func(wire *Reader, state *State)
	Write      func(wire *Writer, state *State)

	send, recv         []Interceptor
	sendLock, recvLock sync.RWMutex
}

func (c *Codec) Clone() Codec {
	return Codec{
		PrefixSize: c.PrefixSize,
		Read:       c.Read,
		Write:      c.Write,

		send: c.send,
		recv: c.recv,
	}
}

func (c *Codec) InterceptRecv(i Interceptor) {
	c.recvLock.Lock()
	c.recv = append(c.recv, i)
	c.recvLock.Unlock()
}

func (c *Codec) InterceptSend(i Interceptor) {
	c.sendLock.Lock()
	c.send = append(c.send, i)
	c.sendLock.Unlock()
}

func (c *Codec) DoRead(r io.Reader, state *State) error {
	var wire *Reader
	var err error

	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	if c.PrefixSize {
		var length uint32

		if err = binary.Read(r, binary.BigEndian, &length); err != nil {
			return err
		}

		if length == 0 {
			return nil
		}

		buf.Set(make([]byte, length))

		n, err := io.ReadFull(r, buf.B)

		if err != nil {
			return errors.Wrap(err, "could not read expected amount of bytes from network")
		}

		if uint32(n) != length {
			return errors.Errorf("only read %d bytes when expected to read %d bytes", n, length)
		}
	} else {
		if _, err := buf.ReadFrom(r); err != nil {
			return errors.Wrap(err, "could not read from network all contents")
		}
	}

	c.recvLock.RLock()
	defer c.recvLock.RUnlock()

	for _, i := range c.recv {
		if buf.B, err = i(buf.B); err != nil {
			return errors.Wrap(err, "failed to apply read interceptor")
		}
	}

	wire = AcquireReader(buf.B)
	defer ReleaseReader(wire)

	c.Read(wire, state)
	return wire.Flush()
}

func (c *Codec) DoWrite(w io.Writer, state *State) error {
	var err error

	wire := AcquireWriter()
	defer ReleaseWriter(wire)

	c.Write(wire, state)

	if err = wire.Flush(); err != nil {
		return err
	}

	c.sendLock.RLock()
	defer c.sendLock.RUnlock()

	for _, i := range c.send {
		if wire.buf.B, err = i(wire.buf.B); err != nil {
			return errors.Wrap(err, "failed to apply write interceptor")
		}
	}

	if c.PrefixSize {
		var length [4]byte
		binary.BigEndian.PutUint32(length[:], uint32(wire.buf.Len()))

		wire.buf.B = append(length[:], wire.buf.B...)
	}

	n, err := wire.buf.WriteTo(w)

	if err != nil {
		return errors.Wrap(err, "could not write wire contents to buf")
	}

	if int(n) != wire.buf.Len() {
		return errors.Wrap(io.ErrUnexpectedEOF, "did not write enough of wire contents to buf")
	}

	return nil
}

type Reader struct {
	buf *bytes.Reader
	err error
}

func (p Reader) Flush() error {
	return p.err
}

func (p *Reader) Fail(err error) {
	if p.err == nil && err != nil {
		p.err = errors.WithStack(err)
	}
}

// BytesRead returns the number of bytes that have been
// read so far from over-the-wire.
func (p *Reader) BytesLeft() int {
	return p.buf.Len()
}

func (p *Reader) ReadUint64(order binary.ByteOrder) (res uint64) {
	if p.err != nil {
		return
	}

	p.Fail(binary.Read(p.buf, order, &res))
	return
}

func (p *Reader) ReadByte() (res byte) {
	if p.err != nil {
		return
	}

	p.Fail(binary.Read(p.buf, binary.BigEndian, &res))
	return
}

func (p *Reader) ReadBytes(amount int) (buf []byte) {
	if p.err != nil {
		return
	}

	if amount == 0 {
		return nil
	}

	buf = make([]byte, amount)

	n, err := p.buf.Read(buf)
	p.Fail(err)

	if n != int(amount) {
		p.Fail(io.ErrUnexpectedEOF)
	}

	return buf
}

type Writer struct {
	buf *bytebufferpool.ByteBuffer
	err error
}

func (w Writer) Flush() error {
	return w.err
}

func (w *Writer) Fail(err error) {
	if w.err == nil && err != nil {
		w.err = errors.WithStack(err)
	}
}

func (w *Writer) WriteUint64(order binary.ByteOrder, val uint64) {
	w.Fail(binary.Write(w.buf, order, val))
}

func (w *Writer) WriteByte(val byte) {
	w.Fail(binary.Write(w.buf, binary.BigEndian, val))
}

func (w *Writer) WriteBytes(buf []byte) {
	n, err := w.buf.Write(buf)
	w.Fail(err)

	if n != len(buf) {
		w.Fail(io.ErrUnexpectedEOF)
	}
}
