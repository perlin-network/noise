package transport

import (
	"fmt"
	"net"
)

var _ Layer = (*tcp)(nil)

type tcp struct{}

func (t tcp) String() string {
	return "tcp"
}

func (t tcp) Listen(host string, port uint16) (net.Listener, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, err
	}

	return listener, nil
}

func (t tcp) Dial(address string) (net.Conn, error) {
	resolved, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTCP("tcp", nil, resolved)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (t tcp) IP(address net.Addr) net.IP {
	return address.(*net.TCPAddr).IP
}

func (t tcp) Port(address net.Addr) uint16 {
	return uint16(address.(*net.TCPAddr).Port)
}

// NewTCP returns a new tcp instance
func NewTCP() tcp {
	return tcp{}
}
