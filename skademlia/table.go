// Copyright (c) 2019 Perlin
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package skademlia

import (
	"bytes"
	"container/list"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"sort"
	"sync"
	"sync/atomic"
)

var (
	ErrBucketFull = errors.New("bucket is full")
)

type Table struct {
	self *ID

	buckets    []*Bucket
	bucketSize uint32
}

func NewTable(self *ID) *Table {
	t := &Table{
		self: self,

		buckets:    make([]*Bucket, len(self.checksum)*8),
		bucketSize: 16,
	}

	for i := range t.buckets {
		t.buckets[i] = new(Bucket)
	}

	b := t.buckets[len(t.buckets)-1]
	b.PushFront(self)

	return t
}

func (t *Table) getBucketSize() int {
	return int(atomic.LoadUint32(&t.bucketSize))
}

func (t *Table) setBucketSize(size int) {
	atomic.StoreUint32(&t.bucketSize, uint32(size))
}

func (t *Table) Find(b *Bucket, target *ID) *list.Element {
	if target == nil {
		return nil
	}

	var element *list.Element

	b.RLock()

	for e := b.Front(); e != nil; e = e.Next() {
		if e.Value.(*ID).checksum == target.checksum {
			element = e
			break
		}
	}

	b.RUnlock()

	return element
}

func (t *Table) Delete(b *Bucket, target *ID) bool {
	e := t.Find(b, target)

	if e == nil {
		return false
	}

	b.Lock()
	defer b.Unlock()

	return b.Remove(e) != nil
}

func (t *Table) Update(target *ID) error {
	if target == nil {
		return nil
	}

	b := t.buckets[getBucketID(t.self.checksum, target.checksum)]

	if found := t.Find(b, target); found != nil {
		b.Lock()

		// address might differ for same public key (checksum
		found.Value.(*ID).SetAddress(target.address)

		b.MoveToFront(found)
		b.Unlock()

		return nil
	}

	if b.Len() < t.getBucketSize() {
		b.Lock()
		b.PushFront(target)
		b.Unlock()

		return nil
	}

	return ErrBucketFull
}

func (t *Table) FindClosest(target *ID, k int) []*ID {
	var checksum [blake2b.Size256]byte

	if target != nil {
		checksum = target.checksum
	}

	var closest []*ID

	f := func(b *Bucket) {
		b.RLock()

		for e := b.Front(); e != nil; e = e.Next() {
			if id := e.Value.(*ID); id.checksum != checksum {
				closest = append(closest, id)
			}
		}

		b.RUnlock()
	}

	idx := getBucketID(t.self.checksum, checksum)

	f(t.buckets[idx])

	for i := 1; len(closest) < k && (idx-i >= 0 || idx+i < len(t.buckets)); i++ {
		if idx-i >= 0 {
			f(t.buckets[idx-i])
		}

		if idx+i < len(t.buckets) {
			f(t.buckets[idx+i])
		}
	}

	sort.Slice(closest, func(i, j int) bool {
		return bytes.Compare(xor(closest[i].checksum[:], checksum[:]), xor(closest[j].checksum[:], checksum[:])) == -1
	})

	if len(closest) > k {
		closest = closest[:k]
	}

	return closest
}

func getBucketID(self, target [blake2b.Size256]byte) int {
	l := prefixLen(xor(target[:], self[:]))
	if l == blake2b.Size256*8 {
		return l - 1
	}

	return l
}

type Bucket struct {
	sync.RWMutex
	list.List
}

func (b *Bucket) Len() int {
	b.RLock()
	size := b.List.Len()
	b.RUnlock()

	return size
}
