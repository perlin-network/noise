package transport

import (
	"net"
)

type Layer interface {
	Listen(port uint16) (net.Listener, error)
	Dial(address string) (net.Conn, error)

	IP(address net.Addr) net.IP
	Port(address net.Addr) uint16
}
