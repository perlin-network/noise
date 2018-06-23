package network

import (
	"testing"
	"net"
	"strings"
)

func TestToUnifiedAddress(t *testing.T) {
	addr, err := ToUnifiedAddress("localhost:1000")
	if err != nil {
		panic(err)
	}

	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		panic(err)
	}

	if !strings.HasPrefix(ip, "127.") && !strings.HasPrefix(ip, "::") {
		panic("localhost resolved to invalid address " + ip)
	}
	if port != "1000" {
		panic("port mismatch")
	}
}
