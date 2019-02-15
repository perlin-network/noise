package noise

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"testing/quick"
)

func TestEncodeMessage(t *testing.T) {
	resetOpcodes()

	o := RegisterMessage(Opcode(123), (*testMsg)(nil))
	assert.Equal(t, o, RegisterMessage(o, (*testMsg)(nil)))

	p := newPeer(nil, nil)

	f := func(msg testMsg) bool {
		bytes, err := p.EncodeMessage(msg)
		assert.Nil(t, err)
		assert.Equal(t, append([]byte{byte(o)}, msg.Write()...), bytes)

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestDecodeMessage(t *testing.T) {
	resetOpcodes()

	o := RegisterMessage(Opcode(45), (*testMsg)(nil))
	assert.Equal(t, o, RegisterMessage(o, (*testMsg)(nil)))

	p := newPeer(nil, nil)

	f := func(msg testMsg) bool {
		resultO, resultM, err := p.DecodeMessage(append([]byte{byte(o)}, msg.Write()...))
		assert.Nil(t, err)
		assert.Equal(t, o, resultO)
		assert.Equal(t, msg, resultM)

		return true
	}

	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}
