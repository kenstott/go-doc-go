package temporal

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
)

// NormalizeTemporal applies light normalization to temporal values for better embeddings
func NormalizeTemporal(value string, temporalType TemporalType) string {
	switch temporalType {
	case TemporalTypeDate:
		return normalizeDate(value)
	case TemporalTypeDateTime:
		return normalizeDateTime(value)
	case TemporalTypeTime:
		return normalizeTime(value)
	case TemporalTypeTimeRange:
		return normalizeTimeRange(value)
	default:
		return value
	}
}

// normalizeDate normalizes date to 'Jan 1, 2021' format
func normalizeDate(dateStr string) string {
	parsed, err := parseDate(dateStr)
	if err != nil {
		return dateStr
	}

	// Format as "Jan 1, 2021"
	return parsed.Format("Jan 2, 2006")
}

// normalizeDateTime normalizes datetime to 'Jan 1, 2021 2:30 PM' format
func normalizeDateTime(datetimeStr string) string {
	parsed, err := parseDate(datetimeStr)
	if err != nil {
		return datetimeStr
	}

	// Format date and time parts
	datePart := parsed.Format("Jan 2, 2006")
	timePart := parsed.Format("3:04 PM")
	return fmt.Sprintf("%s %s", datePart, timePart)
}

// normalizeTime normalizes time to '2:30 PM' format
func normalizeTime(timeStr string) string {
	// Handle special cases
	lower := strings.ToLower(timeStr)
	if lower == "noon" {
		return "12:00 PM"
	} else if lower == "midnight" {
		return "12:00 AM"
	}

	// Try to parse as time
	t := parseTimeString(timeStr)
	if t == nil {
		// Try parsing as full date and extract time
		if dt, err := parseDate(timeStr); err == nil {
			t = &dt
		} else {
			return timeStr
		}
	}

	return t.Format("3:04 PM")
}

// normalizeTimeRange normalizes time range to '2:00 PM - 4:00 PM' format
func normalizeTimeRange(rangeStr string) string {
	// Split on various separators
	separatorPattern := regexp.MustCompile(`\s*[-–—to]+\s*`)
	parts := separatorPattern.Split(rangeStr, 2)

	if len(parts) != 2 {
		return rangeStr
	}

	start := strings.TrimSpace(parts[0])
	end := strings.TrimSpace(parts[1])

	// Normalize each part
	startNorm := normalizeTime(start)
	endNorm := normalizeTime(end)

	return fmt.Sprintf("%s - %s", startNorm, endNorm)
}

// CreateTemporalValue creates a complete temporal_value object with precomputed parts
func CreateTemporalValue(value string, temporalType TemporalType) map[string]interface{} {
	temporalValue := map[string]interface{}{
		"type":       strings.ToLower(temporalTypeName(temporalType)),
		"original":   value,
		"normalized": NormalizeTemporal(value, temporalType),
	}

	switch temporalType {
	case TemporalTypeDate:
		addDateParts(temporalValue, value)
	case TemporalTypeDateTime:
		addDateTimeParts(temporalValue, value)
	case TemporalTypeTime:
		addTimeParts(temporalValue, value)
	case TemporalTypeTimeRange:
		addTimeRangeParts(temporalValue, value)
	}

	return temporalValue
}

// addDateParts adds precomputed date parts to temporal_value
func addDateParts(temporalValue map[string]interface{}, dateStr string) {
	parsed, err := parseDate(dateStr)
	if err != nil {
		log.Printf("Failed to parse date parts for %s: %v", dateStr, err)
		return
	}

	temporalValue["iso_format"] = parsed.Format("2006-01-02")

	parts := map[string]interface{}{
		"year":         parsed.Year(),
		"month":        int(parsed.Month()),
		"month_name":   parsed.Format("January"),
		"month_abbr":   parsed.Format("Jan"),
		"day":          parsed.Day(),
		"day_name":     parsed.Format("Monday"),
		"day_abbr":     parsed.Format("Mon"),
		"quarter":      (int(parsed.Month())-1)/3 + 1,
		"quarter_name": fmt.Sprintf("Q%d", (int(parsed.Month())-1)/3+1),
		"week_of_year": func() int { _, week := parsed.ISOWeek(); return week }(),
		"day_of_year":  parsed.YearDay(),
		"decade":       fmt.Sprintf("%ds", (parsed.Year()/10)*10),
	}

	// Add season
	month := int(parsed.Month())
	switch {
	case month == 12 || month <= 2:
		parts["season"] = "Winter"
	case month >= 3 && month <= 5:
		parts["season"] = "Spring"
	case month >= 6 && month <= 8:
		parts["season"] = "Summer"
	default:
		parts["season"] = "Fall"
	}

	temporalValue["parts"] = parts
}

