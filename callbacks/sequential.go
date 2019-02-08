package callbacks

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

const autoTrimThreshold = 10

type SequentialCallbackManager struct {
	callbacksMutex sync.Mutex // this mutex only protects the `callbacks` pointer itself.

	callbacks           *[]callbackState
	pendingRemovalCount uint64

	reverse bool
}

type callbackState struct {
	cb             Callback
	pendingRemoval uint32
}

func NewSequentialCallbackManager() *SequentialCallbackManager {
	callbacks := make([]callbackState, 0)

	return &SequentialCallbackManager{
		reverse:   false,
		callbacks: &callbacks,
	}
}

func (m *SequentialCallbackManager) pushCallback(cb Callback) {
	callbacks := m.loadCallbacks()
	if len(callbacks) == cap(callbacks) {
		newCallbacks := make([]callbackState, len(callbacks), len(callbacks)*2+1)
		for i := 0; i < len(callbacks); i++ {
			oldCb := &callbacks[i]
			newCallbacks[i].cb = oldCb.cb
			newCallbacks[i].pendingRemoval = atomic.LoadUint32(&oldCb.pendingRemoval)
		}
		callbacks = newCallbacks
	}
	callbacks = append(callbacks, callbackState{
		cb: cb,
	})
	m.storeCallbacks(callbacks)
}

func (m *SequentialCallbackManager) UnsafelySetReverse() *SequentialCallbackManager {
	m.reverse = true
	return m
}

func (m *SequentialCallbackManager) loadCallbacks() []callbackState {
	return *(*[]callbackState)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&m.callbacks))))
}

func (m *SequentialCallbackManager) storeCallbacks(callbacks []callbackState) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&m.callbacks)), unsafe.Pointer(&callbacks))
}

func (m *SequentialCallbackManager) Trim() {
	atomic.StoreUint64(&m.pendingRemovalCount, 0)

	m.callbacksMutex.Lock()
	newCallbacks := make([]callbackState, 0)
	for _, cb := range m.loadCallbacks() {
		if cb.pendingRemoval == 0 {
			newCallbacks = append(newCallbacks, cb)
		}
	}
	m.storeCallbacks(newCallbacks)
	m.callbacksMutex.Unlock()
}

// RegisterCallback atomically registers all callbacks passed in.
func (m *SequentialCallbackManager) RegisterCallback(callbacks ...Callback) {
	m.callbacksMutex.Lock()
	for _, c := range callbacks {
		m.pushCallback(c)
	}
	m.callbacksMutex.Unlock()
}

// RunCallbacks runs all callbacks on a variadic parameter list, and de-registers callbacks
// that throw an error.
func (m *SequentialCallbackManager) RunCallbacks(params ...interface{}) []error {
	callbacks := m.loadCallbacks()
	if m.reverse {
		for i := len(callbacks) - 1; i >= 0; i-- {
			c := &callbacks[i]
			if err := m.doRunCallback(c, params...); err != nil {
				return []error{err}
			}
		}
	} else {
		for i := 0; i < len(callbacks); i++ {
			c := &callbacks[i]
			if err := m.doRunCallback(c, params...); err != nil {
				return []error{err}
			}
		}
	}

	if atomic.LoadUint64(&m.pendingRemovalCount) >= autoTrimThreshold {
		m.Trim()
	}

	return nil
}

func (m *SequentialCallbackManager) doRunCallback(c *callbackState, params ...interface{}) error {
	if atomic.LoadUint32(&c.pendingRemoval) == 0 {
		err := c.cb(params...)
		if err != nil {
			atomic.StoreUint32(&c.pendingRemoval, 1)
			atomic.AddUint64(&m.pendingRemovalCount, 1)
			if err != DeregisterCallback {
				return err
			}
		}
	}

	return nil
}
