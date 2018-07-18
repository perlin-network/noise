package network

import (
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/perlin-network/noise/types/lru"
	"github.com/pkg/errors"
)

var domainLookupCache = lru.NewCache(1000)

// AddressInfo represents a network URL.
type AddressInfo struct {
	Protocol string
	Host     string
	Port     uint16
}

const (
	networkClientName = "noise"
)

// Errors
var (
	// ErrStrInvalidAddress returns if an invalid address was given
	ErrStrInvalidAddress       = "address: invalid address"
	ErrStrAddressEmpty         = "address: cannot dial, address was empty"
	ErrStrNoAvailableAddresses = "address: no available addresses"
)

// NewAddressInfo creates a new AddressInfo instance.
func NewAddressInfo(protocol string, host string, port uint16) *AddressInfo {
	return &AddressInfo{
		Protocol: protocol,
		Host:     host,
		Port:     port,
	}
}

// Network returns the name of the network client.
func (info *AddressInfo) Network() string {
	return networkClientName
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

// HostPort returns the address wihout protocol, in the format `host:port`.
func (info *AddressInfo) HostPort() string {
	return net.JoinHostPort(info.Host, strconv.Itoa(int(info.Port)))
}

// FormatAddress properly marshals a destinations information into a string.
func FormatAddress(protocol string, host string, port uint16) string {
	return NewAddressInfo(protocol, host, port).String()
}

// ParseAddress derives a network scheme, host and port of a destinations
// information. Errors should the provided destination address be malformed.
func ParseAddress(address string) (*AddressInfo, error) {
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
				return "", errors.New(ErrStrNoAvailableAddresses)
			}
			if len(addresses) == 0 {
				return "", errors.New(ErrStrNoAvailableAddresses)
			}

			host = addresses[0]

			// Hacky localhost fix.
			if host == "::1" {
				host = "127.0.0.1"
			}
		}

		return host, nil
	})

	if unifiedHost == nil {
		return "", errors.New(ErrStrNoAvailableAddresses)
	}

	return unifiedHost.(string), err
}

// ToUnifiedAddress resolves and normalizes a network address.
func ToUnifiedAddress(address string) (string, error) {
	address = strings.TrimSpace(address)
	if len(address) == 0 {
		return "", errors.New(ErrStrAddressEmpty)
	}

	info, err := ParseAddress(address)
	if err != nil {
		return "", err
	}

	info.Host, err = ToUnifiedHost(info.Host)
	if err != nil {
		return "", err
	}

	return info.String(), nil
}
