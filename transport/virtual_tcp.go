package transport

import (
	"fmt"
	"github.com/perlin-network/noise/transport/bufconn"
	"github.com/pkg/errors"
	"net"
	"strconv"
	"strings"
)

var _ Layer = (*virtualTCP)(nil)

type virtualTCP struct {
	listeners map[string]*bufconn.Listener
}

func (t virtualTCP) Listen(port uint16) (net.Listener, error) {
	addr := fmt.Sprintf("%d", port)
	if l, ok := t.listeners[addr]; ok {
		return l, nil
	}
	t.listeners[addr] = bufconn.Listen(7, port)
	return t.listeners[addr], nil
}

func (t virtualTCP) Dial(address string) (net.Conn, error) {
	split := strings.Split(address, ":")
	addr := split[len(split)-1]
	if l, ok := t.listeners[addr]; ok {
		return l.Dial()
	}
	return nil, errors.Errorf("no listener setup for address %s, port %s", address, addr)
}

func (t virtualTCP) IP(address net.Addr) net.IP {
	split := strings.Split(address.String(), ":")
	addr := split[0]
	return net.IP(addr)
}

func (t virtualTCP) Port(address net.Addr) uint16 {
	split := strings.Split(address.String(), ":")
	addr := split[len(split)-1]
	port, err := strconv.Atoi(addr)
	if err != nil {
		panic(err)
	}
	return uint16(port)
}

// NewVirtualTCP returns a new virtualTCP instance
func NewVirtualTCP() virtualTCP {
	return virtualTCP{
		listeners: map[string]*bufconn.Listener{},
	}
}
