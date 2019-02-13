package noise

import (
	"github.com/perlin-network/noise/payload"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
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

	RegisterMessage(Opcode(1), (*testMsg)(nil))

	// an opcode should still exist after the loop
	msg, err = MessageFromOpcode(Opcode(1))
	assert.Nil(t, err)
	assert.Equal(t, testMsg{}, msg)

	DebugOpcodes()
}

func TestEncodeMessage(t *testing.T) {
	resetOpcodes()

	p := newPeer(nil, nil)
	msg := testMsg{
		text: "hello",
	}

	// test an unregister messsage for an error
	_, err := p.EncodeMessage(msg)
	assert.NotNil(t, err)

	// register and encode again
	o := RegisterMessage(Opcode(123), (*testMsg)(nil))
	bytes, err := p.EncodeMessage(msg)
	assert.Nil(t, err)
	assert.Equal(t, append([]byte{byte(o)}, msg.Write()...), bytes)
}

func TestDecodeMessage(t *testing.T) {
	resetOpcodes()

	p := newPeer(nil, nil)
	msg := testMsg{
		text: "world",
	}
	o := Opcode(45)

	// decode an unregistered msg
	_, _, err := p.DecodeMessage(append([]byte{byte(o)}, msg.Write()...))
	assert.NotNil(t, err)

	// decode a registered one
	o = RegisterMessage(o, (*testMsg)(nil))
	assert.Equal(t, o, RegisterMessage(o, (*testMsg)(nil)))

	resultO, resultM, err := p.DecodeMessage(append([]byte{byte(o)}, msg.Write()...))
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
