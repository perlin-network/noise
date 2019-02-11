package payload

import (
	"bytes"
	"encoding/binary"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	testBytes = []byte{'a', 'b'}
	testByte byte = 'c'
	testString = "str"
	testUint16 uint16 = 10
	testUint32 uint32 = 11
	testUint64 uint64 = 12
)

func generatePayload(t *testing.T) *bytes.Buffer {
	var payload = bytes.NewBuffer(nil)
	var err error

	err = binary.Write(payload, binary.LittleEndian, uint32(len(testBytes)))
	assert.Nil(t, err, "error write length into buffer")
	_, err = payload.Write(testBytes)
	assert.Nil(t, err, "error write into buffer")

	err = binary.Write(payload, binary.LittleEndian, uint32(len(testString)))
	assert.Nil(t, err, "error write length into buffer")
	payload.WriteString(testString)

	err = payload.WriteByte(testByte)
	assert.Nil(t, err, "error write into buffer")

	err = binary.Write(payload, binary.LittleEndian, testUint16)
	assert.Nil(t, err, "error write into buffer")

	err = binary.Write(payload, binary.LittleEndian, testUint32)
	assert.Nil(t, err, "error write into buffer")

	err = binary.Write(payload, binary.LittleEndian, testUint64)
	assert.Nil(t, err, "error write into buffer")

	return payload
}

func TestReader(t *testing.T) {
	payload := generatePayload(t)

	reader := NewReader(payload.Bytes())

	actualBytes, err := reader.ReadBytes()
	assert.Nil(t, err, "error read bytes")
	assert.Equal(t, testBytes, actualBytes, "invalid bytes")

	actualString, err := reader.ReadString()
	assert.Nil(t, err, "error read string")
	assert.Equal(t, testString, actualString, "invalid bytes")

	actualByte, err := reader.ReadByte()
	assert.Nil(t, err, "error read byte")
	assert.Equal(t, testByte, actualByte, "invalid bytes")

	actualUint16, err := reader.ReadUint16()
	assert.Nil(t, err, "error read uint16")
	assert.Equal(t, testUint16, actualUint16, "invalid bytes")

	actualUint32, err := reader.ReadUint32()
	assert.Nil(t, err, "error read uint32")
	assert.Equal(t, testUint32, actualUint32, "invalid bytes")

	actualUint64, err := reader.ReadUint64()
	assert.Nil(t, err, "error read uint64")
	assert.Equal(t, testUint64, actualUint64, "invalid bytes")
}