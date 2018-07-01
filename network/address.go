package network

import (
	"net"
	"strconv"
)

type AddressInfo struct {
	Protocol string
	Host     string
	Port     uint16
}

func NewAddressInfo(protocol, host string, port uint16) *AddressInfo {
	return &AddressInfo{
		Protocol: protocol,
		Host:     host,
		Port:     port,
	}
}

func (info *AddressInfo) String() string {
	address := net.JoinHostPort(info.Host, strconv.Itoa(int(info.Port)))
	if len(info.Protocol) > 0 {
		address = info.Protocol + "://" + address
	}
	return address
}
