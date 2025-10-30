package ontology

import (
	"strings"
	"unicode"
)

// WordNetValidator provides filtering based on linguistic patterns and word semantics
// This is a simple implementation - can be extended with actual WordNet integration
type WordNetValidator struct {
	// Common nouns that are definitely not person names
	commonNouns map[string]bool
	// Common verbs in gerund form (ending in -ing)
	commonVerbs map[string]bool
	// Geographic/place terms
	placeTerms map[string]bool
	// Temporal terms
	temporalTerms map[string]bool
	// Navigation/UI terms
	uiTerms map[string]bool
}

// NewWordNetValidator creates a new WordNet-based validator
func NewWordNetValidator() *WordNetValidator {
	return &WordNetValidator{
		commonNouns: map[string]bool{
			// Geographic
			"national": true, "park": true, "capitol": true, "reef": true,
			"mountain": true, "river": true, "ocean": true, "lake": true,
			"city": true, "state": true, "country": true, "island": true,

			// Academic/Publishing
			"academic": true, "university": true, "press": true, "publishing": true,
			"publisher": true, "journal": true, "review": true,

			// Abstract concepts
			"concept": true, "theory": true, "principle": true, "method": true,
			"approach": true, "system": true, "process": true,

			// Time
			"century": true, "decade": true, "year": true, "era": true,
			"period": true, "age": true,

			// Document structure
			"chapter": true, "section": true, "appendix": true, "index": true,
			"table": true, "figure": true, "reference": true,
		},
		commonVerbs: map[string]bool{
			"reading": true, "writing": true, "learning": true, "teaching": true,
			"thinking": true, "understanding": true, "analyzing": true,
		},
		placeTerms: map[string]bool{
			"bloomsbury": true, "oxford": true, "cambridge": true, "harvard": true,
			"stanford": true, "yale": true, "princeton": true,
		},
		temporalTerms: map[string]bool{
			"today": true, "tomorrow": true, "yesterday": true,
			"morning": true, "evening": true, "night": true,
		},
		uiTerms: map[string]bool{
			"donate": true, "create": true, "login": true, "logout": true,
			"signup": true, "subscribe": true, "download": true, "upload": true,
			"search": true, "filter": true, "sort": true, "edit": true,
			"delete": true, "save": true, "cancel": true, "submit": true,
		},
	}
}

// CheckWordNetFilter validates text against a WordNet filter configuration
// Returns true if text PASSES the filter (should be accepted), false if REJECTED
func (v *WordNetValidator) CheckWordNetFilter(text string, filter *WordNetFilter) bool {
	if filter == nil {
		return true // No filter = accept
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return false // Empty text = reject
	}

	// Count how many words are in WordNet vs unknown
	knownWords := 0
	unknownWords := 0
	wordPOS := make([]string, len(words))

	for i, word := range words {
		wordLower := strings.ToLower(word)
		pos := v.getPartOfSpeech(wordLower)
		wordPOS[i] = pos

		if pos == "unknown" {
			unknownWords++
		} else {
			knownWords++
		}
	}

	totalWords := len(words)
	knownRatio := float64(knownWords) / float64(totalWords)

	// Check: require at least one unknown word (proper noun not in WordNet)
	if filter.RequireUnknownWords && unknownWords == 0 {
		return false // Reject - no unknown words found
	}

	// Check: max known words ratio
	if filter.MaxKnownWordsRatio > 0 && knownRatio > filter.MaxKnownWordsRatio {
		return false // Reject - too many known words
	}

	// Check: reject if ALL words match certain POS
	if len(filter.RejectIfAllPOS) > 0 {
		allMatch := true
		for _, pos := range wordPOS {
			matchesAny := false
			for _, rejectPOS := range filter.RejectIfAllPOS {
				if pos == rejectPOS {
					matchesAny = true
					break
				}
			}
			if !matchesAny {
				allMatch = false
				break
			}
		}
		if allMatch {
			return false // Reject - all words match rejected POS
		}
	}

	// Check: reject specific POS combinations
	if len(filter.RejectPOSCombinations) > 0 {
		for _, combination := range filter.RejectPOSCombinations {
			if len(combination) != len(wordPOS) {
				continue // Different length, skip
			}
			matches := true
			for i, expectedPOS := range combination {
				if wordPOS[i] != expectedPOS {
					matches = false
					break
				}
			}
			if matches {
				return false // Reject - matches forbidden POS combination
			}
		}
	}

	return true // Passed all filters
}

