package kademlia

import "go.uber.org/zap"

type IteratorOption func(it *Iterator)

func WithIteratorLogger(logger *zap.Logger) IteratorOption {
	return func(it *Iterator) {
		it.logger = logger
	}
}

func WithIteratorMaxNumResults(maxNumResults int) IteratorOption {
	return func(it *Iterator) {
		it.maxNumResults = maxNumResults
	}
}

func WithIteratorNumParallelLookups(numParallelLookups int) IteratorOption {
	return func(it *Iterator) {
		it.numParallelLookups = numParallelLookups
	}
}

func WithIteratorNumParallelRequestsPerLookup(numParallelRequestsPerLookup int) IteratorOption {
	return func(it *Iterator) {
		it.numParallelRequestsPerLookup = numParallelRequestsPerLookup
	}
}
