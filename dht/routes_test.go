package dht

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"

	"github.com/perlin-network/noise/crypto"
	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/peer"
)

var (
	keys               = crypto.RandomKeyPair()
	host               = "127.0.0.1"
	port               = 12345
	expectedBucketSize = 20
)

func TestBucketSize(t *testing.T) {
	if dht.BucketSize != expectedBucketSize {
		t.Fatalf("bucket size is expected %d but found %d", expectedBucketSize, dht.BucketSize)
	}
}

func TestCreateRoutingTable(t *testing.T) {

	id := peer.CreateID(host+":"+strconv.Itoa(port), keys.PublicKey)
	routes := dht.CreateRoutingTable(id)
	if routes.Self().Address != fmt.Sprintf("%s:%d", host, port) {
		t.Fatalf("wrong address: %s", routes.Self().Address)
	}
	if !bytes.Equal(routes.Self().PublicKey, keys.PublicKey) {
		t.Fatalf("wrong public key: %v", routes.Self().PublicKey)
	}
}

//

func TestPeerExists(t *testing.T) {

	id1 := peer.CreateID(host+":"+strconv.Itoa(port), keys.PublicKey)
	routes := dht.CreateRoutingTable(id1)
	if !routes.PeerExists(id1) {
		t.Fatal("peerexists() failed")
	}
}
func TestGetPeers(t *testing.T) {

	id1 := peer.CreateID(host+":"+strconv.Itoa(port), keys.PublicKey)
	//id2
	routes := dht.CreateRoutingTable(id1)

	peer := routes.GetPeers()
	fmt.Printf("%v", peer)

	// id2 := peer.CreateID(host+":"+strconv.Itoa(port+1), keys.PublicKey)
	// id3 := peer.CreateID(host+":"+strconv.Itoa(port+2), keys.PublicKey)

	// routes.Update(id2)
	// routes.Update(id3)
	// bucketID := id2.Xor(id1).PrefixLen()
	// bucket := routes.Bucket(bucketID)
	// fmt.Printf("%v", bucket.List.Len())

	// fmt.Printf("%v", bucket.List.Front())

	// fmt.Printf("%v", bucket.List.Back())

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

// routes.Bucket(1)
// routes.FindClosestPeers()
// routes.GetPeerAddresses()
// routes.GetPeers()
// routes.PeerExists()
// routes.RemovePeer()
// routes.Self()
// routes.Update()
// net := &network.Network{
// 	Keys: keys,
// 	Host: host,
// 	Port: port,
// 	ID:   id,

// 	Processors: &network.StringMessageProcessorSyncMap{},

// 	Routes: dht.CreateRoutingTable(id),

// 	Peers: &network.StringPeerClientSyncMap{},

// 	Listening: make(chan struct{}),
// }
