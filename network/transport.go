package network

import (
	"net"
)

var (
	_ TransportInterface = (*Transport)(nil)
)

// TransportInterface is used to proxy callbacks to a particular Plugin instance.
type TransportInterface interface {
	// Dial establishes a bidirectional connection to an address.
	Dial(addr string) (net.Conn, error)

	// Callback for when the network stops listening for peers.
	Cleanup()

	// NewListener creates a new transport protocol listener using given address
	NewListener(addr string) (net.Listener, error)

	// Listener returns the listener for the transport
	Listen(net *Network)

	// GetAddress returns the address of the transport protocol listener
	GetAddress() net.Addr
}

// Transport is an abstract class which all plugins extend.
type Transport struct {
	address string
	lis     net.Listener
}

// Hook callbacks of transport protocol

// Dial establishes a bidirectional connection to an address.
func (*Transport) Dial(addr string) (net.Conn, error) {
	return nil, nil
}

// Cleanup is called only once after network stops listening
func (p *Transport) Cleanup() {
	if p.lis != nil {
		p.lis.Close()
	}
}

// NewListener creates a new transport protocol listener using given address
func (*Transport) NewListener(addr string) (net.Listener, error) {
	return nil, nil
}

// Listen starts listening on the transport protocol
func (p *Transport) Listen(net *Network) {}

// GetAddress returns the address of the transport protocol listener
func (p *Transport) GetAddress() net.Addr {
	if p.lis != nil {
		return p.lis.Addr()
	}
	return nil
}
