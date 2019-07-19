package payload

import (
	"bytes"
	"encoding/binary"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWriter(t *testing.T) {
	writer := NewWriter(nil)

	var result = bytes.NewBuffer(nil)
	var length = 0
	var err error

	// write bytes
	testBytes := make([]byte, 2)
	writer.WriteBytes(testBytes)

	// check length
	length += 4 + len(testBytes)
	assert.Equal(t, length, writer.Len(), "found invalid length after WriteBytes")

	// check bytes
	err = binary.Write(result, binary.LittleEndian, uint32(len(testBytes)))
	assert.Nil(t, err, "error write length into buffer")
	_, err = result.Write(testBytes)
	assert.Nil(t, err, "error write into buffer")
	assert.Equal(t, result.Bytes(), writer.Bytes(), "found invalid bytes after WriteBytes")

	// write string
	testString := "str"
	writer.WriteString(testString)

	// check length
	length += 4 + len([]byte(testString))
	assert.Equal(t, length, writer.Len(), "found invalid length after WriteString")

	// check bytes
	err = binary.Write(result, binary.LittleEndian, uint32(len(testString)))
	assert.Nil(t, err, "error write length into buffer")
	result.WriteString(testString)
	assert.Equal(t, result.Bytes(), writer.Bytes(), "found invalid bytes after WriteString")

	// write byte
	var testByte byte
	writer.WriteByte(testByte)

	// check length
	length += 1
	assert.Equal(t, length, writer.Len(), "found invalid length after WriteByte")

	// check bytes
	err = result.WriteByte(testByte)
	assert.Nil(t, err, "error write into buffer")
	assert.Equal(t, result.Bytes(), writer.Bytes(), "found invalid bytes after WriteByte")

	// write uint16
	var testUint16 uint16 = 10
	writer.WriteUint16(testUint16)

	// check length
	length += 2
	assert.Equal(t, length, writer.Len(), "found invalid length after WriteUint16")

	// check bytes
	err = binary.Write(result, binary.LittleEndian, testUint16)
	assert.Nil(t, err, "error write into buffer")
	assert.Equal(t, result.Bytes(), writer.Bytes(), "found invalid bytes after WriteUint16")

	// write uint32
	var testUint32 uint32 = 10
	writer.WriteUint32(testUint32)

	// check length
	length += 4
	assert.Equal(t, length, writer.Len(), "found invalid length after WriteUint32")

	err = binary.Write(result, binary.LittleEndian, testUint32)
	assert.Nil(t, err, "error write into buffer")
	assert.Equal(t, result.Bytes(), writer.Bytes(), "found invalid bytes after WriteUint32")

	// write uint64
	var testUint64 uint64 = 10
	writer.WriteUint64(testUint64)

	// check length
	length += 8
	assert.Equal(t, length, writer.Len(), "found invalid length after WriteUint64")

	err = binary.Write(result, binary.LittleEndian, testUint64)
	assert.Nil(t, err, "error write into buffer")
	assert.Equal(t, result.Bytes(), writer.Bytes(), "found invalid bytes after WriteUint64")
}
