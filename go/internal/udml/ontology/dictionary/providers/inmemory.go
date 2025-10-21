package providers

import (
	"strings"

	"github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology/dictionary"
)

// InMemoryProvider provides fast dictionary lookups from in-memory word lists
type InMemoryProvider struct {
	name         string
	priority     int
	commonNouns  map[string]bool
	commonVerbs  map[string]bool
	placeTerms   map[string]bool
	temporalTerms map[string]bool
	uiTerms      map[string]bool
	organizations map[string]bool
}

// NewInMemoryProvider creates a new in-memory dictionary provider
func NewInMemoryProvider() *InMemoryProvider {
	return &InMemoryProvider{
		name:     "inmemory",
		priority: 100, // Highest priority (checked first)
		commonNouns: map[string]bool{
			// Geographic
			"national": true, "park": true, "capitol": true, "reef": true,
			"mountain": true, "river": true, "ocean": true, "lake": true,
			"city": true, "state": true, "country": true, "island": true,
			"forest": true, "valley": true, "beach": true, "desert": true,

			// Academic/Publishing
			"academic": true, "university": true, "press": true, "publishing": true,
			"publisher": true, "journal": true, "review": true, "college": true,
			"institute": true, "school": true, "library": true,

			// Abstract concepts
			"concept": true, "theory": true, "principle": true, "method": true,
			"approach": true, "system": true, "process": true, "model": true,
			"framework": true, "strategy": true,

			// Time
			"century": true, "decade": true, "year": true, "era": true,
			"period": true, "age": true, "epoch": true,

			// Document structure
			"chapter": true, "section": true, "appendix": true, "index": true,
			"table": true, "figure": true, "reference": true, "page": true,

			// Business/Organization terms
			"company": true, "corporation": true, "firm": true, "business": true,
			"enterprise": true, "organization": true, "association": true,
			"society": true, "foundation": true, "trust": true,

			// General nouns
			"world": true, "health": true, "production": true, "silver": true,
			"mining": true, "medicine": true, "education": true, "learning": true,
			"science": true, "research": true, "development": true,
		},
		commonVerbs: map[string]bool{
			"reading": true, "writing": true, "learning": true, "teaching": true,
			"thinking": true, "understanding": true, "analyzing": true,
			"processing": true, "developing": true, "creating": true,
		},
		placeTerms: map[string]bool{
			// Universities/Colleges
			"bloomsbury": true, "oxford": true, "cambridge": true, "harvard": true,
			"stanford": true, "yale": true, "princeton": true, "mit": true,

			// Cities
			"london": true, "paris": true, "tokyo": true, "beijing": true,
			"newyork": true, "losangeles": true, "chicago": true,

			// Regions
			"europe": true, "asia": true, "africa": true, "america": true,
		},
		temporalTerms: map[string]bool{
			"today": true, "tomorrow": true, "yesterday": true,
			"morning": true, "evening": true, "night": true,
			"january": true, "february": true, "march": true, "april": true,
			"may": true, "june": true, "july": true, "august": true,
			"september": true, "october": true, "november": true, "december": true,
		},
		uiTerms: map[string]bool{
			"donate": true, "create": true, "login": true, "logout": true,
			"signup": true, "subscribe": true, "download": true, "upload": true,
			"search": true, "filter": true, "sort": true, "edit": true,
			"delete": true, "save": true, "cancel": true, "submit": true,
			"print": true, "export": true, "import": true, "share": true,
		},
		organizations: map[string]bool{
			// Known organizations/publishers
			"who": true, "unesco": true, "nato": true, "un": true,
			"springer": true, "elsevier": true, "wiley": true, "sage": true,
		},
	}
}

// Lookup performs dictionary lookup
func (p *InMemoryProvider) Lookup(word string) (*dictionary.LexicalEntry, error) {
	word = strings.ToLower(word)

	entry := &dictionary.LexicalEntry{
		Word:       word,
		POS:        []string{},
		Categories: []string{},
		Source:     p.name,
	}

	found := false

	// Check common nouns
	if p.commonNouns[word] {
		entry.POS = append(entry.POS, "noun")
		entry.Categories = append(entry.Categories, "common_noun")
		found = true
	}

	// Check verbs
	if p.commonVerbs[word] {
		entry.POS = append(entry.POS, "verb")
		found = true
	}

	// Check place terms
	if p.placeTerms[word] {
		entry.POS = append(entry.POS, "proper_noun")
		entry.Categories = append(entry.Categories, "place")
		entry.IsProper = true
		found = true
	}

	// Check temporal terms
	if p.temporalTerms[word] {
		entry.POS = append(entry.POS, "noun")
		entry.Categories = append(entry.Categories, "temporal")
		found = true
	}

	// Check UI terms
	if p.uiTerms[word] {
		entry.POS = append(entry.POS, "verb")
		entry.Categories = append(entry.Categories, "ui_action")
		found = true
	}

	// Check organizations
	if p.organizations[word] {
		entry.POS = append(entry.POS, "proper_noun")
		entry.Categories = append(entry.Categories, "organization")
		entry.IsProper = true
		found = true
	}

	if !found {
		return nil, nil // Not found in this provider
	}

	return entry, nil
}

// MightContain performs fast check if word might be in dictionary
func (p *InMemoryProvider) MightContain(word string) bool {
	word = strings.ToLower(word)

	// Simple Bloom filter: check if in any of our maps
	return p.commonNouns[word] ||
		p.commonVerbs[word] ||
		p.placeTerms[word] ||
		p.temporalTerms[word] ||
		p.uiTerms[word] ||
		p.organizations[word]
}

// Name returns provider name
func (p *InMemoryProvider) Name() string {
	return p.name
}

// Priority returns provider priority
func (p *InMemoryProvider) Priority() int {
	return p.priority
}
