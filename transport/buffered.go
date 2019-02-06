package transport

import (
	"fmt"
	"github.com/perlin-network/noise/transport/bufconn"
	"github.com/pkg/errors"
	"net"
	"strconv"
	"strings"
	"sync"
)

var _ Layer = (*Buffered)(nil)

type Buffered struct {
	sync.Mutex
	listeners map[string]*bufconn.Listener
}

func (t *Buffered) Listen(port uint16) (net.Listener, error) {
	t.Lock()
	defer t.Unlock()

	addr := fmt.Sprintf("%d", port)
	if l, ok := t.listeners[addr]; ok {
		return l, nil
	}
	t.listeners[addr] = bufconn.Listen(port)
	return t.listeners[addr], nil
}

func (t *Buffered) Dial(address string) (net.Conn, error) {
	t.Lock()
	defer t.Unlock()

	split := strings.Split(address, ":")
	addr := split[len(split)-1]
	if l, ok := t.listeners[addr]; ok {
		return l.Dial()
	}
	return nil, errors.Errorf("no listener setup for address %s, port %s", address, addr)
}

func (t *Buffered) IP(address net.Addr) net.IP {
	split := strings.Split(address.String(), ":")
	addr := split[0]
	return net.IP(addr)
}

func (t *Buffered) Port(address net.Addr) uint16 {
	split := strings.Split(address.String(), ":")
	addr := split[len(split)-1]
	port, err := strconv.Atoi(addr)
	if err != nil {
		panic(err)
	}
	return uint16(port)
}

// NewVirtualTCP returns a new virtualTCP instance
func NewBuffered() *Buffered {
	return &Buffered{
		listeners: map[string]*bufconn.Listener{},
	}
}
