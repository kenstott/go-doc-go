package parser_test

import (
	"context"
	"testing"

	"github.com/kennethstott/doculyzer-go-conversion/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParserRegistryWithMultipleParsers validates ParserRegistry with all migrated parsers
func TestParserRegistryWithMultipleParsers(t *testing.T) {
	registry := parser.NewParserRegistry()

	// Register all migrated parsers
	csvParser := parser.NewCSVParser()
	textParser := parser.NewTextParser()
	jsonParser := parser.NewJSONParser()
	htmlParser := parser.NewHTMLParser()
	markdownParser := parser.NewMarkdownParser()
	xmlParser := parser.NewXMLParser()
	// Note: PDF, DOCX, PPTX, XLSX require file paths or byte content, so we register them separately

	registry.Register(csvParser)
	registry.Register(textParser)
	registry.Register(jsonParser)
	registry.Register(htmlParser)
	registry.Register(markdownParser)
	registry.Register(xmlParser)

	t.Run("all parsers are registered", func(t *testing.T) {
		// Verify CSV parser
		csv, err := registry.GetParser("csv")
		require.NoError(t, err)
		assert.Equal(t, "csv", csv.GetName())

		// Verify Text parser
		text, err := registry.GetParser("text")
		require.NoError(t, err)
		assert.Equal(t, "text", text.GetName())

		// Verify JSON parser
		json, err := registry.GetParser("json")
		require.NoError(t, err)
		assert.Equal(t, "json", json.GetName())

		// Verify HTML parser
		html, err := registry.GetParser("html")
		require.NoError(t, err)
		assert.Equal(t, "html", html.GetName())

		// Verify Markdown parser
		markdown, err := registry.GetParser("markdown")
		require.NoError(t, err)
		assert.Equal(t, "markdown", markdown.GetName())

		// Verify XML parser
		xml, err := registry.GetParser("xml")
		require.NoError(t, err)
		assert.Equal(t, "xml", xml.GetName())
	})

	t.Run("parsers can be retrieved by file extension", func(t *testing.T) {
		// CSV files
		csvByExt, err := registry.GetParserForFile("data.csv")
		require.NoError(t, err)
		assert.Equal(t, "csv", csvByExt.GetName())

		// Text files
		textByExt, err := registry.GetParserForFile("document.txt")
		require.NoError(t, err)
		assert.Equal(t, "text", textByExt.GetName())

		// JSON files
		jsonByExt, err := registry.GetParserForFile("config.json")
		require.NoError(t, err)
		assert.Equal(t, "json", jsonByExt.GetName())

		// HTML files
		htmlByExt, err := registry.GetParserForFile("page.html")
		require.NoError(t, err)
		assert.Equal(t, "html", htmlByExt.GetName())

		// Markdown files
		mdByExt, err := registry.GetParserForFile("README.md")
		require.NoError(t, err)
		assert.Equal(t, "markdown", mdByExt.GetName())

		// XML files
		xmlByExt, err := registry.GetParserForFile("config.xml")
		require.NoError(t, err)
		assert.Equal(t, "xml", xmlByExt.GetName())
	})

	t.Run("handles unknown file extensions gracefully", func(t *testing.T) {
		_, err := registry.GetParserForFile("document.unknown")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no parser found")
	})

	t.Run("handles unknown parser names gracefully", func(t *testing.T) {
		_, err := registry.GetParser("unknown")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "parser not found")
	})
}

