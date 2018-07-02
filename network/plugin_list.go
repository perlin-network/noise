package network

import (
	"github.com/umpc/go-sortedmap"
)

// PluginInfo wraps a priority level with a plugin interface.
type PluginInfo struct {
	Priority int
	Plugin   PluginInterface
}

// PluginList holds a statically-typed sorted map of plugins
// registered on Noise.
type PluginList struct {
	inner *sortedmap.SortedMap
}

// Highest priority is considered to be on the lower end of the spectrum.
func byPriority(a, b interface{}) bool {
	return a.(*PluginInfo).Priority < b.(*PluginInfo).Priority
}

// NewPluginList creates a new instance of a sorted plugin list.
func NewPluginList() *PluginList {
	return &PluginList{inner: sortedmap.New(4, byPriority)}
}

// PutInfo places a new plugins info onto the list.
func (m *PluginList) PutInfo(key string, plugin *PluginInfo) bool {
	return m.inner.Insert(key, plugin)
}


// Put places a new plugin with a set priority onto the list.
func (m *PluginList) Put(key string, priority int, plugin PluginInterface) bool {
	return m.PutInfo(key, &PluginInfo{Priority: priority, Plugin: plugin})
}

// Len returns the number of plugins in the plugin list.
func (m *PluginList) Len() int {
	return m.inner.Len()
}

// Delete deletes a plugin by its ID from the list. Returns false if not exist.
func (m *PluginList) Delete(key string) bool {
	return m.inner.Delete(key)
}

// GetInfo gets the priority and plugin interface given a plugin ID. Returns nil if not exists.
func (m *PluginList) GetInfo(key string) (*PluginInfo, bool) {
	info, exists := m.inner.Get(key)
	return info.(*PluginInfo), exists
}

// Get returns the plugin interface given a plugin ID. Returns nil if not exists.
func (m *PluginList) Get(key string) (PluginInterface, bool) {
	info, exists := m.inner.Get(key)

	if exists {
		return info.(*PluginInfo).Plugin, exists
	} else {
		return nil, false
	}
}

// Each goes through every plugin in ascending order of priority of the plugin list.
func (m *PluginList) Each(f func(key string, value PluginInterface)) {
	m.inner.IterFunc(false, func(rec sortedmap.Record) bool {
		f(rec.Key.(string), rec.Val.(*PluginInfo).Plugin)
		return true
	})
}
