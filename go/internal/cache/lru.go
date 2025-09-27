package cache

import (
	"container/list"
	"sync"
	"time"
)

// CacheEntry represents a single cache entry with value and timestamp
type CacheEntry struct {
	Key       string
	Value     interface{}
	Timestamp time.Time
}

// LRUCache is a thread-safe LRU cache with TTL support
type LRUCache struct {
	maxSize int
	ttl     time.Duration
	cache   map[string]*list.Element
	lru     *list.List
	mu      sync.RWMutex
}

// NewLRUCache creates a new LRU cache instance
func NewLRUCache(maxSize int, ttlSeconds int) *LRUCache {
	return &LRUCache{
		maxSize: maxSize,
		ttl:     time.Duration(ttlSeconds) * time.Second,
		cache:   make(map[string]*list.Element),
		lru:     list.New(),
	}
}

// Get retrieves a value from the cache
func (c *LRUCache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	element, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	entry := element.Value.(*CacheEntry)

	// Check if entry is expired
	if time.Since(entry.Timestamp) > c.ttl {
		// Remove expired entry
		c.lru.Remove(element)
		delete(c.cache, key)
		return nil, false
	}

	// Move to front (most recently used)
	c.lru.MoveToFront(element)
	return entry.Value, true
}

// Set adds or updates a value in the cache
func (c *LRUCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if key already exists
	if element, exists := c.cache[key]; exists {
		// Update existing entry
		c.lru.MoveToFront(element)
		entry := element.Value.(*CacheEntry)
		entry.Value = value
		entry.Timestamp = time.Now()
		return
	}

	// Evict least recently used if cache is full
	if c.lru.Len() >= c.maxSize {
		oldest := c.lru.Back()
		if oldest != nil {
			c.lru.Remove(oldest)
			oldEntry := oldest.Value.(*CacheEntry)
			delete(c.cache, oldEntry.Key)
		}
	}

	// Add new entry
	entry := &CacheEntry{
		Key:       key,
		Value:     value,
		Timestamp: time.Now(),
	}
	element := c.lru.PushFront(entry)
	c.cache[key] = element
}

// Clear removes all entries from the cache
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*list.Element)
	c.lru.Init()
}

// Size returns the current number of items in the cache
func (c *LRUCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lru.Len()
}

// CleanupExpired removes all expired entries from the cache
func (c *LRUCache) CleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	var next *list.Element
	for e := c.lru.Back(); e != nil; e = next {
		next = e.Prev()
		entry := e.Value.(*CacheEntry)
		if now.Sub(entry.Timestamp) > c.ttl {
			c.lru.Remove(e)
			delete(c.cache, entry.Key)
		}
	}
}

// Export returns all cache entries for persistence
func (c *LRUCache) Export() map[string]*CacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	exported := make(map[string]*CacheEntry)
	for e := c.lru.Back(); e != nil; e = e.Prev() {
		entry := e.Value.(*CacheEntry)
		// Only export non-expired entries
		if time.Since(entry.Timestamp) <= c.ttl {
			exported[entry.Key] = entry
		}
	}
	return exported
}

// Import loads entries into the cache
func (c *LRUCache) Import(entries map[string]*CacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clear existing cache
	c.cache = make(map[string]*list.Element)
	c.lru.Init()

	// Import entries sorted by timestamp (oldest first)
	type kv struct {
		key   string
		entry *CacheEntry
	}

	var sorted []kv
	for k, v := range entries {
		sorted = append(sorted, kv{k, v})
	}

	// Sort by timestamp
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].entry.Timestamp.After(sorted[j].entry.Timestamp) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Add to cache
	now := time.Now()
	for _, item := range sorted {
		// Only import non-expired entries
		if now.Sub(item.entry.Timestamp) <= c.ttl {
			element := c.lru.PushFront(item.entry)
			c.cache[item.key] = element
		}
	}
}

// GetOrder returns the keys in LRU order (most recent first)
func (c *LRUCache) GetOrder() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var order []string
	for e := c.lru.Front(); e != nil; e = e.Next() {
		entry := e.Value.(*CacheEntry)
		order = append(order, entry.Key)
	}
	return order
}