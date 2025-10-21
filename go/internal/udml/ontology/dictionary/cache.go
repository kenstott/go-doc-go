package dictionary

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// PersistentLRUCache implements a two-tier LRU cache (memory + disk)
type PersistentLRUCache struct {
	db          *sql.DB
	memCache    map[string]*CacheEntry // Hot cache in memory
	maxMemSize  int
	maxDiskSize int
	defaultTTL  time.Duration
	mu          sync.RWMutex
}

// NewPersistentLRUCache creates a new persistent LRU cache
func NewPersistentLRUCache(dbPath string, maxMemSize, maxDiskSize int, defaultTTL time.Duration) (*PersistentLRUCache, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache database: %w", err)
	}

	// Create schema
	schema := `
	CREATE TABLE IF NOT EXISTS lexical_cache (
		word TEXT PRIMARY KEY,
		entry_json TEXT,
		not_found BOOLEAN,
		cached_at INTEGER,
		last_access INTEGER,
		hit_count INTEGER DEFAULT 0,
		ttl_seconds INTEGER,
		source TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_lru ON lexical_cache(last_access ASC);
	CREATE INDEX IF NOT EXISTS idx_ttl ON lexical_cache(cached_at);
	CREATE INDEX IF NOT EXISTS idx_popular ON lexical_cache(hit_count DESC);
	`

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to create cache schema: %w", err)
	}

	cache := &PersistentLRUCache{
		db:          db,
		memCache:    make(map[string]*CacheEntry),
		maxMemSize:  maxMemSize,
		maxDiskSize: maxDiskSize,
		defaultTTL:  defaultTTL,
	}

	// Warmup: load hot entries into memory
	cache.warmup()

	return cache, nil
}

// Get retrieves a cache entry (memory → disk → nil)
func (c *PersistentLRUCache) Get(word string) *CacheEntry {
	// 1. Check memory cache (hot)
	c.mu.RLock()
	if entry, ok := c.memCache[word]; ok {
		c.mu.RUnlock()
		entry.HitCount++
		entry.LastAccess = time.Now()
		return entry
	}
	c.mu.RUnlock()

	// 2. Check disk cache
	entry := c.loadFromDisk(word)
	if entry == nil {
		return nil
	}

	// 3. Check if expired
	if entry.IsExpired() {
		c.invalidate(word)
		return nil
	}

	// 4. Update access metadata
	c.updateAccess(word, entry)

	// 5. Promote to memory if frequently accessed
	if entry.HitCount > 10 {
		c.promoteToMemory(word, entry)
	}

	return entry
}

// Put adds an entry to the cache (memory + disk)
func (c *PersistentLRUCache) Put(word string, entry *LexicalEntry, notFound bool, ttl time.Duration) {
	if ttl == 0 {
		ttl = c.defaultTTL
	}

	cacheEntry := &CacheEntry{
		Word:       word,
		Entry:      entry,
		NotFound:   notFound,
		CachedAt:   time.Now(),
		LastAccess: time.Now(),
		HitCount:   1,
		TTL:        ttl,
		Source:     "",
	}

	if entry != nil {
		cacheEntry.Source = entry.Source
	}

	// Add to memory cache
	c.addToMemory(word, cacheEntry)

	// Persist to disk
	c.saveToDisk(cacheEntry)

	// Evict if disk cache too large
	c.evictLRU()
}

// loadFromDisk loads a cache entry from SQLite
func (c *PersistentLRUCache) loadFromDisk(word string) *CacheEntry {
	var entryJSON sql.NullString
	var notFound bool
	var cachedAt, lastAccess, ttlSeconds int64
	var hitCount int64
	var source string

	err := c.db.QueryRow(`
		SELECT entry_json, not_found, cached_at, last_access, hit_count, ttl_seconds, source
		FROM lexical_cache
		WHERE word = ?
	`, word).Scan(&entryJSON, &notFound, &cachedAt, &lastAccess, &hitCount, &ttlSeconds, &source)

	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		log.Printf("ERROR: Failed to load cache entry for '%s': %v", word, err)
		return nil
	}

	entry := &CacheEntry{
		Word:       word,
		NotFound:   notFound,
		CachedAt:   time.Unix(cachedAt, 0),
		LastAccess: time.Unix(lastAccess, 0),
		HitCount:   hitCount,
		TTL:        time.Duration(ttlSeconds) * time.Second,
		Source:     source,
	}

	if entryJSON.Valid && entryJSON.String != "" {
		var lexEntry LexicalEntry
		if err := json.Unmarshal([]byte(entryJSON.String), &lexEntry); err != nil {
			log.Printf("ERROR: Failed to unmarshal cache entry for '%s': %v", word, err)
			return nil
		}
		entry.Entry = &lexEntry
	}

	return entry
}

