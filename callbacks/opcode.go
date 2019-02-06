package callbacks

import (
	"math"
	"sync"
)

// OpcodeCallbackManager maps opcodes to a sequential list of callback functions.
// We assumes that there are at most 1 << 8 - 1 callbacks (opcodes are represented as uint32).
type OpcodeCallbackManager struct {
	sync.Mutex

	callbacks [math.MaxUint8 + 1]*SequentialCallbackManager
}

func NewOpcodeCallbackManager() *OpcodeCallbackManager {
	return &OpcodeCallbackManager{}
}

func (m *OpcodeCallbackManager) RegisterCallback(opcode byte, c callback) {
	m.Lock()

	if m.callbacks[opcode] == nil {
		m.callbacks[opcode] = NewSequentialCallbackManager()
	}

	manager := m.callbacks[opcode]

	m.Unlock()

	manager.RegisterCallback(c)
}

// RunCallbacks runs all callbacks on a variadic parameter list, and de-registers callbacks
// that throw an error.
func (m *OpcodeCallbackManager) RunCallbacks(opcode byte, params ...interface{}) (errs []error) {
	m.Lock()
	manager := m.callbacks[opcode]
	m.Unlock()

	if manager == nil {
		return
	}

	errs = manager.RunCallbacks(params...)

	return errs
}
