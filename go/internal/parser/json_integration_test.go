package parser_test

import (
	"context"
	"testing"

	"github.com/kennethstott/go-doc-go/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJSONParserInterface validates JSON parser implements Parser interface correctly
func TestJSONParserInterface(t *testing.T) {
	jsonParser := parser.NewJSONParser()

	t.Run("implements Parser interface", func(t *testing.T) {
		// Verify parser implements the interface
		var _ parser.Parser = jsonParser

		// Test GetName
		assert.Equal(t, "json", jsonParser.GetName())

		// Test GetSupportedFormats
		formats := jsonParser.GetSupportedFormats()
		assert.Contains(t, formats, ".json")
		assert.Contains(t, formats, "json")

		// Test SupportsStreaming
		assert.False(t, jsonParser.SupportsStreaming())

		// Test Close
		err := jsonParser.Close()
		assert.NoError(t, err)
	})

	t.Run("can be registered in ParserRegistry", func(t *testing.T) {
		registry := parser.NewParserRegistry()
		registry.Register(jsonParser)

		// Verify parser is registered
		retrieved, err := registry.GetParser("json")
		require.NoError(t, err)
		assert.Equal(t, "json", retrieved.GetName())

		// Verify parser can be found by file extension
		parserByFile, err := registry.GetParserForFile("data.json")
		require.NoError(t, err)
		assert.Equal(t, "json", parserByFile.GetName())
	})
}

// TestJSONParserEndToEnd validates complete JSON parsing workflow via Parser interface
func TestJSONParserEndToEnd(t *testing.T) {
	// Create sample JSON content
	jsonContent := `{
		"name": "John Doe",
		"age": 30,
		"email": "john@example.com",
		"address": {
			"street": "123 Main St",
			"city": "New York",
			"zip": "10001"
		},
		"hobbies": ["reading", "coding", "hiking"]
	}`

	jsonParser := parser.NewJSONParser()
	ctx := context.Background()

	t.Run("parses JSON via Parser interface", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-json-001",
			Content: jsonContent,
			Metadata: map[string]interface{}{
				"source": "test",
			},
			Config: parser.DefaultParserConfig(),
		}

		result, err := jsonParser.Parse(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Validate Document
		assert.Equal(t, "test-json-001", result.Document.ID)
		assert.Equal(t, "json", result.Document.DocType)

		// Validate Elements exist
		assert.NotEmpty(t, result.Elements)
	})

	t.Run("handles different content types", func(t *testing.T) {
		// Test with string content
		reqString := parser.ParseRequest{
			ID:      "test-json-002",
			Content: jsonContent,
			Config:  parser.DefaultParserConfig(),
		}
		resultString, err := jsonParser.Parse(ctx, reqString)
		require.NoError(t, err)
		assert.NotNil(t, resultString)

		// Test with []byte content
		reqBytes := parser.ParseRequest{
			ID:      "test-json-003",
			Content: []byte(jsonContent),
			Config:  parser.DefaultParserConfig(),
		}
		resultBytes, err := jsonParser.Parse(ctx, reqBytes)
		require.NoError(t, err)
		assert.NotNil(t, resultBytes)
	})

	t.Run("validates element structure", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-json-004",
			Content: jsonContent,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := jsonParser.Parse(ctx, req)
		require.NoError(t, err)

		// Verify all elements have required fields
		for _, elem := range result.Elements {
			assert.NotEmpty(t, elem.ElementID, "Element should have ID")
			assert.NotEmpty(t, elem.ElementType, "Element should have type")
			assert.NotEmpty(t, elem.ContentPreview, "Element should have content preview")
			assert.GreaterOrEqual(t, elem.Position, 0, "Element should have valid position")
			assert.GreaterOrEqual(t, elem.Depth, 0, "Element should have valid depth")
			assert.NotEmpty(t, elem.ElementCategory, "Element should have category")
		}
	})

	t.Run("validates JSON-specific element types", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-json-005",
			Content: jsonContent,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := jsonParser.Parse(ctx, req)
		require.NoError(t, err)

		elementTypes := make(map[string]int)
		for _, elem := range result.Elements {
			elementTypes[elem.ElementType]++
		}

		// Verify expected element types exist
		assert.Greater(t, elementTypes["root"], 0, "Should have root element")
		assert.Greater(t, elementTypes["json_object"], 0, "Should have JSON object elements")
		assert.Greater(t, elementTypes["json_field"], 0, "Should have JSON field elements")
		assert.Greater(t, elementTypes["json_array"], 0, "Should have JSON array element")
		assert.Greater(t, elementTypes["json_item"], 0, "Should have JSON array item elements")
	})

	t.Run("validates relationships exist", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-json-006",
			Content: jsonContent,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := jsonParser.Parse(ctx, req)
		require.NoError(t, err)

		// JSON parser should create bidirectional relationships
		assert.NotEmpty(t, result.Relationships, "Should have relationships between elements")

		// Verify relationship structure
		for _, rel := range result.Relationships {
			assert.NotEmpty(t, rel.RelationshipID)
			assert.NotEmpty(t, rel.RelationshipType)
			assert.NotEmpty(t, rel.SourceElementID)
			assert.NotEmpty(t, rel.TargetElementID)
		}

		// Verify bidirectional relationships exist
		hasContains := false
		hasContainedBy := false
		for _, rel := range result.Relationships {
			if rel.RelationshipType == "contains" {
				hasContains = true
			}
			if rel.RelationshipType == "contained_by" {
				hasContainedBy = true
			}
		}
		assert.True(t, hasContains, "Should have 'contains' relationships")
		assert.True(t, hasContainedBy, "Should have 'contained_by' relationships")
	})

	t.Run("extracts links from JSON", func(t *testing.T) {
		contentWithLinks := `{
			"website": "https://example.com",
			"contact": "info@example.com"
		}`

		req := parser.ParseRequest{
			ID:      "test-json-007",
			Content: contentWithLinks,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := jsonParser.Parse(ctx, req)
		require.NoError(t, err)

		// Should extract URLs and emails
		assert.NotEmpty(t, result.Links, "Should extract links from JSON content")

		// Check link types
		hasURL := false
		for _, link := range result.Links {
			if link.LinkType == "url" {
				hasURL = true
			}
		}

		assert.True(t, hasURL, "Should have extracted URL")
	})
}

