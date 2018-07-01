package network

import (
	"net"
	"net/url"
	"reflect"
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

func TestFilterPeers(t *testing.T) {
	result := FilterPeers("tcp://10.0.0.3:3000", []string{
		"tcp://10.0.0.5:3000",
		"tcp://10.0.0.1:3000",
		"tcp://10.0.0.1:3000",
		"",
		"tcp://10.0.0.1:2000",
		"tcp://10.0.0.3:3000",
		"kcp://10.0.0.3:3000",
		"",
		"tcp://10.0.0.6:3000",
		"tcp://localhost:3004",
		"tcp://::1:3005",
	})
	expected := []string{
		"tcp://10.0.0.5:3000",
		"tcp://10.0.0.1:3000",
		// "tcp://10.0.0.1:3000" is a duplicate
		"tcp://10.0.0.1:2000",
		// "tcp://10.0.0.3:3000" is filtered
		"kcp://10.0.0.3:3000",
		"tcp://10.0.0.6:3000",
		"tcp://127.0.0.1:3004",
		// "tcp://::1:3005" will be removed
	}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("Unexpected got %v, but expected %v", result, expected)
	}
}
