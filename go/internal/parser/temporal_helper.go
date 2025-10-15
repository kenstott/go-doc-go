package parser

import (
	"fmt"
	"strings"

	"github.com/kennethstott/doculyzer-go-conversion/internal/temporal"
)

// ExtractTemporalFromText is a helper function used by all parsers to extract temporal metadata
func ExtractTemporalFromText(text string) map[string]interface{} {
	if text == "" {
		return nil
	}

	temporalData := make(map[string]interface{})
	foundDates := []string{}

	// First check the entire text for temporal content
	if temporal.DetectTemporalType(text) != temporal.TemporalTypeNone {
		foundDates = append(foundDates, text)
		meta := temporal.GenerateTemporalMetadata(text)
		if meta != nil {
			temporalData["date_1"] = meta
		}
	}

	// Also check individual words
	words := strings.Fields(text)
	for i, word := range words {
		// Clean the word of common punctuation
		cleanWord := strings.Trim(word, ".,;:!?()[]{}\"'")

		// Check if this word/phrase is temporal
		tempType := temporal.DetectTemporalType(cleanWord)
		if tempType != temporal.TemporalTypeNone {
			foundDates = append(foundDates, cleanWord)

			// Get temporal metadata for the first few dates found
			if len(foundDates) <= 5 { // Limit to avoid too much processing
				meta := temporal.GenerateTemporalMetadata(cleanWord)
				if meta != nil {
					// Add with a unique key
					temporalData[fmt.Sprintf("date_%d", len(foundDates))] = meta
				}
			}
		}

		// Check for multi-word patterns like "Q1 2024"
		if i < len(words)-1 {
			twoWords := cleanWord + " " + strings.Trim(words[i+1], ".,;:!?()[]{}\"'")
			if temporal.DetectTemporalType(twoWords) != temporal.TemporalTypeNone {
				foundDates = append(foundDates, twoWords)
				if len(foundDates) <= 5 {
					meta := temporal.GenerateTemporalMetadata(twoWords)
					if meta != nil {
						temporalData[fmt.Sprintf("date_%d", len(foundDates))] = meta
					}
				}
			}
		}
	}

	if len(foundDates) > 0 {
		temporalData["temporal_values_found"] = foundDates
		temporalData["temporal_count"] = len(foundDates)
		return temporalData
	}

	return nil
}

// ProcessTemporalContent checks if content contains temporal data and enriches metadata
func ProcessTemporalContent(content string, metadata map[string]interface{}) {
	if content == "" || metadata == nil {
		return
	}

	tempMeta := ExtractTemporalFromText(content)
	if tempMeta != nil {
		for k, v := range tempMeta {
			metadata[k] = v
		}
	}
}