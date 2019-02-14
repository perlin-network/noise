package transport

import (
	"fmt"
	"net"
)

type Layer interface {
	fmt.Stringer

	Listen(port uint16) (net.Listener, error)
	Dial(address string) (net.Conn, error)

	IP(address net.Addr) net.IP
	Port(address net.Addr) uint16
}
