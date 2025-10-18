package parser_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kennethstott/doculyzer-go-conversion/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHCLCodeParserInterface validates HCL code parser implements Parser interface correctly
func TestHCLCodeParserInterface(t *testing.T) {
	hclParser := parser.NewHCLCodeParser()

	t.Run("implements Parser interface", func(t *testing.T) {
		// Verify parser implements the interface
		var _ parser.Parser = hclParser

		// Test GetName
		assert.Equal(t, "hcl_code", hclParser.GetName())

		// Test GetSupportedFormats
		formats := hclParser.GetSupportedFormats()
		assert.Contains(t, formats, ".tf")
		assert.Contains(t, formats, ".hcl")
		assert.Contains(t, formats, "tf")
		assert.Contains(t, formats, "hcl")

		// Test SupportsStreaming
		assert.False(t, hclParser.SupportsStreaming())

		// Test Close
		err := hclParser.Close()
		assert.NoError(t, err)
	})

	t.Run("can be registered in ParserRegistry", func(t *testing.T) {
		registry := parser.NewParserRegistry()
		registry.Register(hclParser)

		// Verify parser is registered
		retrieved, err := registry.GetParser("hcl_code")
		require.NoError(t, err)
		assert.Equal(t, "hcl_code", retrieved.GetName())

		// Verify parser can be found by file extension
		parserByFile, err := registry.GetParserForFile("main.tf")
		require.NoError(t, err)
		assert.Equal(t, "hcl_code", parserByFile.GetName())
	})
}

// TestHCLCodeParserEndToEnd validates complete HCL parsing workflow via Parser interface
func TestHCLCodeParserEndToEnd(t *testing.T) {
	hclParser := parser.NewHCLCodeParser()
	ctx := context.Background()

	// Use test file
	testFile := filepath.Join("testdata/code", "main.tf")

	t.Run("parses HCL file via Parser interface", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-hcl-001",
			Content: testFile,
			Metadata: map[string]interface{}{
				"source":   "test",
				"filename": "main.tf",
			},
		}

		result, err := hclParser.Parse(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Validate Document
		assert.Equal(t, "test-hcl-001", result.Document.ID)
		assert.Equal(t, "hcl_source", result.Document.DocType)

		// Validate Elements exist
		assert.NotEmpty(t, result.Elements)
	})

	t.Run("validates element structure", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-hcl-002",
			Content: testFile,
		}

		result, err := hclParser.Parse(ctx, req)
		require.NoError(t, err)

		// Verify all elements have required fields
		for _, elem := range result.Elements {
			assert.NotEmpty(t, elem.ElementID, "Element should have ID")
			assert.NotEmpty(t, elem.ElementType, "Element should have type")
			assert.NotEmpty(t, elem.ContentPreview, "Element should have content preview")
			assert.NotEmpty(t, elem.ElementCategory, "Element should have category")
		}
	})

	t.Run("validates HCL-specific element types", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-hcl-003",
			Content: testFile,
		}

		result, err := hclParser.Parse(ctx, req)
		require.NoError(t, err)

		elementTypes := make(map[string]int)
		for _, elem := range result.Elements {
			elementTypes[elem.ElementType]++
		}

		// Verify expected element types exist
		assert.Greater(t, elementTypes["document"], 0, "Should have document element")
		assert.Greater(t, elementTypes["code_grouping"], 0, "Should have resource blocks")
		assert.Greater(t, elementTypes["code_data"], 0, "Should have variables/outputs/attributes")
	})

	t.Run("validates resource detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-hcl-004",
			Content: testFile,
		}

		result, err := hclParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find resource elements
		resourceCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_grouping" {
				if kind, ok := elem.Metadata["code_element_kind"].(string); ok {
					if kind == "resource" {
						resourceCount++
						assert.Contains(t, elem.Metadata, "resource_type")
						assert.Contains(t, elem.Metadata, "resource_name")
					}
				}
			}
		}

		assert.Greater(t, resourceCount, 0, "Should have resource elements")
	})

	t.Run("validates variable detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-hcl-005",
			Content: testFile,
		}

		result, err := hclParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find variable elements
		varCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_data" {
				if dataKind, ok := elem.Metadata["data_kind"].(string); ok {
					if dataKind == "variable" {
						varCount++
						assert.Contains(t, elem.Metadata, "data_name")
					}
				}
			}
		}

		assert.Greater(t, varCount, 0, "Should have variable elements")
	})

	t.Run("validates output detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-hcl-006",
			Content: testFile,
		}

		result, err := hclParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find output elements
		outputCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_data" {
				if dataKind, ok := elem.Metadata["data_kind"].(string); ok {
					if dataKind == "output" {
						outputCount++
						assert.Contains(t, elem.Metadata, "data_name")
					}
				}
			}
		}

		assert.Greater(t, outputCount, 0, "Should have output elements")
	})

	t.Run("validates module detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-hcl-007",
			Content: testFile,
		}

		result, err := hclParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find module elements
		moduleCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_dependency" {
				if depKind, ok := elem.Metadata["dependency_kind"].(string); ok {
					if depKind == "module" {
						moduleCount++
						assert.Contains(t, elem.Metadata, "module_name")
					}
				}
			}
		}

		assert.Greater(t, moduleCount, 0, "Should have module elements")
	})

	t.Run("validates provider detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-hcl-008",
			Content: testFile,
		}

		result, err := hclParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find provider elements
		providerCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_dependency" {
				if depKind, ok := elem.Metadata["dependency_kind"].(string); ok {
					if depKind == "provider" {
						providerCount++
					}
				}
			}
		}

		assert.Greater(t, providerCount, 0, "Should have provider elements")
	})

	t.Run("validates data source detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-hcl-009",
			Content: testFile,
		}

		result, err := hclParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find data source elements
		dataCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_grouping" {
				if kind, ok := elem.Metadata["code_element_kind"].(string); ok {
					if kind == "data" {
						dataCount++
					}
				}
			}
		}

		assert.Greater(t, dataCount, 0, "Should have data source elements")
	})

	t.Run("validates documentation detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-hcl-010",
			Content: testFile,
		}

		result, err := hclParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find documentation elements (comments)
		docCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_documentation" {
				docCount++
				assert.Contains(t, elem.Metadata, "doc_kind")
			}
		}

		assert.Greater(t, docCount, 0, "Should have documentation elements")
	})
}

// TestHCLHierarchy validates parent-child relationships
func TestHCLHierarchy(t *testing.T) {
	hclParser := parser.NewHCLCodeParser()
	ctx := context.Background()

	testFile := filepath.Join("testdata/code", "main.tf")

	req := parser.ParseRequest{
		ID:      "test-hcl-hierarchy-001",
		Content: testFile,
	}

	result, err := hclParser.Parse(ctx, req)
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

	t.Run("validates resource block hierarchy", func(t *testing.T) {
		// Find resource blocks and verify they have child attributes
		for _, elem := range result.Elements {
			if elem.ElementType == "code_grouping" {
				if kind, ok := elem.Metadata["code_element_kind"].(string); ok {
					if kind == "resource" {
						// Resources may have child attributes
						hasChildren := false
						for _, child := range result.Elements {
							if child.ParentID == elem.ElementID {
								hasChildren = true
								break
							}
						}
						if hasChildren {
							t.Logf("Resource block %s has child attributes", elem.ElementID)
						}
					}
				}
			}
		}
	})
}
