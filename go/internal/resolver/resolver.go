package resolver

import (
	"fmt"
	"sync"
)

// DefaultContentResolver implements ContentResolver using a map of parsers
type DefaultContentResolver struct {
	parsers map[string]ParserResolver
	cache   map[string]string // Cache for resolved content (key: selector+source+text flag)
	mu      sync.RWMutex      // Mutex for thread-safe cache access
}

// NewContentResolver creates a new content resolver with the given parsers
func NewContentResolver(parsers map[string]ParserResolver) *DefaultContentResolver {
	return &DefaultContentResolver{
		parsers: parsers,
		cache:   make(map[string]string),
	}
}

// ResolveContent resolves element content using the appropriate parser
func (r *DefaultContentResolver) ResolveContent(contentLocation map[string]interface{}, text bool) (string, error) {
	if contentLocation == nil {
		return "", nil
	}

	// Create cache key from content_location
	cacheKey := r.makeCacheKey(contentLocation, text)

	// Check cache first
	r.mu.RLock()
	if cached, ok := r.cache[cacheKey]; ok {
		r.mu.RUnlock()
		return cached, nil
	}
	r.mu.RUnlock()

	// Find a parser that supports this location
	var selectedParser ParserResolver
	for _, parser := range r.parsers {
		if parser.SupportsLocation(contentLocation) {
			selectedParser = parser
			break
		}
	}

	if selectedParser == nil {
		// No parser found - this is common for web content where original HTML isn't cached
		// Silently return error to fall back to direct element content
		return "", fmt.Errorf("no parser supports this content location")
	}

	// Resolve using the appropriate method
	var result string
	var err error
	if text {
		result, err = selectedParser.ResolveElementText(contentLocation, "")
	} else {
		result, err = selectedParser.ResolveElementContent(contentLocation, "")
	}

	if err != nil {
		return "", err
	}

	// Cache the result
	r.mu.Lock()
	r.cache[cacheKey] = result
	r.mu.Unlock()

	return result, nil
}

// makeCacheKey creates a unique cache key from content_location
func (r *DefaultContentResolver) makeCacheKey(contentLocation map[string]interface{}, text bool) string {
	source := ""
	locator := ""

	if s, ok := contentLocation["source"].(string); ok {
		source = s
	}

	// HTML uses "selector", XML uses "path"
	if sel, ok := contentLocation["selector"].(string); ok {
		locator = sel
	} else if path, ok := contentLocation["path"].(string); ok {
		locator = path
	}

	textFlag := "html"
	if text {
		textFlag = "text"
	}

	return fmt.Sprintf("%s|%s|%s", source, locator, textFlag)
}
