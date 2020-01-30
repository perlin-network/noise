package kademlia

import "go.uber.org/zap"

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
