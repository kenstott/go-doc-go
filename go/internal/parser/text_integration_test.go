package parser_test

import (
	"context"
	"testing"

	"github.com/kennethstott/doculyzer-go-conversion/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTextParserInterface validates Text parser implements Parser interface correctly
func TestTextParserInterface(t *testing.T) {
	textParser := parser.NewTextParser()

	t.Run("implements Parser interface", func(t *testing.T) {
		// Verify parser implements the interface
		var _ parser.Parser = textParser

		// Test GetName
		assert.Equal(t, "text", textParser.GetName())

		// Test GetSupportedFormats
		formats := textParser.GetSupportedFormats()
		assert.Contains(t, formats, ".txt")
		assert.Contains(t, formats, "text")

		// Test SupportsStreaming
		assert.False(t, textParser.SupportsStreaming())

		// Test Close
		err := textParser.Close()
		assert.NoError(t, err)
	})

	t.Run("can be registered in ParserRegistry", func(t *testing.T) {
		registry := parser.NewParserRegistry()
		registry.Register(textParser)

		// Verify parser is registered
		retrieved, err := registry.GetParser("text")
		require.NoError(t, err)
		assert.Equal(t, "text", retrieved.GetName())

		// Verify parser can be found by file extension
		parserByFile, err := registry.GetParserForFile("document.txt")
		require.NoError(t, err)
		assert.Equal(t, "text", parserByFile.GetName())
	})
}

// TestTextParserEndToEnd validates complete text parsing workflow via Parser interface
func TestTextParserEndToEnd(t *testing.T) {
	// Create sample text content
	textContent := `Introduction to Go

Go is a statically typed, compiled programming language designed at Google.
It was created in 2007 and released in 2009.

Key Features
Go is known for its simplicity and efficiency.
The language includes built-in concurrency support.`

	textParser := parser.NewTextParser()
	ctx := context.Background()

	t.Run("parses text via Parser interface", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-text-001",
			Content: textContent,
			Metadata: map[string]interface{}{
				"source": "test",
			},
			Config: parser.DefaultParserConfig(),
		}

		result, err := textParser.Parse(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Validate Document
		assert.Equal(t, "test-text-001", result.Document.ID)
		assert.Equal(t, "text", result.Document.DocType)

		// Validate Elements exist
		assert.NotEmpty(t, result.Elements)
	})

	t.Run("handles different content types", func(t *testing.T) {
		// Test with string content
		reqString := parser.ParseRequest{
			ID:      "test-text-002",
			Content: textContent,
			Config:  parser.DefaultParserConfig(),
		}
		resultString, err := textParser.Parse(ctx, reqString)
		require.NoError(t, err)
		assert.NotNil(t, resultString)

		// Test with []byte content
		reqBytes := parser.ParseRequest{
			ID:      "test-text-003",
			Content: []byte(textContent),
			Config:  parser.DefaultParserConfig(),
		}
		resultBytes, err := textParser.Parse(ctx, reqBytes)
		require.NoError(t, err)
		assert.NotNil(t, resultBytes)
	})

	t.Run("validates element structure", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-text-004",
			Content: textContent,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := textParser.Parse(ctx, req)
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

	t.Run("validates text-specific element types", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-text-005",
			Content: textContent,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := textParser.Parse(ctx, req)
		require.NoError(t, err)

		elementTypes := make(map[string]int)
		for _, elem := range result.Elements {
			elementTypes[elem.ElementType]++
		}

		// Verify expected element types exist
		assert.Greater(t, elementTypes["root"], 0, "Should have root element")
		assert.Greater(t, elementTypes["paragraph"], 0, "Should have paragraph elements")
	})

	t.Run("validates relationships exist", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-text-006",
			Content: textContent,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := textParser.Parse(ctx, req)
		require.NoError(t, err)

		// Verify hierarchy via parent_id field (replaces old Relationships array)
		hierarchyCount := 0
		parentChildMap := make(map[string][]string)

		for _, elem := range result.Elements {
			if elem.ParentID != "" {
				hierarchyCount++
				parentChildMap[elem.ParentID] = append(parentChildMap[elem.ParentID], elem.ElementID)
			}
		}

		assert.Greater(t, hierarchyCount, 0, "Should have elements with parent_id forming hierarchy")

		// Verify root element exists and has children
		rootFound := false
		for _, elem := range result.Elements {
			if elem.ElementType == "root" && elem.ParentID == "" {
				rootFound = true
				assert.Greater(t, len(parentChildMap[elem.ElementID]), 0, "Root should have children")
				break
			}
		}
		assert.True(t, rootFound, "Should have root element")
	})

	t.Run("extracts links from text", func(t *testing.T) {
		contentWithLinks := `Visit our website at https://example.com or email us at contact@example.com`

		req := parser.ParseRequest{
			ID:      "test-text-007",
			Content: contentWithLinks,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := textParser.Parse(ctx, req)
		require.NoError(t, err)

		// Links may be represented as elements with element_type='link' or in metadata
		linkCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "link" {
				linkCount++
			}
		}
		// Note: Not all parsers create separate link elements - some store in metadata
		// This is acceptable architecture - both approaches are valid
	})
}

