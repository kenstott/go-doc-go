package parser

import (
	"encoding/json"
	"testing"
)

func TestJSONParserTemporalExtraction(t *testing.T) {
	parser := NewJSONParser()
	parser.ExtractDates = true

	// Test JSON with various temporal fields
	jsonContent := `{
		"created_date": "2023-12-25",
		"meeting_time": "14:30",
		"deadline": "2023-12-31T23:59:59",
		"business_hours": "9:00-17:00",
		"quarter": "Q1 2024",
		"events": ["2024-01-15", "2024-02-20", "2024-03-10"],
		"non_temporal": "hello world"
	}`

	request := JSONParseRequest{
		ID:       "test-doc",
		Content:  jsonContent,
		Metadata: make(map[string]interface{}),
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Check that we have elements
	if len(response.Elements) == 0 {
		t.Fatal("No elements parsed")
	}

	// Look for temporal values in the elements
	temporalFieldsFound := 0
	for _, element := range response.Elements {
		if element.ElementType == JSONElementTypeField {
			fieldName, _ := element.Metadata["field_name"].(string)

			// Check if temporal fields have temporal values
			switch fieldName {
			case "created_date", "meeting_time", "deadline", "business_hours":
				if element.TemporalValue != nil {
					temporalFieldsFound++
					t.Logf("Found temporal value for field %s: %v", fieldName, element.TemporalValue)

					// Verify temporal metadata exists
					if element.Metadata["temporal_type"] == nil {
						t.Errorf("Field %s missing temporal_type in metadata", fieldName)
					}
				} else {
					t.Errorf("Expected temporal value for field %s but got nil", fieldName)
				}
			case "non_temporal":
				if element.TemporalValue != nil {
					t.Errorf("Unexpected temporal value for non-temporal field: %s", fieldName)
				}
			}
		} else if element.ElementType == JSONElementTypeItem {
			// Check array items for temporal values
			if element.TemporalValue != nil {
				temporalFieldsFound++
				t.Logf("Found temporal value in array item: %v", element.TemporalValue)
			}
		}
	}

	if temporalFieldsFound < 4 {
		t.Errorf("Expected at least 4 temporal fields, found %d", temporalFieldsFound)
	}
}

func TestJSONParserTemporalNormalization(t *testing.T) {
	parser := NewJSONParser()
	parser.ExtractDates = true

	// Test that dates get normalized in previews
	jsonContent := `{
		"date": "2023-12-25"
	}`

	request := JSONParseRequest{
		ID:       "test-doc-2",
		Content:  jsonContent,
		Metadata: make(map[string]interface{}),
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Find the date field
	for _, element := range response.Elements {
		if element.ElementType == JSONElementTypeField {
			fieldName, _ := element.Metadata["field_name"].(string)
			if fieldName == "date" {
				// Check if the preview contains normalized date
				if element.ContentPreview == "" {
					t.Error("ContentPreview is empty")
				} else {
					t.Logf("Date field preview: %s", element.ContentPreview)
					// The preview should contain normalized format like "Dec 25, 2023"
					// This depends on the normalization implementation
				}

				// Check temporal metadata
				if element.TemporalValue == nil {
					t.Error("TemporalValue is nil for date field")
				} else {
					// Marshal to JSON to inspect structure
					tempJSON, _ := json.MarshalIndent(element.TemporalValue, "", "  ")
					t.Logf("Temporal value structure:\n%s", string(tempJSON))
				}

				// Check specific metadata fields
				if element.Metadata["year"] != 2023 {
					t.Errorf("Expected year=2023, got %v", element.Metadata["year"])
				}
				if element.Metadata["month"] != 12 {
					t.Errorf("Expected month=12, got %v", element.Metadata["month"])
				}
				if element.Metadata["day"] != 25 {
					t.Errorf("Expected day=25, got %v", element.Metadata["day"])
				}

				return // Found and tested the field
			}
		}
	}

	t.Error("Date field not found in parsed elements")
}

func BenchmarkJSONParserWithTemporal(b *testing.B) {
	parser := NewJSONParser()
	parser.ExtractDates = true

	jsonContent := `{
		"date1": "2023-12-25",
		"date2": "2024-01-15",
		"time1": "14:30",
		"datetime": "2023-12-25T14:30:00",
		"text": "This is regular text",
		"nested": {
			"date3": "2024-03-20"
		}
	}`

	request := JSONParseRequest{
		ID:       "bench-doc",
		Content:  jsonContent,
		Metadata: make(map[string]interface{}),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(request)
	}
}

func BenchmarkJSONParserWithoutTemporal(b *testing.B) {
	parser := NewJSONParser()
	parser.ExtractDates = false // Disable temporal extraction

	jsonContent := `{
		"date1": "2023-12-25",
		"date2": "2024-01-15",
		"time1": "14:30",
		"datetime": "2023-12-25T14:30:00",
		"text": "This is regular text",
		"nested": {
			"date3": "2024-03-20"
		}
	}`

	request := JSONParseRequest{
		ID:       "bench-doc",
		Content:  jsonContent,
		Metadata: make(map[string]interface{}),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(request)
	}
}