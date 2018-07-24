package network

import (
	"fmt"
	"net"

	"github.com/golang/glog"
	"github.com/perlin-network/noise/types"
	"github.com/pkg/errors"
)

// TCPPlugin provides pluggable transport protocol
type TCPPlugin struct {
	*Transport
}

var (
	// TCPPluginID to reference transport plugin
	TCPPluginID                    = (*TCPPlugin)(nil)
	_           TransportInterface = (*TCPPlugin)(nil)
)

// Dial establishes a bidirectional connection to an address.
func (*TCPPlugin) Dial(addr string) (net.Conn, error) {
	addrInfo, err := types.ParseAddress(addr)
	if err != nil {
		return nil, err
	}

	address, err := net.ResolveTCPAddr("tcp", addrInfo.HostPort())
	if err != nil {
		return nil, err
	}

	dialer, err := net.DialTCP("tcp", nil, address)
	if err != nil {
		return nil, err
	}
	dialer.SetWriteBuffer(10000)
	dialer.SetNoDelay(false)

	return dialer, nil
}

// NewListener creates a new transport protocol listener using given address
func (*TCPPlugin) NewListener(addr string) (net.Listener, error) {
	addrInfo, err := types.ParseAddress(addr)
	if err != nil {
		return nil, err
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", addrInfo.Port))
	if err != nil {
		return nil, err
	}

	return lis, nil
}

// Listen starts listening on the transport protocol
func (p *TCPPlugin) Listen(net *Network) {
	lis, err := p.NewListener(p.address)
	if err != nil {
		glog.Errorf("transport: %+v", err)
		return
	}

	p.lis = lis

	// Handle new clients.
	go func() {
		for {
			if conn, err := p.lis.Accept(); err == nil {
				go net.Accept(conn)
			} else {
				glog.Errorf("transport: %+v", err)
			}
		}
	}()
}

// NewTCPTransport returns a new TCP protocol plugin
func NewTCPTransport(address string) (TransportInterface, error) {
	if len(address) > 5 && address[:6] != "tcp://" {
		return nil, errors.New("transport: address must begin with tcp://")
	}

	p := &Transport{
		address: address,
	}
	tcp := &TCPPlugin{
		p,
	}

	return tcp, nil
}
