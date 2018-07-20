package network

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	flag.Set("alsologtostderr", fmt.Sprintf("%t", true))
	var logLevel string
	flag.StringVar(&logLevel, "logLevel", "4", "test")
	flag.Lookup("v").Value.Set(logLevel)
}

func getListenerAddress(lis net.Listener) string {
	return fmt.Sprintf("%s://%s", lis.Addr().Network(), lis.Addr().String())
}

func TestListen(t *testing.T) {
	testCases := []struct {
		protocol string
	}{
		{"tcp"},
		{"tcp4"},
		{"tcp6"},
		{"unix"},
	}

	for _, tt := range testCases {
		addr := "localhost:0"
		switch tt.protocol {
		case "unix":
			name := fmt.Sprintf("testsocket-%d", time.Now().UnixNano())
			ioutil.TempFile("/tmp", name)
			addr = fmt.Sprintf("/tmp/%s", name)
			syscall.Unlink(addr)
		}
		lis1, err := net.Listen(tt.protocol, addr)

		b := NewBuilder()
		n, err := b.Build()
		assert.Equal(t, nil, err, err)

		assert.Equal(t, nil, err, err)
		assert.NotEqual(t, nil, lis1)
		go n.Listen(lis1)
		time.Sleep(200 * time.Millisecond)

		switch tt.protocol {
		case "unix":
			name := fmt.Sprintf("testsocket-%d", time.Now().UnixNano())
			ioutil.TempFile("/tmp", name)
			addr = fmt.Sprintf("/tmp/%s", name)
			syscall.Unlink(addr)
		}
		lis2, err := net.Listen(tt.protocol, addr)
		n2, err := b.Build()
		assert.Equal(t, nil, err, err)
		assert.NotEqual(t, getListenerAddress(lis1), getListenerAddress(lis2))

		n.BlockUntilListening()
		_, err = n2.Client(getListenerAddress(lis1))
		if err != nil {
			t.Errorf("Client() = %+v, expected <nil>", err)
		}
	}
}
