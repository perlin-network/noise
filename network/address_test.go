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

func BenchmarkParseAddress(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := ParseAddress("tcp://127.0.0.1:3000")
		if err != nil {
			panic(err)
		}
	}
}
