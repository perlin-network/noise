package network

import (
	"errors"
	"net"
	"net/url"
)

// ToUnifiedHost resolves a domain host.
func ToUnifiedHost(host string) (string, error) {
	if net.ParseIP(host) == nil {
		// Probably a domain name is provided.
		addresses, err := net.LookupHost(host)
		if err != nil {
			return "", err
		}
		if len(addresses) == 0 {
			return "", errors.New("no available addresses")
		}

		host = addresses[0]

		// Hacky localhost fix.
		if host == "::1" {
			host = "127.0.0.1"
		}
	}

	return host, nil
}

// ToUnifiedAddress resolves and normalizes a network address.
func ToUnifiedAddress(address string) (string, error) {
	urlInfo, err := url.Parse(address)
	if err != nil {
		return "", err
	}

	host, port, err := net.SplitHostPort(urlInfo.Host)
	if err != nil {
		return "", err
	}

	host, err = ToUnifiedHost(host)
	if err != nil {
		return "", err
	}

	return urlInfo.Scheme + "://" + net.JoinHostPort(host, port), nil
}

// FilterPeers filters out duplicate/empty addresses.
func FilterPeers(address string, peers []string) (filtered []string) {
	visited := make(map[string]struct{})
	visited[address] = struct{}{}

	for _, peerAddress := range peers {
		if len(peerAddress) == 0 {
			continue
		}

		resolved, err := ToUnifiedAddress(peerAddress)
		if err != nil {
			continue
		}
		if _, exists := visited[resolved]; !exists {
			filtered = append(filtered, resolved)
			visited[resolved] = struct{}{}
		}
	}
	return filtered
}
