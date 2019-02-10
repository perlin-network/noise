package callbacks

type ReduceCallbackManager struct {
	seqMgr *SequentialCallbackManager
}

func NewReduceCallbackManager() *ReduceCallbackManager {
	return &ReduceCallbackManager{
		seqMgr: NewSequentialCallbackManager(),
	}
}

func (m *ReduceCallbackManager) UnsafelySetReverse() *ReduceCallbackManager {
	m.seqMgr.UnsafelySetReverse()
	return m
}

func (m *ReduceCallbackManager) RegisterCallback(c ReduceCallback) {
	m.seqMgr.RegisterCallback(func(params ...interface{}) error {
		valueOut := params[0].(*interface{})
		var err error
		*valueOut, err = c(*valueOut, params[1:]...)
		return err
	})
}

// RunCallbacks runs all callbacks on a variadic parameter list, and de-registers callbacks
// that throw an error.
func (m *ReduceCallbackManager) RunCallbacks(in interface{}, params ...interface{}) (res interface{}, errs []error) {
	errs = m.seqMgr.RunCallbacks(append([]interface{}{&in}, params...)...)
	res = in
	return
}

// MustRunCallbacks runs all callbacks on a variadic parameter list, and de-registers callbacks
// that throw an error. Errors are ignored.
func (m *ReduceCallbackManager) MustRunCallbacks(in interface{}, params ...interface{}) interface{} {
	out, _ := m.RunCallbacks(in, params...)
	return out
}
