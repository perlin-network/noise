package noise

import (
	"github.com/perlin-network/noise/payload"
	"github.com/pkg/errors"
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

func TestEncodeMessageError(t *testing.T) {
	resetOpcodes()
	var err error

	p := newPeer(nil, nil)

	// test encode before register opcode
	_, err = p.EncodeMessage(testMsg{Text: "hello"})
	assert.Error(t, err)

	o := RegisterMessage(NextAvailableOpcode(), (*testMsg)(nil))
	assert.Equal(t, o, RegisterMessage(o, (*testMsg)(nil)))

	_, err = p.EncodeMessage(testMsg{Text: "hello"})
	assert.Nil(t, err)

	// test encode header error
	var headerError = errors.New("encode header error")
	p.OnEncodeHeader(func(node *Node, peer *Peer, header, msg []byte) (bytes []byte, e error) {
		return nil, headerError
	})
	_, err = p.EncodeMessage(testMsg{Text: "hello"})
	assert.Error(t, err)
	assert.Equal(t, headerError, errors.Cause(err))

	// test encode footer error
	var footerError = errors.New("encode footer error")
	p.OnEncodeFooter(func(node *Node, peer *Peer, header, msg []byte) (bytes []byte, e error) {
		return nil, footerError
	})
	_, err = p.EncodeMessage(testMsg{Text: "hello"})
	assert.Error(t, err)
	assert.Equal(t, footerError, errors.Cause(err))
}

func TestDecodeMessageError(t *testing.T) {
	p := newPeer(nil, nil)
	testMessage := testMsg{Text: "hello"}

	var o = NextAvailableOpcode()

	var resultO Opcode
	var resultM Message
	var err error

	// test decode before register opcode
	resultO, resultM, err = p.DecodeMessage(append([]byte{byte(o)}, testMessage.Write()...))
	assert.Equal(t, o, resultO)
	assert.Nil(t, resultM)
	assert.Error(t, err)

	resetOpcodes()
	assert.Equal(t, o, RegisterMessage(o, (*testMsg)(nil)))

	// test decode invalid opcode
	resultO, resultM, err = p.DecodeMessage(append([]byte{222}, testMessage.Write()...))
	assert.Equal(t, Opcode(byte(222)), resultO)
	assert.Nil(t, resultM)
	assert.Error(t, err)

	// test decode nil
	resultO, resultM, err = p.DecodeMessage(nil)
	assert.Equal(t, OpcodeNil, resultO)
	assert.Equal(t, nil, resultM)
	assert.Error(t, err)

	// test decode with nil content
	resultO, resultM, err = p.DecodeMessage(append([]byte{byte(o)}))
	assert.Equal(t, o, resultO)
	assert.Nil(t, resultM)
	assert.Error(t, err)

	// test encode header error
	var headerError = errors.New("decode header error")
	p.OnDecodeHeader(func(node *Node, peer *Peer, reader payload.Reader) error {
		return headerError
	})
	resultO, resultM, err = p.DecodeMessage(append([]byte{byte(o)}, testMessage.Write()...))
	assert.Equal(t, OpcodeNil, resultO)
	assert.Equal(t, nil, resultM)
	assert.Error(t, err)
	assert.Equal(t, headerError, errors.Cause(err))

	// test encode footer error
	var footerError = errors.New("decode footer error")
	p.OnDecodeFooter(func(node *Node, peer *Peer, msg []byte, reader payload.Reader) error {
		return footerError
	})
	resultO, resultM, err = p.DecodeMessage(append([]byte{byte(o)}, testMessage.Write()...))
	assert.Equal(t, OpcodeNil, resultO)
	assert.Equal(t, nil, resultM)
	assert.Error(t, err)
	assert.Equal(t, footerError, errors.Cause(err))
}