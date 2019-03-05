package noise

import (
	"fmt"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/transport"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"testing/quick"
	"time"
)

func TestNewNode(t *testing.T) {
	tests := []struct {
		name    string
		params  parameters
		wantErr bool
	}{
		{
			name: "bad port 1",
			params: func() parameters {
				p := DefaultParams()
				p.Port = 1
				return p
			}(),
			wantErr: true,
		},
		{
			name: "bad port 1023",
			params: func() parameters {
				p := DefaultParams()
				p.Port = 1023
				return p
			}(),
			wantErr: true,
		},
		{
			name: "good port 0",
			params: func() parameters {
				p := DefaultParams()
				p.Port = 0
				return p
			}(),
			wantErr: false,
		},
		{
			name: "bad transport",
			params: func() parameters {
				p := DefaultParams()
				p.Transport = nil
				return p
			}(),
			wantErr: true,
		},
		{
			name: "many parameters",
			params: func() parameters {
				p := DefaultParams()
				p.Metadata["a"] = "b"
				p.Metadata["1"] = 2
				p.Metadata["f"] = 3.0
				return p
			}(),
			wantErr: false,
		},
		{
			name: "bad host",
			params: func() parameters {
				p := DefaultParams()
				p.Host = "bad host"
				p.Port = 1234
				return p
			}(),
			wantErr: true,
		},
		{
			name: "use mock transport",
			params: func() parameters {
				p := DefaultParams()
				p.Transport = transport.NewBuffered()
				return p
			}(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := NewNode(tt.params)
			if tt.wantErr {
				assert.NotNil(t, err)
				return
			}
			assert.Nil(t, err)
			assert.NotNil(t, node)

			// check the metadata
			for key, val := range tt.params.Metadata {
				assert.Equal(t, val, node.Get(key))
			}

			// check the port
			if tt.params.Port == 0 {
				assert.True(t, node.InternalPort() > 1024)
			} else {
				assert.Equal(t, tt.params.Port, node.InternalPort())
			}
			assert.Equal(t, fmt.Sprintf("%s:%d", tt.params.Host, node.InternalPort()), node.ExternalAddress())
		})
	}
}

func TestNodeMetadata(t *testing.T) {
	node, err := NewNode(DefaultParams())
	assert.Nil(t, err)

	assert.Equal(t, false, node.Has("missing"))
	assert.Nil(t, node.Get("missing"))

	checkInterface := func(key1 string, val1 interface{}, val2 interface{}) bool {
		key2 := fmt.Sprintf("%s2", key1)

		node.Set(key1, val1)
		assert.Equal(t, true, node.Has(key1))
		assert.Equal(t, val1, node.Get(key1))

		node.Delete(key1)
		assert.Equal(t, false, node.Has(key1))
		assert.Nil(t, node.Get(key1))

		two := node.LoadOrStore(key2, val2)
		assert.Equal(t, val2, two)
		assert.Equal(t, true, node.Has(key2))
		assert.Equal(t, val2, node.Get(key2))

		notTwo := node.LoadOrStore(key2, val1)
		assert.Equal(t, val2, notTwo)
		assert.Equal(t, true, node.Has(key2))
		assert.Equal(t, val2, node.Get(key2))

		node.Delete(key2)

		return true
	}

	checkInts := func(key string, val1 int, val2 int) bool {
		return checkInterface(key, val1, val2)
	}
	checkFloats := func(key string, val1 float32, val2 float32) bool {
		return checkInterface(key, val1, val2)
	}
	checkStrings := func(key string, val1 string, val2 string) bool {
		return checkInterface(key, val1, val2)
	}

	// quick test all the parameter types
	assert.Nil(t, quick.Check(checkInts, nil))
	assert.Nil(t, quick.Check(checkFloats, nil))
	assert.Nil(t, quick.Check(checkStrings, nil))
}

