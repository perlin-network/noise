package lru

import (
	"container/list"
	"sync"
)

type cacheItem struct {
	key     string
	value   interface{}
	element *list.Element
}

// Cache is a simple concurrent-safe cache with a least-recently-used (LRU)
// eviction policy.
type Cache struct {
	order *list.List
	items map[string]*cacheItem

	limit int

	mutex *sync.Mutex
}

// NewCache instantiates a new concurrent-safe LRU cache.
func NewCache(limit int) *Cache {
	return &Cache{order: list.New(), items: make(map[string]*cacheItem), limit: limit, mutex: &sync.Mutex{}}
}

// Get returns a cached value for a key, and initializes it otherwise should it not exist.
func (c *Cache) Get(key string, init func() (interface{}, error)) (interface{}, error) {
	c.mutex.Lock()

	// If key exists, mark it as used and return it.
	if item, exists := c.items[key]; exists {
		c.order.MoveToFront(item.element)

		c.mutex.Unlock()
		return item.value, nil
	}

	// Evict least recently used.
	if c.order.Len() >= c.limit {
		// Pop last element.
		item := c.order.Remove(c.order.Back()).(*cacheItem)
		delete(c.items, item.key)
	}

	// If key does not exist, push it to the front.
	value, err := init()
	if err != nil {
		c.mutex.Unlock()
		return nil, err
	}

	item := &cacheItem{key: key, value: value}
	item.element = c.order.PushFront(item)
	c.items[key] = item

	c.mutex.Unlock()
	return item.value, nil
}
