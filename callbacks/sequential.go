package callbacks

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)

type wrappedCallback struct {
	file       string
	line       int
	createdAt  time.Time
	label      string
	createdIdx int
	cb         *callback
}

type SequentialCallbackManager struct {
	sync.Mutex

	callbacks []*wrappedCallback
	reverse   bool

	logMetadata *LogMetadata
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

	offset := 1
	if m.logMetadata != nil {
		offset = m.logMetadata.RuntimeCallerOffset
	}
	_, file, no, _ := runtime.Caller(offset)
	wc := &wrappedCallback{
		file:       file,
		line:       no,
		createdAt:  time.Now(),
		createdIdx: len(m.callbacks),
		cb:         &c,
	}

	if m.reverse {
		m.callbacks = append([]*wrappedCallback{wc}, m.callbacks...)
	} else {
		m.callbacks = append(m.callbacks, wc)
	}

	m.Unlock()
}

// RunCallbacks runs all callbacks on a variadic parameter list, and de-registers callbacks
// that throw an error.
func (m *SequentialCallbackManager) RunCallbacks(params ...interface{}) (errs []error) {
	m.Lock()

	cpy := make([]*wrappedCallback, len(m.callbacks))
	copy(cpy, m.callbacks)

	m.callbacks = make([]*wrappedCallback, 0)

	m.Unlock()

	var remaining []*wrappedCallback
	var err error

	for _, c := range cpy {
		if err = (*c.cb)(params...); err != nil {
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

func (m *SequentialCallbackManager) SetLogMetadata(l *LogMetadata) {
	m.logMetadata = l
}

func (m *SequentialCallbackManager) ListCallbacks() []string {
	m.Lock()

	nodeID := "unset"
	peerID := "unset"
	if m.logMetadata != nil {
		nodeID = m.logMetadata.NodeID
		peerID = m.logMetadata.PeerID
	}

	var results []string
	for i, cb := range m.callbacks {
		path := strings.Split(cb.file, "/")
		results = append(results, fmt.Sprintf("%d node=%s,peer=%s,%d|r %s:%d %s %d %d",
			time.Now().UnixNano(), nodeID, peerID,
			i, strings.Join(path[len(path)-3:], "/"), cb.line, cb.label, cb.createdIdx, cb.createdAt.UnixNano()))
	}

	m.Unlock()

	return results
}
