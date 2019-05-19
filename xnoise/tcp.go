package xnoise

import (
	"github.com/perlin-network/noise"
	"github.com/pkg/errors"
	"net"
	"strconv"
	"time"
)

func DialTCP(n *noise.Node, address string) (*noise.Peer, error) {
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)

	if err != nil {
		return nil, errors.Wrap(err, "failed to dial peer")
	}

	c := conn.(*net.TCPConn)

	if err := c.SetWriteBuffer(10000); err != nil {
		return nil, err
	}

	peer := n.Wrap(conn)
	go peer.Start()

	return peer, nil
}

func ListenTCP(port uint) (*noise.Node, error) {
	listener, err := net.Listen("tcp", ":"+strconv.FormatUint(uint64(port), 10))

	if err != nil {
		return nil, err
	}

	node := noise.NewNode(listener)

	go func() {
		for {
			conn, err := listener.Accept()

			if err != nil {
				break
			}

			peer := node.Wrap(conn)
			go peer.Start()
		}
	}()

	return node, nil
}
