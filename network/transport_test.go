package network

import (
	"flag"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	flag.Set("alsologtostderr", fmt.Sprintf("%t", true))
	var logLevel string
	flag.StringVar(&logLevel, "logLevel", "4", "test")
	flag.Lookup("v").Value.Set(logLevel)
}

func TestTransportPlugin(t *testing.T) {
	testCases := []struct {
		protocol string
	}{
		{"tcp"},
		{"kcp"},
	}

	for _, tt := range testCases {
		addr := fmt.Sprintf("%s://%s", tt.protocol, "localhost:0")
		b := NewBuilder()
		var p1 TransportInterface
		var err error
		switch tt.protocol {
		case "tcp":
			p1, err = NewTCPTransport(addr)
			assert.Equal(t, nil, err, "%s: %+v", tt.protocol, err)
			b.RegisterTransportLayer(p1)
		case "kcp":
			p1, err = NewKCPTransport(addr)
			assert.Equal(t, nil, err, "%s: %+v", tt.protocol, err)
			b.RegisterTransportLayer(p1)
		}

		n, err := b.Build()
		assert.Equal(t, nil, err, "%s: %+v", tt.protocol, err)
		go n.Listen()

		b2 := NewBuilder()
		switch tt.protocol {
		case "tcp":
			p, err := NewTCPTransport(addr)
			assert.Equal(t, nil, err, "%s: %+v", tt.protocol, err)
			b2.RegisterTransportLayer(p)
		case "kcp":
			p, err := NewKCPTransport(addr)
			assert.Equal(t, nil, err, "%s: %+v", tt.protocol, err)
			b2.RegisterTransportLayer(p)
		}
		n2, err := b2.Build()
		assert.Equal(t, nil, err, "%s: %+v", tt.protocol, err)

		n.BlockUntilListening()
		addr = fmt.Sprintf("%s://%s", p1.GetAddress().Network(), p1.GetAddress().String())
		_, err = n2.Client(addr)
		assert.Equal(t, nil, err, "%s: %+v", tt.protocol, err)
	}
}

func TestMultipleTransport(t *testing.T) {
	tcpAddr := "tcp://localhost:0"
	kcpAddr := "kcp://localhost:0"
	b := NewBuilder()
	p1, err := NewTCPTransport(tcpAddr)
	assert.Equal(t, nil, err, "%+v", err)
	b.RegisterTransportLayer(p1)
	p2, err := NewKCPTransport(kcpAddr)
	assert.Equal(t, nil, err, "%+v", err)
	b.RegisterTransportLayer(p2)

	n, err := b.Build()
	assert.Equal(t, nil, err, "%+v", err)
	go n.Listen()

	b2 := NewBuilder()
	n2, err := b2.Build()
	assert.Equal(t, nil, err, "%+v", err)

	n.BlockUntilListening()
	addr1 := fmt.Sprintf("%s://%s", p1.GetAddress().Network(), p1.GetAddress().String())
	_, err = n2.Client(addr1)
	assert.Equal(t, nil, err, "%+v", err)

	addr2 := fmt.Sprintf("%s://%s", p1.GetAddress().Network(), p1.GetAddress().String())
	_, err = n2.Client(addr2)
	assert.Equal(t, nil, err, "%+v", err)
}
