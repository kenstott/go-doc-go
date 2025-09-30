package temporal

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// TemporalType represents different types of temporal data
type TemporalType int

const (
	TemporalTypeNone      TemporalType = 0 // Not a temporal string
	TemporalTypeDate      TemporalType = 1 // Date only (no time component)
	TemporalTypeTime      TemporalType = 2 // Time only (no date component)
	TemporalTypeDateTime  TemporalType = 3 // Combined date and time
	TemporalTypeTimeRange TemporalType = 4 // Time range (start and end times)
)

// Common date/time patterns for detection
var (
	// Time range patterns
	timeRangePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^\d{1,2}:\d{2}\s*[-–—to]+\s*\d{1,2}:\d{2}$`),                        // 14:00-16:00
		regexp.MustCompile(`(?i)^\d{1,2}:\d{2}\s*(?:am|pm)\s*[-–—to]+\s*\d{1,2}:\d{2}\s*(?:am|pm)$`), // 2:00pm-4:00pm
		regexp.MustCompile(`(?i)^\d{1,2}\s*(?:am|pm)\s*[-–—to]+\s*\d{1,2}\s*(?:am|pm)$`),             // 2pm-4pm
		regexp.MustCompile(`(?i)^\d{1,2}[-–—]\d{1,2}\s*(?:am|pm)$`),                                  // 2-4pm
	}

	// Time only patterns
	timePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^[0-1]?\d:[0-5]\d(:[0-5]\d)?\s*[ap]\.?m\.?$`),        // 3:45pm, 03:45 pm
		regexp.MustCompile(`^[0-2]?\d:[0-5]\d(:[0-5]\d)?$`),                           // 15:30, 13:45:30 (24-hour)
		regexp.MustCompile(`(?i)^[0]?\d\s*[ap]\.?m\.?|1[0-2]\s*[ap]\.?m\.?$`),        // 3pm, 11 a.m.
		regexp.MustCompile(`(?i)^(noon|midnight)$`),                                   // noon, midnight
	}

	// Date patterns - common formats
	datePatterns = []*regexp.Regexp{
		regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`),                                     // 2023-12-25
		regexp.MustCompile(`^\d{2}/\d{2}/\d{4}$`),                                     // 12/25/2023
		regexp.MustCompile(`^\d{1,2}/\d{1,2}/\d{2,4}$`),                               // 1/5/23 or 1/5/2023
		regexp.MustCompile(`(?i)^(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)`), // Month names
		regexp.MustCompile(`(?i)^\d{1,2}[-\s](jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)[-\s]\d{2,4}$`), // 25-Dec-2023
		regexp.MustCompile(`(?i)^Q[1-4]\s+\d{4}$`),                                    // Q1 2024
		regexp.MustCompile(`(?i)^Q[1-4]$`),                                            // Q1
	}

	// DateTime patterns
	dateTimePatterns = []*regexp.Regexp{
		regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}`),                      // ISO format
		regexp.MustCompile(`\d{1,2}/\d{1,2}/\d{2,4}\s+\d{1,2}:\d{2}`),                // 12/25/2023 14:30
		regexp.MustCompile(`(?i)\d{1,2}:\d{2}\s*(?:am|pm).*\d{1,2}/\d{1,2}/\d{2,4}`), // Time before date
	}

	// Simple number pattern to exclude
	simpleNumberPattern = regexp.MustCompile(`^\d+(\.\d+)?$`)
)

