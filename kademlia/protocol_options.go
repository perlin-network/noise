package kademlia

import (
	"go.uber.org/zap"
	"time"
)

// ProtocolOption represents a functional option which may be passed to New to configure a Protocol.
type ProtocolOption func(p *Protocol)

// WithProtocolEvents configures an event listener for a Protocol.
func WithProtocolEvents(events Events) ProtocolOption {
	return func(p *Protocol) {
		p.events = events
	}
}

// WithProtocolLogger configures the logger instance for an iterator. By default, the logger used is the logger of
// the node which the iterator is bound to.
func WithProtocolLogger(logger *zap.Logger) ProtocolOption {
	return func(p *Protocol) {
		p.logger = logger
	}
}

// WithProtocolPingTimeout configures the amount of time to wait for until we declare a ping to have failed. Peers
// typically either are pinged through a call of (*Protocol).Ping, or in amidst the execution of Kademlia's peer
// eviction policy. By default, it is set to 3 seconds.
func WithProtocolPingTimeout(pingTimeout time.Duration) ProtocolOption {
	return func(p *Protocol) {
		p.pingTimeout = pingTimeout
	}
}
