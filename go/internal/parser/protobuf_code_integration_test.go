package parser_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kennethstott/doculyzer-go-conversion/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProtobufCodeParserInterface validates Protobuf code parser implements Parser interface correctly
func TestProtobufCodeParserInterface(t *testing.T) {
	protoParser := parser.NewProtobufCodeParser()

	t.Run("implements Parser interface", func(t *testing.T) {
		// Verify parser implements the interface
		var _ parser.Parser = protoParser

		// Test GetName
		assert.Equal(t, "protobuf_code", protoParser.GetName())

		// Test GetSupportedFormats
		formats := protoParser.GetSupportedFormats()
		assert.Contains(t, formats, ".proto")
		assert.Contains(t, formats, "proto")

		// Test SupportsStreaming
		assert.False(t, protoParser.SupportsStreaming())

		// Test Close
		err := protoParser.Close()
		assert.NoError(t, err)
	})

	t.Run("can be registered in ParserRegistry", func(t *testing.T) {
		registry := parser.NewParserRegistry()
		registry.Register(protoParser)

		// Verify parser is registered
		retrieved, err := registry.GetParser("protobuf_code")
		require.NoError(t, err)
		assert.Equal(t, "protobuf_code", retrieved.GetName())

		// Verify parser can be found by file extension
		parserByFile, err := registry.GetParserForFile("user_service.proto")
		require.NoError(t, err)
		assert.Equal(t, "protobuf_code", parserByFile.GetName())
	})
}

// TestProtobufCodeParserEndToEnd validates complete Protobuf parsing workflow via Parser interface
func TestProtobufCodeParserEndToEnd(t *testing.T) {
	protoParser := parser.NewProtobufCodeParser()
	ctx := context.Background()

	// Use test file
	testFile := filepath.Join("testdata/code", "user_service.proto")

	t.Run("parses Protobuf file via Parser interface", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-proto-001",
			Content: testFile,
			Metadata: map[string]interface{}{
				"source":   "test",
				"filename": "user_service.proto",
			},
		}

		result, err := protoParser.Parse(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Validate Document
		assert.Equal(t, "test-proto-001", result.Document.ID)
		assert.Equal(t, "protobuf_source", result.Document.DocType)

		// Validate Elements exist
		assert.NotEmpty(t, result.Elements)
	})

	t.Run("validates element structure", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-proto-002",
			Content: testFile,
		}

		result, err := protoParser.Parse(ctx, req)
		require.NoError(t, err)

		// Verify all elements have required fields
		for _, elem := range result.Elements {
			assert.NotEmpty(t, elem.ElementID, "Element should have ID")
			assert.NotEmpty(t, elem.ElementType, "Element should have type")
			assert.NotEmpty(t, elem.ContentPreview, "Element should have content preview")
			assert.NotEmpty(t, elem.ElementCategory, "Element should have category")
		}
	})

	t.Run("validates Protobuf-specific element types", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-proto-003",
			Content: testFile,
		}

		result, err := protoParser.Parse(ctx, req)
		require.NoError(t, err)

		elementTypes := make(map[string]int)
		for _, elem := range result.Elements {
			elementTypes[elem.ElementType]++
		}

		// Verify expected element types exist
		assert.Greater(t, elementTypes["document"], 0, "Should have document element")
		assert.Greater(t, elementTypes["code_grouping"], 0, "Should have message definitions")
		assert.Greater(t, elementTypes["code_function"], 0, "Should have RPC methods")
		assert.Greater(t, elementTypes["code_type"], 0, "Should have enum definitions")
		assert.Greater(t, elementTypes["code_data"], 0, "Should have field definitions")
	})

	t.Run("validates package detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-proto-004",
			Content: testFile,
		}

		result, err := protoParser.Parse(ctx, req)
		require.NoError(t, err)

		// Verify package is detected in document metadata
		assert.Contains(t, result.Document.Metadata, "package")
		assert.Equal(t, "user.service.v1", result.Document.Metadata["package"])
	})

	t.Run("validates import detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-proto-005",
			Content: testFile,
		}

		result, err := protoParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find import elements
		importCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_dependency" {
				if depKind, ok := elem.Metadata["dependency_kind"].(string); ok {
					if depKind == "import" {
						importCount++
						assert.Contains(t, elem.Metadata, "import_path")
					}
				}
			}
		}

		assert.Greater(t, importCount, 0, "Should have import statements")
	})

	t.Run("validates message detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-proto-006",
			Content: testFile,
		}

		result, err := protoParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find message elements
		messageCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_grouping" {
				if codeKind, ok := elem.Metadata["code_element_kind"].(string); ok {
					if codeKind == "message" {
						messageCount++
						assert.Contains(t, elem.Metadata, "message_name")
					}
				}
			}
		}

		assert.Greater(t, messageCount, 0, "Should have message definitions")
	})

	t.Run("validates field detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-proto-007",
			Content: testFile,
		}

		result, err := protoParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find field elements
		fieldCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_data" {
				if dataKind, ok := elem.Metadata["data_kind"].(string); ok {
					if dataKind == "field" {
						fieldCount++
						assert.Contains(t, elem.Metadata, "data_name")
						assert.Contains(t, elem.Metadata, "data_type")
						assert.Contains(t, elem.Metadata, "field_number")
					}
				}
			}
		}

		assert.Greater(t, fieldCount, 0, "Should have field definitions")
	})

	t.Run("validates service detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-proto-008",
			Content: testFile,
		}

		result, err := protoParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find service elements
		serviceCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_grouping" {
				if codeKind, ok := elem.Metadata["code_element_kind"].(string); ok {
					if codeKind == "service" {
						serviceCount++
						assert.Contains(t, elem.Metadata, "service_name")
					}
				}
			}
		}

		assert.Greater(t, serviceCount, 0, "Should have service definitions")
	})

	t.Run("validates RPC detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-proto-009",
			Content: testFile,
		}

		result, err := protoParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find RPC elements
		rpcCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_function" {
				if codeKind, ok := elem.Metadata["code_element_kind"].(string); ok {
					if codeKind == "rpc" {
						rpcCount++
						assert.Contains(t, elem.Metadata, "rpc_name")
						assert.Contains(t, elem.Metadata, "request_type")
						assert.Contains(t, elem.Metadata, "response_type")
					}
				}
			}
		}

		assert.Greater(t, rpcCount, 0, "Should have RPC method definitions")
	})

	t.Run("validates enum detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-proto-010",
			Content: testFile,
		}

		result, err := protoParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find enum elements
		enumCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_type" {
				if typeKind, ok := elem.Metadata["type_kind"].(string); ok {
					if typeKind == "enum" {
						enumCount++
						assert.Contains(t, elem.Metadata, "enum_name")
					}
				}
			}
		}

		assert.Greater(t, enumCount, 0, "Should have enum definitions")
	})

	t.Run("validates enum value detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-proto-011",
			Content: testFile,
		}

		result, err := protoParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find enum value elements
		enumValueCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "code_data" {
				if dataKind, ok := elem.Metadata["data_kind"].(string); ok {
					if dataKind == "enum_value" {
						enumValueCount++
						assert.Contains(t, elem.Metadata, "data_name")
						assert.Contains(t, elem.Metadata, "value")
					}
				}
			}
		}

		assert.Greater(t, enumValueCount, 0, "Should have enum value definitions")
	})

	t.Run("validates nested message detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-proto-012",
			Content: testFile,
		}

		result, err := protoParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find nested messages (messages with parent_id pointing to another message)
		nestedMessageCount := 0
		messageIDs := make(map[string]bool)

		// First pass: collect all message IDs
		for _, elem := range result.Elements {
			if elem.ElementType == "code_grouping" {
				if codeKind, ok := elem.Metadata["code_element_kind"].(string); ok {
					if codeKind == "message" {
						messageIDs[elem.ElementID] = true
					}
				}
			}
		}

		// Second pass: find messages with parent_id pointing to another message
		for _, elem := range result.Elements {
			if elem.ElementType == "code_grouping" {
				if codeKind, ok := elem.Metadata["code_element_kind"].(string); ok {
					if codeKind == "message" && elem.ParentID != "" {
						if messageIDs[elem.ParentID] {
							nestedMessageCount++
						}
					}
				}
			}
		}

		assert.Greater(t, nestedMessageCount, 0, "Should have nested message definitions")
	})

	t.Run("validates documentation detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-proto-013",
			Content: testFile,
		}

		result, err := protoParser.Parse(ctx, req)
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

