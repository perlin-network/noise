package network

import (
	"errors"
	"fmt"
	"github.com/perlin-network/noise/types/lru"
	"net"
	"net/url"
	"strconv"
	"strings"
)

var domainLookupCache = lru.NewCache(1000)

// FormatAddress properly marshals a destinations information into a string.
func FormatAddress(protocol, host string, port uint16) string {
	return NewAddressInfo(protocol, host, port).String()
}

// ExtractAddressInfo derives a network scheme, host and port of a destinations
// information. Errors should the provided destination address be malformed.
func ExtractAddressInfo(address string) (*AddressInfo, error) {
	urlInfo, err := url.Parse(address)
	if err != nil {
		return nil, err
	}

	host, rawPort, err := net.SplitHostPort(urlInfo.Host)
	if err != nil {
		return nil, err
	}

	port, err := strconv.ParseUint(rawPort, 10, 16)
	if err != nil {
		return nil, err
	}

	return &AddressInfo{
		Protocol: urlInfo.Scheme,
		Host:     host,
		Port:     uint16(port),
	}, nil
}

// ToUnifiedHost resolves a domain host.
func ToUnifiedHost(host string) (string, error) {
	unifiedHost, err := domainLookupCache.Get(host, func() (interface{}, error) {
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
	})

	return unifiedHost.(string), err
}

// ToUnifiedAddress resolves and normalizes a network address.
func ToUnifiedAddress(address string) (string, error) {
	address = strings.TrimSpace(address)
	if len(address) == 0 {
		return "", fmt.Errorf("cannot dial, address was empty")
	}

	info, err := ExtractAddressInfo(address)
	if err != nil {
		return "", err
	}

	info.Host, err = ToUnifiedHost(info.Host)
	if err != nil {
		return "", err
	}

	return info.String(), nil
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
