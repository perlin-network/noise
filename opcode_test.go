package noise

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

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
}
