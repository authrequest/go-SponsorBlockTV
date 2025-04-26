package api

import (
	"sync"
	"time"
)

// cacheEntry represents a cached value with expiration
type cacheEntry struct {
	value      interface{}
	expiration time.Time
}

// Cache implements a thread-safe cache with TTL
type Cache struct {
	entries map[string]cacheEntry
	mu      sync.RWMutex
	ttl     time.Duration
	maxSize int
}

// NewCache creates a new cache with the specified size and TTL
func NewCache(maxSize int, ttl time.Duration) *Cache {
	return &Cache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
	}
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}

	if time.Now().After(entry.expiration) {
		delete(c.entries, key)
		return nil, false
	}

	return entry.value, true
}

// Set stores a value in the cache
func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove expired entries
	for k, entry := range c.entries {
		if time.Now().After(entry.expiration) {
			delete(c.entries, k)
		}
	}

	// Remove oldest entry if cache is full
	if len(c.entries) >= c.maxSize {
		var oldestKey string
		var oldestTime time.Time
		for k, entry := range c.entries {
			if oldestTime.IsZero() || entry.expiration.Before(oldestTime) {
				oldestKey = k
				oldestTime = entry.expiration
			}
		}
		delete(c.entries, oldestKey)
	}

	c.entries[key] = cacheEntry{
		value:      value,
		expiration: time.Now().Add(c.ttl),
	}
}

// Delete removes a value from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

// Clear removes all values from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]cacheEntry)
}
