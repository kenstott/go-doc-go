package query

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseJSONPath validates JSONPath parsing
func TestParseJSONPath(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected int // number of segments (including root)
		wantErr  bool
	}{
		{"root_only", "$", 1, false},
		{"simple_key", "$.field", 2, false},
		{"nested_keys", "$.field.nested", 3, false},
		{"array_index", "$.field[0]", 3, false},
		{"array_wildcard", "$.field[*]", 3, false},
		{"wildcard_key", "$.*", 2, false},
		{"recursive", "$..field", 2, false},
		{"quoted_key", "$['field']", 2, false},
		{"double_quoted", "$[\"field\"]", 2, false},
		{"complex", "$.store.book[0].title", 5, false},

		// Error cases
		{"empty", "", 0, true},
		{"no_root", "field", 0, true},
		{"unclosed_bracket", "$.field[0", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jp, err := ParseJSONPath(tt.expr)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, jp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, jp)
				assert.Equal(t, tt.expected, len(jp.Segments))
				assert.Equal(t, tt.expr, jp.Expression)
			}
		})
	}
}

// TestParseJSONPath_SegmentTypes validates segment type parsing
func TestParseJSONPath_SegmentTypes(t *testing.T) {
	tests := []struct {
		name         string
		expr         string
		segmentIndex int
		expectedType SegmentType
		expectedKey  string
		expectedIdx  int
	}{
		{"root_segment", "$", 0, SegmentRoot, "", 0},
		{"key_segment", "$.field", 1, SegmentKey, "field", 0},
		{"index_segment", "$[0]", 1, SegmentIndex, "", 0},
		{"wildcard_bracket", "$[*]", 1, SegmentWildcard, "", 0},
		{"wildcard_dot", "$.*", 1, SegmentWildcard, "", 0},
		{"recursive_segment", "$..field", 1, SegmentRecursive, "field", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jp, err := ParseJSONPath(tt.expr)
			require.NoError(t, err)
			require.Greater(t, len(jp.Segments), tt.segmentIndex)

			segment := jp.Segments[tt.segmentIndex]
			assert.Equal(t, tt.expectedType, segment.Type)

			if tt.expectedType == SegmentKey || tt.expectedType == SegmentRecursive {
				assert.Equal(t, tt.expectedKey, segment.Key)
			}

			if tt.expectedType == SegmentIndex {
				assert.Equal(t, tt.expectedIdx, segment.Index)
			}

			if tt.expectedType == SegmentWildcard {
				assert.True(t, segment.IsWildcard)
			}
		})
	}
}

// TestEvaluate_SimpleKey validates simple key evaluation
func TestEvaluate_SimpleKey(t *testing.T) {
	data := map[string]interface{}{
		"name":  "John",
		"age":   30,
		"email": "john@example.com",
	}

	tests := []struct {
		name     string
		expr     string
		expected interface{}
	}{
		{"name", "$.name", "John"},
		{"age", "$.age", 30},
		{"email", "$.email", "john@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jp, err := ParseJSONPath(tt.expr)
			require.NoError(t, err)

			results, err := jp.Evaluate(data)
			require.NoError(t, err)
			require.Len(t, results, 1)
			assert.Equal(t, tt.expected, results[0])
		})
	}
}

// TestEvaluate_NestedKeys validates nested key evaluation
func TestEvaluate_NestedKeys(t *testing.T) {
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"profile": map[string]interface{}{
				"name": "Alice",
				"age":  25,
			},
		},
	}

	jp, err := ParseJSONPath("$.user.profile.name")
	require.NoError(t, err)

	results, err := jp.Evaluate(data)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Alice", results[0])
}

// TestEvaluate_ArrayIndex validates array index evaluation
func TestEvaluate_ArrayIndex(t *testing.T) {
	data := map[string]interface{}{
		"items": []interface{}{"apple", "banana", "cherry"},
	}

	tests := []struct {
		name     string
		expr     string
		expected interface{}
	}{
		{"first", "$.items[0]", "apple"},
		{"second", "$.items[1]", "banana"},
		{"third", "$.items[2]", "cherry"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jp, err := ParseJSONPath(tt.expr)
			require.NoError(t, err)

			results, err := jp.Evaluate(data)
			require.NoError(t, err)
			require.Len(t, results, 1)
			assert.Equal(t, tt.expected, results[0])
		})
	}
}

