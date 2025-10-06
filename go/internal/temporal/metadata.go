package temporal

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// GenerateTemporalMetadata generates structured temporal metadata for a value if it's temporal
func GenerateTemporalMetadata(value string) map[string]interface{} {
	temporalType := DetectTemporalType(value)

	if temporalType == TemporalTypeNone {
		return nil
	}

	switch temporalType {
	case TemporalTypeDate:
		return generateDateMetadata(value)
	case TemporalTypeDateTime:
		return generateDateTimeMetadata(value)
	case TemporalTypeTime:
		return generateTimeMetadata(value)
	case TemporalTypeTimeRange:
		return generateTimeRangeMetadata(value)
	default:
		return nil
	}
}

// generateDateMetadata generates metadata for a date value
func generateDateMetadata(dateStr string) map[string]interface{} {
	// Parse the date
	parsedDate, err := parseDate(dateStr)
	if err != nil {
		log.Printf("Failed to parse date %s: %v", dateStr, err)
		return map[string]interface{}{
			"temporal_type": "date",
			"raw_value":     dateStr,
		}
	}

	// Calculate quarter
	quarter := (int(parsedDate.Month())-1)/3 + 1

	// Calculate week of month
	weekOfMonth := (parsedDate.Day()-1)/7 + 1

	// Calculate decade
	decade := fmt.Sprintf("%ds", (parsedDate.Year()/10)*10)

	metadata := map[string]interface{}{
		"temporal_type":    "date",
		"raw_value":        dateStr,
		"year":             parsedDate.Year(),
		"quarter":          quarter,
		"quarter_label":    fmt.Sprintf("Q%d", quarter),
		"month":            int(parsedDate.Month()),
		"month_name":       parsedDate.Format("January"),
		"month_abbr":       parsedDate.Format("Jan"),
		"week_of_month":    weekOfMonth,
		"day":              parsedDate.Day(),
		"day_of_week":      parsedDate.Format("Monday"),
		"day_of_week_abbr": parsedDate.Format("Mon"),
		"decade":           decade,
		"iso_date":         parsedDate.Format("2006-01-02"),
	}

	// Add season
	if season := getSeason(int(parsedDate.Month())); season != "" {
		metadata["season"] = season
	}

	return metadata
}

// generateDateTimeMetadata generates metadata for a datetime value
func generateDateTimeMetadata(datetimeStr string) map[string]interface{} {
	// Parse the datetime
	parsedDT, err := parseDate(datetimeStr)
	if err != nil {
		log.Printf("Failed to parse datetime %s: %v", datetimeStr, err)
		return map[string]interface{}{
			"temporal_type": "datetime",
			"raw_value":     datetimeStr,
		}
	}

	// Start with date metadata
	metadata := generateDateMetadata(parsedDT.Format("2006-01-02"))

	// Override temporal type
	metadata["temporal_type"] = "datetime"
	metadata["raw_value"] = datetimeStr

	// Add time components
	metadata["hour"] = parsedDT.Hour()
	metadata["minute"] = parsedDT.Minute()
	metadata["second"] = parsedDT.Second()
	metadata["time_of_day"] = getTimeOfDay(parsedDT.Hour())
	metadata["business_hours"] = parsedDT.Hour() >= 9 && parsedDT.Hour() < 17
	metadata["iso_datetime"] = parsedDT.Format(time.RFC3339)

	return metadata
}

// generateTimeMetadata generates metadata for a time value
func generateTimeMetadata(timeStr string) map[string]interface{} {
	// Parse as time
	parsedTime := parseTimeString(timeStr)
	if parsedTime == nil {
		// Try parsing as full date and extract time
		if dt, err := parseDate(timeStr); err == nil {
			parsedTime = &dt
		} else {
			log.Printf("Failed to parse time %s", timeStr)
			return map[string]interface{}{
				"temporal_type": "time",
				"raw_value":     timeStr,
			}
		}
	}

	metadata := map[string]interface{}{
		"temporal_type":  "time",
		"raw_value":      timeStr,
		"hour":           parsedTime.Hour(),
		"minute":         parsedTime.Minute(),
		"second":         parsedTime.Second(),
		"time_of_day":    getTimeOfDay(parsedTime.Hour()),
		"business_hours": parsedTime.Hour() >= 9 && parsedTime.Hour() < 17,
		"iso_time":       parsedTime.Format("15:04:05"),
	}

	return metadata
}

// generateTimeRangeMetadata generates metadata for a time range value
func generateTimeRangeMetadata(rangeStr string) map[string]interface{} {
	// Try to parse common range formats
	var parts []string

	// Try different separators
	separators := []string{" to ", " - ", "-", "–", "—"}
	for _, sep := range separators {
		if strings.Contains(rangeStr, sep) {
			parts = strings.Split(rangeStr, sep)
			break
		}
	}

	if len(parts) == 2 {
		startStr := strings.TrimSpace(parts[0])
		endStr := strings.TrimSpace(parts[1])

		// Try to parse start and end
		startMeta := generateTimeMetadata(startStr)
		endMeta := generateTimeMetadata(endStr)

		metadata := map[string]interface{}{
			"temporal_type": "time_range",
			"raw_value":     rangeStr,
			"start":         startMeta,
			"end":           endMeta,
		}

		// Add range-specific metadata if both parsed successfully
		if startHour, ok := startMeta["hour"].(int); ok {
			if endHour, ok := endMeta["hour"].(int); ok {
				durationHours := float64(endHour - startHour)

				if startMin, ok := startMeta["minute"].(int); ok {
					if endMin, ok := endMeta["minute"].(int); ok {
						durationHours += float64(endMin-startMin) / 60
					}
				}

				metadata["duration_hours"] = durationHours

				// Check if it's business hours
				startBusiness, _ := startMeta["business_hours"].(bool)
				endBusiness, _ := endMeta["business_hours"].(bool)
				metadata["business_hours"] = startBusiness && endBusiness
			}
		}

		return metadata
	}

	// Fallback
	return map[string]interface{}{
		"temporal_type": "time_range",
		"raw_value":     rangeStr,
	}
}

// getSeason returns the season for a month (Northern hemisphere)
func getSeason(month int) string {
	seasons := map[int]string{
		12: "winter", 1: "winter", 2: "winter",
		3: "spring", 4: "spring", 5: "spring",
		6: "summer", 7: "summer", 8: "summer",
		9: "fall", 10: "fall", 11: "fall",
	}
	return seasons[month]
}

// getTimeOfDay returns time of day description
func getTimeOfDay(hour int) string {
	switch {
	case hour >= 0 && hour < 6:
		return "early_morning"
	case hour >= 6 && hour < 12:
		return "morning"
	case hour == 12:
		return "noon"
	case hour > 12 && hour < 17:
		return "afternoon"
	case hour >= 17 && hour < 20:
		return "evening"
	default:
		return "night"
	}
}