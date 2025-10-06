package temporal

import (
	"strings"
	"testing"
	"time"
)

func TestDetectTemporalType(t *testing.T) {
	tests := []struct {
		input    string
		expected TemporalType
	}{
		// Date patterns
		{"2023-12-25", TemporalTypeDate},
		{"12/25/2023", TemporalTypeDate},
		{"25-Dec-2023", TemporalTypeDate},
		{"December 25, 2023", TemporalTypeDate},
		{"Jan 1, 2024", TemporalTypeDate},

		// Time patterns
		{"3:45pm", TemporalTypeTime},
		{"15:30", TemporalTypeTime},
		{"3:45 PM", TemporalTypeTime},
		{"noon", TemporalTypeTime},
		{"midnight", TemporalTypeTime},

		// DateTime patterns
		{"2023-12-25 14:30", TemporalTypeDateTime},
		{"2023-12-25T14:30:00", TemporalTypeDateTime},
		{"12/25/2023 2:30 PM", TemporalTypeDateTime},

		// Time range patterns
		{"14:00-16:00", TemporalTypeTimeRange},
		{"2:00pm-4:00pm", TemporalTypeTimeRange},
		{"2pm-4pm", TemporalTypeTimeRange},
		{"9-5pm", TemporalTypeTimeRange},

		// Non-temporal
		{"hello world", TemporalTypeNone},
		{"123", TemporalTypeNone},
		{"123.45", TemporalTypeNone},
		{"", TemporalTypeNone},
	}

	for _, test := range tests {
		result := DetectTemporalType(test.input)
		if result != test.expected {
			t.Errorf("DetectTemporalType(%q) = %v, want %v", test.input, result, test.expected)
		}
	}
}

func TestParseTimeRange(t *testing.T) {
	tests := []struct {
		input      string
		wantStart  int // hour
		wantEnd    int // hour
		shouldFail bool
	}{
		{"9-5pm", 9, 17, false},
		{"14:00-16:00", 14, 16, false},
		{"2:00pm-4:00pm", 14, 16, false},
		{"2pm-4pm", 14, 16, false},
		{"invalid", 0, 0, true},
	}

	for _, test := range tests {
		start, end := ParseTimeRange(test.input)

		if test.shouldFail {
			if start != nil || end != nil {
				t.Errorf("ParseTimeRange(%q) should have failed", test.input)
			}
			continue
		}

		if start == nil || end == nil {
			t.Errorf("ParseTimeRange(%q) failed to parse", test.input)
			continue
		}

		if start.Hour() != test.wantStart {
			t.Errorf("ParseTimeRange(%q) start hour = %d, want %d", test.input, start.Hour(), test.wantStart)
		}

		if end.Hour() != test.wantEnd {
			t.Errorf("ParseTimeRange(%q) end hour = %d, want %d", test.input, end.Hour(), test.wantEnd)
		}
	}
}

func TestNormalizeTemporal(t *testing.T) {
	tests := []struct {
		input        string
		temporalType TemporalType
		expected     string
	}{
		// Date normalization
		{"2023-12-25", TemporalTypeDate, "Dec 25, 2023"},
		{"12/25/2023", TemporalTypeDate, "Dec 25, 2023"},

		// Time normalization
		{"15:30", TemporalTypeTime, "3:30 PM"},
		{"noon", TemporalTypeTime, "12:00 PM"},
		{"midnight", TemporalTypeTime, "12:00 AM"},

		// DateTime normalization
		{"2023-12-25 14:30", TemporalTypeDateTime, "Dec 25, 2023 2:30 PM"},

		// Time range normalization
		{"14:00-16:00", TemporalTypeTimeRange, "2:00 PM - 4:00 PM"},
	}

	for _, test := range tests {
		result := NormalizeTemporal(test.input, test.temporalType)
		if result != test.expected {
			t.Errorf("NormalizeTemporal(%q, %v) = %q, want %q", test.input, test.temporalType, result, test.expected)
		}
	}
}

func TestGenerateTemporalMetadata(t *testing.T) {
	// Test date metadata
	metadata := GenerateTemporalMetadata("2023-12-25")
	if metadata == nil {
		t.Fatal("GenerateTemporalMetadata returned nil for valid date")
	}

	// Check key fields
	if metadata["temporal_type"] != "date" {
		t.Errorf("Expected temporal_type = 'date', got %v", metadata["temporal_type"])
	}

	if metadata["year"] != 2023 {
		t.Errorf("Expected year = 2023, got %v", metadata["year"])
	}

	if metadata["month"] != 12 {
		t.Errorf("Expected month = 12, got %v", metadata["month"])
	}

	if metadata["day"] != 25 {
		t.Errorf("Expected day = 25, got %v", metadata["day"])
	}

	if metadata["quarter"] != 4 {
		t.Errorf("Expected quarter = 4, got %v", metadata["quarter"])
	}

	if metadata["season"] != "winter" {
		t.Errorf("Expected season = 'winter', got %v", metadata["season"])
	}

	// Test time metadata
	timeMetadata := GenerateTemporalMetadata("14:30")
	if timeMetadata == nil {
		t.Fatal("GenerateTemporalMetadata returned nil for valid time")
	}

	if timeMetadata["temporal_type"] != "time" {
		t.Errorf("Expected temporal_type = 'time', got %v", timeMetadata["temporal_type"])
	}

	if timeMetadata["hour"] != 14 {
		t.Errorf("Expected hour = 14, got %v", timeMetadata["hour"])
	}

	if timeMetadata["minute"] != 30 {
		t.Errorf("Expected minute = 30, got %v", timeMetadata["minute"])
	}

	if timeMetadata["business_hours"] != true {
		t.Errorf("Expected business_hours = true, got %v", timeMetadata["business_hours"])
	}
}

