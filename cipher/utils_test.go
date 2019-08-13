package cipher

import (
	"bytes"
	"encoding/binary"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

func TestSliceForAppend(t *testing.T) {
	// Test without capacity
	buf := make([]byte, 128)
	rand.Read(buf)
	testSliceForAppend(t, buf)

	// Test with capacity
	buf = make([]byte, 128, 256)
	rand.Read(buf)
	testSliceForAppend(t, buf)
}

func testSliceForAppend(t *testing.T, buf []byte) {
	head, tail := sliceForAppend(buf, 32)

	assert.Len(t, head, len(buf)+32)
	assert.Equal(t, buf, head[:len(buf)])
	assert.Len(t, tail, 32)
	assert.Equal(t, tail, make([]byte, 32))

	// Check if allocation is performed or not

	if cap(buf) > len(buf) {
		// The input has sufficient capacity so, allocation is not needed.
		assert.True(t, &buf[0] == &head[0], "The underlying array must same")
	} else {
		// The input does not has sufficient capacity so, allocation is performed.
		assert.True(t, &buf[0] != &head[0], "The underlying array must be different")
	}
}

func TestParseFrame(t *testing.T) {
	// Test msg length lower than MsgLenFieldSize (4)
	buf := make([]byte, 1)
	rand.Read(buf)
	current, next, err := parseFrame(buf, 8)
	assert.Nil(t, current)
	assert.Equal(t, buf, next)
	assert.NoError(t, err)

	// Test msg length higher than maxLength
	buf = getMsg(t, 128, 128)
	current, next, err = parseFrame(buf, 16)
	assert.Nil(t, current)
	assert.Nil(t, next)
	assert.Error(t, err)

	// Test incomplete frame
	buf = getMsg(t, 64, 128)
	current, next, err = parseFrame(buf, 128)
	assert.Nil(t, current)
	assert.Equal(t, buf, next)
	assert.NoError(t, err)
}

func getMsg(t *testing.T, size int, frameSize uint32) []byte {
	msg := make([]byte, size)
	rand.Read(msg)

	msgLength := make([]byte, 4)
	binary.BigEndian.PutUint32(msgLength, frameSize)

	var buf = bytes.NewBuffer(msgLength)
	assert.NoError(t, binary.Write(buf, binary.LittleEndian, msg))

	return buf.Bytes()
}