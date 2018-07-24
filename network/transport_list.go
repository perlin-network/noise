package network

import (
	"reflect"
	"sort"
)

// TransportInfo wraps a priority level with a Transport interface.
type TransportInfo struct {
	Priority  int
	Transport TransportInterface
}

// TransportList holds a statically-typed sorted map of Transports
// registered on Noise.
type TransportList struct {
	keys   map[reflect.Type]*TransportInfo
	values []*TransportInfo
}

// NewTransportList creates a new instance of a sorted Transport list.
func NewTransportList() *TransportList {
	return &TransportList{
		keys:   make(map[reflect.Type]*TransportInfo),
		values: make([]*TransportInfo, 0),
	}
}

// SortByPriority sorts the Transports list by each Transports priority.
func (m *TransportList) SortByPriority() {
	sort.SliceStable(m.values, func(i, j int) bool {
		return m.values[i].Priority < m.values[j].Priority
	})
}

// PutInfo places a new Transports info onto the list.
func (m *TransportList) PutInfo(Transport *TransportInfo) bool {
	ty := reflect.TypeOf(Transport.Transport)
	if _, ok := m.keys[ty]; ok {
		return false
	}
	m.keys[ty] = Transport
	m.values = append(m.values, Transport)
	return true
}

// Put places a new Transport with a set priority onto the list.
func (m *TransportList) Put(priority int, Transport TransportInterface) bool {
	return m.PutInfo(&TransportInfo{
		Priority:  priority,
		Transport: Transport,
	})
}

// Len returns the number of Transports in the Transport list.
func (m *TransportList) Len() int {
	return len(m.keys)
}

// GetInfo gets the priority and Transport interface given a Transport ID. Returns nil if not exists.
func (m *TransportList) GetInfo(withTy interface{}) (*TransportInfo, bool) {
	item, ok := m.keys[reflect.TypeOf(withTy)]
	return item, ok
}

// Get returns the Transport interface given a Transport ID. Returns nil if not exists.
func (m *TransportList) Get(withTy interface{}) (TransportInterface, bool) {
	if info, ok := m.GetInfo(withTy); ok {
		return info.Transport, true
	} else {
		return nil, false
	}
}

// Each goes through every Transport in ascending order of priority of the Transport list.
func (m *TransportList) Each(f func(value TransportInterface)) {
	for _, item := range m.values {
		f(item.Transport)
	}
}
