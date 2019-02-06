package transport

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"net"
	"testing"
	"time"
)

func TestBuffered(t *testing.T) {
	port := uint16(3000)
	layer := NewBuffered()
	errChan := make(chan error)
	var lisConn net.Conn
	go func() {
		lis, err := layer.Listen(port)
		assert.Nil(t, err)
		if lisConn, err = lis.Accept(); err != nil {
			errChan <- err
		}
		close(errChan)
	}()
	// seems Accept isn't instant, need a bit of setup
	time.Sleep(1 * time.Millisecond)

	// dial
	dialConn, err := layer.Dial(fmt.Sprintf(":%d", port))
	assert.Nilf(t, err, "Dial error: %v", err)
	err = <-errChan
	assert.Nilf(t, err, "Listen error: %v", err)

	// check the IP and port
	assert.Equal(t, "bufconn", string(layer.IP(dialConn.RemoteAddr())))
	assert.Equal(t, port, layer.Port(dialConn.RemoteAddr()))

	// Write some data on both sides of the connection.
	n, err := dialConn.Write([]byte("hello"))
	assert.Truef(t, n == 5 && err == nil, "dialConn.Write([]byte{\"hello\"}) = %v, %v; want 5, <nil>", n, err)

	n, err = lisConn.Write([]byte("hello"))
	assert.Truef(t, n == 5 && err == nil, "lisConn.Write([]byte{\"hello\"}) = %v, %v; want 5, <nil>", n, err)

	// Close dial-side; writes from either side should fail.
	dialConn.Close()
	_, err = lisConn.Write([]byte("hello"))
	assert.Equalf(t, err, io.ErrClosedPipe, "lisConn.Write() = _, <nil>; want _, <non-nil>")
	_, err = dialConn.Write([]byte("hello"))
	assert.Equalf(t, err, io.ErrClosedPipe, "dialConn.Write() = _, <nil>; want _, <non-nil>")

	// Read from both sides; reads on lisConn should work, but dialConn should fail.
	buf := make([]byte, 6)
	_, err = dialConn.Read(buf)
	assert.Equalf(t, err, io.ErrClosedPipe, "dialConn.Read(buf) = %v, %v; want _, io.ErrClosedPipe", n, err)
	n, err = lisConn.Read(buf)
	assert.Truef(t, n == 5 && err == nil, "lisConn.Read(buf) = %v, %v; want 5, <nil>", n, err)

}