// DetectTemporalType detects if a string represents a date, time, datetime, time range, or none
func DetectTemporalType(inputString string) TemporalType {
	// Check if it's an obviously non-temporal string
	if inputString == "" || len(inputString) > 50 || len(strings.Fields(inputString)) > 8 {
		return TemporalTypeNone
	}

	inputString = strings.TrimSpace(inputString)

	// Check if it's just a number
	if simpleNumberPattern.MatchString(inputString) {
		return TemporalTypeNone
	}

	// Check for time range patterns first
	for _, pattern := range timeRangePatterns {
		if pattern.MatchString(inputString) {
			return TemporalTypeTimeRange
		}
	}

	// Check for time-only patterns
	for _, pattern := range timePatterns {
		if pattern.MatchString(inputString) {
			return TemporalTypeTime
		}
	}

	// Check for datetime patterns
	for _, pattern := range dateTimePatterns {
		if pattern.MatchString(inputString) {
			return TemporalTypeDateTime
		}
	}

	// Check for date patterns
	for _, pattern := range datePatterns {
		if pattern.MatchString(inputString) {
			return TemporalTypeDate
		}
	}

	// Try parsing with Go's time package
	// Common date/time formats to try
	formats := []string{
		"2006-01-02",                   // ISO date
		"01/02/2006",                   // US date
		"02/01/2006",                   // EU date
		"Jan 2, 2006",                  // Month name
		"2 Jan 2006",                   // Day month year
		"2006-01-02 15:04:05",          // ISO datetime
		"01/02/2006 15:04",             // US datetime
		"2006-01-02T15:04:05",          // ISO 8601
		"2006-01-02T15:04:05Z07:00",    // ISO 8601 with timezone
		"Mon Jan 2 15:04:05 2006",      // Unix date
		"3:04 PM",                      // Time with AM/PM
		"15:04",                        // 24-hour time
	}

	for _, format := range formats {
		if _, err := time.Parse(format, inputString); err == nil {
			// Successfully parsed
			if strings.Contains(inputString, ":") ||
			   strings.Contains(strings.ToLower(inputString), "am") ||
			   strings.Contains(strings.ToLower(inputString), "pm") {
				// Has time component
				if regexp.MustCompile(`\d{4}|\d{1,2}/\d{1,2}`).MatchString(inputString) {
					return TemporalTypeDateTime // Has both date and time
				}
				return TemporalTypeTime // Time only
			}
			return TemporalTypeDate // Date only
		}
	}

	return TemporalTypeNone
}

// ParseTimeRange parses a time range string into start and end time
func ParseTimeRange(timeRangeStr string) (*time.Time, *time.Time) {
	// Normalize the separator
	normalized := regexp.MustCompile(`[-–—to]+`).ReplaceAllString(timeRangeStr, "-")

	// Check for simple range with AM/PM at end (e.g., "9-5pm")
	amPmEndPattern := regexp.MustCompile(`(?i)(\d{1,2})[-–—](\d{1,2})\s*([ap]\.?m\.?)`)
	if matches := amPmEndPattern.FindStringSubmatch(normalized); len(matches) == 4 {
		startHour := parseInt(matches[1])
		endHour := parseInt(matches[2])
		amPm := strings.ToLower(matches[3])

		// Adjust for PM
		if strings.Contains(amPm, "p") && endHour < 12 {
			endHour += 12
			if startHour < endHour-12 {
				startHour += 12
			}
		}

		startTime := time.Date(0, 1, 1, startHour, 0, 0, 0, time.UTC)
		endTime := time.Date(0, 1, 1, endHour, 0, 0, 0, time.UTC)
		return &startTime, &endTime
	}

	// Split on separator
	parts := strings.Split(normalized, "-")
	if len(parts) != 2 {
		return nil, nil
	}

	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])

	// Parse both parts
	startTime := parseTimeString(startStr)
	endTime := parseTimeString(endStr)

	return startTime, endTime
}

// parseTimeString attempts to parse a time string
func parseTimeString(timeStr string) *time.Time {
	timeStr = strings.TrimSpace(timeStr)

	timeFormats := []string{
		"3:04 PM",
		"3:04PM",
		"3:04pm",
		"3:04 pm",
		"15:04",
		"15:04:05",
		"3PM",
		"3 PM",
		"3pm",
		"3 pm",
	}

	// Try case-insensitive parsing for AM/PM
	lowerStr := strings.ToLower(timeStr)
	upperStr := strings.ToUpper(timeStr)

	for _, format := range timeFormats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return &t
		}
		// Try lowercase version
		if t, err := time.Parse(format, lowerStr); err == nil {
			return &t
		}
		// Try uppercase version
		if t, err := time.Parse(format, upperStr); err == nil {
			return &t
		}
	}

	// Try simple hour parsing for formats like "9" or "5"
	if matches := regexp.MustCompile(`^\d{1,2}$`).FindStringSubmatch(timeStr); len(matches) > 0 {
		hour := parseInt(matches[0])
		t := time.Date(0, 1, 1, hour, 0, 0, 0, time.UTC)
		return &t
	}

	return nil
}

