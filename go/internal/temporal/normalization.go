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

// NormalizeDatesInText finds and normalizes all dates in text
func NormalizeDatesInText(text string) string {
	// This is a simplified version - in production you'd want more comprehensive detection
	// For now, just return the original text
	return text
}

// ProcessFieldValue processes a field value to detect and normalize temporal data
func ProcessFieldValue(fieldName, fieldValue string) (interface{}, map[string]interface{}) {
	// Detect temporal type
	temporalType := DetectTemporalType(fieldValue)

	if temporalType == TemporalTypeNone {
		return fieldValue, nil
	}

	// Create temporal value
	temporalValue := CreateTemporalValue(fieldValue, temporalType)

	// Return normalized value and metadata
	if normalized, ok := temporalValue["normalized"].(string); ok {
		return normalized, temporalValue
	}

	return fieldValue, temporalValue
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