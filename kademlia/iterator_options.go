package kademlia

import (
	"go.uber.org/zap"
	"time"
)

// IteratorOption represents a functional option which may be passed to NewIterator, or to (*Protocol).Find or
// (*Protocol).Discover to configure Iterator.
type IteratorOption func(it *Iterator)

// WithIteratorLogger configures the logger instance for an iterator. By default, the logger used is the logger of
// the node which the iterator is bound to.
func WithIteratorLogger(logger *zap.Logger) IteratorOption {
	return func(it *Iterator) {
		it.logger = logger
	}
}

// WithIteratorMaxNumResults sets the max number of resultant peer IDs from a single (*Iterator).Find call. By default,
// it is set to the max capacity of a routing table bucket which is 16 based on the S/Kademlia paper.
func WithIteratorMaxNumResults(maxNumResults int) IteratorOption {
	return func(it *Iterator) {
		it.maxNumResults = maxNumResults
	}
}

// WithIteratorNumParallelLookups sets the max number of parallel disjoint lookups that may occur at once while
// executing (*Iterator).Find. By default, it is set to 3 based on the S/Kademlia paper.
func WithIteratorNumParallelLookups(numParallelLookups int) IteratorOption {
	return func(it *Iterator) {
		it.numParallelLookups = numParallelLookups
	}
}

// WithIteratorNumParallelRequestsPerLookup sets the max number of parallel requests a single disjoint lookup
// may make during the execution of (*Iterator).Find. By default, it is set to 8 based on the S/Kademlia paper.
func WithIteratorNumParallelRequestsPerLookup(numParallelRequestsPerLookup int) IteratorOption {
	return func(it *Iterator) {
		it.numParallelRequestsPerLookup = numParallelRequestsPerLookup
	}
}

// WithIteratorLookupTimeout sets the max duration to wait until we declare a lookup request sent in amidst
// a single disjoint lookup to have timed out. By default, it is set to 3 seconds.
func WithIteratorLookupTimeout(lookupTimeout time.Duration) IteratorOption {
	return func(it *Iterator) {
		it.lookupTimeout = lookupTimeout
	}
}
