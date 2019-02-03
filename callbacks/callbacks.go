package callbacks

import (
	"github.com/pkg/errors"
	"math"
	"sync"
)

var DeregisterCallback = errors.New("callback deregistered")

type callback func(params ...interface{}) error
type reduceCallback func(in interface{}, params ...interface{}) (interface{}, error)

type ReduceCallbackManager struct {
	sync.Mutex

	callbacks []*reduceCallback
	reverse   bool
}

func NewReduceCallbackManager() *ReduceCallbackManager {
	return &ReduceCallbackManager{reverse: false}
}

func (m *ReduceCallbackManager) Reverse() *ReduceCallbackManager {
	m.reverse = true
	return m
}

func (m *ReduceCallbackManager) RegisterCallback(c reduceCallback) {
	m.Lock()

	if m.reverse {
		m.callbacks = append([]*reduceCallback{&c}, m.callbacks...)
	} else {
		m.callbacks = append(m.callbacks, &c)
	}

	m.Unlock()
}

// RunCallbacks runs all callbacks on a variadic parameter list, and de-registers callbacks
// that throw an error.
func (m *ReduceCallbackManager) RunCallbacks(in interface{}, params ...interface{}) (res interface{}, errs []error) {
	m.Lock()

	callbacksCopy := make([]*reduceCallback, len(m.callbacks))
	copy(callbacksCopy, m.callbacks)

	m.Unlock()

	var remaining []*reduceCallback
	var err error

	for _, c := range callbacksCopy {
		if in, err = (*c)(in, params...); err != nil {
			if errors.Cause(err) != DeregisterCallback {
				errs = append(errs, err)
			}
		} else {
			remaining = append(remaining, c)
		}
	}

	m.Lock()

	m.callbacks = remaining

	m.Unlock()

	return in, errs
}

// MustRunCallbacks runs all callbacks on a variadic parameter list, and de-registers callbacks
// that throw an error. Errors are ignored.
func (m *ReduceCallbackManager) MustRunCallbacks(in interface{}, params ...interface{}) interface{} {
	out, _ := m.RunCallbacks(in, params...)
	return out
}

type SequentialCallbackManager struct {
	sync.Mutex

	callbacks []*callback
	reverse   bool
}

func NewSequentialCallbackManager() *SequentialCallbackManager {
	return &SequentialCallbackManager{reverse: false}
}

func (m *SequentialCallbackManager) Reverse() *SequentialCallbackManager {
	m.reverse = true
	return m
}

func (m *SequentialCallbackManager) RegisterCallback(c callback) {
	m.Lock()

	if m.reverse {
		m.callbacks = append([]*callback{&c}, m.callbacks...)
	} else {
		m.callbacks = append(m.callbacks, &c)
	}

	m.Unlock()
}

// RunCallbacks runs all callbacks on a variadic parameter list, and de-registers callbacks
// that throw an error.
func (m *SequentialCallbackManager) RunCallbacks(params ...interface{}) (errs []error) {
	m.Lock()

	callbacksCopy := make([]*callback, len(m.callbacks))
	copy(callbacksCopy, m.callbacks)

	m.Unlock()

	var remaining []*callback

	for _, c := range callbacksCopy {
		if err := (*c)(params...); err != nil {
			if errors.Cause(err) != DeregisterCallback {
				errs = append(errs, err)
			}
		} else {
			remaining = append(remaining, c)
		}
	}

	m.Lock()

	m.callbacks = remaining

	m.Unlock()

	return
}

// OpcodeCallbackManager maps opcodes to a sequential list of callback functions.
// We assumes that there are at most 1 << 8 - 1 callbacks (opcodes are represented as uint32).
type OpcodeCallbackManager struct {
	sync.Mutex

	callbacks [math.MaxUint8]*SequentialCallbackManager
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
