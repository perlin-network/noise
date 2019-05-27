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
