package nat

import (
	"fmt"
	"github.com/pkg/errors"
	"net"
	"time"
)

// Provider represents a barebones generic interface to a NAT traversal network protocol.
type Provider interface {
	ExternalIP() (net.IP, error)
	AddMapping(protocol string, externalPort, internalPort uint16, expiry time.Duration) error
	DeleteMapping(protocol string, externalPort, internalPort uint16) (err error)
}

var privateBlocks = []*net.IPNet{
	parseCIDR("127.0.0.1/8"),
	parseCIDR("::1/128"),
	parseCIDR("fe80::/10"),
	parseCIDR("10.0.0.0/8"),
	parseCIDR("172.16.0.0/12"),
	parseCIDR("192.168.0.0/16"),
}

// parseCIDR is a wrapper over `net.ParseCIDR` which panics should an error occur.
func parseCIDR(s string) *net.IPNet {
	_, block, err := net.ParseCIDR(s)
	if err != nil {
		panic(fmt.Sprintf("Bad CIDR %s: %s", s, err))
	}

	return block
}

// IsPrivateIP returns whether or not an IP is within a private range.
func IsPrivateIP(ip net.IP) bool {
	for _, ipnet := range privateBlocks {
		if ipnet.Contains(ip) {
			return true
		}
	}
	return false
}

// activateGateways returns all online private network gateways on this PC.
func activeGateways() ([]net.IP, error) {
	var gateways []net.IP

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load network interfaces")
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		addresses, err := iface.Addrs()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get interface's addresses")
		}

		for _, address := range addresses {
			address, ok := address.(*net.IPNet)
			if !ok {
				continue
			}

			if IsPrivateIP(address.IP) {
				if ip := address.IP.Mask(address.Mask).To4(); ip != nil {
					ip[3] = ip[3] | 0x01
					gateways = append(gateways, ip)
				}
			}
		}
	}

	return gateways, nil
}
