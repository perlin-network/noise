package nat

import "net"

type Provider interface {
	ExternalIP() net.IP
}
