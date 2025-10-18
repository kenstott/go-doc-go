package parser_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kennethstott/doculyzer-go-conversion/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBashCodeParserInterface validates Bash code parser implements Parser interface correctly
func TestBashCodeParserInterface(t *testing.T) {
	bashParser := parser.NewBashCodeParser()

	t.Run("implements Parser interface", func(t *testing.T) {
		// Verify parser implements the interface
		var _ parser.Parser = bashParser

		// Test GetName
		assert.Equal(t, "bash_code", bashParser.GetName())

		// Test GetSupportedFormats
		formats := bashParser.GetSupportedFormats()
		assert.Contains(t, formats, ".sh")
		assert.Contains(t, formats, ".bash")
		assert.Contains(t, formats, "sh")
		assert.Contains(t, formats, "bash")

		// Test SupportsStreaming
		assert.False(t, bashParser.SupportsStreaming())

		// Test Close
		err := bashParser.Close()
		assert.NoError(t, err)
	})

	t.Run("can be registered in ParserRegistry", func(t *testing.T) {
		registry := parser.NewParserRegistry()
		registry.Register(bashParser)

		// Verify parser is registered
		retrieved, err := registry.GetParser("bash_code")
		require.NoError(t, err)
		assert.Equal(t, "bash_code", retrieved.GetName())

		// Verify parser can be found by file extension
		parserByFile, err := registry.GetParserForFile("deploy.sh")
		require.NoError(t, err)
		assert.Equal(t, "bash_code", parserByFile.GetName())
	})
}

// TestBashCodeParserEndToEnd validates complete Bash parsing workflow via Parser interface
func TestBashCodeParserEndToEnd(t *testing.T) {
	bashParser := parser.NewBashCodeParser()
	ctx := context.Background()

	// Use test file
	testFile := filepath.Join("testdata/code", "deploy.sh")

	t.Run("parses Bash file via Parser interface", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-bash-001",
			Content: testFile,
			Metadata: map[string]interface{}{
				"source":   "test",
				"filename": "deploy.sh",
			},
		}

		result, err := bashParser.Parse(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Validate Document
		assert.Equal(t, "test-bash-001", result.Document.ID)
		assert.Equal(t, "bash_source", result.Document.DocType)

		// Validate Elements exist
		assert.NotEmpty(t, result.Elements)
	})

	t.Run("validates element structure", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-bash-002",
			Content: testFile,
		}

		result, err := bashParser.Parse(ctx, req)
		require.NoError(t, err)

		// Verify all elements have required fields
		for _, elem := range result.Elements {
			assert.NotEmpty(t, elem.ElementID, "Element should have ID")
			assert.NotEmpty(t, elem.ElementType, "Element should have type")
			assert.NotEmpty(t, elem.ContentPreview, "Element should have content preview")
			assert.NotEmpty(t, elem.ElementCategory, "Element should have category")
		}
	})

	t.Run("validates Bash-specific element types", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-bash-003",
			Content: testFile,
		}

		result, err := bashParser.Parse(ctx, req)
		require.NoError(t, err)

		elementTypes := make(map[string]int)
		for _, elem := range result.Elements {
			elementTypes[elem.ElementType]++
		}

		// Verify expected element types exist
		assert.Greater(t, elementTypes["document"], 0, "Should have document element")
		assert.Greater(t, elementTypes["code_function"], 0, "Should have function elements")
		// Variables and other elements may or may not be present
	})

	t.Run("validates function detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-bash-004",
			Content: testFile,
		}

		result, err := bashParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find function elements
		functionCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_function" {
				if kind, ok := elem.Metadata["code_element_kind"].(string); ok {
					if kind == "function" {
						functionCount++
						// Verify function metadata
						assert.Contains(t, elem.Metadata, "language")
						assert.Equal(t, "bash", elem.Metadata["language"])
						if elem.FunctionName != nil {
							assert.NotEmpty(t, *elem.FunctionName)
						}
					}
				}
			}
		}

		assert.Greater(t, functionCount, 0, "Should have function elements")
	})

	t.Run("validates variable detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-bash-005",
			Content: testFile,
		}

		result, err := bashParser.Parse(ctx, req)
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

	t.Run("validates conditional detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-bash-006",
			Content: testFile,
		}

		result, err := bashParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find conditional elements (if statements)
		condCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_grouping" {
				if kind, ok := elem.Metadata["code_element_kind"].(string); ok {
					if kind == "conditional" {
						condCount++
					}
				}
			}
		}

		assert.Greater(t, condCount, 0, "Should have conditional elements")
	})

	t.Run("validates loop detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-bash-007",
			Content: testFile,
		}

		result, err := bashParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find loop elements
		loopCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_grouping" {
				if kind, ok := elem.Metadata["code_element_kind"].(string); ok {
					if kind == "for_loop" || kind == "while_loop" {
						loopCount++
					}
				}
			}
		}

		assert.Greater(t, loopCount, 0, "Should have loop elements")
	})

	t.Run("validates case statement detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-bash-008",
			Content: testFile,
		}

		result, err := bashParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find case statement elements
		caseCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_grouping" {
				if kind, ok := elem.Metadata["code_element_kind"].(string); ok {
					if kind == "case_statement" {
						caseCount++
					}
				}
			}
		}

		assert.Greater(t, caseCount, 0, "Should have case statement elements")
	})

	t.Run("validates command detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-bash-009",
			Content: testFile,
		}

		result, err := bashParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find command elements
		cmdCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_function" {
				if kind, ok := elem.Metadata["code_element_kind"].(string); ok {
					if kind == "command" || kind == "pipeline" {
						cmdCount++
					}
				}
			}
		}

		assert.Greater(t, cmdCount, 0, "Should have command/pipeline elements")
	})

	t.Run("validates documentation detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-bash-010",
			Content: testFile,
		}

		result, err := bashParser.Parse(ctx, req)
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

// TestBashHierarchy validates parent-child relationships
func TestBashHierarchy(t *testing.T) {
	bashParser := parser.NewBashCodeParser()
	ctx := context.Background()

	testFile := filepath.Join("testdata/code", "deploy.sh")

	req := parser.ParseRequest{
		ID:      "test-bash-hierarchy-001",
		Content: testFile,
	}

	result, err := bashParser.Parse(ctx, req)
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

	t.Run("validates function body hierarchy", func(t *testing.T) {
		// Find function elements and verify they have child elements
		for _, elem := range result.Elements {
			if elem.ElementType == "code_function" {
				if kind, ok := elem.Metadata["code_element_kind"].(string); ok {
					if kind == "function" {
						// Functions should potentially have child elements (commands, variables, etc.)
						hasChildren := false
						for _, child := range result.Elements {
							if child.ParentID == elem.ElementID {
								hasChildren = true
								break
							}
						}
						// Some functions may have children, some may not - just log
						if hasChildren {
							t.Logf("Function %s has child elements", elem.ElementID)
						}
					}
				}
			}
		}
	})
}
