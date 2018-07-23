package recvwindow

import (
	"math/rand"
	"sync"
	"testing"

	"github.com/golang/glog"
	"github.com/stretchr/testify/assert"
)

type Entry struct {
	idx int
	val uint64
}

func TestRecvWindowBasic(t *testing.T) {
	t.Parallel()
	rwSize := 1024
	batchSize := 333
	numBatches := 543

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

func TestRecvWindowConcurrency(t *testing.T) {
	t.Parallel()
	rwSize := 1024
	batchSize := 33
	numBatches := 1234

	rw := NewRecvWindow(rwSize)

	wg := &sync.WaitGroup{}

	var expected sync.Map

	// loop over several batches to work the ring buffer
	for i := 0; i < numBatches; i++ {
		wg.Add(1)
		for j := 0; j < batchSize; j++ {
			entry := &Entry{
				idx: j + i*batchSize,
				val: rand.Uint64(),
			}

			expected.Store(entry.idx, entry)
			assert.Equalf(t, nil, rw.Insert(entry), "should not error for entry %d", j+i*batchSize)
		}
		glog.Infof("Inserted %d items\n", batchSize)

		go func() {
			defer wg.Done()
			// get them out and check them
			window := rw.PopWindow()
			glog.Infof("Window had %d items\n", len(window))
			for j, val := range window {
				entry, ok := val.(*Entry)
				assert.Equal(t, true, ok, "should match entry %d", j)
				assert.NotEqualf(t, nil, entry, "should not match entry %d", j)
				expect, loaded := expected.Load(entry.idx)
				assert.Equal(t, true, loaded, "should match entry %d", j)
				e := expect.(*Entry)
				assert.NotEqualf(t, nil, e, "should not match entry %d", j)
				assert.Equalf(t, e.val, entry.val, "should match entry %d", j)
				assert.Equalf(t, e.idx, entry.idx, "should match entry %d", j)
			}
		}()
	}
	glog.Infof("Finished inserting %d items\n", numBatches*batchSize)

	wg.Wait()

	// make sure nothing is left
	assert.Equal(t, 0, len(rw.PopWindow()))
}