// addDateTimeParts adds precomputed datetime parts to temporal_value
func addDateTimeParts(temporalValue map[string]interface{}, datetimeStr string) {
	parsed, err := parseDate(datetimeStr)
	if err != nil {
		log.Printf("Failed to parse datetime parts for %s: %v", datetimeStr, err)
		return
	}

	temporalValue["iso_format"] = parsed.Format(time.RFC3339)

	// Start with date parts
	addDateParts(temporalValue, datetimeStr)

	// Add time components to existing parts
	if parts, ok := temporalValue["parts"].(map[string]interface{}); ok {
		parts["hour"] = parsed.Hour()
		parts["minute"] = parsed.Minute()
		parts["second"] = parsed.Second()
		parts["hour_12"] = parsed.Hour() % 12
		if parts["hour_12"] == 0 {
			parts["hour_12"] = 12
		}
		parts["am_pm"] = parsed.Format("PM")
		parts["time_of_day"] = getTimeOfDay(parsed.Hour())
		parts["business_hours"] = parsed.Hour() >= 9 && parsed.Hour() < 17
	}
}

// addTimeParts adds precomputed time parts to temporal_value
func addTimeParts(temporalValue map[string]interface{}, timeStr string) {
	// Parse as time
	t := parseTimeString(timeStr)
	if t == nil {
		// Try parsing as full date and extract time
		if dt, err := parseDate(timeStr); err == nil {
			t = &dt
		} else {
			log.Printf("Failed to parse time parts for %s", timeStr)
			return
		}
	}

	temporalValue["iso_format"] = t.Format("15:04:05")

	parts := map[string]interface{}{
		"hour":   t.Hour(),
		"minute": t.Minute(),
		"second": t.Second(),
		"hour_12": t.Hour() % 12,
		"am_pm":  t.Format("PM"),
		"time_of_day": getTimeOfDay(t.Hour()),
		"business_hours": t.Hour() >= 9 && t.Hour() < 17,
	}

	if parts["hour_12"] == 0 {
		parts["hour_12"] = 12
	}

	temporalValue["parts"] = parts
}

// addTimeRangeParts adds precomputed time range parts to temporal_value
func addTimeRangeParts(temporalValue map[string]interface{}, rangeStr string) {
	startTime, endTime := ParseTimeRange(rangeStr)

	if startTime == nil || endTime == nil {
		return
	}

	parts := map[string]interface{}{
		"start": map[string]interface{}{
			"hour":           startTime.Hour(),
			"minute":         startTime.Minute(),
			"normalized":     startTime.Format("3:04 PM"),
			"iso_format":     startTime.Format("15:04:05"),
		},
		"end": map[string]interface{}{
			"hour":           endTime.Hour(),
			"minute":         endTime.Minute(),
			"normalized":     endTime.Format("3:04 PM"),
			"iso_format":     endTime.Format("15:04:05"),
		},
	}

	// Calculate duration
	duration := endTime.Sub(*startTime)
	parts["duration_hours"] = duration.Hours()
	parts["duration_minutes"] = duration.Minutes()

	// Check if within business hours
	parts["business_hours"] = startTime.Hour() >= 9 && endTime.Hour() <= 17

	temporalValue["parts"] = parts
}

// TemporalMatch represents a temporal pattern found in text
type TemporalMatch struct {
	Start      int
	End        int
	Original   string
	Normalized string
	Type       TemporalType
}

// NormalizeDatesInText finds and normalizes all dates in text
func NormalizeDatesInText(text string) string {
	if text == "" {
		return text
	}

	// Find all temporal patterns
	matches := findTemporalMatches(text)
	if len(matches) == 0 {
		return text
	}

	// Sort by position (reverse) to maintain positions during replacement
	// Replace from end to beginning
	result := text
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		if match.Start >= 0 && match.End > match.Start && match.End <= len(result) {
			result = result[:match.Start] + match.Normalized + result[match.End:]
		}
	}

	return result
}

