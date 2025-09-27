package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"
)

// MmapCache provides a memory-mapped cache implementation
type MmapCache struct {
	filePath string
	fileSize int
	data     []byte
	mu       sync.RWMutex
	maxSize  int
	ttl      time.Duration
}

// CacheState represents the entire cache state in memory
type CacheState struct {
	Version   int                    `json:"version"`
	Entries   map[string]*CacheEntry `json:"entries"`
	LRUOrder  []string               `json:"lru_order"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// NewMmapCache creates a new memory-mapped cache
func NewMmapCache(filePath string, maxSize int, ttl int, fileSize int) (*MmapCache, error) {
	mc := &MmapCache{
		filePath: filePath,
		fileSize: fileSize,
		maxSize:  maxSize,
		ttl:      time.Duration(ttl) * time.Second,
	}

	if err := mc.initialize(); err != nil {
		return nil, err
	}

	return mc, nil
}

// initialize sets up the memory-mapped file
func (mc *MmapCache) initialize() error {
	// Ensure file exists with correct size
	file, err := os.OpenFile(mc.filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get current size
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	// Resize if needed
	if stat.Size() != int64(mc.fileSize) {
		if err := file.Truncate(int64(mc.fileSize)); err != nil {
			return err
		}
	}

	// Memory map the file
	mc.data, err = syscall.Mmap(int(file.Fd()), 0, mc.fileSize,
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return err
	}

	// Initialize with empty cache if new file
	if stat.Size() == 0 || mc.data[0] == 0 {
		state := &CacheState{
			Version:   1,
			Entries:   make(map[string]*CacheEntry),
			LRUOrder:  []string{},
			UpdatedAt: time.Now(),
		}
		mc.saveState(state)
	}

	return nil
}

// loadState reads the current cache state from memory
func (mc *MmapCache) loadState() (*CacheState, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Find the end of JSON data (null-terminated)
	endIdx := 0
	for i, b := range mc.data {
		if b == 0 {
			endIdx = i
			break
		}
	}

	if endIdx == 0 {
		// Empty cache
		return &CacheState{
			Version:  1,
			Entries:  make(map[string]*CacheEntry),
			LRUOrder: []string{},
		}, nil
	}

	var state CacheState
	if err := json.Unmarshal(mc.data[:endIdx], &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// saveState writes the cache state to memory
func (mc *MmapCache) saveState(state *CacheState) error {
	state.UpdatedAt = time.Now()

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	if len(data) > mc.fileSize-1 {
		return fmt.Errorf("cache data too large for mapped file")
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Clear the buffer
	for i := range mc.data {
		mc.data[i] = 0
	}

	// Write new data
	copy(mc.data, data)

	// Force sync to disk
	// Note: On macOS, we can use syscall.Sync() or skip explicit sync
	// as the OS will handle it when the memory is unmapped
	// For cross-platform compatibility, we'll skip explicit sync here

	return nil
}

// Get retrieves a value from the cache
func (mc *MmapCache) Get(key string) (interface{}, bool) {
	state, err := mc.loadState()
	if err != nil {
		return nil, false
	}

	entry, exists := state.Entries[key]
	if !exists {
		return nil, false
	}

	// Check TTL
	if time.Since(entry.Timestamp) > mc.ttl {
		// Remove expired entry
		delete(state.Entries, key)
		mc.removeLRUOrder(state, key)
		mc.saveState(state)
		return nil, false
	}

	// Update LRU order - move to front
	mc.removeLRUOrder(state, key)
	state.LRUOrder = append([]string{key}, state.LRUOrder...)

	mc.saveState(state)
	return entry.Value, true
}

// Set adds or updates a value in the cache
func (mc *MmapCache) Set(key string, value interface{}) error {
	state, err := mc.loadState()
	if err != nil {
		return err
	}

	// Check if key exists
	if _, exists := state.Entries[key]; exists {
		// Update existing entry
		mc.removeLRUOrder(state, key)
	} else {
		// Check if we need to evict
		if len(state.Entries) >= mc.maxSize {
			// Evict least recently used
			if len(state.LRUOrder) > 0 {
				lruKey := state.LRUOrder[len(state.LRUOrder)-1]
				delete(state.Entries, lruKey)
				state.LRUOrder = state.LRUOrder[:len(state.LRUOrder)-1]
			}
		}
	}

	// Add new entry
	state.Entries[key] = &CacheEntry{
		Key:       key,
		Value:     value,
		Timestamp: time.Now(),
	}

	// Add to front of LRU order
	state.LRUOrder = append([]string{key}, state.LRUOrder...)

	return mc.saveState(state)
}

// Clear removes all entries from the cache
func (mc *MmapCache) Clear() error {
	state := &CacheState{
		Version:  1,
		Entries:  make(map[string]*CacheEntry),
		LRUOrder: []string{},
	}
	return mc.saveState(state)
}

// Size returns the number of entries
func (mc *MmapCache) Size() int {
	state, err := mc.loadState()
	if err != nil {
		return 0
	}
	return len(state.Entries)
}

// removeLRUOrder removes a key from the LRU order list
func (mc *MmapCache) removeLRUOrder(state *CacheState, key string) {
	newOrder := []string{}
	for _, k := range state.LRUOrder {
		if k != key {
			newOrder = append(newOrder, k)
		}
	}
	state.LRUOrder = newOrder
}

// Close unmaps the memory
func (mc *MmapCache) Close() error {
	if mc.data != nil {
		return syscall.Munmap(mc.data)
	}
	return nil
}