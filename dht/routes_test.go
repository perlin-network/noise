package dht

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/perlin-network/noise/peer"
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"
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
