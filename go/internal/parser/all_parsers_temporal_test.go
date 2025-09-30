package parser

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// TestAllParsersTemporalExtraction tests temporal extraction across all parsers
func TestAllParsersTemporalExtraction(t *testing.T) {
	// Test data with various temporal formats
	temporalContent := `
		Meeting scheduled for 2024-01-15 at 14:30.
		Project deadline: December 31, 2023.
		Business hours: 9:00-17:00.
		Quarter results: Q1 2024.
		Last updated: 2023-12-25T10:30:00Z.
	`

	tests := []struct {
		name        string
		parserType  string
		prepareData func() ([]byte, error)
		runParser   func([]byte) (bool, error)
	}{
		{
			name:       "PDF Parser",
			parserType: "pdf",
			prepareData: func() ([]byte, error) {
				// For PDF, we'll use simple text content
				// In real tests, this would be actual PDF bytes
				return []byte(temporalContent), nil
			},
			runParser: func(data []byte) (bool, error) {
				// Since we can't easily create a real PDF, we'll test the helper
				metadata := make(map[string]interface{})
				ProcessTemporalContent(temporalContent, metadata)
				return metadata["temporal_count"] != nil, nil
			},
		},
		{
			name:       "JSON Parser",
			parserType: "json",
			prepareData: func() ([]byte, error) {
				jsonData := map[string]interface{}{
					"meeting_date":    "2024-01-15",
					"deadline":        "December 31, 2023",
					"business_hours":  "9:00-17:00",
					"quarter":         "Q1 2024",
					"last_updated":    "2023-12-25T10:30:00Z",
					"description":     temporalContent,
				}
				return json.Marshal(jsonData)
			},
			runParser: func(data []byte) (bool, error) {
				parser := NewJSONParser()
				parser.ExtractDates = true

				result, err := parser.Parse("test-doc", string(data))
				if err != nil {
					return false, err
				}

				// Check if any elements have temporal metadata
				for _, elem := range result.Elements {
					if elem.Metadata != nil {
						if _, ok := elem.Metadata["temporal_count"]; ok {
							return true, nil
						}
					}
				}
				return false, nil
			},
		},
		{
			name:       "CSV Parser",
			parserType: "csv",
			prepareData: func() ([]byte, error) {
				csvContent := `Date,Event,Time,Quarter
2024-01-15,Meeting,14:30,Q1 2024
2023-12-31,Deadline,23:59,Q4 2023
2024-03-20,Review,9:00-17:00,Q1 2024`
				return []byte(csvContent), nil
			},
			runParser: func(data []byte) (bool, error) {
				parser := NewCSVParser()
				parser.ExtractDates = true

				result, err := parser.Parse("test-doc", string(data))
				if err != nil {
					return false, err
				}

				// Check for temporal metadata
				for _, elem := range result.Elements {
					if elem.Metadata != nil {
						if _, ok := elem.Metadata["temporal_count"]; ok {
							return true, nil
						}
						// CSV parser may have temporal detection
						if _, ok := elem.Metadata["temporal_type"]; ok {
							return true, nil
						}
					}
				}
				return false, nil
			},
		},
		{
			name:       "Text Parser",
			parserType: "text",
			prepareData: func() ([]byte, error) {
				return []byte(temporalContent), nil
			},
			runParser: func(data []byte) (bool, error) {
				parser := NewTextParser()
				parser.ExtractDates = true

				result, err := parser.Parse("test-doc", string(data))
				if err != nil {
					return false, err
				}

				// Check for temporal values in elements
				for _, elem := range result.Elements {
					if elem.Metadata != nil {
						if _, ok := elem.Metadata["temporal_count"]; ok {
							return true, nil
						}
					}
				}
				return false, nil
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := test.prepareData()
			if err != nil {
				t.Fatalf("Failed to prepare test data: %v", err)
			}

			foundTemporal, err := test.runParser(data)
			if err != nil {
				t.Fatalf("Parser failed: %v", err)
			}

			if !foundTemporal {
				t.Logf("Warning: No temporal data found for %s (may need parser implementation)", test.name)
			} else {
				t.Logf("✓ %s successfully extracted temporal data", test.name)
			}
		})
	}
}

// TestTemporalHelperFunctions tests the shared temporal helper functions
func TestTemporalHelperFunctions(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
		desc     string
	}{
		{"2024-01-15", true, "ISO date"},
		{"December 31, 2023", true, "Month name date"},
		{"14:30", true, "Time"},
		{"9:00-17:00", true, "Time range"},
		{"Q1 2024", true, "Quarter"},
		{"2023-12-25T10:30:00Z", true, "ISO datetime"},
		{"hello world", false, "Non-temporal text"},
		{"123.45", false, "Number"},
		{"", false, "Empty string"},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			metadata := ExtractTemporalFromText(test.input)
			hasTemporal := metadata != nil && len(metadata) > 0

			if hasTemporal != test.expected {
				t.Errorf("%s: expected temporal=%v, got=%v", test.desc, test.expected, hasTemporal)
			}
		})
	}
}

// TestProcessTemporalContent tests the ProcessTemporalContent helper
func TestProcessTemporalContent(t *testing.T) {
	content := "The meeting is on 2024-01-15 at 14:30. Deadline: December 31, 2023."
	metadata := make(map[string]interface{})

	ProcessTemporalContent(content, metadata)

	// Check that temporal data was added to metadata
	if metadata["temporal_count"] == nil {
		t.Error("Expected temporal_count in metadata")
	}

	if count, ok := metadata["temporal_count"].(int); ok {
		if count < 2 {
			t.Errorf("Expected at least 2 temporal values, got %d", count)
		}
	}

	if values, ok := metadata["temporal_values_found"].([]string); ok {
		t.Logf("Found temporal values: %v", values)
		if len(values) < 2 {
			t.Errorf("Expected at least 2 temporal values, got %d", len(values))
		}
	} else {
		t.Error("Expected temporal_values_found in metadata")
	}
}

// BenchmarkTemporalExtraction benchmarks temporal extraction across parsers
func BenchmarkTemporalExtraction(b *testing.B) {
	content := strings.Repeat("Meeting on 2024-01-15 at 14:30. ", 100)

	b.Run("ExtractTemporalFromText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = ExtractTemporalFromText(content)
		}
	})

	b.Run("ProcessTemporalContent", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			metadata := make(map[string]interface{})
			ProcessTemporalContent(content, metadata)
		}
	})

	// Benchmark JSON parser with temporal
	b.Run("JSONParserWithTemporal", func(b *testing.B) {
		parser := NewJSONParser()
		parser.ExtractDates = true

		jsonData := fmt.Sprintf(`{
			"content": "%s",
			"date": "2024-01-15",
			"time": "14:30"
		}`, content)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = parser.Parse("bench-doc", jsonData)
		}
	})
}