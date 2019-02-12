package skademlia

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/identity/ed25519"
	"github.com/perlin-network/noise/protocol"
	"github.com/stretchr/testify/assert"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"
)

var (
	ttid1 = NewID("0000", ed25519.Random().PublicID())
	ttid2 = NewID("0001", ed25519.Random().PublicID())
	ttid3 = NewID("0002", ed25519.Random().PublicID())
	ttid4 = NewID("0003", ed25519.Random().PublicID())

	ttidBytes = ttid1.PublicID()
)

func MustReadRand(size int) []byte {
	out := make([]byte, size)
	_, err := rand.Read(out)
	if err != nil {
		panic(err)
	}
	return out
}

func RandByte() byte {
	return MustReadRand(1)[0]
}

func TestSelf(t *testing.T) {
	t.Parallel()

	routingTable := newTable(ttid1)
	routingTable.Update(ttid2)
	routingTable.Update(ttid3)

	assert.Equal(t, "0000", routingTable.self.(ID).address)
	assert.EqualValues(t, routingTable.self.PublicID(), ttidBytes)
}

func TestGetPeerAddresses(t *testing.T) {
	t.Parallel()

	routingTable := newTable(ttid1)
	routingTable.Update(ttid2)
	routingTable.Update(ttid3)

	tester := routingTable.GetPeers()
	sort.Strings(tester)
	testee := []string{"0001", "0002"}

	assert.EqualValues(t, tester, testee)
}

func TestGet(t *testing.T) {
	t.Parallel()

	routingTable := newTable(ttid1)
	routingTable.Update(ttid2)

	// exists
	found, ok := routingTable.Get(ttid1)
	assert.True(t, ok)
	assert.NotNil(t, found)
	assert.EqualValues(t, ttid1, found)

	// doesn't exist
	found, ok = routingTable.Get(ttid3)
	assert.False(t, ok)
	assert.Nil(t, found)

	// add new
	routingTable.Update(ttid3)
	found, ok = routingTable.Get(ttid3)
	assert.True(t, ok)
	assert.NotNil(t, found)
	assert.EqualValues(t, ttid3, found)

	routingTable.Delete(ttid1)
	found, ok = routingTable.Get(ttid1)
	assert.False(t, ok)
	assert.Nil(t, found)
}

func TestGetPeers(t *testing.T) {
	t.Parallel()

	routingTable := newTable(ttid1)
	routingTable.Update(ttid2)

	// return the other, exclude self
	peers := routingTable.GetPeers()
	assert.Equal(t, 1, len(peers))
	assert.EqualValues(t, []string{ttid2.address}, peers)
}

func TestRemovePeer(t *testing.T) {
	t.Parallel()

	routingTable := newTable(ttid1)
	routingTable.Update(ttid2)
	routingTable.Update(ttid3)

	routingTable.Delete(ttid2)
	testee := routingTable.GetPeers()
	sort.Strings(testee)
	tester := []string{"0002"}

	assert.EqualValues(t, tester, testee)
}

func TestUpdate(t *testing.T) {
	// modify the bucket size for this test
	defaultBucketSize := BucketSize()
	defer func() {
		bucketSize = defaultBucketSize
	}()

	// self key generates bucket id 255
	idKey1 := []byte{124, 224, 147, 208, 211, 103, 166, 113, 153, 104, 83, 62, 61, 145, 8, 211, 144, 164, 224, 191, 177, 205, 198, 94, 92, 35, 76, 83, 229, 46, 219, 110}
	id1 := NewID("0001", idKey1)

	// key generates bucket id 8
	idKey2 := []byte{210, 127, 212, 137, 47, 66, 40, 189, 231, 239, 210, 168, 52, 15, 223, 66, 199, 199, 156, 61, 132, 56, 102, 223, 32, 175, 169, 241, 156, 46, 83, 98}
	id2 := NewID("0002", idKey2)

	// key generates bucket id 8
	idKey3 := []byte{228, 61, 230, 169, 243, 78, 244, 44, 82, 76, 54, 56, 98, 135, 227, 158, 114, 251, 56, 160, 208, 60, 121, 41, 197, 63, 235, 41, 236, 66, 222, 219}
	id3 := NewID("0003", idKey3)

	bucketSize = 1
	routingTable := newTable(id1)

	bucketID2 := routingTable.bucketID(idKey2)
	bucketID3 := routingTable.bucketID(idKey3)
	assert.Equal(t, bucketID2, bucketID3, "only 1 bucket, should be the same")

	err := routingTable.Update(id2)
	assert.Nil(t, err)
	err = routingTable.Update(id3)
	assert.Equal(t, err, ErrBucketFull)

	bucketSize = 2
	err = routingTable.Update(id3)
	assert.Nil(t, err, "should not be full with 2 entries anymore")
}

