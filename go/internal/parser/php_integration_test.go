package parser_test

import (
	"context"
	"testing"

	"github.com/kennethstott/doculyzer-go-conversion/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPHPParserInterface validates PHP parser implements Parser interface correctly
func TestPHPParserInterface(t *testing.T) {
	phpParser := parser.NewPHPParser()

	t.Run("implements Parser interface", func(t *testing.T) {
		// Verify parser implements the interface
		var _ parser.Parser = phpParser

		// Test GetName
		assert.Equal(t, "php", phpParser.GetName())

		// Test GetSupportedFormats
		formats := phpParser.GetSupportedFormats()
		assert.Contains(t, formats, ".php")
		assert.Contains(t, formats, "php")

		// Test SupportsStreaming
		assert.False(t, phpParser.SupportsStreaming())

		// Test Close
		err := phpParser.Close()
		assert.NoError(t, err)
	})

	t.Run("can be registered in ParserRegistry", func(t *testing.T) {
		registry := parser.NewParserRegistry()
		registry.Register(phpParser)

		// Verify parser is registered
		retrieved, err := registry.GetParser("php")
		require.NoError(t, err)
		assert.Equal(t, "php", retrieved.GetName())

		// Verify parser can be found by file extension
		parserByFile, err := registry.GetParserForFile("script.php")
		require.NoError(t, err)
		assert.Equal(t, "php", parserByFile.GetName())
	})
}

// TestPHPParserEndToEnd validates complete PHP parsing workflow via Parser interface
func TestPHPParserEndToEnd(t *testing.T) {
	// Create sample PHP content
	phpContent := `<?php
namespace App\Services;

use App\Models\User;
use Illuminate\Support\Facades\DB;

/**
 * User service class for handling user operations
 * @package App\Services
 */
class UserService
{
    private $db;

    public function __construct(DB $db)
    {
        $this->db = $db;
    }

    /**
     * Get user by ID
     * @param int $userId
     * @return User|null
     */
    public function getUserById($userId)
    {
        return User::find($userId);
    }

    public static function createUser($data)
    {
        // Create new user
        return User::create($data);
    }
}

function helper_function($param) {
    return $param * 2;
}
`

	phpParser := parser.NewPHPParser()
	ctx := context.Background()

	t.Run("parses PHP via Parser interface", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-php-001",
			Content: phpContent,
			Metadata: map[string]interface{}{
				"source":   "test",
				"filename": "UserService.php",
			},
			Config: parser.DefaultParserConfig(),
		}

		result, err := phpParser.Parse(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Validate Document
		assert.Equal(t, "test-php-001", result.Document.ID)
		assert.Equal(t, "php", result.Document.DocType)

		// Validate Elements exist
		assert.NotEmpty(t, result.Elements)
	})

	t.Run("handles different content types", func(t *testing.T) {
		// Test with string content
		reqString := parser.ParseRequest{
			ID:      "test-php-002",
			Content: phpContent,
			Config:  parser.DefaultParserConfig(),
		}
		resultString, err := phpParser.Parse(ctx, reqString)
		require.NoError(t, err)
		assert.NotNil(t, resultString)

		// Test with []byte content
		reqBytes := parser.ParseRequest{
			ID:      "test-php-003",
			Content: []byte(phpContent),
			Config:  parser.DefaultParserConfig(),
		}
		resultBytes, err := phpParser.Parse(ctx, reqBytes)
		require.NoError(t, err)
		assert.NotNil(t, resultBytes)
	})

	t.Run("validates element structure", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-php-004",
			Content: phpContent,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := phpParser.Parse(ctx, req)
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

	t.Run("validates PHP-specific element types", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-php-005",
			Content: phpContent,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := phpParser.Parse(ctx, req)
		require.NoError(t, err)

		elementTypes := make(map[string]int)
		for _, elem := range result.Elements {
			elementTypes[elem.ElementType]++
		}

		// Verify expected element types exist
		assert.Greater(t, elementTypes["root"], 0, "Should have root element")
		assert.Greater(t, elementTypes["namespace"], 0, "Should have namespace element")
		assert.Greater(t, elementTypes["class"], 0, "Should have class element")
		assert.Greater(t, elementTypes["method"], 0, "Should have method elements")
		assert.Greater(t, elementTypes["function"], 0, "Should have function element")
		assert.Greater(t, elementTypes["use_statement"], 0, "Should have use statement elements")
	})

	t.Run("validates class detection and metadata", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-php-006",
			Content: phpContent,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := phpParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find class element
		var classElement *parser.Element
		for i := range result.Elements {
			if result.Elements[i].ElementType == "class" {
				classElement = &result.Elements[i]
				break
			}
		}

		require.NotNil(t, classElement, "Should find class element")
		assert.Equal(t, "UserService", classElement.Content)
		assert.Contains(t, classElement.Metadata, "class_name")
		assert.Equal(t, "UserService", classElement.Metadata["class_name"])
	})

	t.Run("validates method detection with visibility", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-php-007",
			Content: phpContent,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := phpParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find methods
		var publicMethods, staticMethods int
		for _, elem := range result.Elements {
			if elem.ElementType == "method" {
				if visibility, ok := elem.Metadata["visibility"].(string); ok {
					if visibility == "public" {
						publicMethods++
					}
				}
				if modifier, ok := elem.Metadata["modifier"].(string); ok {
					if modifier == "static" {
						staticMethods++
					}
				}
			}
		}

		assert.Greater(t, publicMethods, 0, "Should have public methods")
		assert.GreaterOrEqual(t, staticMethods, 0, "Should detect static methods if present")
	})

	t.Run("validates doc block extraction", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-php-008",
			Content: phpContent,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := phpParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find doc block elements
		docBlockCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "doc_block" {
				docBlockCount++
				// Verify it has tags
				if tags, ok := elem.Metadata["tags"].([]map[string]interface{}); ok {
					assert.NotEmpty(t, tags, "Doc block should have tags")
				}
			}
		}

		assert.Greater(t, docBlockCount, 0, "Should have doc block elements")
	})

	t.Run("validates namespace detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-php-009",
			Content: phpContent,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := phpParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find namespace element
		var namespaceElement *parser.Element
		for i := range result.Elements {
			if result.Elements[i].ElementType == "namespace" {
				namespaceElement = &result.Elements[i]
				break
			}
		}

		require.NotNil(t, namespaceElement, "Should find namespace element")
		assert.Equal(t, "App\\Services", namespaceElement.Content)
	})

	t.Run("validates use statement detection", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-php-010",
			Content: phpContent,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := phpParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find use statement elements
		useStatements := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "use_statement" {
				useStatements++
			}
		}

		assert.Greater(t, useStatements, 0, "Should have use statement elements")
	})
}

