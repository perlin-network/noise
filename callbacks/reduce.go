package callbacks

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)

type wrappedReduceCallback struct {
	file       string
	line       int
	createdAt  time.Time
	label      string
	createdIdx int
	cb         *reduceCallback
}

type ReduceCallbackManager struct {
	sync.Mutex

	callbacks []*wrappedReduceCallback
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

	_, file, no, _ := runtime.Caller(1)
	wc := &wrappedReduceCallback{
		file:       file,
		line:       no,
		createdAt:  time.Now(),
		createdIdx: len(m.callbacks),
		cb:         &c,
	}
	if m.reverse {
		m.callbacks = append([]*wrappedReduceCallback{wc}, m.callbacks...)
	} else {
		m.callbacks = append(m.callbacks, wc)
	}

	m.Unlock()
}

// RunCallbacks runs all callbacks on a variadic parameter list, and de-registers callbacks
// that throw an error.
func (m *ReduceCallbackManager) RunCallbacks(in interface{}, params ...interface{}) (res interface{}, errs []error) {
	m.Lock()

	cpy := make([]*wrappedReduceCallback, len(m.callbacks))
	copy(cpy, m.callbacks)

	m.callbacks = make([]*wrappedReduceCallback, 0)

	m.Unlock()

	var remaining []*wrappedReduceCallback
	var err error

	for _, c := range cpy {
		if in, err = (*c.cb)(in, params...); err != nil {
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

func (m *ReduceCallbackManager) ListCallbacks() []string {
	m.Lock()

	var results []string
	for i, cb := range m.callbacks {
		path := strings.Split(cb.file, "/")
		results = append(results, fmt.Sprintf("%d| %s:%d", i, strings.Join(path[len(path)-3:], "/"), cb.line))
	}

	m.Unlock()

	return results
}