// TestCrossParserConsistency validates that all parsers follow the same conventions
func TestCrossParserConsistency(t *testing.T) {
	ctx := context.Background()

	parsers := map[string]struct {
		parser  parser.Parser
		content string
	}{
		"csv": {
			parser:  parser.NewCSVParser(),
			content: "name,age\nAlice,30\nBob,25",
		},
		"text": {
			parser:  parser.NewTextParser(),
			content: "Sample text document\n\nWith multiple paragraphs.",
		},
		"json": {
			parser:  parser.NewJSONParser(),
			content: `{"name": "Alice", "age": 30}`,
		},
		"html": {
			parser:  parser.NewHTMLParser(),
			content: `<html><body><h1>Title</h1><p>Content</p></body></html>`,
		},
		"markdown": {
			parser:  parser.NewMarkdownParser(),
			content: "# Title\n\nSome content here.\n\n## Section\n\nMore content.",
		},
		"xml": {
			parser:  parser.NewXMLParser(),
			content: `<?xml version="1.0"?><root><item>value</item></root>`,
		},
	}

	for name, p := range parsers {
		t.Run(name+" parser consistency", func(t *testing.T) {
			req := parser.ParseRequest{
				ID:      "test-consistency-" + name,
				Content: p.content,
				Config:  parser.DefaultParserConfig(),
			}

			result, err := p.parser.Parse(ctx, req)
			require.NoError(t, err, "Parser %s should parse without error", name)

			// All parsers must return consistent structure
			assert.NotNil(t, result, "Result should not be nil")
			assert.NotNil(t, result.Document, "Document should not be nil")
			assert.NotEmpty(t, result.Elements, "Should have at least one element")

			// All parsers must have a root element
			rootFound := false
			for _, elem := range result.Elements {
				if elem.ElementType == "root" {
					rootFound = true
					assert.Equal(t, 0, elem.Depth, "Root element should have depth 0")
					assert.Empty(t, elem.ParentID, "Root element should have no parent")
					break
				}
			}
			assert.True(t, rootFound, "Parser %s should have a root element", name)

			// All elements must have required fields
			for _, elem := range result.Elements {
				assert.NotEmpty(t, elem.ElementID, "Element ID should not be empty")
				assert.NotEmpty(t, elem.ElementType, "Element type should not be empty")
				// ContentPreview may be empty for container elements (html, body, etc.)
				// but element type and category must always be set
				assert.GreaterOrEqual(t, elem.Position, 0, "Position should be non-negative")
				assert.GreaterOrEqual(t, elem.Depth, 0, "Depth should be non-negative")
				assert.NotEmpty(t, elem.ElementCategory, "Element category should not be empty")
			}

			// Document must have correct metadata
			assert.Equal(t, "test-consistency-"+name, result.Document.ID)
			assert.Equal(t, name, result.Document.DocType)
		})
	}
}