// saveToDisk persists a cache entry to SQLite
func (c *PersistentLRUCache) saveToDisk(entry *CacheEntry) {
	var entryJSON sql.NullString

	if entry.Entry != nil {
		data, err := json.Marshal(entry.Entry)
		if err != nil {
			log.Printf("ERROR: Failed to marshal cache entry for '%s': %v", entry.Word, err)
			return
		}
		entryJSON = sql.NullString{String: string(data), Valid: true}
	}

	_, err := c.db.Exec(`
		INSERT OR REPLACE INTO lexical_cache
		(word, entry_json, not_found, cached_at, last_access, hit_count, ttl_seconds, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, entry.Word, entryJSON, entry.NotFound,
		entry.CachedAt.Unix(), entry.LastAccess.Unix(),
		entry.HitCount, int64(entry.TTL.Seconds()), entry.Source)

	if err != nil {
		log.Printf("ERROR: Failed to save cache entry for '%s': %v", entry.Word, err)
	}
}

// updateAccess updates last access time and hit count
func (c *PersistentLRUCache) updateAccess(word string, entry *CacheEntry) {
	entry.LastAccess = time.Now()
	entry.HitCount++

	// Update in background to avoid blocking
	go func() {
		_, err := c.db.Exec(`
			UPDATE lexical_cache
			SET last_access = ?, hit_count = hit_count + 1
			WHERE word = ?
		`, entry.LastAccess.Unix(), word)

		if err != nil {
			log.Printf("ERROR: Failed to update access for '%s': %v", word, err)
		}
	}()
}

// addToMemory adds entry to in-memory cache with LRU eviction
func (c *PersistentLRUCache) addToMemory(word string, entry *CacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict from memory if full (simple LRU: evict oldest access)
	if len(c.memCache) >= c.maxMemSize {
		var oldestWord string
		var oldestTime time.Time

		for w, e := range c.memCache {
			if oldestWord == "" || e.LastAccess.Before(oldestTime) {
				oldestWord = w
				oldestTime = e.LastAccess
			}
		}

		if oldestWord != "" {
			delete(c.memCache, oldestWord)
		}
	}

	c.memCache[word] = entry
}

// promoteToMemory promotes frequently accessed entry to memory cache
func (c *PersistentLRUCache) promoteToMemory(word string, entry *CacheEntry) {
	c.addToMemory(word, entry)
}

// evictLRU evicts least recently used entries from disk when cache is full
func (c *PersistentLRUCache) evictLRU() {
	var count int
	c.db.QueryRow("SELECT COUNT(*) FROM lexical_cache").Scan(&count)

	if count > c.maxDiskSize {
		toEvict := count - c.maxDiskSize + (c.maxDiskSize / 10) // Evict 10% extra

		_, err := c.db.Exec(`
			DELETE FROM lexical_cache
			WHERE word IN (
				SELECT word FROM lexical_cache
				ORDER BY last_access ASC
				LIMIT ?
			)
		`, toEvict)

		if err != nil {
			log.Printf("ERROR: Failed to evict LRU entries: %v", err)
		}
	}
}

// invalidate removes an entry from cache
func (c *PersistentLRUCache) invalidate(word string) {
	c.mu.Lock()
	delete(c.memCache, word)
	c.mu.Unlock()

	c.db.Exec("DELETE FROM lexical_cache WHERE word = ?", word)
}

// warmup loads hot entries into memory on startup
func (c *PersistentLRUCache) warmup() {
	rows, err := c.db.Query(`
		SELECT word, entry_json, not_found, cached_at, last_access, hit_count, ttl_seconds, source
		FROM lexical_cache
		ORDER BY hit_count DESC
		LIMIT ?
	`, c.maxMemSize/2) // Load 50% of memory cache capacity

	if err != nil {
		log.Printf("ERROR: Failed to warmup cache: %v", err)
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var word, source string
		var entryJSON sql.NullString
		var notFound bool
		var cachedAt, lastAccess, ttlSeconds, hitCount int64

		if err := rows.Scan(&word, &entryJSON, &notFound, &cachedAt, &lastAccess, &hitCount, &ttlSeconds, &source); err != nil {
			continue
		}

		entry := &CacheEntry{
			Word:       word,
			NotFound:   notFound,
			CachedAt:   time.Unix(cachedAt, 0),
			LastAccess: time.Unix(lastAccess, 0),
			HitCount:   hitCount,
			TTL:        time.Duration(ttlSeconds) * time.Second,
			Source:     source,
		}

		if entryJSON.Valid && entryJSON.String != "" {
			var lexEntry LexicalEntry
			if err := json.Unmarshal([]byte(entryJSON.String), &lexEntry); err == nil {
				entry.Entry = &lexEntry
			}
		}

		c.mu.Lock()
		c.memCache[word] = entry
		c.mu.Unlock()

		count++
	}

	log.Printf("Dictionary cache warmed up with %d hot entries", count)
}

// Close closes the database connection
func (c *PersistentLRUCache) Close() error {
	return c.db.Close()
}

// Stats returns cache statistics
type CacheStats struct {
	MemorySize    int
	DiskSize      int
	TotalHits     int64
	MemoryHitRate float64
}

func (c *PersistentLRUCache) Stats() *CacheStats {
	stats := &CacheStats{}

	c.mu.RLock()
	stats.MemorySize = len(c.memCache)
	c.mu.RUnlock()

	var diskSize int
	var totalHits int64
	c.db.QueryRow("SELECT COUNT(*), COALESCE(SUM(hit_count), 0) FROM lexical_cache").Scan(&diskSize, &totalHits)

	stats.DiskSize = diskSize
	stats.TotalHits = totalHits

	return stats
}
