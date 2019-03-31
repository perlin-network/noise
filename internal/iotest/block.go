package iotest

import (
	"io"
	"time"
)

type BlockingReadWriter struct {
	Unblock chan struct{}
}

func NewBlockingReadWriter() *BlockingReadWriter {
	return &BlockingReadWriter{Unblock: make(chan struct{})}
}

func (w BlockingReadWriter) Close() error {
	return nil
}

var zero time.Time

func (w BlockingReadWriter) SetReadDeadline(t time.Time) error {
	if t != zero {
		close(w.Unblock)
	}
	return nil
}

func (w BlockingReadWriter) SetWriteDeadline(t time.Time) error {
	if t != zero {
		close(w.Unblock)
	}
	return nil
}

func (w BlockingReadWriter) Write([]byte) (int, error) {
	<-w.Unblock
	return 0, io.EOF
}

func (w BlockingReadWriter) Read([]byte) (int, error) {
	<-w.Unblock
	return 0, io.EOF
}
