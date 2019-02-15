// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !amd64

// This file contains field element logic that is representation-dependent.

package edwards25519

// This code is a port of the public domain, “ref10” implementation of ed25519
// from SUPERCOP.

// FieldElement represents an element of the field GF(2^255 - 19).  An element
// t, entries t[0]...t[9], represents the integer t[0]+2^26 t[1]+2^51 t[2]+2^77
// t[3]+2^102 t[4]+...+2^230 t[9].  Bounds on each t[i] vary depending on
// context.
type FieldElement [10]int32

func FeAdd(dst, a, b *FieldElement) {
	dst[0] = a[0] + b[0]
	dst[1] = a[1] + b[1]
	dst[2] = a[2] + b[2]
	dst[3] = a[3] + b[3]
	dst[4] = a[4] + b[4]
	dst[5] = a[5] + b[5]
	dst[6] = a[6] + b[6]
	dst[7] = a[7] + b[7]
	dst[8] = a[8] + b[8]
	dst[9] = a[9] + b[9]
}

func FeSub(dst, a, b *FieldElement) {
	dst[0] = a[0] - b[0]
	dst[1] = a[1] - b[1]
	dst[2] = a[2] - b[2]
	dst[3] = a[3] - b[3]
	dst[4] = a[4] - b[4]
	dst[5] = a[5] - b[5]
	dst[6] = a[6] - b[6]
	dst[7] = a[7] - b[7]
	dst[8] = a[8] - b[8]
	dst[9] = a[9] - b[9]
}

// Replace (f,g) with (g,g) if b == 1;
// replace (f,g) with (f,g) if b == 0.
//
// Preconditions: b in {0,1}.
func FeCMove(f, g *FieldElement, b int32) {
	b = -b
	f[0] ^= b & (f[0] ^ g[0])
	f[1] ^= b & (f[1] ^ g[1])
	f[2] ^= b & (f[2] ^ g[2])
	f[3] ^= b & (f[3] ^ g[3])
	f[4] ^= b & (f[4] ^ g[4])
	f[5] ^= b & (f[5] ^ g[5])
	f[6] ^= b & (f[6] ^ g[6])
	f[7] ^= b & (f[7] ^ g[7])
	f[8] ^= b & (f[8] ^ g[8])
	f[9] ^= b & (f[9] ^ g[9])
}

func FeFromBytes(dst *FieldElement, src *[32]byte) {
	h0 := load4(src[:])
	h1 := load3(src[4:]) << 6
	h2 := load3(src[7:]) << 5
	h3 := load3(src[10:]) << 3
	h4 := load3(src[13:]) << 2
	h5 := load4(src[16:])
	h6 := load3(src[20:]) << 7
	h7 := load3(src[23:]) << 5
	h8 := load3(src[26:]) << 4
	h9 := (load3(src[29:]) & 8388607) << 2

	FeCombine(dst, h0, h1, h2, h3, h4, h5, h6, h7, h8, h9)
}

