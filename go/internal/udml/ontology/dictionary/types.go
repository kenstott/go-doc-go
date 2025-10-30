package dictionary

import "time"

// DictionaryProvider defines the interface for dictionary lookup services
type DictionaryProvider interface {
	// Lookup returns lexical information for a word
	Lookup(word string) (*LexicalEntry, error)

	// MightContain performs a fast probabilistic check if word might be in dictionary
	// Used to avoid expensive API calls for words definitely not present
	MightContain(word string) bool

	// Name returns the provider name (e.g., "wordnet", "geonames")
	Name() string

	// Priority returns the provider priority (higher = checked first)
	Priority() int
}

// LexicalEntry represents lexical/semantic information about a word
type LexicalEntry struct {
	Word        string                 `json:"word"`
	POS         []string               `json:"pos"`        // Part of speech: ["noun", "verb"]
	Categories  []string               `json:"categories"` // ["place", "medical_condition", "organization"]
	Definitions []string               `json:"definitions"`
	Synonyms    []string               `json:"synonyms"`
	Hypernyms   []string               `json:"hypernyms"` // Parent concepts
	Source      string                 `json:"source"`    // Which provider found it
	IsProper    bool                   `json:"is_proper"` // Proper noun vs common noun
	Metadata    map[string]interface{} `json:"metadata"`
}

// HasPOS checks if entry has a specific part of speech
func (e *LexicalEntry) HasPOS(pos string) bool {
	for _, p := range e.POS {
		if p == pos {
			return true
		}
	}
	return false
}

// HasCategory checks if entry belongs to a specific category
func (e *LexicalEntry) HasCategory(category string) bool {
	for _, c := range e.Categories {
		if c == category {
			return true
		}
	}
	return false
}

// LookupHint provides hints to optimize dictionary lookups
type LookupHint struct {
	PreferredCategories []string // Prefer providers with these categories
	PreferredSources    []string // Prefer specific provider names
	Context             string   // Contextual information for disambiguation
}

// CacheEntry represents a cached dictionary lookup result
type CacheEntry struct {
	Word       string
	Entry      *LexicalEntry // nil if not found
	NotFound   bool          // Explicitly mark as "checked and not found"
	CachedAt   time.Time
	LastAccess time.Time
	HitCount   int64
	TTL        time.Duration
	Source     string
}

// IsExpired checks if cache entry has exceeded its TTL
func (c *CacheEntry) IsExpired() bool {
	return time.Since(c.CachedAt) > c.TTL
}

// IsStale checks if entry should be refreshed (e.g., 80% of TTL elapsed)
func (c *CacheEntry) IsStale() bool {
	return time.Since(c.CachedAt) > (c.TTL * 8 / 10)
}
