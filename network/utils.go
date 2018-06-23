package network

import (
	"net"
	"errors"
)

func ToUnifiedHost(host string) (string, error) {
	if net.ParseIP(host) == nil {
		// Probably a domain name is provided.
		addrs, err := net.LookupHost(host)
		if err != nil {
			return "", err
		}
		if len(addrs) == 0 {
			return "", errors.New("no available addresses")
		}

		host = addrs[0]
	}

	return host, nil
}

func ToUnifiedAddress(address string) (string, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", err
	}

	host, err = ToUnifiedHost(host)
	if err != nil {
		return "", err
	}

	return net.JoinHostPort(host, port), nil
}