// TestProtobufHierarchy validates parent-child relationships
func TestProtobufHierarchy(t *testing.T) {
	protoParser := parser.NewProtobufCodeParser()
	ctx := context.Background()

	testFile := filepath.Join("testdata/code", "user_service.proto")

	req := parser.ParseRequest{
		ID:      "test-proto-hierarchy-001",
		Content: testFile,
	}

	result, err := protoParser.Parse(ctx, req)
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

	t.Run("validates message field hierarchy", func(t *testing.T) {
		// Find messages and verify they have child fields
		for _, elem := range result.Elements {
			if elem.ElementType == "code_grouping" {
				if codeKind, ok := elem.Metadata["code_element_kind"].(string); ok {
					if codeKind == "message" {
						// Messages may have child fields
						hasChildren := false
						for _, child := range result.Elements {
							if child.ParentID == elem.ElementID {
								hasChildren = true
								break
							}
						}
						if hasChildren {
							t.Logf("Message %s has child fields", elem.ElementID)
						}
					}
				}
			}
		}
	})

	t.Run("validates service RPC hierarchy", func(t *testing.T) {
		// Find services and verify they have child RPCs
		for _, elem := range result.Elements {
			if elem.ElementType == "code_grouping" {
				if codeKind, ok := elem.Metadata["code_element_kind"].(string); ok {
					if codeKind == "service" {
						// Services should have child RPCs
						hasRPCs := false
						for _, child := range result.Elements {
							if child.ParentID == elem.ElementID && child.ElementType == "code_function" {
								hasRPCs = true
								break
							}
						}
						assert.True(t, hasRPCs, "Service should have RPC children")
					}
				}
			}
		}
	})

	t.Run("validates enum value hierarchy", func(t *testing.T) {
		// Find enums and verify they have child enum values
		for _, elem := range result.Elements {
			if elem.ElementType == "code_type" {
				if typeKind, ok := elem.Metadata["type_kind"].(string); ok {
					if typeKind == "enum" {
						// Enums should have child enum values
						hasValues := false
						for _, child := range result.Elements {
							if child.ParentID == elem.ElementID {
								if dataKind, ok := child.Metadata["data_kind"].(string); ok {
									if dataKind == "enum_value" {
										hasValues = true
										break
									}
								}
							}
						}
						assert.True(t, hasValues, "Enum should have enum value children")
					}
				}
			}
		}
	})
}
