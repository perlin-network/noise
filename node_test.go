package noise

import (
	"github.com/fortytw2/leaktest"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net"
	"testing"
)

func TestShutdownNodeByCallingShutdown(t *testing.T) {
	defer leaktest.Check(t)()

	conn, _ := net.Pipe()

	node := NewNode(ioutil.NopCloser(conn))
	defer assert.Empty(t, node.Peers())
	defer node.Shutdown()

	go node.Wrap(conn).Start()
	assert.NotEmpty(t, node.Peers())
	assert.NotNil(t, node.PeerByAddr(conn.RemoteAddr().String()))
}

func TestShutdownPeerByClosingConn(t *testing.T) {
	defer leaktest.Check(t)()

	conn, _ := net.Pipe()

	node := NewNode(ioutil.NopCloser(conn))

	peer := node.Wrap(conn)
	go peer.Start()

	assert.NoError(t, conn.Close())
}

func TestNodeCannotHaveDuplicatePeers(t *testing.T) {
	defer leaktest.Check(t)()

	conn, _ := net.Pipe()

	node := NewNode(ioutil.NopCloser(conn))
	defer node.Shutdown()

	p1 := node.Wrap(conn)
	p2 := node.Wrap(conn)

	p1.killed <- struct{}{}

	assert.Equal(t, p1, p2)
}
