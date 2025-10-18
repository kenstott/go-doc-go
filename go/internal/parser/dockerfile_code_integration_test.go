package parser_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kennethstott/doculyzer-go-conversion/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDockerfileParserInterface validates Dockerfile parser implements Parser interface correctly
func TestDockerfileParserInterface(t *testing.T) {
	dockerfileParser := parser.NewDockerfileParser()

	t.Run("implements Parser interface", func(t *testing.T) {
		// Verify parser implements the interface
		var _ parser.Parser = dockerfileParser

		// Test GetName
		assert.Equal(t, "dockerfile_code", dockerfileParser.GetName())

		// Test GetSupportedFormats
		formats := dockerfileParser.GetSupportedFormats()
		assert.Contains(t, formats, "Dockerfile")
		assert.Contains(t, formats, ".dockerfile")
		assert.Contains(t, formats, "dockerfile")

		// Test SupportsStreaming
		assert.False(t, dockerfileParser.SupportsStreaming())

		// Test Close
		err := dockerfileParser.Close()
		assert.NoError(t, err)
	})

	t.Run("can be registered in ParserRegistry", func(t *testing.T) {
		registry := parser.NewParserRegistry()
		registry.Register(dockerfileParser)

		// Verify parser is registered
		retrieved, err := registry.GetParser("dockerfile_code")
		require.NoError(t, err)
		assert.Equal(t, "dockerfile_code", retrieved.GetName())

		// Verify parser can be found by file name
		parserByFile, err := registry.GetParserForFile("Dockerfile")
		require.NoError(t, err)
		assert.Equal(t, "dockerfile_code", parserByFile.GetName())
	})
}

