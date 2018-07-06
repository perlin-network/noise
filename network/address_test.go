package network

import (
	"net"
	"net/url"
	"strings"
	"testing"
)

func TestToUnifiedAddress(t *testing.T) {
	address, err := ToUnifiedAddress("tcp://localhost:1000")
	if err != nil {
		t.Fatal(err)
	}

	urlInfo, err := url.Parse(address)
	if err != nil {
		t.Fatal(err)
	}

	ip, port, err := net.SplitHostPort(urlInfo.Host)
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