func TestFindClosestPeers(t *testing.T) {
	t.Parallel()

	nodes := []ID{}

	nodes = append(nodes,
		ID{address: "0000", hash: []byte("12345678901234567890123456789010")},
		ID{address: "0001", hash: []byte("12345678901234567890123456789011")},
		ID{address: "0002", hash: []byte("12345678901234567890123456789012")},
		ID{address: "0003", hash: []byte("12345678901234567890123456789013")},
		ID{address: "0004", hash: []byte("12345678901234567890123456789014")},
		ID{address: "0005", hash: []byte("00000000000000000000000000000000")},
	)
	for _, node := range nodes {
		node.publicKey = node.hash
	}

	routingTable := newTable(nodes[0])
	for i := 1; i <= 5; i++ {
		routingTable.Update(nodes[i])
	}
	testee := []ID{}
	for _, peer := range FindClosestPeers(routingTable, nodes[5].Hash(), 3) {
		testee = append(testee, peer.(ID))
	}
	if len(testee) != 3 {
		t.Fatalf("findclosestpeers() error, size of return should be 3, but found %d", len(testee))
	}
	answerKeys := []int{5, 2, 3}
	for i := 0; i <= 2; i++ {
		_answer := nodes[answerKeys[i]]
		assert.EqualValues(t, _answer, testee[i])
	}

	testee = []ID{}
	for _, peer := range FindClosestPeers(routingTable, nodes[4].Hash(), 2) {
		testee = append(testee, peer.(ID))
	}
	if len(testee) != 2 {
		t.Fatalf("findclosestpeers() error, size of return should be 2, but found %d", len(testee))
	}
	answerKeys = []int{4, 5}
	// TODO: should be {4, 2} not {4, 5}
	//answerKeys = []int{4, 2}
	for i := 0; i <= 1; i++ {
		_answer := nodes[answerKeys[i]]
		assert.EqualValues(t, _answer, testee[i])
	}

	// make sure the bucket size is right too
	assert.Nil(t, routingTable.bucket(len(ttid1.Hash())*8+1))
	assert.NotNil(t, routingTable.bucket(len(ttid1.Hash())*8-1))

}

func TestFindClosestConcurrent(t *testing.T) {
	t.Parallel()

	const IDPoolSize = 16
	const concurrentCount = 16

	ids := make([]unsafe.Pointer, IDPoolSize) // Element type: *peer.Id

	id := NewID("0000", ed25519.Random().PublicID())
	table := newTable(id)

	wg := &sync.WaitGroup{}
	wg.Add(concurrentCount)

	for i := 0; i < concurrentCount; i++ {
		go func() {
			defer func() {
				wg.Done()
			}()
			indices := MustReadRand(16)

			for _, indice := range indices {
				switch int(indice) % 4 {
				case 0:
					{
						addrRaw := MustReadRand(8)
						addr := hex.EncodeToString(addrRaw)

						id := NewID(addr, ed25519.Random().PublicID())
						table.Update(id)

						atomic.StorePointer(&ids[int(RandByte())%IDPoolSize], unsafe.Pointer(&id))
					}
				case 1:
					{
						id := (*ID)(atomic.LoadPointer(&ids[int(RandByte())%IDPoolSize]))
						if id != nil {
							table.Delete(id)
						}
					}
				case 2:
					{
						id := (*ID)(atomic.LoadPointer(&ids[int(RandByte())%IDPoolSize]))
						if id != nil {
							table.Get(id)
						}
					}
				case 3:
					{
						id := (*ID)(atomic.LoadPointer(&ids[int(RandByte())%IDPoolSize]))
						if id != nil {
							FindClosestPeers(table, id.Hash(), 5)
						}
					}
				}
			}
		}()
	}

	wg.Wait()
}

func TestTable(t *testing.T) {
	// make the node and table
	params := noise.DefaultParams()
	params.ID = ed25519.Random()
	params.Port = uint16(3000)
	id := NewID(fmt.Sprintf("127.0.0.1:%d", params.Port), params.ID.PublicID())

	node, err := noise.NewNode(params)
	assert.Nil(t, err)

	p := protocol.New()
	p.Register(New())
	p.Enforce(node)

	// test the table
	table := Table(node)
	assert.NotNil(t, table)
	assert.EqualValues(t, id, table.self)
}
