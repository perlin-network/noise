package dht_test

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"github.com/perlin-network/noise/skademlia"
	"github.com/perlin-network/noise/skademlia/dht"
	"github.com/perlin-network/noise/skademlia/peer"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"
)

var (
	id1 peer.ID
	id2 peer.ID
	id3 peer.ID
	id4 peer.ID

	idBytes []byte
)

func init() {
	id1 = dht.NewID(skademlia.NewIdentityAdapter(8, 8).MyIdentity(), "0000")
	id2 = dht.NewID(skademlia.NewIdentityAdapter(8, 8).MyIdentity(), "0001")
	id3 = dht.NewID(skademlia.NewIdentityAdapter(8, 8).MyIdentity(), "0002")
	id4 = dht.NewID(skademlia.NewIdentityAdapter(8, 8).MyIdentity(), "0003")

	idBytes = id1.Id
}

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

	routingTable := dht.NewRoutingTable(id1)
	routingTable.Update(id2)
	routingTable.Update(id3)

	if routingTable.Self().Address != "0000" {
		t.Fatalf("wrong address: %s", routingTable.Self().Address)
	}
	if !bytes.Equal(routingTable.Self().Id, idBytes) {
		t.Fatalf("wrong node id: %v", routingTable.Self().Id)
	}
}

func TestGetPeerAddresses(t *testing.T) {
	t.Parallel()

	routingTable := dht.NewRoutingTable(id1)
	routingTable.Update(id2)
	routingTable.Update(id3)

	tester := routingTable.GetPeerAddresses()
	sort.Strings(tester)
	testee := []string{"0001", "0002"}

	if !reflect.DeepEqual(tester, testee) {
		t.Fatalf("GetPeerAddresses() failed got: %v, expected : %v", routingTable.GetPeerAddresses(), testee)
	}
}

func TestGetPeer(t *testing.T) {
	t.Parallel()

	routingTable := dht.NewRoutingTable(id1)
	routingTable.Update(id2)

	found, ok := routingTable.GetPeer(id1.Id)
	if !ok && found == nil {
		t.Errorf("GetPeer() expected to find id1")
	}
	if !reflect.DeepEqual(id1, *found) {
		t.Fatalf("GetPeer() expected found peer %+v to be equal to id1 %+v", found, id1)
	}

	found, ok = routingTable.GetPeer(id3.Id)
	if ok && found != nil {
		t.Errorf("GetPeer() expected not to find id3")
	}
	routingTable.Update(id3)
	found, ok = routingTable.GetPeer(id3.Id)
	if !ok && found == nil {
		t.Errorf("GetPeer() expected to find id3")
	}
	if !reflect.DeepEqual(id3, *found) {
		t.Fatalf("GetPeer() expected found peer to be equal to id1")
	}

	routingTable.RemovePeer(id1.Id)
	found, ok = routingTable.GetPeer(id1.Id)
	if ok && found != nil {
		t.Errorf("GetPeer() expected not to find id1 after deletion")
	}
}

func TestGetPeers(t *testing.T) {
	t.Parallel()

	routingTable := dht.NewRoutingTable(id1)
	routingTable.Update(id2)

	peers := routingTable.GetPeers()
	if len(peers) != 1 {
		t.Errorf("len(peers) = %d, expected 1", len(peers))
	}
	peer1 := peers[0]
	if !bytes.Equal(peer1.Id, id2.Id) {
		t.Errorf("'%v'.Equals(%v) = false, expected true", peer1, id2)
	}
}

