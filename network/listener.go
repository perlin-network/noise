package network

import (
	"fmt"
	"net"

	"github.com/xtaci/kcp-go"
)

// NewTcpListener is a convenience function to return a new TCP listener
func NewTcpListener(addr string) (net.Listener, error) {
	addrInfo, err := ParseAddress(addr)
	if err != nil {
		return nil, err
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", addrInfo.Port))
	if err != nil {
		return nil, err
	}

	return lis, nil
}

// NewKcpListener is a convenience function to return a new KCP listener
func NewKcpListener(addr string) (net.Listener, error) {
	addrInfo, err := ParseAddress(addr)
	if err != nil {
		return nil, err
	}

	lis, err := kcp.ListenWithOptions(fmt.Sprintf("localhost:%d", addrInfo.Port), nil, 0, 0)
	if err != nil {
		return nil, err
	}

	return lis, nil
}
