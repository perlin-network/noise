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
		return nil, errors.Wrap(err, "failed to read test message")
	}

	return testMsg{text: text}, nil
}

func (m testMsg) Write() []byte {
	return payload.NewWriter(nil).WriteString(m.text).Bytes()
}

func TestEncodeMessage(t *testing.T) {
	resetOpcodes()
	o := RegisterMessage(Opcode(123), (*testMsg)(nil))

	msg := testMsg{
		text: "hello",
	}

	p := newPeer(nil, nil)

	bytes, err := p.EncodeMessage(msg)
	assert.Nil(t, err)
	assert.Equal(t, append([]byte{byte(o)}, msg.Write()...), bytes)
}

func TestDecodeMessage(t *testing.T) {
	resetOpcodes()
	o := RegisterMessage(Opcode(45), (*testMsg)(nil))

	msg := testMsg{
		text: "world",
	}
	assert.Equal(t, o, RegisterMessage(o, (*testMsg)(nil)))

	p := newPeer(nil, nil)

	resultO, resultM, err := p.DecodeMessage(append([]byte{byte(o)}, msg.Write()...))
	assert.Nil(t, err)
	assert.Equal(t, o, resultO)
	assert.Equal(t, msg, resultM)
}
