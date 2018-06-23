package network

import (
	"net"
	"strings"
	"testing"
)

func TestToUnifiedAddress(t *testing.T) {
	addr, err := ToUnifiedAddress("localhost:1000")
	if err != nil {
		t.Fatal(err)
	}

	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(ip, "127.") && !strings.HasPrefix(ip, "::") {
		t.Fatal("localhost resolved to invalid address", ip)
	}
	if port != "1000" {
		t.Fatal("port mismatch")
	}
}
