package kademlia

import (
	"bytes"
	"github.com/perlin-network/noise"
	"math/bits"
	"sort"
)

// XOR allocates a new byte slice with the computed result of XOR(a, b).
func XOR(a, b []byte) []byte {
	if len(a) != len(b) {
		return a
	}

	c := make([]byte, len(a))

	for i := 0; i < len(a); i++ {
		c[i] = a[i] ^ b[i]
	}

	return c
}

// PrefixDiff counts the number of equal prefixed bits of a and b.
func PrefixDiff(a, b []byte, n int) int {
	buf, total := XOR(a, b), 0

	for i, b := range buf {
		if 8*i >= n {
			break
		}

		if n > 8*i && n < 8*(i+1) {
			shift := 8 - uint(n%8)
			b >>= shift
		}

		total += bits.OnesCount8(b)
	}

	return total
}

// PrefixLen returns the number of prefixed zero bits of a.
func PrefixLen(a []byte) int {
	for i, b := range a {
		if b != 0 {
			return i*8 + bits.LeadingZeros8(b)
		}
	}

	return len(a) * 8
}

// SortByDistance sorts ids by descending XOR distance with respect to id.
func SortByDistance(id noise.PublicKey, ids []noise.ID) []noise.ID {
	sort.Slice(ids, func(i, j int) bool {
		return bytes.Compare(XOR(ids[i].ID[:], id[:]), XOR(ids[j].ID[:], id[:])) == -1
	})

	return ids
}
