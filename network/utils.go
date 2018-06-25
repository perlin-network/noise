package network

import (
	"errors"
	"fmt"
	"net"
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

		// Hacky localhost fix.
		if host == "::1" {
			host = "127.0.0.1"
		}
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

// Filter out duplicate addresses.
func FilterPeers(host string, port int, peers []string) (filtered []string) {
	address := fmt.Sprintf("%s:%d", host, port)

	visited := make(map[string]struct{})
	visited[address] = struct{}{}

	for _, peer := range peers {
		if _, exists := visited[peer]; !exists {
			filtered = append(filtered, peer)
			visited[peer] = struct{}{}
		}
	}
	return filtered
}

// resolves a list of addresses, ignores bad ones
func unifyAddresses(addresses []string) []string {
	retVal := []string{}
	for _, address := range addresses {
		resolved, err := ToUnifiedAddress(address)
		if err != nil {
			continue
		}
		retVal = append(retVal, resolved)
	}
	return retVal
}