// TestEvaluate_Wildcard validates wildcard evaluation
func TestEvaluate_Wildcard(t *testing.T) {
	data := map[string]interface{}{
		"users": map[string]interface{}{
			"user1": map[string]interface{}{"name": "Alice", "age": 25},
			"user2": map[string]interface{}{"name": "Bob", "age": 30},
			"user3": map[string]interface{}{"name": "Charlie", "age": 35},
		},
	}

	jp, err := ParseJSONPath("$.users.*")
	require.NoError(t, err)

	results, err := jp.Evaluate(data)
	require.NoError(t, err)
	assert.Len(t, results, 3)
}

// TestEvaluate_WildcardArray validates wildcard on arrays
func TestEvaluate_WildcardArray(t *testing.T) {
	data := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"name": "Item1", "price": 10},
			map[string]interface{}{"name": "Item2", "price": 20},
			map[string]interface{}{"name": "Item3", "price": 30},
		},
	}

	jp, err := ParseJSONPath("$.items[*]")
	require.NoError(t, err)

	results, err := jp.Evaluate(data)
	require.NoError(t, err)
	assert.Len(t, results, 3)
}

// TestEvaluate_RecursiveDescent validates recursive descent
func TestEvaluate_RecursiveDescent(t *testing.T) {
	data := map[string]interface{}{
		"store": map[string]interface{}{
			"book": []interface{}{
				map[string]interface{}{
					"title": "Book1",
					"author": map[string]interface{}{
						"name": "Author1",
					},
				},
				map[string]interface{}{
					"title": "Book2",
					"author": map[string]interface{}{
						"name": "Author2",
					},
				},
			},
		},
	}

	jp, err := ParseJSONPath("$..name")
	require.NoError(t, err)

	results, err := jp.Evaluate(data)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Contains(t, results, "Author1")
	assert.Contains(t, results, "Author2")
}

// TestEvaluate_MissingKey validates behavior with missing keys
func TestEvaluate_MissingKey(t *testing.T) {
	data := map[string]interface{}{
		"name": "John",
		"age":  30,
	}

	jp, err := ParseJSONPath("$.email")
	require.NoError(t, err)

	results, err := jp.Evaluate(data)
	require.NoError(t, err)
	assert.Empty(t, results)
}

// TestExtractFromJSON validates JSON string extraction
func TestExtractFromJSON(t *testing.T) {
	jsonStr := `{"name": "John", "age": 30, "email": "john@example.com"}`

	results, err := ExtractFromJSON(jsonStr, "$.name")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "John", results[0])
}

// TestExtractFromJSON_NestedJSON validates nested JSON extraction
func TestExtractFromJSON_NestedJSON(t *testing.T) {
	jsonStr := `{
		"user": {
			"profile": {
				"name": "Alice",
				"age": 25
			}
		}
	}`

	results, err := ExtractFromJSON(jsonStr, "$.user.profile.age")
	require.NoError(t, err)
	require.Len(t, results, 1)

	// JSON unmarshaling converts numbers to float64
	assert.Equal(t, float64(25), results[0])
}

// TestExtractFromJSON_Array validates array extraction
func TestExtractFromJSON_Array(t *testing.T) {
	jsonStr := `{"items": ["apple", "banana", "cherry"]}`

	results, err := ExtractFromJSON(jsonStr, "$.items[1]")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "banana", results[0])
}

// TestExtractFromJSON_InvalidJSON validates error handling
func TestExtractFromJSON_InvalidJSON(t *testing.T) {
	jsonStr := `{invalid json}`

	results, err := ExtractFromJSON(jsonStr, "$.name")
	assert.Error(t, err)
	assert.Nil(t, results)
}

