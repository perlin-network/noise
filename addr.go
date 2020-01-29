package noise

import (
	"net"
)

// ResolveAddress resolves an address using net.ResolveTCPAddress("tcp", (*net.Conn).RemoteAddr()) and nullifies the
// IP if the IP is unspecified or is a loopback address. It then returns the string representation of the address, or
// an error if the resolution of the address fails.
func ResolveAddress(address string) (string, error) {
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return "", err
	}

	addr.IP = resolveIP(addr.IP)

	return addr.String(), nil
}

func resolveIP(ip net.IP) net.IP {
	if ip.IsLoopback() || ip.IsUnspecified() {
		return nil
	}

	return ip
}

func normalizeIP(ip net.IP) string {
	ip = resolveIP(ip)

	str := ip.String()
	if str == "<nil>" {
		return ""
	}

	return str
}