// CreateSemanticTimeRangeExpression converts a time range into semantic expression
func CreateSemanticTimeRangeExpression(timeRangeStr string) string {
	startTime, endTime := ParseTimeRange(timeRangeStr)

	if startTime == nil || endTime == nil {
		return timeRangeStr // Return original if parsing failed
	}

	// Generate semantic expressions for both times
	startSemantic := CreateSemanticTimeExpression(startTime)
	endSemantic := CreateSemanticTimeExpression(endTime)

	// Check for common business hours
	businessTerms := []string{}

	if startTime.Hour() == 9 && startTime.Minute() == 0 &&
		endTime.Hour() == 17 && endTime.Minute() == 0 {
		businessTerms = append(businessTerms, "nine to five", "9-5", "standard business hours", "regular office hours")
	} else if startTime.Hour() == 8 && startTime.Minute() == 0 &&
		endTime.Hour() == 17 && endTime.Minute() == 0 {
		businessTerms = append(businessTerms, "eight to five", "8-5", "extended business hours")
	} else if startTime.Hour() == 12 && endTime.Hour() == 13 {
		businessTerms = append(businessTerms, "lunch hour", "lunch break", "midday break")
	}

	result := fmt.Sprintf("from %s until %s", startSemantic, endSemantic)
	if len(businessTerms) > 0 {
		result += fmt.Sprintf(", %s", strings.Join(businessTerms, ", "))
	}

	return result
}

// CreateSemanticTimeExpression converts a time into semantic expression
func CreateSemanticTimeExpression(t *time.Time) string {
	if t == nil {
		return ""
	}

	hour := t.Hour()
	minute := t.Minute()

	essentialTerms := []string{fmt.Sprintf("%02d%02d", hour, minute)}

	// Common business time expressions
	if hour == 12 && minute == 0 {
		essentialTerms = append(essentialTerms, "noon", "midday")
	} else if hour == 0 && minute == 0 {
		essentialTerms = append(essentialTerms, "midnight")
	} else if minute == 30 {
		hour12 := hour % 12
		if hour12 == 0 {
			hour12 = 12
		}
		essentialTerms = append(essentialTerms, fmt.Sprintf("half past %d", hour12))
	}

	// Time of day
	if hour >= 9 && hour < 17 {
		essentialTerms = append(essentialTerms, "business hours")
	}
	if hour >= 12 && hour < 14 {
		essentialTerms = append(essentialTerms, "lunch time")
	}

	result := t.Format("15:04")
	if len(essentialTerms) > 0 {
		result += fmt.Sprintf(", %s", strings.Join(essentialTerms, ", "))
	}

	return result
}

