package callbacks

import "sync"

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

	cpy := make([]*reduceCallback, len(m.callbacks))
	copy(cpy, m.callbacks)

	m.callbacks = make([]*reduceCallback, 0)

	m.Unlock()

	var remaining []*reduceCallback
	var err error

	for _, c := range cpy {
		if in, err = (*c)(in, params...); err != nil {
			if err != DeregisterCallback {
				errs = append(errs, err)
			}
		} else {
			remaining = append(remaining, c)
		}
	}

	m.Lock()

	m.callbacks = append(m.callbacks, remaining...)

	m.Unlock()

	return in, errs
}

// MustRunCallbacks runs all callbacks on a variadic parameter list, and de-registers callbacks
// that throw an error. Errors are ignored.
func (m *ReduceCallbackManager) MustRunCallbacks(in interface{}, params ...interface{}) interface{} {
	out, _ := m.RunCallbacks(in, params...)
	return out
}