func TestCallbacks(t *testing.T) {
	log.Disable()
	defer log.Enable()

	layer := transport.NewBuffered()
	numNodes := 2
	var nodes []*Node
	var callbacks []*counter
	allTypes := []string{
		"OnListenerError",
		"OnPeerConnected",
		"OnPeerDisconnected",
		"OnPeerDialed",
		"OnPeerInit",
	}

	for i := 0; i < numNodes; i++ {
		p := DefaultParams()
		p.Port = uint16(7000 + i)
		p.Transport = layer

		n, err := NewNode(p)
		assert.Nil(t, err)
		nodes = append(nodes, n)

		cb := NewCounter()
		callbacks = append(callbacks, cb)

		// setup callbacks
		n.OnListenerError(func(_ *Node, _ error) error {
			cb.Incr("OnListenerError")
			return nil
		})
		n.OnPeerConnected(func(_ *Node, _ *Peer) error {
			cb.Incr("OnPeerConnected")
			return nil
		})
		n.OnPeerDisconnected(func(_ *Node, _ *Peer) error {
			cb.Incr("OnPeerDisconnected")
			return nil
		})
		n.OnPeerDialed(func(_ *Node, _ *Peer) error {
			cb.Incr("OnPeerDialed")
			return nil
		})
		n.OnPeerInit(func(_ *Node, _ *Peer) error {
			cb.Incr("OnPeerInit")
			return nil
		})

		go n.Listen()
	}

	compareCB := func(actuals *counter, expected map[string]int) {
		for _, key := range allTypes {
			if _, ok := expected[key]; ok {
				assert.Equalf(t, expected[key], actuals.Get(key), "count mismatch for key %s", key)
			} else {
				assert.Equalf(t, 0, actuals.Get(key), "count mismatch for key %s", key)
			}
		}
	}

	clearCounters := func() {
		for _, cb := range callbacks {
			cb.Clear()
		}
	}

	for i := 0; i < numNodes; i++ {
		clearCounters()

		// dial the next node
		src := i
		dst := (i + 1) % numNodes
		peer, err := nodes[src].Dial(nodes[dst].ExternalAddress())
		assert.Nil(t, err)

		time.Sleep(1 * time.Millisecond)

		// check that the expected callbacks were called on the dialer
		compareCB(callbacks[src], map[string]int{
			"OnPeerDialed": 1,
			"OnPeerInit":   1,
		})

		// check that the expected callbacks were called on the reciever
		compareCB(callbacks[dst], map[string]int{
			"OnPeerConnected": 1,
			"OnPeerInit":      1,
		})

		clearCounters()

		peer.Disconnect()

		time.Sleep(10 * time.Millisecond)

		// check that the expected callbacks were called on the dialer
		compareCB(callbacks[src], map[string]int{
			"OnPeerDisconnected": 1,
		})

		// check that the expected callbacks were called on the receiver
		//compareCB(callbacks[dst], map[string]int{
		//	"OnPeerDisconnected": 1,
		//})
	}
}

func TestNodeKill(t *testing.T) {
	numNodes := 2

	var nodes []*Node
	layer := transport.NewBuffered()

	for i := 0; i < numNodes; i++ {
		p := DefaultParams()
		p.Port = uint16(7100 + i)
		p.Transport = layer

		n, err := NewNode(p)
		assert.Nil(t, err)
		nodes = append(nodes, n)

		go n.Listen()
	}

	_, err := nodes[0].Dial(nodes[1].ExternalAddress())
	assert.NoError(t, err)

	for i := 0; i < numNodes; i++ {
		nodes[i].Kill()
	}
}

// counter is a concurrent safe map of counters
type counter struct {
	mu     sync.Mutex
	values map[string]int
}

func NewCounter() *counter {
	return &counter{
		values: make(map[string]int),
	}
}

func (s *counter) Clear() *counter {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values = make(map[string]int)
	return s
}

func (s *counter) Get(key string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.values[key]
}

func (s *counter) Incr(key string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key]++
	return s.values[key]
}
