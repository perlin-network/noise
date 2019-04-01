package noise

import (
	"github.com/pkg/errors"
	"io"
	"net"
	"time"
)

type Conn interface {
	io.Closer

	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

type Dialer func(n *Node, address string) (*Peer, error)

var defaultDialer Dialer = func(n *Node, address string) (*Peer, error) {
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)

	if err != nil {
		return nil, errors.Wrap(err, "failed to dial peer")
	}

	c := conn.(*net.TCPConn)

	if err := c.SetNoDelay(false); err != nil {
		return nil, err
	}

	if err := c.SetWriteBuffer(10000); err != nil {
		return nil, err
	}

	peer := n.Wrap(conn)
	go peer.Start()

	return peer, nil
}
