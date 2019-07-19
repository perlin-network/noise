package transport

import (
	"net"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

var _ Layer = (*tcp)(nil)

type tcp struct{}

func (t tcp) String() string {
	return "tcp"
}

func (t tcp) Listen(host string, port uint16) (net.Listener, error) {
	if net.ParseIP(host) == nil {
		return nil, errors.Errorf("unable to parse host as IP: %s", host)
	}

	listener, err := net.Listen("tcp", ":"+strconv.Itoa(int(port)))
	if err != nil {
		return nil, err
	}

	return listener, nil
}

func (t tcp) Dial(address string) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
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
