package loop

import (
	"container/list"
	"sync"
)

// LRUCache implements a thread-safe Least Recently Used cache.

type LRUCache struct {
	mu sync.RWMutex

	capacity int

	items map[string]*list.Element

	lru *list.List
}

// cacheEntry represents an entry in the cache.

type cacheEntry struct {
	key string

	value interface{}
}

// NewLRUCache creates a new LRU cache with the specified capacity.

func NewLRUCache(capacity int) *LRUCache {
	if capacity <= 0 {
		capacity = 100
	}

	return &LRUCache{
		capacity: capacity,

		items: make(map[string]*list.Element),

		lru: list.New(),
	}
}

// Get retrieves a value from the cache.

func (c *LRUCache) Get(key string) (interface{}, bool) {
	c.mu.Lock()

	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		// Move to front (most recently used).

		c.lru.MoveToFront(elem)

		return elem.Value.(*cacheEntry).value, true
	}

	return nil, false
}

// Put adds or updates a value in the cache.

func (c *LRUCache) Put(key string, value interface{}) {
	c.mu.Lock()

	defer c.mu.Unlock()

	// Check if key exists.

	if elem, ok := c.items[key]; ok {
		// Update existing entry.

		c.lru.MoveToFront(elem)

		elem.Value.(*cacheEntry).value = value

		return
	}

	// Add new entry.

	entry := &cacheEntry{key: key, value: value}

	elem := c.lru.PushFront(entry)

	c.items[key] = elem

	// Evict if over capacity.

	if c.lru.Len() > c.capacity {
		c.evictLocked()
	}
}

// evictLocked removes the least recently used item (must be called with lock held).

func (c *LRUCache) evictLocked() {
	elem := c.lru.Back()

	if elem != nil {
		c.lru.Remove(elem)

		entry := elem.Value.(*cacheEntry)

		delete(c.items, entry.key)
	}
}

// Remove removes a key from the cache.

func (c *LRUCache) Remove(key string) bool {
	c.mu.Lock()

	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.lru.Remove(elem)

		delete(c.items, key)

		return true
	}

	return false
}

// Clear removes all items from the cache.

func (c *LRUCache) Clear() {
	c.mu.Lock()

	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)

	c.lru.Init()
}

// Size returns the current number of items in the cache.

func (c *LRUCache) Size() int {
	c.mu.RLock()

	defer c.mu.RUnlock()

	return c.lru.Len()
}

// Capacity returns the maximum capacity of the cache.

func (c *LRUCache) Capacity() int {
	return c.capacity
}
