package network

import (
	"fmt"
	"net"
	"net/url"
	"testing"

	"github.com/pkg/errors"
)

func TestFormatAddress(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		protocol string
		host     string
		port     uint16
	}{
		{"kcp", "127.0.0.1", 10000},
		{"tcp", "localhost", 10001},
		{"ppp", "localhost", 10001},
	}
	for _, tt := range testCases {
		address := FormatAddress(tt.protocol, tt.host, tt.port)
		expected := fmt.Sprintf("%s://%s:%d", tt.protocol, tt.host, tt.port)
		if address != expected {
			t.Errorf("FormatAddress() = %+v, expected %+v", address, expected)
		}
	}
}

func TestNetworkName(t *testing.T) {
	t.Parallel()

	address := NewAddressInfo("kcp", "127.0.0.1", 10000)
	expected := "noise"
	if address.Network() != expected {
		t.Errorf("Network() = %s, expected %s", address.Network(), expected)
	}
}

func TestHostPort(t *testing.T) {
	t.Parallel()

	address := NewAddressInfo("kcp", "127.0.0.1", 10000)
	expected := "127.0.0.1:10000"
	if address.HostPort() != expected {
		t.Errorf("HostPort() = %s, expected %s", address.HostPort(), expected)
	}
}

func TestToUnifiedAddress(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		address      string
		expectedPort string
	}{
		{"tcp://localhost:1000", "1000"},
		{"kcp://example.com:2000", "2000"},
	}
	for _, tt := range testCases {
		address, err := ToUnifiedAddress(tt.address)
		if err != nil {
			t.Errorf("ToUnifiedAddress() = %+v, expected <nil>", err)
		}

		urlInfo, err := url.Parse(address)
		if err != nil {
			t.Error(err)
		}

		_, port, err := net.SplitHostPort(urlInfo.Host)
		if err != nil {
			t.Error(err)
		}

		if port != tt.expectedPort {
			t.Error("port mismatch")
		}
	}
}

func TestToUnifiedHost(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		address     string
		err         error
		expected    string
		description string
	}{
		{"tcp://asdf:1000", errors.New(ErrStrNoAvailableAddresses), "", "LookupHost fails to resolve"},
		{"localhost", nil, "127.0.0.1", "should resolve localhost to 127.0.0.1"},
	}
	for _, tt := range testCases {
		address, err := ToUnifiedHost(tt.address)

		if tt.err == nil && err != nil {
			t.Errorf("ToUnifiedHost() = %+v, expected %+v (%s)", err, tt.err, tt.description)
		} else if tt.err != nil && err.Error() != tt.err.Error() {
			t.Errorf("ToUnifiedHost() = %+v, expected %+v (%s)", err, tt.err, tt.description)
		}
		if address != tt.expected {
			t.Errorf("address %s, expected %s", address, tt.expected)
		}
	}
}

func TestParseAddress(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		address     string
		description string
	}{
		{"https://[2b01:e34:ef40:7730:8e70:5aff:fefe:edac]:foo/foo", "url.Parse fails"},
		{"tcp://", "empty url error not triggered"},
		{"tcp://host:k", "port url error not triggered"},
	}
	for _, tt := range testCases {
		_, err := ParseAddress(tt.address)
		if err == nil {
			t.Errorf("ParseAddress() = <nil>, expected %+v (%s)", err, tt.description)
		}
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