// TestDockerfileParserEndToEnd validates complete Dockerfile parsing workflow via Parser interface
func TestDockerfileParserEndToEnd(t *testing.T) {
	dockerfileParser := parser.NewDockerfileParser()
	ctx := context.Background()

	// Use test file
	testFile := filepath.Join("testdata/code", "Dockerfile")

	t.Run("parses Dockerfile via Parser interface", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-dockerfile-001",
			Content: testFile,
			Metadata: map[string]interface{}{
				"source":   "test",
				"filename": "Dockerfile",
			},
		}

		result, err := dockerfileParser.Parse(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Validate Document
		assert.Equal(t, "test-dockerfile-001", result.Document.ID)
		assert.Equal(t, "dockerfile_source", result.Document.DocType)

		// Validate Elements exist
		assert.NotEmpty(t, result.Elements)
	})

	t.Run("validates element structure", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-dockerfile-002",
			Content: testFile,
		}

		result, err := dockerfileParser.Parse(ctx, req)
		require.NoError(t, err)

		// Verify all elements have required fields
		for _, elem := range result.Elements {
			assert.NotEmpty(t, elem.ElementID, "Element should have ID")
			assert.NotEmpty(t, elem.ElementType, "Element should have type")
			assert.NotEmpty(t, elem.ContentPreview, "Element should have content preview")
			assert.NotEmpty(t, elem.ElementCategory, "Element should have category")
		}
	})

	t.Run("validates Dockerfile-specific element types", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-dockerfile-003",
			Content: testFile,
		}

		result, err := dockerfileParser.Parse(ctx, req)
		require.NoError(t, err)

		elementTypes := make(map[string]int)
		for _, elem := range result.Elements {
			elementTypes[elem.ElementType]++
		}

		// Verify expected element types exist
		assert.Greater(t, elementTypes["document"], 0, "Should have document element")
		assert.Greater(t, elementTypes["code_dependency"], 0, "Should have FROM (base image) elements")
		assert.Greater(t, elementTypes["code_function"], 0, "Should have RUN/CMD elements")
		assert.Greater(t, elementTypes["code_data"], 0, "Should have ENV/ARG elements")
	})

	t.Run("validates FROM instruction detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-dockerfile-004",
			Content: testFile,
		}

		result, err := dockerfileParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find FROM elements
		var fromElements []parser.Element
		for i := range result.Elements {
			if result.Elements[i].ElementType == "code_dependency" {
				if depKind, ok := result.Elements[i].Metadata["dependency_kind"].(string); ok {
					if depKind == "base_image" {
						fromElements = append(fromElements, result.Elements[i])
					}
				}
			}
		}

		require.NotEmpty(t, fromElements, "Should find FROM elements")

		// Verify first FROM instruction metadata
		firstFrom := fromElements[0]
		assert.Contains(t, firstFrom.Metadata, "base_image")
		assert.Contains(t, firstFrom.Metadata, "language")
		assert.Equal(t, "dockerfile", firstFrom.Metadata["language"])
	})

	t.Run("validates multi-stage build detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-dockerfile-005",
			Content: testFile,
		}

		result, err := dockerfileParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find FROM elements with stage aliases
		stageCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_dependency" {
				if depKind, ok := elem.Metadata["dependency_kind"].(string); ok {
					if depKind == "base_image" {
						if _, hasAlias := elem.Metadata["stage_alias"]; hasAlias {
							stageCount++
						}
					}
				}
			}
		}

		assert.Greater(t, stageCount, 0, "Should have stages with aliases")
	})

	t.Run("validates RUN instruction detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-dockerfile-006",
			Content: testFile,
		}

		result, err := dockerfileParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find RUN elements
		runCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_function" {
				if kind, ok := elem.Metadata["code_element_kind"].(string); ok {
					if kind == "run_command" {
						runCount++
						// Verify metadata exists
						assert.Contains(t, elem.Metadata, "command")
						assert.Contains(t, elem.Metadata, "language")
						assert.Equal(t, "dockerfile", elem.Metadata["language"])
					}
				}
			}
		}

		assert.Greater(t, runCount, 0, "Should have RUN instruction elements")
	})

	t.Run("validates COPY instruction detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-dockerfile-007",
			Content: testFile,
		}

		result, err := dockerfileParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find COPY elements
		copyCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_function" {
				if kind, ok := elem.Metadata["code_element_kind"].(string); ok {
					if kind == "copy_instruction" {
						copyCount++
						assert.Contains(t, elem.Metadata, "source")
						assert.Contains(t, elem.Metadata, "destination")
					}
				}
			}
		}

		assert.Greater(t, copyCount, 0, "Should have COPY instruction elements")
	})

	t.Run("validates ENV variable detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-dockerfile-008",
			Content: testFile,
		}

		result, err := dockerfileParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find ENV elements
		envCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_data" {
				if dataKind, ok := elem.Metadata["data_kind"].(string); ok {
					if dataKind == "environment_variable" {
						envCount++
						assert.Contains(t, elem.Metadata, "data_name")
						assert.Contains(t, elem.Metadata, "value")
					}
				}
			}
		}

		assert.Greater(t, envCount, 0, "Should have ENV variable elements")
	})

	t.Run("validates ARG detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-dockerfile-009",
			Content: testFile,
		}

		result, err := dockerfileParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find ARG elements
		argCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_data" {
				if dataKind, ok := elem.Metadata["data_kind"].(string); ok {
					if dataKind == "build_argument" {
						argCount++
						assert.Contains(t, elem.Metadata, "data_name")
					}
				}
			}
		}

		assert.Greater(t, argCount, 0, "Should have ARG elements")
	})

	t.Run("validates EXPOSE detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-dockerfile-010",
			Content: testFile,
		}

		result, err := dockerfileParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find EXPOSE elements
		exposeCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_data" {
				if dataKind, ok := elem.Metadata["data_kind"].(string); ok {
					if dataKind == "port_exposure" {
						exposeCount++
						assert.Contains(t, elem.Metadata, "ports")
					}
				}
			}
		}

		assert.Greater(t, exposeCount, 0, "Should have EXPOSE elements")
	})

	t.Run("validates documentation detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-dockerfile-011",
			Content: testFile,
		}

		result, err := dockerfileParser.Parse(ctx, req)
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

// TestDockerfileHierarchy validates parent-child relationships
func TestDockerfileHierarchy(t *testing.T) {
	dockerfileParser := parser.NewDockerfileParser()
	ctx := context.Background()

	testFile := filepath.Join("testdata/code", "Dockerfile")

	req := parser.ParseRequest{
		ID:      "test-dockerfile-hierarchy-001",
		Content: testFile,
	}

	result, err := dockerfileParser.Parse(ctx, req)
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

	t.Run("validates all instructions are children of document", func(t *testing.T) {
		// Find document element
		var docElement *parser.Element
		for i := range result.Elements {
			if result.Elements[i].ElementType == "document" {
				docElement = &result.Elements[i]
				break
			}
		}

		require.NotNil(t, docElement, "Should have document element")

		// Verify all non-document elements have document as parent
		for _, elem := range result.Elements {
			if elem.ElementType != "document" {
				assert.Equal(t, docElement.ElementID, elem.ParentID,
					"All instructions should be children of document")
			}
		}
	})
}
