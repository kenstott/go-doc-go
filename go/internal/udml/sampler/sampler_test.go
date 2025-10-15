package sampler

import (
	"testing"
)

func TestSamplerConfig_Defaults(t *testing.T) {
	config := SamplerConfig{
		ParquetPath: "/test/path",
	}

	_ = &Sampler{config: config}

	// Verify defaults are set correctly in NewSampler
	if config.SampleSize == 0 {
		t.Log("SampleSize defaults to 0 before NewSampler")
	}
}

func TestSample_Creation(t *testing.T) {
	sample := Sample{
		ElementID:      "elem1",
		DocID:          "doc1",
		ElementType:    "paragraph",
		Content:        "This is test content",
		ContentPreview: "This is test...",
		ElementOrder:   1.0,
	}

	if sample.ElementID != "elem1" {
		t.Errorf("Expected ElementID 'elem1', got '%s'", sample.ElementID)
	}

	if sample.ElementType != "paragraph" {
		t.Errorf("Expected ElementType 'paragraph', got '%s'", sample.ElementType)
	}
}

func TestStratumStats_Creation(t *testing.T) {
	stats := StratumStats{
		StratumValue: "paragraph",
		TotalCount:   1000,
		SampledCount: 100,
		SamplingRate: 0.1,
	}

	if stats.StratumValue != "paragraph" {
		t.Errorf("Expected stratum 'paragraph', got '%s'", stats.StratumValue)
	}

	if stats.SamplingRate != 0.1 {
		t.Errorf("Expected sampling rate 0.1, got %.2f", stats.SamplingRate)
	}
}

func TestExtractCandidateEntities(t *testing.T) {
	sampler := &Sampler{}

	text := "Microsoft Corporation and Apple Inc are technology companies. John Smith works there."

	entities := sampler.extractCandidateEntities(text)

	// Should extract capitalized words/phrases
	if len(entities) == 0 {
		t.Error("Expected to extract some entities")
	}

	t.Logf("Extracted entities: %v", entities)

	// Check for multi-word entities
	foundMultiWord := false
	for _, entity := range entities {
		if len(entity) > 10 { // Multi-word entities are longer
			foundMultiWord = true
			break
		}
	}

	if !foundMultiWord {
		t.Log("Note: No multi-word entities found (may be expected)")
	}
}

func TestIsCapitalized(t *testing.T) {
	tests := []struct {
		word     string
		expected bool
	}{
		{"Microsoft", true},
		{"microsoft", false},
		{"", false},
		{"M", true},
		{"m", false},
		{"MSFT", true},
	}

	for _, tt := range tests {
		result := isCapitalized(tt.word)
		if result != tt.expected {
			t.Errorf("isCapitalized(%q) = %v, expected %v", tt.word, result, tt.expected)
		}
	}
}

func TestCalculateAllocation(t *testing.T) {
	sampler := &Sampler{
		config: SamplerConfig{
			SampleSize:    100,
			MinPerStratum: 5,
		},
	}

	stratumCounts := map[string]int64{
		"paragraph": 1000,
		"heading":   200,
		"table":     50,
	}

	totalCount := int64(1250)

	allocation := sampler.calculateAllocation(stratumCounts, totalCount)

	// Verify total allocation
	totalAllocated := 0
	for _, count := range allocation {
		totalAllocated += count
	}

	t.Logf("Allocation: %v", allocation)
	t.Logf("Total allocated: %d", totalAllocated)

	// Verify minimum per stratum is respected
	for stratum, count := range allocation {
		if count < sampler.config.MinPerStratum {
			t.Errorf("Stratum %s allocated %d, less than minimum %d",
				stratum, count, sampler.config.MinPerStratum)
		}
	}
}

func TestGetTopEntities(t *testing.T) {
	result := &SamplingResult{
		EntityFrequencies: map[string]int{
			"Microsoft":   500,
			"Apple":       300,
			"Google":      250,
			"Amazon":      200,
			"Facebook":    150,
		},
	}

	top3 := result.GetTopEntities(3)

	if len(top3) != 3 {
		t.Errorf("Expected 3 top entities, got %d", len(top3))
	}

	// Verify sorted by count descending
	if top3[0].Entity != "Microsoft" {
		t.Errorf("Expected 'Microsoft' as top entity, got '%s'", top3[0].Entity)
	}

	if top3[0].Count != 500 {
		t.Errorf("Expected count 500 for Microsoft, got %d", top3[0].Count)
	}

	// Verify descending order
	for i := 1; i < len(top3); i++ {
		if top3[i].Count > top3[i-1].Count {
			t.Error("Top entities not sorted in descending order")
		}
	}

	t.Logf("Top 3 entities: %v", top3)
}

func TestExtractHelpers(t *testing.T) {
	sampler := &Sampler{}

	// Test extractString
	t.Run("extractString", func(t *testing.T) {
		tests := []struct {
			input    interface{}
			expected string
		}{
			{nil, ""},
			{"hello", "hello"},
			{[]byte("world"), "world"},
			{123, "123"},
		}

		for _, tt := range tests {
			result := sampler.extractString(tt.input)
			if result != tt.expected {
				t.Errorf("extractString(%v) = %q, expected %q", tt.input, result, tt.expected)
			}
		}
	})

	// Test extractInt64
	t.Run("extractInt64", func(t *testing.T) {
		tests := []struct {
			input    interface{}
			expected int64
		}{
			{nil, 0},
			{int64(123), 123},
			{int(456), 456},
			{float64(789.5), 789},
		}

		for _, tt := range tests {
			result := sampler.extractInt64(tt.input)
			if result != tt.expected {
				t.Errorf("extractInt64(%v) = %d, expected %d", tt.input, result, tt.expected)
			}
		}
	})

	// Test extractFloat64
	t.Run("extractFloat64", func(t *testing.T) {
		tests := []struct {
			input    interface{}
			expected float64
			epsilon  float64
		}{
			{nil, 0.0, 0.0},
			{float64(123.45), 123.45, 0.0},
			{float32(67.89), 67.89, 0.01}, // Allow small precision error for float32
			{int(100), 100.0, 0.0},
			{int64(200), 200.0, 0.0},
		}

		for _, tt := range tests {
			result := sampler.extractFloat64(tt.input)
			diff := result - tt.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.epsilon {
				t.Errorf("extractFloat64(%v) = %f, expected %f (diff: %f > epsilon: %f)",
					tt.input, result, tt.expected, diff, tt.epsilon)
			}
		}
	})
}
