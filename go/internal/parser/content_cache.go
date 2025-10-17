package parser

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ContentCacheConfig holds configuration for the disk-based content cache
type ContentCacheConfig struct {
	CacheDir       string // Directory to store cached content
	MaxCacheSizeMB int64  // Maximum total cache size in MB (default: 500MB)
	MaxItemSizeMB  int64  // Maximum size for a single cached item in MB (default: 10MB)
	Enabled        bool   // Whether disk caching is enabled
}

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() ContentCacheConfig {
	homeDir, _ := os.UserHomeDir()
	return ContentCacheConfig{
		CacheDir:       filepath.Join(homeDir, ".cache", "go-doc-go", "content"),
		MaxCacheSizeMB: 500,
		MaxItemSizeMB:  10,
		Enabled:        true,
	}
}

// ContentCache provides disk-based LRU caching for HTML/web content
type ContentCache struct {
	config ContentCacheConfig
	mu     sync.RWMutex
}

// cacheEntry represents metadata for a cached item
type cacheEntry struct {
	Key          string
	FilePath     string
	Size         int64
	LastAccessed time.Time
}

// NewContentCache creates a new disk-based content cache
func NewContentCache(config ContentCacheConfig) (*ContentCache, error) {
	if !config.Enabled {
		return &ContentCache{config: config}, nil
	}

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(config.CacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	cache := &ContentCache{
		config: config,
	}

	// Clean up old cache entries if we exceed size limit
	if err := cache.enforceMaxSize(); err != nil {
		// Log warning but don't fail - cache can still work
		fmt.Fprintf(os.Stderr, "Warning: failed to enforce cache size limit: %v\n", err)
	}

	return cache, nil
}

// Get retrieves content from cache if it exists
func (c *ContentCache) Get(key string) (string, bool) {
	if !c.config.Enabled {
		return "", false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	filePath := c.keyToFilePath(key)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", false
	}

	// Read content
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", false
	}

	// Update access time
	go c.touchFile(filePath)

	return string(data), true
}

// Put stores content in cache
func (c *ContentCache) Put(key string, content string) error {
	if !c.config.Enabled {
		return nil
	}

	// Check if content exceeds max item size
	contentSize := int64(len(content))
	maxBytes := c.config.MaxItemSizeMB * 1024 * 1024
	if contentSize > maxBytes {
		// Don't cache items that are too large
		return fmt.Errorf("content size %d MB exceeds max item size %d MB",
			contentSize/(1024*1024), c.config.MaxItemSizeMB)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	filePath := c.keyToFilePath(key)

	// Create parent directory if needed
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache subdirectory: %w", err)
	}

	// Write content to file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	// Enforce max cache size (async to avoid blocking)
	go c.enforceMaxSize()

	return nil
}

// keyToFilePath converts a cache key (URL) to a file path
func (c *ContentCache) keyToFilePath(key string) string {
	// Hash the key to create a safe filename
	hash := sha256.Sum256([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	// Use first 2 chars as subdirectory for better filesystem performance
	subdir := hashStr[:2]
	filename := hashStr[2:] + ".html"

	return filepath.Join(c.config.CacheDir, subdir, filename)
}

// touchFile updates the access time of a cached file
func (c *ContentCache) touchFile(filePath string) {
	now := time.Now()
	os.Chtimes(filePath, now, now)
}

// enforceMaxSize removes least recently used items until cache is under size limit
func (c *ContentCache) enforceMaxSize() error {
	if !c.config.Enabled {
		return nil
	}

	// Get all cache entries with their metadata
	entries, totalSize, err := c.getCacheEntries()
	if err != nil {
		return err
	}

	maxBytes := c.config.MaxCacheSizeMB * 1024 * 1024

	// If we're under the limit, nothing to do
	if totalSize <= maxBytes {
		return nil
	}

	// Sort entries by last access time (oldest first)
	// We'll remove oldest entries until we're under the limit
	sortedEntries := c.sortByAccessTime(entries)

	// Remove entries until we're under the limit
	bytesToRemove := totalSize - maxBytes
	var bytesRemoved int64

	for _, entry := range sortedEntries {
		if bytesRemoved >= bytesToRemove {
			break
		}

		if err := os.Remove(entry.FilePath); err != nil {
			// Log but continue - best effort cleanup
			fmt.Fprintf(os.Stderr, "Warning: failed to remove cache file %s: %v\n", entry.FilePath, err)
			continue
		}

		bytesRemoved += entry.Size
	}

	return nil
}

// getCacheEntries returns all cache entries with metadata
func (c *ContentCache) getCacheEntries() ([]cacheEntry, int64, error) {
	var entries []cacheEntry
	var totalSize int64

	err := filepath.Walk(c.config.CacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip non-HTML files
		if filepath.Ext(path) != ".html" {
			return nil
		}

		entry := cacheEntry{
			FilePath:     path,
			Size:         info.Size(),
			LastAccessed: info.ModTime(),
		}

		entries = append(entries, entry)
		totalSize += info.Size()

		return nil
	})

	if err != nil {
		return nil, 0, err
	}

	return entries, totalSize, nil
}

// sortByAccessTime sorts cache entries by last access time (oldest first)
func (c *ContentCache) sortByAccessTime(entries []cacheEntry) []cacheEntry {
	// Simple bubble sort - good enough for cache management
	sorted := make([]cacheEntry, len(entries))
	copy(sorted, entries)

	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].LastAccessed.After(sorted[j+1].LastAccessed) {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	return sorted
}

// Clear removes all cached content
func (c *ContentCache) Clear() error {
	if !c.config.Enabled {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	return os.RemoveAll(c.config.CacheDir)
}

// GetStats returns cache statistics
func (c *ContentCache) GetStats() (int, int64, error) {
	if !c.config.Enabled {
		return 0, 0, nil
	}

	entries, totalSize, err := c.getCacheEntries()
	if err != nil {
		return 0, 0, err
	}

	return len(entries), totalSize, nil
}