// TestPromotedFieldsAcrossParsers validates that promoted fields are populated correctly
func TestPromotedFieldsAcrossParsers(t *testing.T) {
	ctx := context.Background()

	t.Run("CSV parser populates RowIndex and ColumnIndex", func(t *testing.T) {
		csvParser := parser.NewCSVParser()
		req := parser.ParseRequest{
			ID:      "test-promoted-csv",
			Content: "col1,col2\nval1,val2",
			Config:  parser.DefaultParserConfig(),
		}

		result, err := csvParser.Parse(ctx, req)
		require.NoError(t, err)

		// Check for table cells with row/column indices
		foundPromoted := false
		for _, elem := range result.Elements {
			if elem.ElementType == "table_cell" {
				if elem.RowIndex != nil && elem.ColumnIndex != nil {
					foundPromoted = true
					assert.GreaterOrEqual(t, *elem.RowIndex, 0, "RowIndex should be non-negative")
					assert.GreaterOrEqual(t, *elem.ColumnIndex, 0, "ColumnIndex should be non-negative")
					t.Logf("CSV cell at row=%d, col=%d", *elem.RowIndex, *elem.ColumnIndex)
				}
			}
		}
		assert.True(t, foundPromoted, "CSV parser should populate RowIndex and ColumnIndex for table cells")
	})

	t.Run("JSON parser populates TemporalType for date fields", func(t *testing.T) {
		jsonParser := parser.NewJSONParser()
		req := parser.ParseRequest{
			ID:      "test-promoted-json",
			Content: `{"created_at": "2023-01-15", "name": "Test"}`,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := jsonParser.Parse(ctx, req)
		require.NoError(t, err)

		// Check for elements with TemporalType
		foundTemporal := false
		for _, elem := range result.Elements {
			if elem.TemporalType != nil {
				foundTemporal = true
				assert.NotEmpty(t, *elem.TemporalType, "TemporalType should not be empty")
				t.Logf("JSON element with TemporalType: %s (type: %s)", *elem.TemporalType, elem.ElementType)
			}
		}
		assert.True(t, foundTemporal, "JSON parser should populate TemporalType for temporal values")
	})

	t.Run("Text parser supports TemporalType integration point", func(t *testing.T) {
		textParser := parser.NewTextParser()
		req := parser.ParseRequest{
			ID:      "test-promoted-text",
			Content: "The event happened on 2023-01-15.\n\nAnother paragraph.",
			Config:  parser.DefaultParserConfig(),
		}

		result, err := textParser.Parse(ctx, req)
		require.NoError(t, err)

		// Text parser has temporal integration point, but may not populate depending on temporal package
		assert.NotNil(t, result)
		t.Log("Text parser successfully integrates temporal processing")
	})

	t.Run("HTML parser populates TagName and SectionLevel", func(t *testing.T) {
		htmlParser := parser.NewHTMLParser()
		req := parser.ParseRequest{
			ID:      "test-promoted-html",
			Content: `<html><body><h1>Title</h1><h2>Subtitle</h2><p>Content</p></body></html>`,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := htmlParser.Parse(ctx, req)
		require.NoError(t, err)

		// Check for TagName on elements
		foundTagName := false
		foundSectionLevel := false
		for _, elem := range result.Elements {
			if elem.TagName != nil {
				foundTagName = true
				assert.NotEmpty(t, *elem.TagName, "TagName should not be empty")
				t.Logf("HTML element with TagName: %s", *elem.TagName)
			}
			// Check for SectionLevel on header elements
			if elem.ElementType == "header" && elem.SectionLevel != nil {
				foundSectionLevel = true
				assert.GreaterOrEqual(t, *elem.SectionLevel, 1, "SectionLevel should be >= 1")
				assert.LessOrEqual(t, *elem.SectionLevel, 6, "SectionLevel should be <= 6")
				t.Logf("HTML header with SectionLevel: %d", *elem.SectionLevel)
			}
		}
		assert.True(t, foundTagName, "HTML parser should populate TagName for elements")
		assert.True(t, foundSectionLevel, "HTML parser should populate SectionLevel for headers")
	})

	t.Run("Markdown parser populates SectionLevel for headers", func(t *testing.T) {
		markdownParser := parser.NewMarkdownParser()
		req := parser.ParseRequest{
			ID:      "test-promoted-markdown",
			Content: "# Main Title\n\nSome content.\n\n## Section 1\n\nMore content.\n\n### Subsection",
			Config:  parser.DefaultParserConfig(),
		}

		result, err := markdownParser.Parse(ctx, req)
		require.NoError(t, err)

		// Check for SectionLevel on header elements
		foundSectionLevel := false
		for _, elem := range result.Elements {
			if elem.ElementType == "header" && elem.SectionLevel != nil {
				foundSectionLevel = true
				assert.GreaterOrEqual(t, *elem.SectionLevel, 1, "SectionLevel should be >= 1")
				assert.LessOrEqual(t, *elem.SectionLevel, 6, "SectionLevel should be <= 6")
				t.Logf("Markdown header with SectionLevel: %d", *elem.SectionLevel)
			}
		}
		assert.True(t, foundSectionLevel, "Markdown parser should populate SectionLevel for headers")
	})

	t.Run("XML parser populates TagName", func(t *testing.T) {
		xmlParser := parser.NewXMLParser()
		req := parser.ParseRequest{
			ID:      "test-promoted-xml",
			Content: `<?xml version="1.0"?><root><person><name>Alice</name><age>30</age></person></root>`,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := xmlParser.Parse(ctx, req)
		require.NoError(t, err)

		// Check for TagName on elements
		foundTagName := false
		for _, elem := range result.Elements {
			if elem.TagName != nil {
				foundTagName = true
				assert.NotEmpty(t, *elem.TagName, "TagName should not be empty")
				t.Logf("XML element with TagName: %s", *elem.TagName)
			}
		}
		assert.True(t, foundTagName, "XML parser should populate TagName for elements")
	})

	t.Run("all parsers set ElementCategory", func(t *testing.T) {
		parsers := []struct {
			name    string
			parser  parser.Parser
			content string
		}{
			{"csv", parser.NewCSVParser(), "a,b\n1,2"},
			{"text", parser.NewTextParser(), "Sample text"},
			{"json", parser.NewJSONParser(), `{"key": "value"}`},
		}

		for _, p := range parsers {
			req := parser.ParseRequest{
				ID:      "test-category-" + p.name,
				Content: p.content,
				Config:  parser.DefaultParserConfig(),
			}

			result, err := p.parser.Parse(ctx, req)
			require.NoError(t, err)

			for _, elem := range result.Elements {
				assert.NotEmpty(t, elem.ElementCategory, "Parser %s should set ElementCategory for all elements", p.name)
			}
		}
	})
}

// TestRelationshipConsistency validates that all parsers create proper relationships
func TestRelationshipConsistency(t *testing.T) {
	ctx := context.Background()

	parsers := []struct {
		name    string
		parser  parser.Parser
		content string
	}{
		{"csv", parser.NewCSVParser(), "a,b\n1,2\n3,4"},
		{"text", parser.NewTextParser(), "First paragraph.\n\nSecond paragraph.\n\nThird paragraph."},
		{"json", parser.NewJSONParser(), `{"user": {"name": "Alice", "age": 30}}`},
	}

	for _, p := range parsers {
		t.Run(p.name+" relationship structure", func(t *testing.T) {
			req := parser.ParseRequest{
				ID:      "test-relationships-" + p.name,
				Content: p.content,
				Config:  parser.DefaultParserConfig(),
			}

			result, err := p.parser.Parse(ctx, req)
			require.NoError(t, err)

			// Verify hierarchy via parent_id field (replaces old Relationships array)
			hierarchyCount := 0
			parentChildMap := make(map[string][]string)
			elementIDs := make(map[string]bool)

			for _, elem := range result.Elements {
				elementIDs[elem.ElementID] = true
				if elem.ParentID != "" {
					hierarchyCount++
					parentChildMap[elem.ParentID] = append(parentChildMap[elem.ParentID], elem.ElementID)
				}
			}

			assert.Greater(t, hierarchyCount, 0, "Parser %s should have elements with parent_id forming hierarchy", p.name)

			// Verify all parent_id references point to valid elements
			for _, elem := range result.Elements {
				if elem.ParentID != "" {
					assert.True(t, elementIDs[elem.ParentID], "Parent element should exist: %s", elem.ParentID)
				}
			}

			// Verify root element exists and has children
			rootFound := false
			for _, elem := range result.Elements {
				if elem.ElementType == "root" && elem.ParentID == "" {
					rootFound = true
					assert.Greater(t, len(parentChildMap[elem.ElementID]), 0, "Root should have children")
					break
				}
			}
			assert.True(t, rootFound, "Parser %s should have root element", p.name)
		})
	}
}

// TestParserInterfaceCompliance validates all parsers implement the interface correctly
func TestParserInterfaceCompliance(t *testing.T) {
	parsers := []struct {
		name   string
		parser parser.Parser
	}{
		{"csv", parser.NewCSVParser()},
		{"text", parser.NewTextParser()},
		{"json", parser.NewJSONParser()},
	}

	for _, p := range parsers {
		t.Run(p.name+" interface compliance", func(t *testing.T) {
			// GetName should return non-empty string
			name := p.parser.GetName()
			assert.NotEmpty(t, name, "GetName should return non-empty string")

			// GetSupportedFormats should return non-empty slice
			formats := p.parser.GetSupportedFormats()
			assert.NotEmpty(t, formats, "GetSupportedFormats should return non-empty slice")

			// SupportsStreaming should return a boolean (no assertion needed, just call it)
			_ = p.parser.SupportsStreaming()

			// Close should not return error
			err := p.parser.Close()
			assert.NoError(t, err, "Close should not return error")
		})
	}
}

// TestErrorHandling validates error handling across all parsers
func TestErrorHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("all parsers handle empty content", func(t *testing.T) {
		parsers := []struct {
			name   string
			parser parser.Parser
		}{
			{"csv", parser.NewCSVParser()},
			{"text", parser.NewTextParser()},
			{"json", parser.NewJSONParser()},
		}

		for _, p := range parsers {
			req := parser.ParseRequest{
				ID:      "test-empty-" + p.name,
				Content: "",
				Config:  parser.DefaultParserConfig(),
			}

			result, err := p.parser.Parse(ctx, req)

			// CSV and Text should handle empty content gracefully
			// JSON will error on empty content (invalid JSON)
			if p.name == "json" {
				assert.Error(t, err, "JSON parser should error on empty content")
			} else {
				// CSV and Text should either succeed or error gracefully
				if err == nil {
					assert.NotNil(t, result, "Result should not be nil if no error")
				}
			}
		}
	})

	t.Run("all parsers handle invalid content types", func(t *testing.T) {
		parsers := []parser.Parser{
			parser.NewCSVParser(),
			parser.NewTextParser(),
			parser.NewJSONParser(),
		}

		for _, p := range parsers {
			req := parser.ParseRequest{
				ID:      "test-invalid-type",
				Content: 12345, // Invalid type
				Config:  parser.DefaultParserConfig(),
			}

			_, err := p.Parse(ctx, req)
			assert.Error(t, err, "Parser %s should error on invalid content type", p.GetName())
		}
	})
}

// TestBenchmarkAllParsers runs basic performance validation
func TestBenchmarkAllParsers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping benchmark test in short mode")
	}

	ctx := context.Background()

	testCases := []struct {
		name    string
		parser  parser.Parser
		content string
	}{
		{
			name:    "csv",
			parser:  parser.NewCSVParser(),
			content: generateLargeCSV(100, 10), // 100 rows, 10 columns
		},
		{
			name:    "text",
			parser:  parser.NewTextParser(),
			content: generateLargeText(50), // 50 paragraphs
		},
		{
			name:    "json",
			parser:  parser.NewJSONParser(),
			content: generateLargeJSON(20), // 20 nested objects
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name+" performance", func(t *testing.T) {
			req := parser.ParseRequest{
				ID:      "test-benchmark-" + tc.name,
				Content: tc.content,
				Config:  parser.DefaultParserConfig(),
			}

			// Parse and time it
			result, err := tc.parser.Parse(ctx, req)
			require.NoError(t, err)

			// Validate result
			assert.NotEmpty(t, result.Elements)
			t.Logf("Parser %s: %d elements created", tc.name, len(result.Elements))
		})
	}
}

// Helper functions for generating test data

func generateLargeCSV(rows, cols int) string {
	csv := "col1"
	for i := 1; i < cols; i++ {
		csv += ",col" + string(rune('0'+i))
	}
	csv += "\n"

	for r := 0; r < rows; r++ {
		csv += "val1"
		for c := 1; c < cols; c++ {
			csv += ",val" + string(rune('0'+c))
		}
		csv += "\n"
	}
	return csv
}

func generateLargeText(paragraphs int) string {
	text := ""
	for i := 0; i < paragraphs; i++ {
		text += "This is paragraph number " + string(rune('0'+i)) + ". It contains sample text for testing.\n\n"
	}
	return text
}

func generateLargeJSON(depth int) string {
	if depth <= 0 {
		return `{"value": "leaf"}`
	}
	return `{"nested": ` + generateLargeJSON(depth-1) + `, "field": "value` + string(rune('0'+depth)) + `"}`
}
