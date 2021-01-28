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

	if !isAllowedIP(addr.IP) {
		addr.IP = nil
	}

	return addr.String(), nil
}

func isAllowedIP(ip net.IP) bool {
	return !(ip.IsLoopback() || ip.IsUnspecified() || ip.String() == "<nil>")
}

func normalizeIP(ip net.IP) string {
	if isAllowedIP(ip) {
		return ip.String()
	}

	return ""
}
