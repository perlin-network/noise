package transport

import (
	"fmt"
	"github.com/perlin-network/noise/internal/bufconn"
	"github.com/pkg/errors"
	"math/rand"
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

func (t *Buffered) String() string {
	return "buffered"
}

func (t *Buffered) Listen(host string, port uint16) (net.Listener, error) {
	t.Lock()
	defer t.Unlock()

	if net.ParseIP(host) == nil {
		return nil, errors.Errorf("unable to parse host as IP: %s", host)
	}

	if port == 0 {
		port = uint16(rand.Intn(50000) + 10000)
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	if l, ok := t.listeners[addr]; ok {
		return l, nil
	}

	t.listeners[addr] = bufconn.Listen(host, port)
	return t.listeners[addr], nil
}

func (t *Buffered) Dial(address string) (net.Conn, error) {
	t.Lock()
	defer t.Unlock()

	if l, ok := t.listeners[address]; ok {
		return l.Dial()
	}

	return nil, errors.Errorf("no listener setup for address %s", address)
}

func (t *Buffered) IP(address net.Addr) net.IP {
	split := strings.Split(address.String(), ":")
	addr := split[0]
	return net.ParseIP(addr)
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