func TestCreateTemporalValue(t *testing.T) {
	// Test date temporal value
	value := CreateTemporalValue("2023-12-25", TemporalTypeDate)

	if value["type"] != "date" {
		t.Errorf("Expected type = 'date', got %v", value["type"])
	}

	if value["normalized"] != "Dec 25, 2023" {
		t.Errorf("Expected normalized = 'Dec 25, 2023', got %v", value["normalized"])
	}

	if value["iso_format"] != "2023-12-25" {
		t.Errorf("Expected iso_format = '2023-12-25', got %v", value["iso_format"])
	}

	parts, ok := value["parts"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected parts to be a map")
	}

	if parts["year"] != 2023 {
		t.Errorf("Expected parts.year = 2023, got %v", parts["year"])
	}

	if parts["season"] != "Winter" {
		t.Errorf("Expected parts.season = 'Winter', got %v", parts["season"])
	}
}

func TestSemanticExpressions(t *testing.T) {
	// Test date semantic expression
	dateExpr := CreateSemanticDateExpression("2023-01-15", false)
	if dateExpr == "" {
		t.Error("CreateSemanticDateExpression returned empty string")
	}

	// Should contain quarter info
	if !contains(dateExpr, "Q1") {
		t.Errorf("Expected date expression to contain 'Q1', got %q", dateExpr)
	}

	// Test time semantic expression
	testTime := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
	timeExpr := CreateSemanticTimeExpression(&testTime)
	if timeExpr == "" {
		t.Error("CreateSemanticTimeExpression returned empty string")
	}

	// Should contain business hours
	if !contains(timeExpr, "business hours") {
		t.Errorf("Expected time expression to contain 'business hours', got %q", timeExpr)
	}

	// Test time range semantic expression
	rangeExpr := CreateSemanticTimeRangeExpression("9:00-17:00")
	if rangeExpr == "" {
		t.Error("CreateSemanticTimeRangeExpression returned empty string")
	}

	// Should contain standard business hours references
	if !contains(rangeExpr, "business hours") && !contains(rangeExpr, "9-5") {
		t.Errorf("Expected range expression to contain business hours reference, got %q", rangeExpr)
	}
}

func TestProcessFieldValue(t *testing.T) {
	// Test processing a date field
	normalized, metadata := ProcessFieldValue("date_field", "2023-12-25")

	if normalized != "Dec 25, 2023" {
		t.Errorf("Expected normalized = 'Dec 25, 2023', got %v", normalized)
	}

	if metadata == nil {
		t.Fatal("Expected metadata to be non-nil for date value")
	}

	if metadata["type"] != "date" {
		t.Errorf("Expected metadata.type = 'date', got %v", metadata["type"])
	}

	// Test processing a non-temporal field
	nonTemporal, nonMetadata := ProcessFieldValue("text_field", "hello world")

	if nonTemporal != "hello world" {
		t.Errorf("Expected non-temporal value to remain unchanged, got %v", nonTemporal)
	}

	if nonMetadata != nil {
		t.Errorf("Expected nil metadata for non-temporal value, got %v", nonMetadata)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && strings.Contains(s, substr)
}

// Benchmarks
func BenchmarkDetectTemporalType(b *testing.B) {
	inputs := []string{
		"2023-12-25",
		"3:45pm",
		"2023-12-25 14:30",
		"14:00-16:00",
		"hello world",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range inputs {
			DetectTemporalType(input)
		}
	}
}

func BenchmarkGenerateTemporalMetadata(b *testing.B) {
	inputs := []string{
		"2023-12-25",
		"14:30",
		"2023-12-25 14:30",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range inputs {
			GenerateTemporalMetadata(input)
		}
	}
}

func BenchmarkNormalizeTemporal(b *testing.B) {
	tests := []struct {
		input string
		ttype TemporalType
	}{
		{"2023-12-25", TemporalTypeDate},
		{"15:30", TemporalTypeTime},
		{"2023-12-25 14:30", TemporalTypeDateTime},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, test := range tests {
			NormalizeTemporal(test.input, test.ttype)
		}
	}
}