// FeToBytes marshals h to s.
// Preconditions:
//   |h| bounded by 1.1*2^25,1.1*2^24,1.1*2^25,1.1*2^24,etc.
//
// Write p=2^255-19; q=floor(h/p).
// Basic claim: q = floor(2^(-255)(h + 19 2^(-25)h9 + 2^(-1))).
//
// Proof:
//   Have |h|<=p so |q|<=1 so |19^2 2^(-255) q|<1/4.
//   Also have |h-2^230 h9|<2^230 so |19 2^(-255)(h-2^230 h9)|<1/4.
//
//   Write y=2^(-1)-19^2 2^(-255)q-19 2^(-255)(h-2^230 h9).
//   Then 0<y<1.
//
//   Write r=h-pq.
//   Have 0<=r<=p-1=2^255-20.
//   Thus 0<=r+19(2^-255)r<r+19(2^-255)2^255<=2^255-1.
//
//   Write x=r+19(2^-255)r+y.
//   Then 0<x<2^255 so floor(2^(-255)x) = 0 so floor(q+2^(-255)x) = q.
//
//   Have q+2^(-255)x = 2^(-255)(h + 19 2^(-25) h9 + 2^(-1))
//   so floor(2^(-255)(h + 19 2^(-25) h9 + 2^(-1))) = q.
func FeToBytes(s *[32]byte, h *FieldElement) {
	var carry [10]int32

	q := (19*h[9] + (1 << 24)) >> 25
	q = (h[0] + q) >> 26
	q = (h[1] + q) >> 25
	q = (h[2] + q) >> 26
	q = (h[3] + q) >> 25
	q = (h[4] + q) >> 26
	q = (h[5] + q) >> 25
	q = (h[6] + q) >> 26
	q = (h[7] + q) >> 25
	q = (h[8] + q) >> 26
	q = (h[9] + q) >> 25

	// Goal: Output h-(2^255-19)q, which is between 0 and 2^255-20.
	h[0] += 19 * q
	// Goal: Output h-2^255 q, which is between 0 and 2^255-20.

	carry[0] = h[0] >> 26
	h[1] += carry[0]
	h[0] -= carry[0] << 26
	carry[1] = h[1] >> 25
	h[2] += carry[1]
	h[1] -= carry[1] << 25
	carry[2] = h[2] >> 26
	h[3] += carry[2]
	h[2] -= carry[2] << 26
	carry[3] = h[3] >> 25
	h[4] += carry[3]
	h[3] -= carry[3] << 25
	carry[4] = h[4] >> 26
	h[5] += carry[4]
	h[4] -= carry[4] << 26
	carry[5] = h[5] >> 25
	h[6] += carry[5]
	h[5] -= carry[5] << 25
	carry[6] = h[6] >> 26
	h[7] += carry[6]
	h[6] -= carry[6] << 26
	carry[7] = h[7] >> 25
	h[8] += carry[7]
	h[7] -= carry[7] << 25
	carry[8] = h[8] >> 26
	h[9] += carry[8]
	h[8] -= carry[8] << 26
	carry[9] = h[9] >> 25
	h[9] -= carry[9] << 25
	// h10 = carry9

	// Goal: Output h[0]+...+2^255 h10-2^255 q, which is between 0 and 2^255-20.
	// Have h[0]+...+2^230 h[9] between 0 and 2^255-1;
	// evidently 2^255 h10-2^255 q = 0.
	// Goal: Output h[0]+...+2^230 h[9].

	s[0] = byte(h[0] >> 0)
	s[1] = byte(h[0] >> 8)
	s[2] = byte(h[0] >> 16)
	s[3] = byte((h[0] >> 24) | (h[1] << 2))
	s[4] = byte(h[1] >> 6)
	s[5] = byte(h[1] >> 14)
	s[6] = byte((h[1] >> 22) | (h[2] << 3))
	s[7] = byte(h[2] >> 5)
	s[8] = byte(h[2] >> 13)
	s[9] = byte((h[2] >> 21) | (h[3] << 5))
	s[10] = byte(h[3] >> 3)
	s[11] = byte(h[3] >> 11)
	s[12] = byte((h[3] >> 19) | (h[4] << 6))
	s[13] = byte(h[4] >> 2)
	s[14] = byte(h[4] >> 10)
	s[15] = byte(h[4] >> 18)
	s[16] = byte(h[5] >> 0)
	s[17] = byte(h[5] >> 8)
	s[18] = byte(h[5] >> 16)
	s[19] = byte((h[5] >> 24) | (h[6] << 1))
	s[20] = byte(h[6] >> 7)
	s[21] = byte(h[6] >> 15)
	s[22] = byte((h[6] >> 23) | (h[7] << 3))
	s[23] = byte(h[7] >> 5)
	s[24] = byte(h[7] >> 13)
	s[25] = byte((h[7] >> 21) | (h[8] << 4))
	s[26] = byte(h[8] >> 4)
	s[27] = byte(h[8] >> 12)
	s[28] = byte((h[8] >> 20) | (h[9] << 6))
	s[29] = byte(h[9] >> 2)
	s[30] = byte(h[9] >> 10)
	s[31] = byte(h[9] >> 18)
}

// FeNeg sets h = -f
//
// Preconditions:
//    |f| bounded by 1.1*2^25,1.1*2^24,1.1*2^25,1.1*2^24,etc.
//
// Postconditions:
//    |h| bounded by 1.1*2^25,1.1*2^24,1.1*2^25,1.1*2^24,etc.
func FeNeg(h, f *FieldElement) {
	h[0] = -f[0]
	h[1] = -f[1]
	h[2] = -f[2]
	h[3] = -f[3]
	h[4] = -f[4]
	h[5] = -f[5]
	h[6] = -f[6]
	h[7] = -f[7]
	h[8] = -f[8]
	h[9] = -f[9]
}

