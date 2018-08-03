package transport

import "net"

type Layer interface {
	Listen(port int) (net.Listener, error)
	Dial(address string) (net.Conn, error)
}