func TestRemovePeer(t *testing.T) {
	t.Parallel()

	routingTable := dht.NewRoutingTable(id1)
	routingTable.Update(id2)
	routingTable.Update(id3)

	routingTable.RemovePeer(id2.Id)
	testee := routingTable.GetPeerAddresses()
	sort.Strings(testee)
	tester := []string{"0002"}

	if !reflect.DeepEqual(tester, testee) {
		t.Fatalf("testremovepeer() failed got: %v, expected : %v", routingTable.GetPeerAddresses(), testee)
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	// self key generates bucket id 255
	idKey1 := []byte{124, 224, 147, 208, 211, 103, 166, 113, 153, 104, 83, 62, 61, 145, 8, 211, 144, 164, 224, 191, 177, 205, 198, 94, 92, 35, 76, 83, 229, 46, 219, 110}
	id1 := dht.NewID(idKey1, "0001")

	// key generates bucket id 8
	idKey2 := []byte{210, 127, 212, 137, 47, 66, 40, 189, 231, 239, 210, 168, 52, 15, 223, 66, 199, 199, 156, 61, 132, 56, 102, 223, 32, 175, 169, 241, 156, 46, 83, 98}
	id2 := dht.NewID(idKey2, "0002")

	// key generates bucket id 8
	idKey3 := []byte{228, 61, 230, 169, 243, 78, 244, 44, 82, 76, 54, 56, 98, 135, 227, 158, 114, 251, 56, 160, 208, 60, 121, 41, 197, 63, 235, 41, 236, 66, 222, 219}
	id3 := dht.NewID(idKey3, "0003")

	routingTable := dht.NewRoutingTableWithOptions(id1, dht.WithBucketSize(1))

	bucketID2 := routingTable.GetBucketID(idKey2)
	bucketID3 := routingTable.GetBucketID(idKey3)

	if bucketID2 != bucketID3 {
		t.Errorf("GetBucketID() expected to be equal")
	}

	err := routingTable.Update(id2)
	if err != nil {
		t.Errorf("Update() expected no error, got: %+v", err)
	}
	err = routingTable.Update(id3)
	if err != dht.ErrBucketFull {
		t.Errorf("Update() expected error ErrBucketFull, got: %+v", err)
	}

	routingTable.Opts().BucketSize = 2
	err = routingTable.Update(id3)
	if err != nil {
		t.Errorf("Update() expected no error, got: %+v", err)
	}
}

func TestFindClosestPeers(t *testing.T) {
	t.Parallel()

	nodes := []*peer.ID{}

	nodes = append(nodes,
		&peer.ID{Address: "0000", Id: []byte("12345678901234567890123456789010")},
		&peer.ID{Address: "0001", Id: []byte("12345678901234567890123456789011")},
		&peer.ID{Address: "0002", Id: []byte("12345678901234567890123456789012")},
		&peer.ID{Address: "0003", Id: []byte("12345678901234567890123456789013")},
		&peer.ID{Address: "0004", Id: []byte("12345678901234567890123456789014")},
		&peer.ID{Address: "0005", Id: []byte("00000000000000000000000000000000")},
	)
	for _, node := range nodes {
		node.PublicKey = node.Id
	}

	routingTable := dht.NewRoutingTable(*nodes[0])
	for i := 1; i <= 5; i++ {
		routingTable.Update(*nodes[i])
	}
	testee := []peer.ID{}
	for _, peer := range routingTable.FindClosestPeers(*nodes[5], 3) {
		testee = append(testee, peer)
	}
	if len(testee) != 3 {
		t.Fatalf("findclosestpeers() error, size of return should be 3, but found %d", len(testee))
	}
	answerKeys := []int{5, 2, 3}
	for i := 0; i <= 2; i++ {
		_answer := nodes[answerKeys[i]]
		if testee[i].Address != _answer.Address || !bytes.Equal(testee[i].Id, _answer.Id) {
			t.Fatalf("first findclosestpeers(), %d th closest peer is wrong, expected %v, found %v", i, _answer, testee[i])
		}
	}

	testee = []peer.ID{}
	for _, peer := range routingTable.FindClosestPeers(*nodes[4], 2) {
		testee = append(testee, peer)
	}
	if len(testee) != 2 {
		t.Fatalf("findclosestpeers() error, size of return should be 2, but found %d", len(testee))
	}
	answerKeys = []int{4, 2}
	for i := 0; i <= 1; i++ {
		_answer := nodes[answerKeys[i]]
		if testee[i].Address != _answer.Address || !bytes.Equal(testee[i].Id, _answer.Id) {
			t.Fatalf("first findclosestpeers(), %d th closest peer is wrong, expected %v, found %v", i, _answer, testee[i])
		}
	}

}

func TestRoutingTable(t *testing.T) {
	t.Parallel()

	const IDPoolSize = 16
	const concurrentCount = 16

	ids := make([]unsafe.Pointer, IDPoolSize) // Element type: *peer.Id

	id := dht.NewID(skademlia.NewIdentityAdapter(8, 8).MyIdentity(), "0000")
	table := dht.NewRoutingTable(id)

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

						id := dht.NewID(skademlia.NewIdentityAdapter(8, 8).MyIdentity(), addr)
						table.Update(id)

						atomic.StorePointer(&ids[int(RandByte())%IDPoolSize], unsafe.Pointer(&id))
					}
				case 1:
					{
						id := (*peer.ID)(atomic.LoadPointer(&ids[int(RandByte())%IDPoolSize]))
						if id != nil {
							table.RemovePeer(id.Id)
						}
					}
				case 2:
					{
						id := (*peer.ID)(atomic.LoadPointer(&ids[int(RandByte())%IDPoolSize]))
						if id != nil {
							table.GetPeer(id.Id)
						}
					}
				case 3:
					{
						id := (*peer.ID)(atomic.LoadPointer(&ids[int(RandByte())%IDPoolSize]))
						if id != nil {
							table.FindClosestPeers(*id, 5)
						}
					}
				}
			}
		}()
	}

	wg.Wait()
}