// TestCompareJSONPathValue validates JSONPath value comparison
func TestCompareJSONPathValue(t *testing.T) {
	jsonStr := `{
		"metadata": {
			"font_size": 12,
			"color": "#000000",
			"bold": true
		}
	}`

	tests := []struct {
		name     string
		path     string
		operator Operator
		value    interface{}
		expected bool
	}{
		{"equal_int", "$.metadata.font_size", OpEqual, float64(12), true},
		{"equal_string", "$.metadata.color", OpEqual, "#000000", true},
		{"not_equal", "$.metadata.font_size", OpNotEqual, float64(10), true},
		{"greater_than", "$.metadata.font_size", OpGreaterThan, float64(10), true},
		{"less_than", "$.metadata.font_size", OpLessThan, float64(15), true},
		{"greater_equal", "$.metadata.font_size", OpGreaterThanOrEqual, float64(12), true},
		{"less_equal", "$.metadata.font_size", OpLessThanOrEqual, float64(12), true},

		// False cases
		{"equal_false", "$.metadata.font_size", OpEqual, float64(10), false},
		{"greater_false", "$.metadata.font_size", OpGreaterThan, float64(15), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CompareJSONPathValue(jsonStr, tt.path, tt.operator, tt.value)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCompareJSONPathValue_NullHandling validates null handling
func TestCompareJSONPathValue_NullHandling(t *testing.T) {
	jsonStr := `{"metadata": {"font_size": 12}}`

	// Missing field with IsNull
	result, err := CompareJSONPathValue(jsonStr, "$.metadata.missing", OpIsNull, nil)
	require.NoError(t, err)
	assert.True(t, result)

	// Missing field with IsNotNull
	result, err = CompareJSONPathValue(jsonStr, "$.metadata.missing", OpIsNotNull, nil)
	require.NoError(t, err)
	assert.False(t, result)

	// Existing field with IsNotNull
	result, err = CompareJSONPathValue(jsonStr, "$.metadata.font_size", OpIsNotNull, nil)
	require.NoError(t, err)
	assert.True(t, result)
}

// TestValidateJSONPath validates path validation
func TestValidateJSONPath(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{"valid_simple", "$.field", false},
		{"valid_nested", "$.field.nested", false},
		{"valid_array", "$.field[0]", false},
		{"valid_wildcard", "$.*", false},

		{"invalid_empty", "", true},
		{"invalid_no_root", "field", true},
		{"invalid_unclosed", "$.field[0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJSONPath(tt.expr)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestJSONPath_ComplexDocument validates complex document queries
func TestJSONPath_ComplexDocument(t *testing.T) {
	jsonStr := `{
		"store": {
			"book": [
				{
					"category": "reference",
					"author": "Nigel Rees",
					"title": "Sayings of the Century",
					"price": 8.95
				},
				{
					"category": "fiction",
					"author": "Evelyn Waugh",
					"title": "Sword of Honour",
					"price": 12.99
				},
				{
					"category": "fiction",
					"author": "Herman Melville",
					"title": "Moby Dick",
					"isbn": "0-553-21311-3",
					"price": 8.99
				}
			],
			"bicycle": {
				"color": "red",
				"price": 19.95
			}
		}
	}`

	tests := []struct {
		name        string
		path        string
		expectCount int
	}{
		{"all_books", "$.store.book[*]", 3},
		{"first_book", "$.store.book[0]", 1},
		{"bicycle", "$.store.bicycle", 1},
		{"all_authors", "$.store.book[*]", 3}, // Each book contains an author
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ExtractFromJSON(jsonStr, tt.path)
			require.NoError(t, err)
			assert.Len(t, results, tt.expectCount)
		})
	}
}

// TestJSONPath_RealWorldMetadata validates real-world metadata queries
func TestJSONPath_RealWorldMetadata(t *testing.T) {
	// Simulates UDML element metadata field
	metadata := map[string]interface{}{
		"font_size":   12,
		"font_family": "Arial",
		"color":       "#000000",
		"bold":        true,
		"italic":      false,
		"styles": map[string]interface{}{
			"margin_top":    10,
			"margin_bottom": 10,
			"padding_left":  5,
		},
	}

	metadataJSON, err := json.Marshal(metadata)
	require.NoError(t, err)

	tests := []struct {
		name     string
		path     string
		expected interface{}
	}{
		{"font_size", "$.font_size", float64(12)},
		{"font_family", "$.font_family", "Arial"},
		{"nested_margin", "$.styles.margin_top", float64(10)},
		{"bool_value", "$.bold", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ExtractFromJSON(string(metadataJSON), tt.path)
			require.NoError(t, err)
			require.Len(t, results, 1)
			assert.Equal(t, tt.expected, results[0])
		})
	}
}

// TestJSONPath_ContentLocation validates content_location queries
func TestJSONPath_ContentLocation(t *testing.T) {
	// Simulates UDML element content_location field
	contentLocation := map[string]interface{}{
		"x":      100,
		"y":      200,
		"width":  500,
		"height": 300,
		"page":   1,
	}

	locationJSON, err := json.Marshal(contentLocation)
	require.NoError(t, err)

	tests := []struct {
		name     string
		path     string
		operator Operator
		value    interface{}
		expected bool
	}{
		{"x_greater_than_50", "$.x", OpGreaterThan, float64(50), true},
		{"width_equal_500", "$.width", OpEqual, float64(500), true},
		{"page_equal_1", "$.page", OpEqual, float64(1), true},
		{"height_less_than_400", "$.height", OpLessThan, float64(400), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CompareJSONPathValue(string(locationJSON), tt.path, tt.operator, tt.value)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestJSONPath_String validates String() method
func TestJSONPath_String(t *testing.T) {
	expr := "$.field.nested[0]"
	jp, err := ParseJSONPath(expr)
	require.NoError(t, err)

	assert.Equal(t, expr, jp.String())
}
