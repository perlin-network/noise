package network

import (
	"net"
	"reflect"
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

func TestFilterPeers(t *testing.T) {
	result := FilterPeers("10.0.0.3", 3000, []string{
		"10.0.0.5:3000",
		"10.0.0.1:3000",
		"10.0.0.1:3000",
		"10.0.0.1:2000",
		"10.0.0.3:3000",
		"10.0.0.6:3000",
		"localhost:3004",
	})
	expected := []string{
		"10.0.0.5:3000",
		"10.0.0.1:3000",
		"10.0.0.1:2000",
		"10.0.0.6:3000",
		"127.0.0.1:3004",
	}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("Unexpected got %v, but expected %v", result, expected)
	}
}