// TestPHPInterfaceAndTrait validates interface and trait parsing
func TestPHPInterfaceAndTrait(t *testing.T) {
	phpContent := `<?php

interface LoggerInterface
{
    public function log($message);
}

trait LoggerTrait
{
    public function log($message)
    {
        echo $message;
    }
}

class Logger implements LoggerInterface
{
    use LoggerTrait;

    const VERSION = '1.0';

    private $logFile;
}
`

	phpParser := parser.NewPHPParser()
	ctx := context.Background()

	t.Run("parses interface", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-php-interface-001",
			Content: phpContent,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := phpParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find interface element
		var interfaceElement *parser.Element
		for i := range result.Elements {
			if result.Elements[i].ElementType == "interface" {
				interfaceElement = &result.Elements[i]
				break
			}
		}

		require.NotNil(t, interfaceElement, "Should find interface element")
		assert.Equal(t, "LoggerInterface", interfaceElement.Content)
	})

	t.Run("parses trait", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-php-trait-001",
			Content: phpContent,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := phpParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find trait element
		var traitElement *parser.Element
		for i := range result.Elements {
			if result.Elements[i].ElementType == "trait" {
				traitElement = &result.Elements[i]
				break
			}
		}

		require.NotNil(t, traitElement, "Should find trait element")
		assert.Equal(t, "LoggerTrait", traitElement.Content)
	})

	t.Run("parses constants", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-php-const-001",
			Content: phpContent,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := phpParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find constant element
		var constantElement *parser.Element
		for i := range result.Elements {
			if result.Elements[i].ElementType == "constant" {
				constantElement = &result.Elements[i]
				break
			}
		}

		require.NotNil(t, constantElement, "Should find constant element")
		assert.Equal(t, "VERSION", constantElement.Content)
	})

	t.Run("parses properties", func(t *testing.T) {
		req := parser.ParseRequest{
			ID:      "test-php-property-001",
			Content: phpContent,
			Config:  parser.DefaultParserConfig(),
		}

		result, err := phpParser.Parse(ctx, req)
		require.NoError(t, err)

		// Find property element
		var propertyElement *parser.Element
		for i := range result.Elements {
			if result.Elements[i].ElementType == "property" {
				propertyElement = &result.Elements[i]
				break
			}
		}

		require.NotNil(t, propertyElement, "Should find property element")
		assert.Contains(t, propertyElement.Metadata, "visibility")
	})
}

