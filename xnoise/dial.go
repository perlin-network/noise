package xnoise

import (
	"github.com/perlin-network/noise"
	"github.com/pkg/errors"
	"net"
	"time"
)

func DialTCP(n *noise.Node, address string) (*noise.Peer, error) {
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