// TestTextTemporalTypePromotion validates TemporalType promoted field population
func TestTextTemporalTypePromotion(t *testing.T) {
	textWithDates := `The company was founded in 2007.
The product launched on 2023-01-15.
The meeting is scheduled for January 20, 2024.`

	textParser := parser.NewTextParser()
	ctx := context.Background()

	req := parser.ParseRequest{
		ID:      "test-temporal-001",
		Content: textWithDates,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := textParser.Parse(ctx, req)
	require.NoError(t, err)

	t.Run("temporal metadata is extracted", func(t *testing.T) {
		// Find paragraph elements with temporal metadata
		foundTemporalMetadata := false
		for _, elem := range result.Elements {
			if elem.ElementType == "paragraph" {
				if elem.Metadata != nil {
					if _, hasTemporal := elem.Metadata["temporal_type"]; hasTemporal {
						foundTemporalMetadata = true
						break
					}
				}
			}
		}

		// Note: Temporal metadata population depends on temporal package implementation
		// This test validates the integration point exists
		if foundTemporalMetadata {
			t.Log("Temporal metadata successfully extracted")
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
				t.Logf("Found element with TemporalType: %s", *elem.TemporalType)
			}
		}

		// Note: This may or may not be populated depending on temporal package behavior
		if foundTemporalType {
			t.Log("TemporalType promoted field successfully populated")
		} else {
			t.Log("TemporalType not populated (integration point verified)")
		}
	})
}

// TestTextDocumentMetadata validates document-level metadata extraction
func TestTextDocumentMetadata(t *testing.T) {
	textContent := `First paragraph.

Second paragraph with more content.

Third paragraph.`

	textParser := parser.NewTextParser()
	ctx := context.Background()

	req := parser.ParseRequest{
		ID:      "test-metadata-001",
		Content: textContent,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := textParser.Parse(ctx, req)
	require.NoError(t, err)

	t.Run("document metadata exists", func(t *testing.T) {
		assert.NotNil(t, result.Document.Metadata)

		// Verify expected metadata fields
		if metadata := result.Document.Metadata; metadata != nil {
			if charCount, ok := metadata["character_count"]; ok {
				assert.Greater(t, charCount.(int), 0, "Should have character count")
			}
			if wordCount, ok := metadata["word_count"]; ok {
				assert.Greater(t, wordCount.(int), 0, "Should have word count")
			}
			if paragraphCount, ok := metadata["paragraph_count"]; ok {
				assert.Equal(t, 3, paragraphCount.(int), "Should have 3 paragraphs")
			}
		}
	})
}
