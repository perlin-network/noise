package kademlia

import (
	"encoding/binary"
	"github.com/perlin-network/noise"
	"github.com/stretchr/testify/assert"
	"math"
	"math/rand"
	"net"
	"testing"
)

// TestDistanceMetric demonstrates the connectivity of a network where peers have routing tables defined under the
// Kademlia-inspired XOR distance metric.
func TestDistanceMetric(t *testing.T) {
	tables := make([]*Table, 0, 128)
	ids := make([]noise.ID, 0, 128)

	// Create a bunch of random IDs.

	for i := 0; i < cap(ids); i++ {
		pub, _, err := noise.GenerateKeys(nil)
		assert.NoError(t, err)

		host := make(net.IP, 4)
		binary.BigEndian.PutUint32(host[:], uint32(i))

		port := uint16(i % math.MaxUint16)

		id := noise.NewID(pub, host, port)

		tables = append(tables, NewTable(id))
		ids = append(ids, id)
	}

	// Populate each table with at most BucketSize randomly-selected IDs.

	indices := rand.Perm(cap(ids))

	for i := 0; i < cap(ids); i++ {
		for j := 0; j < BucketSize; j++ {
			idx := indices[0]
			indices = append(indices[1:], idx)

			_, _ = tables[i].Update(ids[idx])
		}
	}

	// Assert that if we iterate through each table's closest peers, that we would eventually iterate over every
	// single ID we had generated. This asserts that the network is well-connected due to the choice of distance metric.

	seen := make(map[noise.PublicKey]struct{}, cap(ids))

	for _, table := range tables {
		for _, id := range table.Peers() {
			seen[id.ID] = struct{}{}
		}
	}

	assert.Len(t, seen, cap(ids))
}
