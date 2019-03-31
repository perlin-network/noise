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
	conn, err := net.Dial("tcp", address)

	if err != nil {
		return nil, errors.Wrap(err, "failed to dial peer")
	}

	peer := n.Wrap(conn)
	go peer.Start()

	return peer, nil
}