// findTemporalMatches finds all temporal patterns in text with their positions
func findTemporalMatches(text string) []TemporalMatch {
	var matches []TemporalMatch

	// Split into words and check each word/phrase
	words := strings.Fields(text)
	position := 0

	for i := 0; i < len(words); i++ {
		// Find the start position of this word in the original text
		wordStart := strings.Index(text[position:], words[i])
		if wordStart == -1 {
			continue
		}
		wordStart += position

		// Clean the word of common punctuation for detection
		cleanWord := strings.Trim(words[i], ".,;:!?()[]{}\"'")

		// Check single word
		if tempType := DetectTemporalType(cleanWord); tempType != TemporalTypeNone {
			normalized := NormalizeTemporal(cleanWord, tempType)
			matches = append(matches, TemporalMatch{
				Start:      wordStart,
				End:        wordStart + len(words[i]),
				Original:   words[i],
				Normalized: normalized,
				Type:       tempType,
			})
			position = wordStart + len(words[i])
			continue
		}

		// Check two-word patterns (e.g., "Q1 2024", "Jan 15")
		if i < len(words)-1 {
			nextWord := words[i+1]
			cleanNext := strings.Trim(nextWord, ".,;:!?()[]{}\"'")
			twoWords := cleanWord + " " + cleanNext

			if tempType := DetectTemporalType(twoWords); tempType != TemporalTypeNone {
				// Find the span of both words
				nextStart := strings.Index(text[wordStart:], nextWord)
				if nextStart != -1 {
					endPos := wordStart + nextStart + len(nextWord)
					normalized := NormalizeTemporal(twoWords, tempType)
					matches = append(matches, TemporalMatch{
						Start:      wordStart,
						End:        endPos,
						Original:   text[wordStart:endPos],
						Normalized: normalized,
						Type:       tempType,
					})
					position = endPos
					i++ // Skip next word since we processed it
					continue
				}
			}
		}

		// Check three-word patterns (e.g., "Jan 15, 2024")
		if i < len(words)-2 {
			word2 := words[i+1]
			word3 := words[i+2]
			clean2 := strings.Trim(word2, ".,;:!?()[]{}\"'")
			clean3 := strings.Trim(word3, ".,;:!?()[]{}\"'")
			threeWords := cleanWord + " " + clean2 + " " + clean3

			if tempType := DetectTemporalType(threeWords); tempType != TemporalTypeNone {
				// Find the span of all three words
				word3Start := strings.Index(text[wordStart:], word3)
				if word3Start != -1 {
					endPos := wordStart + word3Start + len(word3)
					normalized := NormalizeTemporal(threeWords, tempType)
					matches = append(matches, TemporalMatch{
						Start:      wordStart,
						End:        endPos,
						Original:   text[wordStart:endPos],
						Normalized: normalized,
						Type:       tempType,
					})
					position = endPos
					i += 2 // Skip next two words
					continue
				}
			}
		}

		position = wordStart + len(words[i])
	}

	return matches
}

// ProcessFieldValue processes a field value to detect and normalize temporal data
// Strategy:
//  1. Short fields (≤50 chars): Detect if 100% temporal, normalize entire value
//  2. Long fields (>50 chars): Extract embedded dates, replace with normalized forms
func ProcessFieldValue(fieldName, fieldValue string) (string, map[string]interface{}) {
	if fieldValue == "" {
		return fieldValue, nil
	}

	trimmed := strings.TrimSpace(fieldValue)

	// SHORT FIELD: Check if 100% temporal
	if len(trimmed) <= 50 {
		temporalType := DetectTemporalType(trimmed)
		if temporalType != TemporalTypeNone {
			// Entire field is temporal - normalize it
			normalized := NormalizeTemporal(trimmed, temporalType)
			temporalValue := CreateTemporalValue(trimmed, temporalType)

			return normalized, map[string]interface{}{
				"treatment":      "short_temporal",
				"temporal_value": temporalValue,
			}
		}
		// Not temporal, return as-is
		return fieldValue, nil
	}

	// LONG FIELD: Check for embedded dates
	normalizedText := NormalizeDatesInText(trimmed)

	// If normalization changed the text, we found temporal content
	if normalizedText != trimmed {
		return normalizedText, map[string]interface{}{
			"treatment":       "long_with_dates",
			"normalized_text": normalizedText,
		}
	}

	// No temporal content found
	return fieldValue, nil
}

// Helper function to get temporal type name
func temporalTypeName(t TemporalType) string {
	switch t {
	case TemporalTypeDate:
		return "date"
	case TemporalTypeTime:
		return "time"
	case TemporalTypeDateTime:
		return "datetime"
	case TemporalTypeTimeRange:
		return "time_range"
	default:
		return "none"
	}
}