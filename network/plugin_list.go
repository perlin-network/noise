package network

import (
	"reflect"
	"sort"
)

// PluginInfo wraps a priority level with a plugin interface.
type PluginInfo struct {
	Priority int
	Plugin   PluginInterface
}

// PluginList holds a statically-typed sorted map of plugins
// registered on Noise.
type PluginList struct {
	keys   map[reflect.Type]*PluginInfo
	values []*PluginInfo
}

// NewPluginList creates a new instance of a sorted plugin list.
func NewPluginList() *PluginList {
	return &PluginList{
		keys:   make(map[reflect.Type]*PluginInfo),
		values: make([]*PluginInfo, 0),
	}
}

// SortByPriority sorts the plugins list by each plugins priority.
func (m *PluginList) SortByPriority() {
	sort.SliceStable(m.values, func(i, j int) bool {
		return m.values[i].Priority < m.values[j].Priority
	})
}

// PutInfo places a new plugins info onto the list.
func (m *PluginList) PutInfo(plugin *PluginInfo) bool {
	ty := reflect.TypeOf(plugin.Plugin)
	if _, ok := m.keys[ty]; ok {
		return false
	}
	m.keys[ty] = plugin
	m.values = append(m.values, plugin)
	return true
}

// Put places a new plugin with a set priority onto the list.
func (m *PluginList) Put(priority int, plugin PluginInterface) bool {
	return m.PutInfo(&PluginInfo{
		Priority: priority,
		Plugin:   plugin,
	})
}

// Len returns the number of plugins in the plugin list.
func (m *PluginList) Len() int {
	return len(m.keys)
}

// GetInfo gets the priority and plugin interface given a plugin ID. Returns nil if not exists.
func (m *PluginList) GetInfo(withTy interface{}) (*PluginInfo, bool) {
	item, ok := m.keys[reflect.TypeOf(withTy)]
	return item, ok
}

// Get returns the plugin interface given a plugin ID. Returns nil if not exists.
func (m *PluginList) Get(withTy interface{}) (PluginInterface, bool) {
	if info, ok := m.GetInfo(withTy); ok {
		return info.Plugin, true
	} else {
		return nil, false
	}
}

// Each goes through every plugin in ascending order of priority of the plugin list.
func (m *PluginList) Each(f func(value PluginInterface)) {
	for _, item := range m.values {
		f(item.Plugin)
	}
}
