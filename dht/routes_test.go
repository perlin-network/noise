package dht

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

	"github.com/perlin-network/noise/crypto/blake2b"
	"github.com/perlin-network/noise/peer"
)

var (
	id1 peer.ID
	id2 peer.ID
	id3 peer.ID

	publicKey []byte
)

func init() {
	publicKey = MustReadRand(32)

	id1 = peer.CreateID("0000", publicKey)
	id2 = peer.CreateID("0001", MustReadRand(32))
	id3 = peer.CreateID("0002", MustReadRand(32))
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
	if !bytes.Equal(routingTable.Self().Id, blake2b.New().HashBytes(publicKey)) {
		t.Fatalf("wrong public key: %v", routingTable.Self().Id)
	}
}

func TestPeerExists(t *testing.T) {
	t.Parallel()

	routingTable := CreateRoutingTable(id1)
	routingTable.Update(id2)
	routingTable.Update(id3)

	if !routingTable.PeerExists(id1) {
		t.Fatal("peerexists() targeting self failed")
	}
	if !routingTable.PeerExists(id2) {
		t.Fatal("peerexists() targeting others failed")
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
		t.Fatalf("getpeeraddress() failed got: %v, expected : %v", routingTable.GetPeerAddresses(), testee)
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
	if !peer1.Equals(id2) {
		t.Errorf("'%v'.Equals(%v) = false, expected true", peer1, id2)
	}
}

func TestRemovePeer(t *testing.T) {
	t.Parallel()

	routingTable := CreateRoutingTable(id1)
	routingTable.Update(id2)
	routingTable.Update(id3)

	routingTable.RemovePeer(id2)
	testee := routingTable.GetPeerAddresses()
	sort.Strings(testee)
	tester := []string{"0002"}

	if !reflect.DeepEqual(tester, testee) {
		t.Fatalf("testremovepeer() failed got: %v, expected : %v", routingTable.GetPeerAddresses(), testee)
	}

}

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

func TestRoutingTable(t *testing.T) {
	t.Parallel()

	const IDPoolSize = 16
	const concurrentCount = 16

	pk0 := MustReadRand(32)
	ids := make([]unsafe.Pointer, IDPoolSize) // Element type: *peer.ID

	table := CreateRoutingTable(peer.CreateID("000", pk0))

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
						pk := MustReadRand(32)

						id := peer.CreateID(addr, pk)
						table.Update(id)

						atomic.StorePointer(&ids[int(RandByte())%IDPoolSize], unsafe.Pointer(&id))
					}
				case 1:
					{
						id := (*peer.ID)(atomic.LoadPointer(&ids[int(RandByte())%IDPoolSize]))
						if id != nil {
							table.RemovePeer(*id)
						}
					}
				case 2:
					{
						id := (*peer.ID)(atomic.LoadPointer(&ids[int(RandByte())%IDPoolSize]))
						if id != nil {
							table.PeerExists(*id)
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