func FeCombine(h *FieldElement, h0, h1, h2, h3, h4, h5, h6, h7, h8, h9 int64) {
	var c0, c1, c2, c3, c4, c5, c6, c7, c8, c9 int64

	/*
	  |h0| <= (1.1*1.1*2^52*(1+19+19+19+19)+1.1*1.1*2^50*(38+38+38+38+38))
	    i.e. |h0| <= 1.2*2^59; narrower ranges for h2, h4, h6, h8
	  |h1| <= (1.1*1.1*2^51*(1+1+19+19+19+19+19+19+19+19))
	    i.e. |h1| <= 1.5*2^58; narrower ranges for h3, h5, h7, h9
	*/

	c0 = (h0 + (1 << 25)) >> 26
	h1 += c0
	h0 -= c0 << 26
	c4 = (h4 + (1 << 25)) >> 26
	h5 += c4
	h4 -= c4 << 26
	/* |h0| <= 2^25 */
	/* |h4| <= 2^25 */
	/* |h1| <= 1.51*2^58 */
	/* |h5| <= 1.51*2^58 */

	c1 = (h1 + (1 << 24)) >> 25
	h2 += c1
	h1 -= c1 << 25
	c5 = (h5 + (1 << 24)) >> 25
	h6 += c5
	h5 -= c5 << 25
	/* |h1| <= 2^24; from now on fits into int32 */
	/* |h5| <= 2^24; from now on fits into int32 */
	/* |h2| <= 1.21*2^59 */
	/* |h6| <= 1.21*2^59 */

	c2 = (h2 + (1 << 25)) >> 26
	h3 += c2
	h2 -= c2 << 26
	c6 = (h6 + (1 << 25)) >> 26
	h7 += c6
	h6 -= c6 << 26
	/* |h2| <= 2^25; from now on fits into int32 unchanged */
	/* |h6| <= 2^25; from now on fits into int32 unchanged */
	/* |h3| <= 1.51*2^58 */
	/* |h7| <= 1.51*2^58 */

	c3 = (h3 + (1 << 24)) >> 25
	h4 += c3
	h3 -= c3 << 25
	c7 = (h7 + (1 << 24)) >> 25
	h8 += c7
	h7 -= c7 << 25
	/* |h3| <= 2^24; from now on fits into int32 unchanged */
	/* |h7| <= 2^24; from now on fits into int32 unchanged */
	/* |h4| <= 1.52*2^33 */
	/* |h8| <= 1.52*2^33 */

	c4 = (h4 + (1 << 25)) >> 26
	h5 += c4
	h4 -= c4 << 26
	c8 = (h8 + (1 << 25)) >> 26
	h9 += c8
	h8 -= c8 << 26
	/* |h4| <= 2^25; from now on fits into int32 unchanged */
	/* |h8| <= 2^25; from now on fits into int32 unchanged */
	/* |h5| <= 1.01*2^24 */
	/* |h9| <= 1.51*2^58 */

	c9 = (h9 + (1 << 24)) >> 25
	h0 += c9 * 19
	h9 -= c9 << 25
	/* |h9| <= 2^24; from now on fits into int32 unchanged */
	/* |h0| <= 1.8*2^37 */

	c0 = (h0 + (1 << 25)) >> 26
	h1 += c0
	h0 -= c0 << 26
	/* |h0| <= 2^25; from now on fits into int32 unchanged */
	/* |h1| <= 1.01*2^24 */

	h[0] = int32(h0)
	h[1] = int32(h1)
	h[2] = int32(h2)
	h[3] = int32(h3)
	h[4] = int32(h4)
	h[5] = int32(h5)
	h[6] = int32(h6)
	h[7] = int32(h7)
	h[8] = int32(h8)
	h[9] = int32(h9)
}