// CreateSemanticDateExpression converts a date string into semantic expression
func CreateSemanticDateExpression(dateStr string, fromDateTime bool) string {
	// Try to parse the date
	t, err := parseDate(dateStr)
	if err != nil {
		return dateStr // Return original on error
	}

	// Check if this has a significant time component
	hasTime := t.Hour() != 0 || t.Minute() != 0 || t.Second() != 0
	if hasTime && !fromDateTime {
		return CreateSemanticDateTimeExpression(dateStr)
	}

	// Get basic date components
	day := t.Day()
	year := t.Year()
	monthNum := int(t.Month())
	dayOfWeek := t.Format("Monday")

	// Calculate quarter
	quarterNum := (monthNum-1)/3 + 1

	// Calculate week of month
	weekOfMonth := (day-1)/7 + 1

	practicalTerms := []string{}

	// Quarter terms
	quarterTerms := map[int][]string{
		1: {"Q1", "first quarter"},
		2: {"Q2", "second quarter"},
		3: {"Q3", "third quarter"},
		4: {"Q4", "fourth quarter"},
	}
	if terms, ok := quarterTerms[quarterNum]; ok {
		practicalTerms = append(practicalTerms, terms...)
	}

	// Month abbreviation
	practicalTerms = append(practicalTerms, t.Format("Jan"))

	// Seasonal terms
	seasonalTerms := map[int]string{
		1: "winter",
		2: "spring",
		3: "summer",
		4: "fall",
	}
	if season, ok := seasonalTerms[quarterNum]; ok {
		practicalTerms = append(practicalTerms, season)
	}

	// Week-based terms
	weekOrdinals := map[int]string{
		1: "first", 2: "second", 3: "third", 4: "fourth", 5: "fifth",
	}
	if ordinal, ok := weekOrdinals[weekOfMonth]; ok {
		practicalTerms = append(practicalTerms, fmt.Sprintf("%s week", ordinal))
	}

	// Month position
	if day <= 7 {
		practicalTerms = append(practicalTerms, "beginning of month")
	} else if day >= 22 {
		practicalTerms = append(practicalTerms, "end of month")
		if quarterNum == 4 && monthNum == 12 {
			practicalTerms = append(practicalTerms, "year end")
		}
	} else if day >= 10 && day <= 20 {
		practicalTerms = append(practicalTerms, "mid month")
	}

	// Day of week
	practicalTerms = append(practicalTerms, dayOfWeek)
	weekday := t.Weekday()
	if weekday >= time.Monday && weekday <= time.Friday {
		practicalTerms = append(practicalTerms, "business day", "weekday")
	} else {
		practicalTerms = append(practicalTerms, "weekend")
	}

	// Business day count
	businessDayCount := 0
	for d := 1; d <= day; d++ {
		testDate := time.Date(year, t.Month(), d, 0, 0, 0, 0, t.Location())
		if testDate.Weekday() >= time.Monday && testDate.Weekday() <= time.Friday {
			businessDayCount++
		}
	}

	if businessDayCount <= 5 {
		practicalTerms = append(practicalTerms, "early business days")
	} else if businessDayCount >= 17 {
		practicalTerms = append(practicalTerms, "late business days")
	}

	// Year and ISO format
	practicalTerms = append(practicalTerms, fmt.Sprintf("%d", year))
	practicalTerms = append(practicalTerms, fmt.Sprintf("%d-%02d", year, monthNum))

	// Fiscal year terms
	if quarterNum == 4 {
		practicalTerms = append(practicalTerms, "fiscal year end", "year end")
	} else if quarterNum == 1 {
		practicalTerms = append(practicalTerms, "fiscal year start")
	}

	result := dateStr
	if len(practicalTerms) > 0 {
		result += fmt.Sprintf(", %s", strings.Join(practicalTerms, ", "))
	}

	return result
}

// CreateSemanticDateTimeExpression converts datetime string to semantic expression
func CreateSemanticDateTimeExpression(dtStr string) string {
	// Parse the datetime
	t, err := parseDate(dtStr)
	if err != nil {
		return dtStr
	}

	// Generate date part
	datePart := CreateSemanticDateExpression(dtStr, true)

	// Generate time part
	timePart := CreateSemanticTimeExpression(&t)

	return fmt.Sprintf("%s, %s", datePart, timePart)
}

// CreateSemanticTemporalExpression creates semantic expression based on detected type
func CreateSemanticTemporalExpression(inputString string) string {
	temporalType := DetectTemporalType(inputString)

	switch temporalType {
	case TemporalTypeDate:
		return CreateSemanticDateExpression(inputString, false)
	case TemporalTypeTime:
		if t := parseTimeString(inputString); t != nil {
			return CreateSemanticTimeExpression(t)
		}
		return inputString
	case TemporalTypeDateTime:
		return CreateSemanticDateTimeExpression(inputString)
	case TemporalTypeTimeRange:
		return CreateSemanticTimeRangeExpression(inputString)
	default:
		return inputString
	}
}

// Helper function to parse dates with multiple formats
func parseDate(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)

	formats := []string{
		"2006-01-02",
		"01/02/2006",
		"02/01/2006",
		"Jan 2, 2006",
		"January 2, 2006",
		"2 Jan 2006",
		"2-Jan-2006",
		"02-Jan-2006",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"01/02/2006 15:04",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z07:00",
		time.RFC3339,
		time.RFC822,
		time.RFC850,
		time.RFC1123,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// Helper function to parse integers
func parseInt(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}