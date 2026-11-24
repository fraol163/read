// Package cache provides a thread-safe, in-memory key-value store with TTL expiration.
package cache

import (
	"errors"
	"sync"
	"time"
)

var (
	// ErrKeyNotFound indicates the requested key does not exist or has expired.
	ErrKeyNotFound = errors.New("key not found or expired")
	// ErrInvalidKey indicates an empty or invalid key was supplied.
	ErrInvalidKey = errors.New("key cannot be empty")
)

type cacheItem struct {
	value      interface{}
	expiration int64 // Unix nanoseconds, 0 means no expiration
}

func (item cacheItem) isExpired(now int64) bool {
	if item.expiration == 0 {
		return false
	}
	return now > item.expiration
}

// Stats captures operational metrics of the cache.
type Stats struct {
	Hits   int64 `json:"hits"`
	Misses int64 `json:"misses"`
	Puts   int64 `json:"puts"`
	Evicts int64 `json:"evicts"`
}

// Cache is a thread-safe in-memory store.
type Cache struct {
	mu         sync.RWMutex
	items      map[string]cacheItem
	defaultTTL time.Duration
	stats      Stats
}

// New creates a new Cache with the given default TTL.
func New(defaultTTL time.Duration) *Cache {
	return &Cache{
		items:      make(map[string]cacheItem),
		defaultTTL: defaultTTL,
	}
}

// Set stores a key-value pair with default TTL.
func (c *Cache) Set(key string, value interface{}) error {
	return c.SetWithTTL(key, value, c.defaultTTL)
}

// SetWithTTL stores a key-value pair with explicit TTL duration.
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) error {
	if key == "" {
		return ErrInvalidKey
	}

	var exp int64
	if ttl > 0 {
		exp = time.Now().Add(ttl).UnixNano()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = cacheItem{
		value:      value,
		expiration: exp,
	}
	c.stats.Puts++
	return nil
}

// Get retrieves an item by key. Returns ErrKeyNotFound if missing or expired.
func (c *Cache) Get(key string) (interface{}, error) {
	if key == "" {
		return nil, ErrInvalidKey
	}

	now := time.Now().UnixNano()

	c.mu.RLock()
	item, exists := c.items[key]
	c.mu.RUnlock()

	if !exists {
		c.mu.Lock()
		c.stats.Misses++
		c.mu.Unlock()
		return nil, ErrKeyNotFound
	}

	if item.isExpired(now) {
		// Lazy eviction
		c.mu.Lock()
		delete(c.items, key)
		c.stats.Evicts++
		c.stats.Misses++
		c.mu.Unlock()
		return nil, ErrKeyNotFound
	}

	c.mu.Lock()
	c.stats.Hits++
	c.mu.Unlock()
	return item.value, nil
}

// Delete removes an item immediately.
func (c *Cache) Delete(key string) bool {
	if key == "" {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.items[key]; exists {
		delete(c.items, key)
		return true
	}
	return false
}

// Len returns the current count of stored items without filtering expired ones.
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// PurgeExpired removes all expired items from the cache and returns count evicted.
func (c *Cache) PurgeExpired() int {
	now := time.Now().UnixNano()

	c.mu.Lock()
	defer c.mu.Unlock()

	evicted := 0
	for k, item := range c.items {
		if item.isExpired(now) {
			delete(c.items, k)
			evicted++
			c.stats.Evicts++
		}
	}
	return evicted
}

// Clear wipes all items from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]cacheItem)
}

// Stats returns a copy of operational statistics.
func (c *Cache) Stats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}
