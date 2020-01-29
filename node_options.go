package noise

import (
	"go.uber.org/zap"
	"net"
	"time"
)

type NodeOption func(n *Node)

func WithNodeMaxDialAttempts(maxDialAttempts int) NodeOption {
	return func(n *Node) {
		n.maxDialAttempts = maxDialAttempts
	}
}

func WithNodeMaxInboundConnections(maxInboundConnections int) NodeOption {
	return func(n *Node) {
		n.maxInboundConnections = maxInboundConnections
	}
}

func WithNodeMaxOutboundConnections(maxOutboundConnections int) NodeOption {
	return func(n *Node) {
		n.maxOutboundConnections = maxOutboundConnections
	}
}

func WithNodeIdleTimeout(idleTimeout time.Duration) NodeOption {
	return func(n *Node) {
		n.idleTimeout = idleTimeout
	}
}

func WithNodeLogger(logger *zap.Logger) NodeOption {
	return func(n *Node) {
		n.logger = logger
	}
}

func WithNodePrivateKey(privateKey PrivateKey) NodeOption {
	return func(n *Node) {
		n.privateKey = privateKey
	}
}

func WithNodeBindHost(host net.IP) NodeOption {
	return func(n *Node) {
		n.host = host
	}
}

func WithNodeBindPort(port uint16) NodeOption {
	return func(n *Node) {
		n.port = port
	}
}

func WithNodeAddress(addr string) NodeOption {
	return func(n *Node) {
		n.addr = addr
	}
}
