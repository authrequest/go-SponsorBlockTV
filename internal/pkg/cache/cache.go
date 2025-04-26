package cache

import (
	"container/list"
	"sync"
	"time"
)

// CacheEntry represents a single cache entry with value and expiration time
type CacheEntry struct {
	value      interface{}
	expiration *time.Time
}

// Cache implements a thread-safe LRU cache with conditional TTL
type Cache struct {
	mu       sync.RWMutex
	capacity int
	ttl      time.Duration
	items    map[string]*list.Element
	lruList  *list.List
}

// NewCache creates a new cache with the specified capacity and TTL
func NewCache(capacity int, ttl time.Duration) *Cache {
	return &Cache{
		capacity: capacity,
		ttl:      ttl,
		items:    make(map[string]*list.Element),
		lruList:  list.New(),
	}
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if element, exists := c.items[key]; exists {
		entry := element.Value.(*CacheEntry)

		// Check if entry has expired
		if entry.expiration != nil && time.Now().After(*entry.expiration) {
			c.mu.RUnlock()
			c.Delete(key)
			c.mu.RLock()
			return nil, false
		}

		// Move to front of LRU list
		c.lruList.MoveToFront(element)
		return entry.value, true
	}

	return nil, false
}

// Set adds or updates a value in the cache
func (c *Cache) Set(key string, value interface{}, ignoreTTL bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Create new entry
	var expiration *time.Time
	if !ignoreTTL && c.ttl > 0 {
		exp := time.Now().Add(c.ttl)
		expiration = &exp
	}

	entry := &CacheEntry{
		value:      value,
		expiration: expiration,
	}

	// Check if key exists
	if element, exists := c.items[key]; exists {
		// Update existing entry
		element.Value = entry
		c.lruList.MoveToFront(element)
	} else {
		// Add new entry
		element := c.lruList.PushFront(entry)
		c.items[key] = element

		// Evict if capacity exceeded
		if c.capacity > 0 && c.lruList.Len() > c.capacity {
			c.evict()
		}
	}
}

// Delete removes a value from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, exists := c.items[key]; exists {
		c.lruList.Remove(element)
		delete(c.items, key)
	}
}

// evict removes the least recently used item
func (c *Cache) evict() {
	if element := c.lruList.Back(); element != nil {
		c.lruList.Remove(element)
		key := element.Value.(*CacheEntry).value.(string)
		delete(c.items, key)
	}
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.lruList.Init()
}

// Len returns the number of items in the cache
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lruList.Len()
}
