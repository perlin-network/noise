package builders

import "sort"

type pluginPriority struct {
	Priority  int
	Name      string
	InsertIdx int
}

type pluginPriorities []*pluginPriority

func (p pluginPriorities) Len() int {
	return len(p)
}

func (p pluginPriorities) Less(i, j int) bool {
	if p[i].InsertIdx == p[j].InsertIdx {
		// if priority is the same, sort ascending my insertion index
		return p[i].InsertIdx < p[j].InsertIdx
	}
	// sort in ascending order by priority (1, 2, 3, ...)
	return p[i].Priority < p[j].Priority
}

func (p pluginPriorities) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p pluginPriorities) GetSortedNames() []string {
	sort.Sort(p)
	var names []string
	for _, n := range p {
		names = append(names, n.Name)
	}
	return names
}
