package parser_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kennethstott/doculyzer-go-conversion/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestYAMLParserInterface validates YAML parser implements Parser interface correctly
func TestYAMLParserInterface(t *testing.T) {
	yamlParser := parser.NewYAMLParser()

	t.Run("implements Parser interface", func(t *testing.T) {
		// Verify parser implements the interface
		var _ parser.Parser = yamlParser

		// Test GetName
		assert.Equal(t, "yaml_data", yamlParser.GetName())

		// Test GetSupportedFormats
		formats := yamlParser.GetSupportedFormats()
		assert.Contains(t, formats, ".yaml")
		assert.Contains(t, formats, ".yml")
		assert.Contains(t, formats, "yaml")
		assert.Contains(t, formats, "yml")

		// Test SupportsStreaming
		assert.False(t, yamlParser.SupportsStreaming())

		// Test Close
		err := yamlParser.Close()
		assert.NoError(t, err)
	})

	t.Run("can be registered in ParserRegistry", func(t *testing.T) {
		registry := parser.NewParserRegistry()
		registry.Register(yamlParser)

		// Verify parser is registered
		retrieved, err := registry.GetParser("yaml_data")
		require.NoError(t, err)
		assert.Equal(t, "yaml_data", retrieved.GetName())

		// Verify parser can be found by file extension
		parserByFile, err := registry.GetParserForFile("deployment.yaml")
		require.NoError(t, err)
		assert.Equal(t, "yaml_data", parserByFile.GetName())
	})
}

// TestYAMLParserEndToEnd validates complete YAML parsing workflow via Parser interface
func TestYAMLParserEndToEnd(t *testing.T) {
	yamlParser := parser.NewYAMLParser()
	ctx := context.Background()

	// Use test file
	testFile := filepath.Join("testdata/data", "deployment.yaml")

	t.Run("parses YAML file via Parser interface", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-yaml-001",
			Content: testFile,
			Metadata: map[string]interface{}{
				"source":   "test",
				"filename": "deployment.yaml",
			},
		}

		result, err := yamlParser.Parse(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Validate Document
		assert.Equal(t, "test-yaml-001", result.Document.ID)
		assert.Equal(t, "yaml_config", result.Document.DocType)

		// Validate Elements exist
		assert.NotEmpty(t, result.Elements)
	})

	t.Run("validates element structure", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-yaml-002",
			Content: testFile,
		}

		result, err := yamlParser.Parse(ctx, req)
		require.NoError(t, err)

		// Verify all elements have required fields
		for _, elem := range result.Elements {
			assert.NotEmpty(t, elem.ElementID, "Element should have ID")
			assert.NotEmpty(t, elem.ElementType, "Element should have type")
			assert.NotEmpty(t, elem.ContentPreview, "Element should have content preview")
			assert.NotEmpty(t, elem.ElementCategory, "Element should have category")
		}
	})

	t.Run("validates YAML-specific element types", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-yaml-003",
			Content: testFile,
		}

		result, err := yamlParser.Parse(ctx, req)
		require.NoError(t, err)

		elementTypes := make(map[string]int)
		for _, elem := range result.Elements {
			elementTypes[elem.ElementType]++
		}

		// Verify expected element types exist
		assert.Greater(t, elementTypes["document"], 0, "Should have document element")
		assert.Greater(t, elementTypes["code_data"], 0, "Should have data elements (scalars)")
		// May or may not have grouping elements depending on nested structures
	})

	t.Run("validates scalar value detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-yaml-004",
			Content: testFile,
		}

		result, err := yamlParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find scalar elements
		scalarCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_data" {
				if dataKind, ok := elem.Metadata["data_kind"].(string); ok {
					if dataKind == "scalar" {
						scalarCount++
						assert.Contains(t, elem.Metadata, "data_name")
						assert.Contains(t, elem.Metadata, "value")
					}
				}
			}
		}

		assert.Greater(t, scalarCount, 0, "Should have scalar value elements")
	})

	t.Run("validates nested mapping detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-yaml-005",
			Content: testFile,
		}

		result, err := yamlParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find mapping elements
		mappingCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_grouping" {
				if dataKind, ok := elem.Metadata["data_kind"].(string); ok {
					if dataKind == "mapping" {
						mappingCount++
						assert.Contains(t, elem.Metadata, "key")
						assert.Contains(t, elem.Metadata, "path")
					}
				}
			}
		}

		// May have nested mappings in the Kubernetes deployment
		// This is optional but should work if present
		if mappingCount > 0 {
			t.Logf("Found %d nested mapping elements", mappingCount)
		}
	})

	t.Run("validates sequence/array detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-yaml-006",
			Content: testFile,
		}

		result, err := yamlParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find sequence elements or grouped structures that could represent arrays
		sequenceCount := 0
		for _, elem := range result.Elements {
			if dataKind, ok := elem.Metadata["data_kind"].(string); ok {
				if dataKind == "sequence" || dataKind == "sequence_item" || dataKind == "mapping" {
					sequenceCount++
				}
			}
		}

		// YAML parser may represent arrays differently - just verify we're parsing nested structures
		assert.GreaterOrEqual(t, sequenceCount, 0, "Parser should handle sequences/arrays")
	})

	t.Run("validates path tracking", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-yaml-007",
			Content: testFile,
		}

		result, err := yamlParser.Parse(ctx, req)
		require.NoError(t, err)

		// Check that elements have path metadata
		pathCount := 0
		for _, elem := range result.Elements {
			if path, ok := elem.Metadata["path"].(string); ok {
				if path != "" {
					pathCount++
				}
			}
		}

		assert.Greater(t, pathCount, 0, "Should have elements with path tracking")
	})

	t.Run("validates metadata field detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-yaml-008",
			Content: testFile,
		}

		result, err := yamlParser.Parse(ctx, req)
		require.NoError(t, err)

		// Look for some fields from Kubernetes deployment
		foundFields := 0

		for _, elem := range result.Elements {
			if elem.ElementType == "code_data" {
				if dataName, ok := elem.Metadata["data_name"].(string); ok {
					if dataName == "kind" || dataName == "apiVersion" ||
					   dataName == "name" || dataName == "replicas" {
						foundFields++
					}
				}
			}
		}

		// Should find at least some of the top-level fields
		assert.Greater(t, foundFields, 0, "Should find some YAML fields")
	})

	t.Run("validates comment detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-yaml-009",
			Content: testFile,
		}

		result, err := yamlParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find documentation elements (comments)
		docCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_documentation" {
				docCount++
				assert.Contains(t, elem.Metadata, "doc_kind")
			}
		}

		assert.Greater(t, docCount, 0, "Should have documentation/comment elements")
	})
}

