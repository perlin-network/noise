package noise

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type connWriterState byte

const (
	connWriterInit connWriterState = iota
	connWriterRunning
	connWriterFlushing
	connWriterClosed
)

type connWriter struct {
	sync.Mutex
	state   connWriterState
	pending [][]byte
	cond    sync.Cond
}

func newConnWriter() *connWriter {
	c := &connWriter{state: connWriterInit}
	c.cond.L = &c.Mutex

	return c
}

func (c *connWriter) close() {
	c.Lock()
	defer c.Unlock()

	if c.state == connWriterInit || c.state == connWriterClosed {
		return
	}

	c.state = connWriterFlushing
	c.cond.Signal()

	for c.state != connWriterClosed {
		c.cond.Wait()
	}
}

func (c *connWriter) write(data []byte) {
	c.Lock()
	defer c.Unlock()

	if c.state != connWriterInit && c.state != connWriterRunning {
		return
	}

	c.pending = append(c.pending, data)
	c.cond.Broadcast()
}

func (c *connWriter) loop(conn net.Conn, timeout time.Duration) error {
	c.Lock()
	c.state = connWriterRunning
	c.Unlock()

	header := make([]byte, 4)
	writer := bufio.NewWriter(conn)

	defer func() {
		c.Lock()
		defer c.Unlock()

		c.state = connWriterClosed
		c.cond.Signal()
	}()

	for {
		if timeout > 0 {
			if err := conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
				return err
			}
		}

		c.Lock()
		for c.state == connWriterRunning && len(c.pending) == 0 {
			c.cond.Wait()
		}
		pending, state := c.pending, c.state
		c.pending = nil
		c.Unlock()

		if len(pending) == 0 && state == connWriterFlushing {
			return nil
		}

		for _, data := range pending {
			binary.BigEndian.PutUint32(header[:4], uint32(len(data)))

			if _, err := writer.Write(header); err != nil {
				return err
			}

			if _, err := writer.Write(data); err != nil {
				return err
			}
		}

		if err := writer.Flush(); err != nil {
			return err
		}
	}
}

type connReader struct {
	pending chan []byte
}

func newConnReader() *connReader {
	return &connReader{pending: make(chan []byte, 1024)}
}

func (c *connReader) loop(conn net.Conn, timeout time.Duration, limit uint32) error {
	defer close(c.pending)

	header := make([]byte, 4)
	reader := bufio.NewReader(conn)

	for {
		if timeout > 0 {
			if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
				return err
			}
		}

		if _, err := io.ReadFull(reader, header); err != nil {
			return err
		}

		size := binary.BigEndian.Uint32(header[:4])

		if limit > 0 && size > limit {
			return fmt.Errorf("got %d bytes, but limit is set to %d: %w", size, limit, ErrMessageTooLarge)
		}

		data := make([]byte, size)
		if _, err := io.ReadFull(reader, data); err != nil {
			return err
		}

		select {
		case c.pending <- data:
		default:
		}
	}
}
