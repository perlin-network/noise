package noise

import (
	"github.com/perlin-network/noise/payload"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

type testMsg struct {
	text string
}

func (testMsg) Read(reader payload.Reader) (Message, error) {
	text, err := reader.ReadString()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read chat msg")
	}

	return testMsg{text: text}, nil
}

func (m testMsg) Write() []byte {
	return payload.NewWriter(nil).WriteString(m.text).Bytes()
}

func TestBytes(t *testing.T) {
	t.Parallel()
	for i := 0; i < 1000; i++ {
		o := Opcode(i)
		assert.Equal(t, [1]byte{byte(i)}, o.Bytes())
	}
}

func TestNextAvailableOpcode(t *testing.T) {
	resetOpcodes()

	// opcode 0 should be an empty message
	msg, err := MessageFromOpcode(Opcode(0))
	assert.Nil(t, err)
	assert.Equal(t, EmptyMessage{}, msg)

	// an unset opcode should be an error
	_, err = MessageFromOpcode(Opcode(1))
	assert.NotNil(t, err)

	type badType struct {
		EmptyMessage
	}

	_, err = OpcodeFromMessage(badType{})
	assert.NotNil(t, err)

	// try adding all the possible values for opcode
	for i := 1; i <= math.MaxUint8; i++ {
		o := NextAvailableOpcode()
		assert.Equal(t, Opcode(i), o)
		assert.Equal(t, o, RegisterMessage(o, (*testMsg)(nil)))

		msg, err = MessageFromOpcode(Opcode(i))
		assert.Nil(t, err)
		assert.Equal(t, testMsg{}, msg)

		actual, err := OpcodeFromMessage(testMsg{})
		assert.Nil(t, err)
		assert.Equal(t, o, actual)
	}

	// an opcode should still exist after the loop
	msg, err = MessageFromOpcode(Opcode(1))
	assert.Nil(t, err)
	assert.Equal(t, testMsg{}, msg)

	DebugOpcodes()
}

func TestEncodeMessage(t *testing.T) {
	o := Opcode(123)
	msg := testMsg{
		text: "hello",
	}
	p := newPeer(nil, nil)
	bytes, err := p.EncodeMessage(o, msg)
	assert.Nil(t, err)
	assert.Equal(t, append([]byte{123}, msg.Write()...), bytes)
}

func TestDecodeMessage(t *testing.T) {
	resetOpcodes()
	o := Opcode(45)
	msg := testMsg{
		text: "world",
	}
	assert.Equal(t, o, RegisterMessage(o, (*testMsg)(nil)))

	p := newPeer(nil, nil)
	resultO, resultM, err := p.DecodeMessage(append([]byte{45}, msg.Write()...))
	assert.Nil(t, err)
	assert.Equal(t, o, resultO)
	assert.Equal(t, msg, resultM)
}

func TestEmptyMessage(t *testing.T) {
	t.Parallel()

	em := EmptyMessage{}

	m, err := em.Read(payload.NewReader(nil))
	assert.Nil(t, err)
	assert.Equal(t, em, m)

	assert.Nil(t, em.Write())
}
