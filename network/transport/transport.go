package transport

import "net"

// Layer represents a transport protocol layer.
type Layer interface {
	Listen(port int) (net.Listener, error)
	Dial(address string) (net.Conn, error)
}
