package dictionary

import (
	"log"
	"sort"
	"strings"
	"sync"
	"time"
)

// UnifiedDictionary orchestrates multiple dictionary providers with caching
type UnifiedDictionary struct {
	providers []DictionaryProvider
	cache     *PersistentLRUCache
	mu        sync.RWMutex
}

// NewUnifiedDictionary creates a new unified dictionary
func NewUnifiedDictionary(cachePath string) (*UnifiedDictionary, error) {
	// Default cache settings
	maxMemSize := 10000                    // 10k hot entries in memory
	maxDiskSize := 1000000                 // 1M entries on disk
	defaultTTL := 30 * 24 * time.Hour      // 30 days default

	cache, err := NewPersistentLRUCache(cachePath, maxMemSize, maxDiskSize, defaultTTL)
	if err != nil {
		return nil, err
	}

	return &UnifiedDictionary{
		providers: []DictionaryProvider{},
		cache:     cache,
	}, nil
}

// AddProvider adds a dictionary provider
func (d *UnifiedDictionary) AddProvider(provider DictionaryProvider) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.providers = append(d.providers, provider)

	// Sort providers by priority (highest first)
	sort.Slice(d.providers, func(i, j int) bool {
		return d.providers[i].Priority() > d.providers[j].Priority()
	})

	log.Printf("Dictionary: Added provider '%s' (priority: %d)", provider.Name(), provider.Priority())
}

// Lookup performs dictionary lookup with caching
func (d *UnifiedDictionary) Lookup(word string) *LexicalEntry {
	word = strings.ToLower(strings.TrimSpace(word))
	if word == "" {
		return nil
	}

	// 1. Check cache (memory → disk)
	if cached := d.cache.Get(word); cached != nil {
		if cached.NotFound {
			return nil // Negative cache: already checked, not found
		}
		return cached.Entry
	}

	// 2. Heuristic: Skip lookups for obvious proper names
	if d.isObviousProperName(word) {
		// Cache negative result to avoid future lookups
		d.cache.Put(word, nil, true, 7*24*time.Hour) // 7 days TTL for negatives
		return nil
	}

	// 3. Query providers in priority order (early termination)
	d.mu.RLock()
	providers := d.providers
	d.mu.RUnlock()

	for _, provider := range providers {
		// Bloom filter check: skip if definitely not present
		if !provider.MightContain(word) {
			continue
		}

		entry, err := provider.Lookup(word)
		if err != nil {
			log.Printf("Dictionary: Provider '%s' error for '%s': %v", provider.Name(), word, err)
			continue
		}

		if entry != nil {
			// Found! Cache positive result
			ttl := d.getTTL(entry)
			d.cache.Put(word, entry, false, ttl)
			return entry
		}
	}

	// 4. Not found in any provider - cache negative result
	d.cache.Put(word, nil, true, 7*24*time.Hour) // 7 days TTL
	return nil
}

// LookupBatch performs batch lookup (more efficient for multiple words)
func (d *UnifiedDictionary) LookupBatch(words []string) map[string]*LexicalEntry {
	results := make(map[string]*LexicalEntry)

	for _, word := range words {
		results[word] = d.Lookup(word)
	}

	return results
}

// isObviousProperName uses heuristics to detect obvious proper names
// Returns true if word is likely a proper name (skip dictionary lookup)
func (d *UnifiedDictionary) isObviousProperName(word string) bool {
	// Single capitalized word (likely first/last name)
	// Examples: "John", "Smith", "Mary"
	if len(word) > 0 && word[0] >= 'A' && word[0] <= 'Z' {
		// Check if rest is lowercase
		allLower := true
		for i := 1; i < len(word); i++ {
			if word[i] < 'a' || word[i] > 'z' {
				allLower = false
				break
			}
		}

		// Single capitalized word with 3-15 chars = likely proper name
		if allLower && len(word) >= 3 && len(word) <= 15 {
			// But exclude common words that might be capitalized
			commonCapitalized := map[string]bool{
				"the": true, "a": true, "an": true,
				"this": true, "that": true, "these": true, "those": true,
			}
			if !commonCapitalized[strings.ToLower(word)] {
				return true // Likely proper name
			}
		}
	}

	return false
}

// getTTL returns appropriate TTL based on entry source
func (d *UnifiedDictionary) getTTL(entry *LexicalEntry) time.Duration {
	if entry == nil {
		return 7 * 24 * time.Hour // Negative cache: 7 days
	}

	switch entry.Source {
	case "inmemory", "wordnet", "geonames":
		return 90 * 24 * time.Hour // Local data: 90 days (stable)
	case "wikidata", "dbpedia":
		return 30 * 24 * time.Hour // API data: 30 days (may change)
	case "umls", "snomed":
		return 180 * 24 * time.Hour // Medical: 6 months (very stable)
	default:
		return 30 * 24 * time.Hour // Default: 30 days
	}
}

// Stats returns dictionary statistics
func (d *UnifiedDictionary) Stats() *CacheStats {
	return d.cache.Stats()
}

// Close closes the dictionary and cache
func (d *UnifiedDictionary) Close() error {
	return d.cache.Close()
}
