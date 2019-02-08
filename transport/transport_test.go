package transport

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net"
	"sync"
	"testing"
	"time"
)

func testTransport(t *testing.T, layer Layer, port uint16, host string) {
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
	// Accept isn't instant, need a bit of setup time
	time.Sleep(10 * time.Millisecond)

	// dial
	dialConn, err := layer.Dial(fmt.Sprintf(":%d", port))
	assert.Nilf(t, err, "Dial error: %v", err)
	err = <-errChan
	assert.Nilf(t, err, "Listen error: %v", err)

	// check the IP and port
	assert.Equal(t, host, string(layer.IP(dialConn.RemoteAddr())))
	assert.Equal(t, port, layer.Port(dialConn.RemoteAddr()))

	// Write some data on both sides of the connection.
	n, err := dialConn.Write([]byte("hello"))
	assert.Truef(t, n == 5 && err == nil, "dialConn.Write([]byte{\"hello\"}) = %v, %v; want 5, <nil>", n, err)

	n, err = lisConn.Write([]byte("hello"))
	assert.Truef(t, n == 5 && err == nil, "lisConn.Write([]byte{\"hello\"}) = %v, %v; want 5, <nil>", n, err)

	// Close dial-side; writes from either side should fail.
	dialConn.Close()
	_, err = lisConn.Write([]byte("hello"))
	assert.Truef(t, err != nil, "lisConn.Write() = _, <nil>; want _, <non-nil>")
	_, err = dialConn.Write([]byte("hello"))
	assert.Truef(t, err != nil, "dialConn.Write() = _, <nil>; want _, <non-nil>")

	// Read from both sides; reads on lisConn should work, but dialConn should fail.
	buf := make([]byte, 6)
	_, err = dialConn.Read(buf)
	assert.Truef(t, err != nil, "dialConn.Read(buf) = %v, %v; want _, io.ErrClosedPipe", n, err)
	n, err = lisConn.Read(buf)
	assert.Truef(t, n == 5 && err == nil, "lisConn.Read(buf) = %v, %v; want 5, <nil>", n, err)

}

func TestBuffered(t *testing.T) {
	layer := NewBuffered()
	var wg sync.WaitGroup

	// run the test over several ports
	for i := 8900; i < 8910; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			testTransport(t, layer, uint16(i), "bufconn")
		}(i)
	}
	wg.Wait()
}

func TestTCP(t *testing.T) {
	layer := NewTCP()
	var wg sync.WaitGroup

	// run the test over several ports
	for i := 8900; i < 8910; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			testTransport(t, layer, uint16(i), "\u007f\x00\x00\x01")
		}(i)
	}
	wg.Wait()
}