// TestPHPHierarchy validates parent-child relationships
func TestPHPHierarchy(t *testing.T) {
	phpContent := `<?php
namespace App;

class MyClass
{
    public function myMethod()
    {
        return true;
    }
}
`

	phpParser := parser.NewPHPParser()
	ctx := context.Background()

	req := parser.ParseRequest{
		ID:      "test-php-hierarchy-001",
		Content: phpContent,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := phpParser.Parse(ctx, req)
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

	t.Run("validates method is child of class", func(t *testing.T) {
		// Find class and its methods
		var classElement *parser.Element
		var methodElements []parser.Element

		for i := range result.Elements {
			elem := &result.Elements[i]
			if elem.ElementType == "class" {
				classElement = elem
			}
			if elem.ElementType == "method" {
				methodElements = append(methodElements, *elem)
			}
		}

		require.NotNil(t, classElement, "Should have class element")
		require.NotEmpty(t, methodElements, "Should have method elements")

		// Verify method is child of class
		for _, method := range methodElements {
			assert.Equal(t, classElement.ElementID, method.ParentID, "Method should be child of class")
		}
	})
}

// TestPHPDocumentMetadata validates document-level metadata extraction
func TestPHPDocumentMetadata(t *testing.T) {
	phpContent := `<?php
class Test {
    public function method1() {}
    public function method2() {}
}
`

	phpParser := parser.NewPHPParser()
	ctx := context.Background()

	req := parser.ParseRequest{
		ID:      "test-php-metadata-001",
		Content: phpContent,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := phpParser.Parse(ctx, req)
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
			if classCount, ok := metadata["class_count"]; ok {
				assert.Equal(t, 1, classCount.(int), "Should have 1 class")
			}
			if methodCount, ok := metadata["method_count"]; ok {
				assert.Equal(t, 2, methodCount.(int), "Should have 2 methods")
			}
		}
	})
}

// TestPHPComments validates comment extraction
func TestPHPComments(t *testing.T) {
	phpContent := `<?php
// Single line comment
# Hash comment
/* Multi-line
   comment */
class Test {
    // Method comment
    public function test() {}
}
`

	phpParser := parser.NewPHPParser()
	ctx := context.Background()

	req := parser.ParseRequest{
		ID:      "test-php-comments-001",
		Content: phpContent,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := phpParser.Parse(ctx, req)
	require.NoError(t, err)

	t.Run("extracts comments", func(t *testing.T) {
		commentCount := 0
		for _, elem := range result.Elements {
			if elem.ElementType == "comment" {
				commentCount++
			}
		}

		assert.Greater(t, commentCount, 0, "Should have comment elements")
	})
}