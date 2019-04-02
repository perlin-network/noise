package noise

import (
	"io"
	"time"
)

type Conn interface {
	io.Closer

	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

type Dialer func(n *Node, address string) (*Peer, error)
