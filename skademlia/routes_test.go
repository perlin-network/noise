package skademlia

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"
)

var (
	id1 ID
	id2 ID
	id3 ID

	idBytes []byte
)

func init() {
	id1 = ID{NewIdentityAdapter(8, 8).id(), "0000"}
	id2 = ID{NewIdentityAdapter(8, 8).id(), "0001"}
	id3 = ID{NewIdentityAdapter(8, 8).id(), "0002"}

	idBytes = id1.id
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

	routingTable := CreateRoutingTable(id1)
	routingTable.Update(id2)
	routingTable.Update(id3)

	if routingTable.Self().Address != "0000" {
		t.Fatalf("wrong address: %s", routingTable.Self().Address)
	}
	if !bytes.Equal(routingTable.Self().id, idBytes) {
		t.Fatalf("wrong node id: %v", routingTable.Self().id)
	}
}

func TestGetPeerAddresses(t *testing.T) {
	t.Parallel()

	routingTable := CreateRoutingTable(id1)
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

	routingTable := CreateRoutingTable(id1)
	routingTable.Update(id2)

	ok, found := routingTable.GetPeer(id1.id)
	if !ok && found == nil {
		t.Errorf("GetPeer() expected to find id1")
	}
	if !reflect.DeepEqual(id1, *found) {
		t.Fatalf("GetPeer() expected found peer %+v to be equal to id1 %+v", found, id1)
	}

	ok, found = routingTable.GetPeer(id3.id)
	if ok && found != nil {
		t.Errorf("GetPeer() expected not to find id3")
	}
	routingTable.Update(id3)
	ok, found = routingTable.GetPeer(id3.id)
	if !ok && found == nil {
		t.Errorf("GetPeer() expected to find id3")
	}
	if !reflect.DeepEqual(id3, *found) {
		t.Fatalf("GetPeer() expected found peer to be equal to id1")
	}

	routingTable.RemovePeer(id1.id)
	ok, found = routingTable.GetPeer(id1.id)
	if ok && found != nil {
		t.Errorf("GetPeer() expected not to find id1 after deletion")
	}
}

func TestGetPeers(t *testing.T) {
	t.Parallel()

	routingTable := CreateRoutingTable(id1)
	routingTable.Update(id2)

	peers := routingTable.GetPeers()
	if len(peers) != 1 {
		t.Errorf("len(peers) = %d, expected 1", len(peers))
	}
	peer1 := peers[0]
	if !bytes.Equal(peer1.id, id2.id) {
		t.Errorf("'%v'.Equals(%v) = false, expected true", peer1, id2)
	}
}

func TestRemovePeer(t *testing.T) {
	t.Parallel()

	routingTable := CreateRoutingTable(id1)
	routingTable.Update(id2)
	routingTable.Update(id3)

	routingTable.RemovePeer(id2.id)
	testee := routingTable.GetPeerAddresses()
	sort.Strings(testee)
	tester := []string{"0002"}

	if !reflect.DeepEqual(tester, testee) {
		t.Fatalf("testremovepeer() failed got: %v, expected : %v", routingTable.GetPeerAddresses(), testee)
	}

}

/*
func TestFindClosestPeers(t *testing.T) {
	t.Parallel()

	nodes := []peer.ID{}

	nodes = append(nodes,
		peer.ID{Address: "0000", Id: []byte("12345678901234567890123456789010")},
		peer.ID{Address: "0001", Id: []byte("12345678901234567890123456789011")},
		peer.ID{Address: "0002", Id: []byte("12345678901234567890123456789012")},
		peer.ID{Address: "0003", Id: []byte("12345678901234567890123456789013")},
		peer.ID{Address: "0004", Id: []byte("12345678901234567890123456789014")},
		peer.ID{Address: "0005", Id: []byte("00000000000000000000000000000000")},
	)
	routingTable := CreateRoutingTable(nodes[0])
	for i := 1; i <= 5; i++ {
		routingTable.Update(nodes[i])
	}
	testee := []peer.ID{}
	for _, peer := range routingTable.FindClosestPeers(nodes[5], 3) {
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
	for _, peer := range routingTable.FindClosestPeers(nodes[4], 2) {
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
*/

func TestRoutingTable(t *testing.T) {
	t.Parallel()

	const IDPoolSize = 16
	const concurrentCount = 16

	ids := make([]unsafe.Pointer, IDPoolSize) // Element type: *peer.ID

	id := ID{NewIdentityAdapter(8, 8).id(), "000"}
	table := CreateRoutingTable(id)

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

						id := ID{NewIdentityAdapter(8, 8).id(), addr}
						table.Update(id)

						atomic.StorePointer(&ids[int(RandByte())%IDPoolSize], unsafe.Pointer(&id))
					}
				case 1:
					{
						id := (*ID)(atomic.LoadPointer(&ids[int(RandByte())%IDPoolSize]))
						if id != nil {
							table.RemovePeer(id.id)
						}
					}
				case 2:
					{
						id := (*ID)(atomic.LoadPointer(&ids[int(RandByte())%IDPoolSize]))
						if id != nil {
							table.GetPeer(id.id)
						}
					}
				case 3:
					{
						id := (*ID)(atomic.LoadPointer(&ids[int(RandByte())%IDPoolSize]))
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
