//go:generate genny -in=$GOFILE -out=gen-string-PeerClient-$GOFILE gen "Key=string Value=*PeerClient"
//go:generate genny -in=$GOFILE -out=gen-uint64-MessageChannel-$GOFILE gen "Key=uint64 Value=MessageChannel"
//go:generate genny -in=$GOFILE -out=gen-string-MessageProcessor-$GOFILE gen "Key=string Value=MessageProcessor"

package network

import (
	"github.com/cheekybits/genny/generic"
	"sync"
)

type Key generic.Type
type Value generic.Type

type KeyValueSyncMap struct {
	inner sync.Map
}

func (m *KeyValueSyncMap) Store(k Key, v Value) {
	m.inner.Store(k, v)
}

func (m *KeyValueSyncMap) Load(k Key) (Value, bool) {
	val, ok := m.inner.Load(k)
	if !ok {
		return nil, false
	}

	return val.(Value), true
}

func (m *KeyValueSyncMap) Range(cb func(Key, Value) bool) {
	m.inner.Range(func(k interface{}, v interface{}) bool {
		return cb(k.(Key), v.(Value))
	})
}

func (m *KeyValueSyncMap) Delete(key Key) {
	m.inner.Delete(key)
}
