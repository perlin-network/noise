package network

import (
	"github.com/umpc/go-sortedmap"
)

type PluginInfo struct {
	Priority int
	Plugin   PluginInterface
}

type PluginList struct {
	inner *sortedmap.SortedMap
}

func byPriority(a, b interface{}) bool {
	return a.(*PluginInfo).Priority < b.(*PluginInfo).Priority
}

func NewPluginList() *PluginList {
	return &PluginList{inner: sortedmap.New(4, byPriority)}
}

func (m *PluginList) Put(key string, plugin *PluginInfo) bool {
	return m.inner.Insert(key, plugin)
}

func (m *PluginList) Len() int {
	return m.inner.Len()
}

func (m *PluginList) Delete(key string) {
	m.inner.Delete(key)
}

func (m *PluginList) GetInfo(key string) (*PluginInfo, bool) {
	info, exists := m.inner.Get(key)
	return info.(*PluginInfo), exists
}

func (m *PluginList) Get(key string) (PluginInterface, bool) {
	info, exists := m.inner.Get(key)

	if exists {
		return info.(*PluginInfo).Plugin, exists
	} else {
		return nil, false
	}
}

func (m *PluginList) Each(f func(key string, value PluginInterface)) {
	m.inner.IterFunc(false, func(rec sortedmap.Record) bool {
		f(rec.Key.(string), rec.Val.(*PluginInfo).Plugin)
		return true
	})
}