// FeMul calculates h = f * g
// Can overlap h with f or g.
//
// Preconditions:
//    |f| bounded by 1.1*2^26,1.1*2^25,1.1*2^26,1.1*2^25,etc.
//    |g| bounded by 1.1*2^26,1.1*2^25,1.1*2^26,1.1*2^25,etc.
//
// Postconditions:
//    |h| bounded by 1.1*2^25,1.1*2^24,1.1*2^25,1.1*2^24,etc.
//
// Notes on implementation strategy:
//
// Using schoolbook multiplication.
// Karatsuba would save a little in some cost models.
//
// Most multiplications by 2 and 19 are 32-bit precomputations;
// cheaper than 64-bit postcomputations.
//
// There is one remaining multiplication by 19 in the carry chain;
// one *19 precomputation can be merged into this,
// but the resulting data flow is considerably less clean.
//
// There are 12 carries below.
// 10 of them are 2-way parallelizable and vectorizable.
// Can get away with 11 carries, but then data flow is much deeper.
//
// With tighter constraints on inputs, can squeeze carries into int32.
func FeMul(h, f, g *FieldElement) {
	f0 := int64(f[0])
	f1 := int64(f[1])
	f2 := int64(f[2])
	f3 := int64(f[3])
	f4 := int64(f[4])
	f5 := int64(f[5])
	f6 := int64(f[6])
	f7 := int64(f[7])
	f8 := int64(f[8])
	f9 := int64(f[9])

	f1_2 := int64(2 * f[1])
	f3_2 := int64(2 * f[3])
	f5_2 := int64(2 * f[5])
	f7_2 := int64(2 * f[7])
	f9_2 := int64(2 * f[9])

	g0 := int64(g[0])
	g1 := int64(g[1])
	g2 := int64(g[2])
	g3 := int64(g[3])
	g4 := int64(g[4])
	g5 := int64(g[5])
	g6 := int64(g[6])
	g7 := int64(g[7])
	g8 := int64(g[8])
	g9 := int64(g[9])

	g1_19 := int64(19 * g[1]) /* 1.4*2^29 */
	g2_19 := int64(19 * g[2]) /* 1.4*2^30; still ok */
	g3_19 := int64(19 * g[3])
	g4_19 := int64(19 * g[4])
	g5_19 := int64(19 * g[5])
	g6_19 := int64(19 * g[6])
	g7_19 := int64(19 * g[7])
	g8_19 := int64(19 * g[8])
	g9_19 := int64(19 * g[9])

	h0 := f0*g0 + f1_2*g9_19 + f2*g8_19 + f3_2*g7_19 + f4*g6_19 + f5_2*g5_19 + f6*g4_19 + f7_2*g3_19 + f8*g2_19 + f9_2*g1_19
	h1 := f0*g1 + f1*g0 + f2*g9_19 + f3*g8_19 + f4*g7_19 + f5*g6_19 + f6*g5_19 + f7*g4_19 + f8*g3_19 + f9*g2_19
	h2 := f0*g2 + f1_2*g1 + f2*g0 + f3_2*g9_19 + f4*g8_19 + f5_2*g7_19 + f6*g6_19 + f7_2*g5_19 + f8*g4_19 + f9_2*g3_19
	h3 := f0*g3 + f1*g2 + f2*g1 + f3*g0 + f4*g9_19 + f5*g8_19 + f6*g7_19 + f7*g6_19 + f8*g5_19 + f9*g4_19
	h4 := f0*g4 + f1_2*g3 + f2*g2 + f3_2*g1 + f4*g0 + f5_2*g9_19 + f6*g8_19 + f7_2*g7_19 + f8*g6_19 + f9_2*g5_19
	h5 := f0*g5 + f1*g4 + f2*g3 + f3*g2 + f4*g1 + f5*g0 + f6*g9_19 + f7*g8_19 + f8*g7_19 + f9*g6_19
	h6 := f0*g6 + f1_2*g5 + f2*g4 + f3_2*g3 + f4*g2 + f5_2*g1 + f6*g0 + f7_2*g9_19 + f8*g8_19 + f9_2*g7_19
	h7 := f0*g7 + f1*g6 + f2*g5 + f3*g4 + f4*g3 + f5*g2 + f6*g1 + f7*g0 + f8*g9_19 + f9*g8_19
	h8 := f0*g8 + f1_2*g7 + f2*g6 + f3_2*g5 + f4*g4 + f5_2*g3 + f6*g2 + f7_2*g1 + f8*g0 + f9_2*g9_19
	h9 := f0*g9 + f1*g8 + f2*g7 + f3*g6 + f4*g5 + f5*g4 + f6*g3 + f7*g2 + f8*g1 + f9*g0

	FeCombine(h, h0, h1, h2, h3, h4, h5, h6, h7, h8, h9)
}

