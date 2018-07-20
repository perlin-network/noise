package network

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListen(t *testing.T) {
	testCases := []struct {
		protocol string
		address  string
	}{
		{"tcp", "tcp://localhost:0"},
		{"tcp4", "tcp4://localhost:0"},
		{"tcp6", "tcp6://[::1]:0"},
		{"unix", "/tmp/go.sock"},
	}

	for _, tt := range testCases {
		b := NewBuilder()
		b.SetAddress(tt.address)
		n, err := b.Build()
		assert.Equal(t, nil, err)

		lis, err := net.Listen(tt.protocol, tt.address)
		assert.Equal(t, nil, err)
		assert.NotEqual(t, nil, lis)
		go n.Listen(lis)

		b.SetAddress(tt.address)
		n2, err := b.Build()
		assert.Equal(t, nil, err)
		assert.NotEqual(t, n2.Address, n.Address)

		n.BlockUntilListening()
		_, err = n.Client(n2.Address)
		assert.Equal(t, nil, err)
	}
}
