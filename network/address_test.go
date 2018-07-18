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
		t.Error("formataddress() error")
	}
	address = FormatAddress("tcp", "localhost", 10001)
	if address != "tcp://localhost:10001" {
		t.Errorf("formataddress() error, got %s", address)
	}
	address = FormatAddress("ppp", "localhost", 10001)
	if address != "ppp://localhost:10001" {
		t.Errorf("formataddress() error, got %s", address)
	}
}

func TestNetworkName(t *testing.T) {
	address := NewAddressInfo("kcp", "127.0.0.1", 10000)
	if "noise" != address.Network() {
		t.Errorf("network name got: %s, expected 'noise'", address.Network())
	}
}

func TestHostPort(t *testing.T) {
	address := NewAddressInfo("kcp", "127.0.0.1", 10000)
	if "127.0.0.1:10000" != address.HostPort() {
		t.Errorf("network name got: %s, expected '127.0.0.1:10000'", address.HostPort())
	}
}

func TestToUnifiedAddress(t *testing.T) {
	address, err := ToUnifiedAddress("tcp://localhost:1000")
	if err != nil {
		t.Error(err)
	}

	urlInfo, err := url.Parse(address)
	if err != nil {
		t.Error(err)
	}

	ip, port, err := net.SplitHostPort(urlInfo.Host)
	if err != nil {
		t.Error(err)
	}

	if !strings.HasPrefix(ip, "127.") && !strings.HasPrefix(ip, "::") {
		t.Error("localhost resolved to invalid address", ip)
	}
	if port != "1000" {
		t.Error("port mismatch")
	}
}

func TestParseAddress(t *testing.T) {
	_, err := ParseAddress("tcp://")
	if err == nil {
		t.Error("empty url error not triggered")
	}
	_, err = ParseAddress("tcp://host:k")
	if err == nil {
		t.Error("port url error not triggered")
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
