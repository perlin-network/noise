package noise

import (
	"go.uber.org/zap"
	"net"
	"time"
)

// NodeOption represents a functional option that may be passed to NewNode for instantiating a new node instance
// with configured values.
type NodeOption func(n *Node)

// WithNodeMaxDialAttempts sets the max number of attempts a connection is dialed before it is determined to have
// failed. By default, the max number of attempts a connection is dialed is 3.
func WithNodeMaxDialAttempts(maxDialAttempts uint) NodeOption {
	return func(n *Node) {
		if maxDialAttempts == 0 {
			maxDialAttempts = 1
		}

		n.maxDialAttempts = maxDialAttempts
	}
}

// WithNodeMaxInboundConnections sets the max number of inbound connections the connection pool a node maintains allows
// at any given moment in time. By default, the max number of inbound connections is 128. Exceeding the max number
// causes the connection pool to release the oldest inbound connection in the pool.
func WithNodeMaxInboundConnections(maxInboundConnections uint) NodeOption {
	return func(n *Node) {
		if maxInboundConnections == 0 {
			maxInboundConnections = 128
		}

		n.maxInboundConnections = maxInboundConnections
	}
}

// WithNodeMaxOutboundConnections sets the max number of outbound connections the connection pool a node maintains
// allows at any given moment in time. By default, the maximum number of outbound connections is 128. Exceeding the
// max number causes the connection pool to release the oldest outbound connection in the pool.
func WithNodeMaxOutboundConnections(maxOutboundConnections uint) NodeOption {
	return func(n *Node) {
		if maxOutboundConnections == 0 {
			maxOutboundConnections = 128
		}

		n.maxOutboundConnections = maxOutboundConnections
	}
}

// WithNodeMaxRecvMessageSize sets the max number of bytes a node is willing to receive from a peer. If the limit is
// ever exceeded, the peer is disconnected with an error. Setting this option to zero will disable the limit. By
// default, the max number of bytes a node is willing to receive from a peer is set to 4MB.
func WithNodeMaxRecvMessageSize(maxRecvMessageSize uint32) NodeOption {
	return func(n *Node) {
		n.maxRecvMessageSize = maxRecvMessageSize
	}
}

// WithNodeNumWorkers sets the max number of workers a node will spawn to handle incoming peer messages. By default,
// the max number of workers a node will spawn is the number of CPUs available to the Go runtime specified by
// runtime.NumCPU(). The minimum number of workers which need to be spawned is 1.
func WithNodeNumWorkers(numWorkers uint) NodeOption {
	return func(n *Node) {
		if numWorkers == 0 {
			numWorkers = 1
		}

		n.numWorkers = numWorkers
	}
}

// WithNodeIdleTimeout sets the duration in which should there be no subsequent reads/writes on a connection, the
// connection shall timeout and have resources related to it released. By default, the timeout is set to be 3 seconds.
// If an idle timeout of 0 is specified, idle timeouts will be disabled.
func WithNodeIdleTimeout(idleTimeout time.Duration) NodeOption {
	return func(n *Node) {
		n.idleTimeout = idleTimeout
	}
}

// WithNodeLogger sets the logger implementation that the node shall use. By default, zap.NewNop() is assigned which
// disables any logs.
func WithNodeLogger(logger *zap.Logger) NodeOption {
	return func(n *Node) {
		if logger == nil {
			logger = zap.NewNop()
		}

		n.logger = logger
	}
}

// WithNodeID sets the nodes ID, and public address. By default, the ID is set with an address that is set to the
// binding host and port upon calling (*Node).Listen should the address not be configured.
func WithNodeID(id ID) NodeOption {
	return func(n *Node) {
		n.id = id
		n.addr = id.Address
	}
}

// WithNodePrivateKey sets the private key of the node. By default, a random private key is generated using
// GenerateKeys should no private key be configured.
func WithNodePrivateKey(privateKey PrivateKey) NodeOption {
	return func(n *Node) {
		n.privateKey = privateKey
	}
}

// WithNodeBindHost sets the TCP host IP address which the node binds itself to and listens for new incoming peer
// connections on. By default, it is unspecified (0.0.0.0).
func WithNodeBindHost(host net.IP) NodeOption {
	return func(n *Node) {
		n.host = host
	}
}

// WithNodeBindPort sets the TCP port which the node binds itself to and listens for new incoming peer connections on.
// By default, a random port is assigned by the operating system.
func WithNodeBindPort(port uint16) NodeOption {
	return func(n *Node) {
		n.port = port
	}
}

// WithNodeAddress sets the public address of this node which is advertised on the ID sent to peers during a handshake
// protocol which is performed when interacting with peers this node has had no live connection to beforehand. By
// default, it is left blank, and initialized to 'binding host:binding port' upon calling (*Node).Listen.
func WithNodeAddress(addr string) NodeOption {
	return func(n *Node) {
		n.addr = addr
	}
}
