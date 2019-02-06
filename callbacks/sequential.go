package callbacks

import "sync"

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

	cpy := make([]*callback, len(m.callbacks))
	copy(cpy, m.callbacks)

	m.callbacks = make([]*callback, 0)

	m.Unlock()

	var remaining []*callback
	var err error

	for _, c := range cpy {
		if err = (*c)(params...); err != nil {
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

	return
}
