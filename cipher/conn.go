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

package cipher

import (
	"crypto/cipher"
	"encoding/binary"
	"math"
	"net"
)

const (
	// MsgLenFieldSize is the byte size of the frame length field of a
	// framed message.
	MsgLenFieldSize = 4
	// The bytes size limit for a ALTS record message.
	RecordLengthLimit = 1024 * 1024 // 1 MiB
	// The default bytes size of a ALTS record message.
	RecordDefaultLength = 4 * 1024 // 4KiB
	// The initial write buffer size.
	WriteBufferInitialSize = 32 * 1024 // 32KiB
	// The maximum write buffer size. This *must* be multiple of
	// altsRecordDefaultLength.
	WriteBufferMaxSize = 512 * 1024 // 512KiB
)

type connAEAD struct {
	localNonce, remoteNonce       uint64
	localNonceBuf, remoteNonceBuf []byte

	readBuf, writeBuf []byte

	protected []byte
	nextFrame []byte

	limit    int
	overhead int

	suite cipher.AEAD
	net.Conn
}

func newConnAEAD(suite cipher.AEAD, conn net.Conn) *connAEAD {
	overhead := MsgLenFieldSize + suite.Overhead()
	limit := RecordDefaultLength - overhead

	protected := make([]byte, 0, 2*RecordDefaultLength-1)

	return &connAEAD{
		Conn: conn,

		overhead: overhead,
		limit:    limit,

		protected: protected,
		nextFrame: protected,

		writeBuf: make([]byte, WriteBufferInitialSize),

		localNonceBuf:  make([]byte, suite.NonceSize()),
		remoteNonceBuf: make([]byte, suite.NonceSize()),

		suite: suite,
	}
}

// Read reads data from the connection.
// Read can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
func (c *connAEAD) Read(b []byte) (n int, err error) {
	if len(c.readBuf) == 0 {
		var currentFrame []byte

		currentFrame, c.nextFrame, err = parseFrame(c.nextFrame, RecordLengthLimit)
		if err != nil {
			return n, err
		}

		if len(currentFrame) == 0 {
			copy(c.protected, c.nextFrame)

			c.protected = c.protected[:len(c.nextFrame)]
			c.nextFrame = c.protected
		}

		for len(currentFrame) == 0 { // Keep reading from conn until we read an entire frame.
			if len(c.protected) == cap(c.protected) {
				tmp := make([]byte, len(c.protected), cap(c.protected)+RecordDefaultLength)
				copy(tmp, c.protected)
				c.protected = tmp
			}

			n, err = c.Conn.Read(c.protected[len(c.protected):min(cap(c.protected), len(c.protected)+RecordDefaultLength)])
			if err != nil {
				return 0, err
			}

			c.protected = c.protected[:len(c.protected)+n]

			currentFrame, c.nextFrame, err = parseFrame(c.protected, RecordLengthLimit)
			if err != nil {
				return 0, err
			}
		}

		ciphertext := currentFrame[MsgLenFieldSize:]

		if c.readBuf, err = c.Decrypt(ciphertext[:0], ciphertext); err != nil {
			return 0, err
		}
	}

	n = copy(b, c.readBuf)
	c.readBuf = c.readBuf[n:]

	return n, nil
}

// Write writes data to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (c *connAEAD) Write(b []byte) (n int, err error) {
	n = len(b)

	numFrames := int(math.Ceil(float64(len(b)) / float64(c.limit)))
	size := len(b) + numFrames*c.overhead

	frameLen := len(b)
	if size > WriteBufferMaxSize {
		size = WriteBufferMaxSize

		frameLen = WriteBufferMaxSize / RecordDefaultLength * c.limit
	}

	if len(c.writeBuf) < size {
		c.writeBuf = make([]byte, size)
	}

	for start := 0; start < n; start += frameLen {
		end := start + frameLen
		if end > len(b) {
			end = len(b)
		}

		i := 0

		frame := b[start:end]

		for len(frame) > 0 {
			payloadLen := len(frame)
			if payloadLen > c.limit {
				payloadLen = c.limit
			}

			buf := frame[:payloadLen]
			frame = frame[payloadLen:]

			msg := c.writeBuf[i+MsgLenFieldSize:]

			// 1. Encrypt the payload.
			msg = c.Encrypt(msg[:0], buf)

			// 2. Fill in the size field.
			binary.BigEndian.PutUint32(c.writeBuf[i:], MsgLenFieldSize+uint32(len(msg)))

			// 3. Increase i.
			i += len(buf) + c.overhead
		}

		nn, err := c.Conn.Write(c.writeBuf[:i])

		if err != nil {
			numOfWrittenFrames := int(math.Floor(float64(nn) / float64(RecordDefaultLength)))
			return start + numOfWrittenFrames*c.limit, err
		}
	}

	return n, nil
}

// Encrypt is the encryption function. dst can contain bytes at the beginning of
// the ciphertext that will not be encrypted but will be authenticated. If dst
// has enough capacity to hold these bytes, the ciphertext and the tag, no
// allocation and copy operations will be performed. dst and plaintext do not
// overlap.
func (c *connAEAD) Encrypt(dst, plaintext []byte) []byte {
	dlen := len(dst)

	dst, out := sliceForAppend(dst, len(plaintext)+c.suite.Overhead())
	data := out[:len(plaintext)]
	copy(data, plaintext)

	binary.LittleEndian.PutUint64(c.localNonceBuf, c.localNonce)
	c.localNonce++

	return c.suite.Seal(dst[:dlen], c.localNonceBuf, data, nil)
}

func (c *connAEAD) Decrypt(dst, ciphertext []byte) ([]byte, error) {
	binary.LittleEndian.PutUint64(c.remoteNonceBuf, c.remoteNonce)
	c.remoteNonce++

	return c.suite.Open(dst, c.remoteNonceBuf, ciphertext, nil)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