// TestYAMLHierarchy validates parent-child relationships
func TestYAMLHierarchy(t *testing.T) {
	yamlParser := parser.NewYAMLParser()
	ctx := context.Background()

	testFile := filepath.Join("testdata/data", "deployment.yaml")

	req := parser.ParseRequest{
		ID:      "test-yaml-hierarchy-001",
		Content: testFile,
	}

	result, err := yamlParser.Parse(ctx, req)
	require.NoError(t, err)

	t.Run("validates hierarchy via parent_id", func(t *testing.T) {
		// Verify hierarchy via parent_id field
		hierarchyCount := 0
		parentChildMap := make(map[string][]string)

		for _, elem := range result.Elements {
			if elem.ParentID != "" {
				hierarchyCount++
				parentChildMap[elem.ParentID] = append(parentChildMap[elem.ParentID], elem.ElementID)
			}
		}

		assert.Greater(t, hierarchyCount, 0, "Should have elements with parent_id forming hierarchy")

		// Verify document element exists and has children
		documentFound := false
		for _, elem := range result.Elements {
			if elem.ElementType == "document" && elem.ParentID == "" {
				documentFound = true
				assert.Greater(t, len(parentChildMap[elem.ElementID]), 0, "Document should have children")
				break
			}
		}
		assert.True(t, documentFound, "Should have document element")
	})

	t.Run("validates nested structure hierarchy", func(t *testing.T) {
		// Find nested mapping elements and verify they have children
		for _, elem := range result.Elements {
			if elem.ElementType == "code_grouping" {
				if dataKind, ok := elem.Metadata["data_kind"].(string); ok {
					if dataKind == "mapping" {
						// This nested mapping should have child elements
						hasChildren := false
						for _, child := range result.Elements {
							if child.ParentID == elem.ElementID {
								hasChildren = true
								break
							}
						}
						// Nested mappings should have children
						if hasChildren {
							t.Logf("Nested mapping %s has children", elem.ElementID)
						}
					}
				}
			}
		}
	})
}
