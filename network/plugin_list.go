package network

import (
	"sort"
	"reflect"
)

// PluginInfo wraps a priority level with a plugin interface.
type PluginInfo struct {
	Priority int
	Plugin PluginInterface
}

// PluginList holds a statically-typed sorted map of plugins
// registered on Noise.
type PluginList struct {
	byType map[reflect.Type]*PluginInfo
	byPriority []*PluginInfo
}

// NewPluginList creates a new instance of a sorted plugin list.
func NewPluginList() *PluginList {
	return &PluginList {
		byType: make(map[reflect.Type]*PluginInfo),
		byPriority: make([]*PluginInfo, 0),
	}
}

func (m *PluginList) Fixup() {
	sort.SliceStable(m.byPriority, func (i, j int) bool {
		return m.byPriority[i].Priority < m.byPriority[j].Priority
	})
}

// PutInfo places a new plugins info onto the list.
func (m *PluginList) PutInfo(plugin *PluginInfo) bool {
	ty := reflect.TypeOf(plugin.Plugin)
	if _, ok := m.byType[ty]; ok {
		return false
	}
	m.byType[ty] = plugin
	m.byPriority = append(m.byPriority, plugin)
	return true
}

// Put places a new plugin with a set priority onto the list.
func (m *PluginList) Put(priority int, plugin PluginInterface) bool {
	return m.PutInfo(&PluginInfo {
		Priority: priority,
		Plugin: plugin,
	})
}

// Len returns the number of plugins in the plugin list.
func (m *PluginList) Len() int {
	return len(m.byType)
}

// GetInfo gets the priority and plugin interface given a plugin ID. Returns nil if not exists.
func (m *PluginList) GetInfo(withTy interface{}) (*PluginInfo, bool) {
	item, ok := m.byType[reflect.TypeOf(withTy)]
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
	for _, item := range m.byPriority {
		f(item.Plugin)
	}
}
