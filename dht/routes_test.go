package dht

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"

	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/peer"
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

func TestBucketSize(t *testing.T) {
	if dht.BucketSize != 20 {
		t.Fatalf("bucket size is expected %d but found %d", 20, dht.BucketSize)
	}
}

func TestSelf(t *testing.T) {
	publicKey := MustReadRand(32)
	id := peer.CreateID("0000", publicKey)
	routes := dht.CreateRoutingTable(id)
	if routes.Self().Address != "0000" {
		t.Fatalf("wrong address: %s", routes.Self().Address)
	}
	if !bytes.Equal(routes.Self().PublicKey, publicKey) {
		t.Fatalf("wrong public key: %v", routes.Self().PublicKey)
	}
}

func TestPeerExists(t *testing.T) {

	id1 := peer.CreateID("0000", MustReadRand(32))
	id2 := peer.CreateID("0001", MustReadRand(32))
	routingTable := dht.CreateRoutingTable(id1)
	routingTable.Update(id2)
	if !routingTable.PeerExists(id1) {
		t.Fatal("peerexists() targeting self failed")
	}
	fmt.Printf("%v", routingTable.GetPeers())
	if !routingTable.PeerExists(id2) {
		t.Fatal("peerexists() targeting others failed")
	}
}
func TestGetPeerAddresses(t *testing.T) {

	id1 := peer.CreateID("0000", MustReadRand(32))
	id2 := peer.CreateID("0001", MustReadRand(32))
	id3 := peer.CreateID("0002", MustReadRand(32))
	routingTable := dht.CreateRoutingTable(id1)
	routingTable.Update(id2)
	routingTable.Update(id3)
	tester := routingTable.GetPeerAddresses()
	sort.Strings(tester)
	testee := []string{"0001", "0002"}

	if !reflect.DeepEqual(tester, testee) {
		t.Fatalf("getpeeraddress() failed got: %v, expected : %v", routingTable.GetPeerAddresses(), testee)
	}

}
func TestRemovePeer(t *testing.T) {

	id1 := peer.CreateID("0000", MustReadRand(32))
	id2 := peer.CreateID("0001", MustReadRand(32))
	id3 := peer.CreateID("0002", MustReadRand(32))
	routingTable := dht.CreateRoutingTable(id1)
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
	nodes := []peer.ID{}

	nodes = append(nodes, peer.CreateID("0000", []byte("12345678901234567890123456789010")))
	nodes = append(nodes, peer.CreateID("0001", []byte("12345678901234567890123456789011")))
	nodes = append(nodes, peer.CreateID("0002", []byte("12345678901234567890123456789012")))
	nodes = append(nodes, peer.CreateID("0003", []byte("12345678901234567890123456789013")))
	nodes = append(nodes, peer.CreateID("0004", []byte("12345678901234567890123456789014")))
	nodes = append(nodes, peer.CreateID("0005", []byte("00000000000000000000000000000000")))
	routingTable := dht.CreateRoutingTable(nodes[0])
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
		if testee[i].Address != _answer.Address || !bytes.Equal(testee[i].PublicKey, _answer.PublicKey) {
			t.Fatalf("first findclosestpeers(), %d th closest peer is wrong, expected %v, found %v", i, _answer, testee[i])
		}
	}

	testee = []peer.ID{}
	for _, peer := range routingTable.FindClosestPeers(nodes[4], 2) {
		testee = append(testee, peer)
	}
	if len(testee) != 2 {
		t.Fatalf("findclosestpeers() error, size of return should be 3, but found %d", len(testee))
	}
	answerKeys = []int{4, 2}
	for i := 0; i <= 1; i++ {
		_answer := nodes[answerKeys[i]]
		if testee[i].Address != _answer.Address || !bytes.Equal(testee[i].PublicKey, _answer.PublicKey) {
			t.Fatalf("first findclosestpeers(), %d th closest peer is wrong, expected %v, found %v", i, _answer, testee[i])
		}
	}

}

func TestRoutingTable(t *testing.T) {
	const ID_POOL_SIZE = 16
	const CONCURRENT_COUNT = 16

	pk0 := MustReadRand(32)
	ids := make([]unsafe.Pointer, ID_POOL_SIZE) // Element type: *peer.ID

	table := CreateRoutingTable(peer.CreateID("000", pk0))

	wg := &sync.WaitGroup{}
	wg.Add(CONCURRENT_COUNT)

	for i := 0; i < CONCURRENT_COUNT; i++ {
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

						atomic.StorePointer(&ids[int(RandByte())%ID_POOL_SIZE], unsafe.Pointer(&id))
					}
				case 1:
					{
						id := (*peer.ID)(atomic.LoadPointer(&ids[int(RandByte())%ID_POOL_SIZE]))
						if id != nil {
							table.RemovePeer(*id)
						}
					}
				case 2:
					{
						id := (*peer.ID)(atomic.LoadPointer(&ids[int(RandByte())%ID_POOL_SIZE]))
						if id != nil {
							table.PeerExists(*id)
						}
					}
				case 3:
					{
						id := (*peer.ID)(atomic.LoadPointer(&ids[int(RandByte())%ID_POOL_SIZE]))
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
