package transport

import (
	"net"
	"strconv"

	"github.com/xtaci/kcp-go"
)

// KCP represents the KCP transport protocol with its respective configurable options.
type KCP struct {
	DataShards     int
	ParityShards   int
	SendWindowSize int
	RecvWindowSize int
}

// NewKCP instantiates a new instance of the KCP protocol.
func NewKCP() *KCP {
	return &KCP{
		DataShards:     0,
		ParityShards:   0,
		SendWindowSize: 10000,
		RecvWindowSize: 10000,
	}
}

// Listen listens for incoming KCP connections on a specified port.
func (t *KCP) Listen(port int) (net.Listener, error) {
	listener, err := kcp.ListenWithOptions(":"+strconv.Itoa(port), nil, t.DataShards, t.ParityShards)

	if err != nil {
		return nil, err
	}

	return listener, nil
}

// Dial dials an address via. the KCP protocol, with optional Reed-Solomon message sharding.
func (t *KCP) Dial(address string) (net.Conn, error) {
	conn, err := kcp.DialWithOptions(address, nil, t.DataShards, t.ParityShards)

	if err != nil {
		return nil, err
	}

	conn.SetWindowSize(t.SendWindowSize, t.RecvWindowSize)

	return conn, nil
}