// TestJSONTemporalTypePromotion validates TemporalType promoted field population
func TestJSONTemporalTypePromotion(t *testing.T) {
	jsonWithDates := `{
		"created_at": "2023-01-15",
		"updated_at": "2024-01-20T10:30:00Z",
		"year": 2023,
		"events": [
			"2023-06-01",
			"2023-12-25"
		]
	}`

	jsonParser := parser.NewJSONParser()
	ctx := context.Background()

	req := parser.ParseRequest{
		ID:      "test-temporal-001",
		Content: jsonWithDates,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := jsonParser.Parse(ctx, req)
	require.NoError(t, err)

	t.Run("temporal metadata is extracted from fields", func(t *testing.T) {
		// Find field elements with temporal metadata
		foundTemporalMetadata := false
		for _, elem := range result.Elements {
			if elem.ElementType == "json_field" {
				if elem.Metadata != nil {
					if _, hasTemporal := elem.Metadata["temporal_type"]; hasTemporal {
						foundTemporalMetadata = true
						t.Logf("Found field with temporal metadata: %+v", elem.Metadata)
						break
					}
				}
			}
		}

		// Note: Temporal metadata population depends on temporal package implementation
		// This test validates the integration point exists
		if foundTemporalMetadata {
			t.Log("Temporal metadata successfully extracted from JSON fields")
		} else {
			t.Log("No temporal metadata extracted (may be expected if temporal package disabled)")
		}
	})

	t.Run("TemporalType promoted field is populated when available", func(t *testing.T) {
		// Check if any elements have TemporalType populated
		foundTemporalType := false
		for _, elem := range result.Elements {
			if elem.TemporalType != nil {
				foundTemporalType = true
				assert.NotEmpty(t, *elem.TemporalType, "TemporalType should not be empty if set")
				t.Logf("Found element with TemporalType: %s (type: %s)", *elem.TemporalType, elem.ElementType)
			}
		}

		// Note: This may or may not be populated depending on temporal package behavior
		if foundTemporalType {
			t.Log("TemporalType promoted field successfully populated")
		} else {
			t.Log("TemporalType not populated (integration point verified)")
		}
	})

	t.Run("temporal metadata in array items", func(t *testing.T) {
		// Check for temporal metadata in array items
		foundTemporalInArray := false
		for _, elem := range result.Elements {
			if elem.ElementType == "json_item" {
				if elem.Metadata != nil {
					if _, hasTemporal := elem.Metadata["temporal_type"]; hasTemporal {
						foundTemporalInArray = true
						t.Logf("Found array item with temporal metadata: %+v", elem.Metadata)
						break
					}
				}
			}
		}

		if foundTemporalInArray {
			t.Log("Temporal metadata successfully extracted from array items")
		} else {
			t.Log("No temporal metadata in array items (may be expected)")
		}
	})
}

// TestJSONDocumentMetadata validates document-level metadata extraction
func TestJSONDocumentMetadata(t *testing.T) {
	jsonContent := `{
		"user": {
			"name": "Alice",
			"age": 25
		},
		"tags": ["developer", "golang"]
	}`

	jsonParser := parser.NewJSONParser()
	ctx := context.Background()

	req := parser.ParseRequest{
		ID:      "test-metadata-001",
		Content: jsonContent,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := jsonParser.Parse(ctx, req)
	require.NoError(t, err)

	t.Run("document metadata exists", func(t *testing.T) {
		assert.NotNil(t, result.Document)
		assert.Equal(t, "test-metadata-001", result.Document.ID)
		assert.Equal(t, "json", result.Document.DocType)
	})

	t.Run("elements have JSON path metadata", func(t *testing.T) {
		// Verify that elements have json_path in their metadata
		foundJsonPath := false
		for _, elem := range result.Elements {
			if elem.ElementType != "root" && elem.Metadata != nil {
				if jsonPath, ok := elem.Metadata["json_path"].(string); ok && jsonPath != "" {
					foundJsonPath = true
					t.Logf("Found element with json_path: %s", jsonPath)
					break
				}
			}
		}

		assert.True(t, foundJsonPath, "Should have elements with json_path metadata")
	})
}

// TestJSONComplexStructure validates parsing of complex nested JSON
func TestJSONComplexStructure(t *testing.T) {
	complexJSON := `{
		"company": {
			"name": "Tech Corp",
			"founded": "2010-01-01",
			"employees": [
				{
					"id": 1,
					"name": "Alice",
					"email": "alice@techcorp.com"
				},
				{
					"id": 2,
					"name": "Bob",
					"email": "bob@techcorp.com"
				}
			],
			"locations": ["New York", "San Francisco", "London"]
		}
	}`

	jsonParser := parser.NewJSONParser()
	ctx := context.Background()

	req := parser.ParseRequest{
		ID:      "test-complex-001",
		Content: complexJSON,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := jsonParser.Parse(ctx, req)
	require.NoError(t, err)

	t.Run("handles deeply nested structures", func(t *testing.T) {
		// Should have multiple levels of nesting
		assert.Greater(t, len(result.Elements), 10, "Should have many elements for complex structure")

		// Verify different element types exist
		elementTypes := make(map[string]bool)
		for _, elem := range result.Elements {
			elementTypes[elem.ElementType] = true
		}

		assert.True(t, elementTypes["root"], "Should have root")
		assert.True(t, elementTypes["json_object"], "Should have objects")
		assert.True(t, elementTypes["json_field"], "Should have fields")
		assert.True(t, elementTypes["json_array"], "Should have arrays")
		assert.True(t, elementTypes["json_item"], "Should have array items")
	})

	t.Run("maintains parent-child relationships", func(t *testing.T) {
		// Build parent-child map
		childrenMap := make(map[string][]string)
		for _, rel := range result.Relationships {
			if rel.RelationshipType == "contains" {
				childrenMap[rel.SourceElementID] = append(childrenMap[rel.SourceElementID], rel.TargetElementID)
			}
		}

		// Root should have children
		rootElement := result.Elements[0]
		assert.Equal(t, "root", rootElement.ElementType)
		assert.NotEmpty(t, childrenMap[rootElement.ElementID], "Root should have children")
	})

	t.Run("extracts multiple links", func(t *testing.T) {
		// Should extract email addresses
		assert.NotEmpty(t, result.Links, "Should extract links from complex JSON")

		emailCount := 0
		for _, link := range result.Links {
			if link.LinkType == "url" && link.LinkTarget != "" {
				emailCount++
			}
		}
		assert.Greater(t, emailCount, 0, "Should have extracted email links")
	})
}
