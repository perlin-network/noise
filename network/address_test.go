package network

import (
	"net"
	"net/url"
	"strings"
	"testing"
)

func TestFormatAddress(t *testing.T) {
	address := FormatAddress("kcp", "127.0.0.1", 10000)
	if address != "kcp://127.0.0.1:10000" {
		t.Fatal("formataddress() error")
	}
	address = FormatAddress("tcp", "localhost", 10001)
	if address != "tcp://localhost:10001" {
		t.Fatalf("formataddress() error, got %s", address)
	}
	address = FormatAddress("ppp", "localhost", 10001)
	if address != "ppp://localhost:10001" {
		t.Fatalf("formataddress() error, got %s", address)
	}
}

func TestNetworkName(t *testing.T) {
	address := NewAddressInfo("kcp", "127.0.0.1", 10000)
	if "noise" != address.Network() {
		t.Fatalf("network name got: %s, expected 'noise'", address.Network())
	}
}

func TestHostPort(t *testing.T) {
	address := NewAddressInfo("kcp", "127.0.0.1", 10000)
	if "127.0.0.1:10000" != address.HostPort() {
		t.Fatalf("network name got: %s, expected '127.0.0.1:10000'", address.HostPort())
	}
}

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

func TestParseAddress(t *testing.T) {
	_, err := ParseAddress("tcp://")
	if err == nil {
		t.Fatal("empty url error not triggered")
	}
	_, err = ParseAddress("tcp://host:k")
	if err == nil {
		t.Fatal("port url error not triggered")
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
