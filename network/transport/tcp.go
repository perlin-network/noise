package transport

import (
	"net"
	"strconv"
)

type TCP struct {
	WriteBufferSize int
	ReadBufferSize  int
	NoDelay         bool
}

func NewTCP() *TCP {
	return &TCP{
		WriteBufferSize: 10000,
		ReadBufferSize:  10000,
		NoDelay:         false,
	}
}

func (t *TCP) Listen(port int) (net.Listener, error) {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return nil, err
	}

	return listener, nil
}

func (t *TCP) Dial(address string) (net.Conn, error) {
	resolved, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTCP("tcp", nil, resolved)
	if err != nil {
		return nil, err
	}

	conn.SetWriteBuffer(t.WriteBufferSize)
	conn.SetReadBuffer(t.ReadBufferSize)
	conn.SetNoDelay(t.NoDelay)

	return conn, nil
}
