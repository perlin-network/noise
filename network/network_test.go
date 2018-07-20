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
		lis, err := net.Listen(tt.protocol, addr)

		b := NewBuilder()
		n, err := b.Build()
		assert.Equal(t, nil, err, err)

		assert.Equal(t, nil, err, err)
		assert.NotEqual(t, nil, lis)
		go n.Listen(lis)
		time.Sleep(200 * time.Millisecond)

		lis, err = net.Listen(tt.protocol, addr)
		n2, err := b.Build()
		assert.Equal(t, nil, err, err)
		assert.NotEqual(t, n2.Address, n.Address)

		n.BlockUntilListening()
		_, err = n2.Client(n.Address)
		if err != nil {
			t.Errorf("Client() = %+v, expected <nil>", err)
		}
	}
}
