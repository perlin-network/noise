package skademlia

import (
	"math/bits"
)

// xor allocates a new byte slice with the computed result of xor(a, b).
func xor(a, b []byte) []byte {
	if len(a) != len(b) {
		return a
	}

	c := make([]byte, len(a))

	for i := 0; i < len(a); i++ {
		c[i] = a[i] ^ b[i]
	}

	return c
}

// prefixDiff counts the number of equal prefixed bits of a and b.
func prefixDiff(a, b []byte, n int) int {
	buf, total := xor(a, b), 0

	for i, b := range buf {
		if 8*i >= n {
			break
		}

		if n > 8*i && n < 8*(i+1) {
			shift := 8 - uint(n%8)
			b = b >> shift
		}

		total += bits.OnesCount8(uint8(b))
	}
	return total
}

// prefixLen returns the number of prefixed zero bits of a.
func prefixLen(a []byte) int {
	for i, b := range a {
		if b != 0 {
			return i*8 + bits.LeadingZeros8(uint8(b))
		}
	}

	return len(a)*8 - 1
}