// getPartOfSpeech returns the part-of-speech for a word (from our simple WordNet)
func (v *WordNetValidator) getPartOfSpeech(word string) string {
	if v.commonNouns[word] {
		return "noun"
	}
	if v.commonVerbs[word] {
		return "verb"
	}
	if v.uiTerms[word] {
		return "verb" // UI terms are mostly verbs
	}
	if v.placeTerms[word] {
		return "proper_noun" // Place names
	}
	if v.temporalTerms[word] {
		return "noun" // Time nouns
	}
	return "unknown" // Not in our WordNet - likely a proper noun or rare word
}

// IsProbablyNotPerson checks if a string is unlikely to be a person name
// Returns true if the string should be REJECTED (not a person)
func (v *WordNetValidator) IsProbablyNotPerson(text string) bool {
	// Quick structural checks
	if v.hasStructuralIssues(text) {
		return true // Reject
	}

	// Check individual words
	words := strings.Fields(text)
	if len(words) == 0 {
		return true // Empty - reject
	}

	// Single word checks
	if len(words) == 1 {
		word := strings.ToLower(words[0])
		// Single common nouns are not person names
		if v.commonNouns[word] || v.uiTerms[word] || v.temporalTerms[word] {
			return true // Reject
		}
	}

	// Two-word checks
	if len(words) == 2 {
		word1 := strings.ToLower(words[0])
		word2 := strings.ToLower(words[1])

		// "National Park", "Capitol Reef", etc.
		if (v.commonNouns[word1] || v.placeTerms[word1]) &&
			(v.commonNouns[word2] || v.placeTerms[word2]) {
			return true // Reject - compound noun phrase
		}

		// "Bloomsbury Academic" - publisher name
		if v.placeTerms[word1] && v.commonNouns[word2] {
			return true // Reject - place + noun (likely org/publisher)
		}

		// UI commands: "Create Account", "Download PDF"
		if v.uiTerms[word1] {
			return true // Reject - starts with UI verb
		}
	}

	// Three+ word checks
	if len(words) >= 3 {
		// "Some Things You Should Know" - sentence fragment
		// Check if contains common sentence words
		lowerText := strings.ToLower(text)
		if strings.Contains(lowerText, " you ") ||
			strings.Contains(lowerText, " should ") ||
			strings.Contains(lowerText, " what ") ||
			strings.Contains(lowerText, " how ") ||
			strings.Contains(lowerText, " when ") ||
			strings.Contains(lowerText, " where ") {
			return true // Reject - contains question/instruction words
		}
	}

	return false // Looks OK - might be a person
}

// hasStructuralIssues checks for obvious non-name patterns
func (v *WordNetValidator) hasStructuralIssues(text string) bool {
	// Contains newlines or tabs
	if strings.ContainsAny(text, "\n\r\t") {
		return true
	}

	// All uppercase (UI commands like "DONATE", "CREATE")
	if text == strings.ToUpper(text) && text != "" {
		hasLower := false
		for _, r := range text {
			if unicode.IsLower(r) {
				hasLower = true
				break
			}
		}
		if !hasLower && len(text) > 0 {
			return true
		}
	}

	// Contains special patterns that indicate non-names
	if strings.Contains(text, "\n\t") || strings.Contains(text, "\t\t") {
		return true
	}

	// Starts or ends with whitespace
	if text != strings.TrimSpace(text) {
		return true
	}

	return false
}

// GetWordNetInfo returns semantic information about a word (placeholder for full WordNet)
// This could be expanded to use actual WordNet database
func (v *WordNetValidator) GetWordNetInfo(word string) map[string]interface{} {
	word = strings.ToLower(word)
	info := make(map[string]interface{})

	if v.commonNouns[word] {
		info["pos"] = "noun"
		info["is_common_noun"] = true
	}
	if v.commonVerbs[word] {
		info["pos"] = "verb"
		info["is_verb"] = true
	}
	if v.placeTerms[word] {
		info["category"] = "place"
		info["likely_proper_noun"] = true
	}
	if v.uiTerms[word] {
		info["category"] = "ui_action"
		info["is_command"] = true
	}

	return info
}