func feSquare(f *FieldElement) (h0, h1, h2, h3, h4, h5, h6, h7, h8, h9 int64) {
	f0 := int64(f[0])
	f1 := int64(f[1])
	f2 := int64(f[2])
	f3 := int64(f[3])
	f4 := int64(f[4])
	f5 := int64(f[5])
	f6 := int64(f[6])
	f7 := int64(f[7])
	f8 := int64(f[8])
	f9 := int64(f[9])
	f0_2 := int64(2 * f[0])
	f1_2 := int64(2 * f[1])
	f2_2 := int64(2 * f[2])
	f3_2 := int64(2 * f[3])
	f4_2 := int64(2 * f[4])
	f5_2 := int64(2 * f[5])
	f6_2 := int64(2 * f[6])
	f7_2 := int64(2 * f[7])
	f5_38 := 38 * f5 // 1.31*2^30
	f6_19 := 19 * f6 // 1.31*2^30
	f7_38 := 38 * f7 // 1.31*2^30
	f8_19 := 19 * f8 // 1.31*2^30
	f9_38 := 38 * f9 // 1.31*2^30

	h0 = f0*f0 + f1_2*f9_38 + f2_2*f8_19 + f3_2*f7_38 + f4_2*f6_19 + f5*f5_38
	h1 = f0_2*f1 + f2*f9_38 + f3_2*f8_19 + f4*f7_38 + f5_2*f6_19
	h2 = f0_2*f2 + f1_2*f1 + f3_2*f9_38 + f4_2*f8_19 + f5_2*f7_38 + f6*f6_19
	h3 = f0_2*f3 + f1_2*f2 + f4*f9_38 + f5_2*f8_19 + f6*f7_38
	h4 = f0_2*f4 + f1_2*f3_2 + f2*f2 + f5_2*f9_38 + f6_2*f8_19 + f7*f7_38
	h5 = f0_2*f5 + f1_2*f4 + f2_2*f3 + f6*f9_38 + f7_2*f8_19
	h6 = f0_2*f6 + f1_2*f5_2 + f2_2*f4 + f3_2*f3 + f7_2*f9_38 + f8*f8_19
	h7 = f0_2*f7 + f1_2*f6 + f2_2*f5 + f3_2*f4 + f8*f9_38
	h8 = f0_2*f8 + f1_2*f7_2 + f2_2*f6 + f3_2*f5_2 + f4*f4 + f9*f9_38
	h9 = f0_2*f9 + f1_2*f8 + f2_2*f7 + f3_2*f6 + f4_2*f5

	return
}

// FeSquare calculates h = f*f. Can overlap h with f.
//
// Preconditions:
//    |f| bounded by 1.1*2^26,1.1*2^25,1.1*2^26,1.1*2^25,etc.
//
// Postconditions:
//    |h| bounded by 1.1*2^25,1.1*2^24,1.1*2^25,1.1*2^24,etc.
func FeSquare(h, f *FieldElement) {
	h0, h1, h2, h3, h4, h5, h6, h7, h8, h9 := feSquare(f)
	FeCombine(h, h0, h1, h2, h3, h4, h5, h6, h7, h8, h9)
}

// FeSquare2 sets h = 2 * f * f
//
// Can overlap h with f.
//
// Preconditions:
//    |f| bounded by 1.65*2^26,1.65*2^25,1.65*2^26,1.65*2^25,etc.
//
// Postconditions:
//    |h| bounded by 1.01*2^25,1.01*2^24,1.01*2^25,1.01*2^24,etc.
// See fe_mul.c for discussion of implementation strategy.
func FeSquare2(h, f *FieldElement) {
	h0, h1, h2, h3, h4, h5, h6, h7, h8, h9 := feSquare(f)

	h0 += h0
	h1 += h1
	h2 += h2
	h3 += h3
	h4 += h4
	h5 += h5
	h6 += h6
	h7 += h7
	h8 += h8
	h9 += h9

	FeCombine(h, h0, h1, h2, h3, h4, h5, h6, h7, h8, h9)
}
