package recvwindow

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Entry struct {
	idx int
	val uint64
}

func TestRecvWindow(t *testing.T) {
	rwSize := 1024
	batchSize := 333
	numBatches := 1234

	rw := NewRecvWindow(rwSize)

	// loop over several batches to work the ring buffer
	for i := 0; i < numBatches; i++ {
		var expected []*Entry

		// insert items
		for j := 0; j < batchSize; j++ {
			entry := &Entry{
				idx: j + i*batchSize,
				val: rand.Uint64(),
			}
			expected = append(expected, entry)
			rw.Insert(entry)
		}

		// get them out and check them
		ready := rw.PopWindow()
		assert.Equal(t, batchSize, len(expected))
		assert.Equal(t, batchSize, len(ready))
		for j, val := range expected {
			entry, ok := ready[j].(*Entry)
			assert.Equal(t, true, ok, "should match entry %d", j)
			assert.Equalf(t, j+i*batchSize, entry.idx, "should match entry %d", j)
			assert.Equalf(t, val.val, entry.val, "should match entry %d", j)
		}
	}
}
