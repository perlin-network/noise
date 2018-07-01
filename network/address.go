package network

import (
	"net"
	"strconv"
)

// AddressInfo represents a network URL.
type AddressInfo struct {
	Protocol string
	Host     string
	Port     uint16
}

// NewAddressInfo creates a new address info instance.
func NewAddressInfo(protocol, host string, port uint16) *AddressInfo {
	return &AddressInfo{
		Protocol: protocol,
		Host:     host,
		Port:     port,
	}
}

// String prints out either the URL representation of the address info, or
// solely just a joined host and port should a network scheme not be defined.
func (info *AddressInfo) String() string {
	address := net.JoinHostPort(info.Host, strconv.Itoa(int(info.Port)))
	if len(info.Protocol) > 0 {
		address = info.Protocol + "://" + address
	}
	return address
}
