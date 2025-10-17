package parser_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kennethstott/doculyzer-go-conversion/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJavaCodeParser_BasicParsing validates the Java code parser with a simple Java file
func TestJavaCodeParser_BasicParsing(t *testing.T) {
	ctx := context.Background()

	// Create a temporary Java file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "Test.java")

	testCode := `package com.example;

import java.util.List;
import java.util.ArrayList;

/**
 * User class for managing user data.
 */
public class User {
    private String name;
    private int age;

    /**
     * Creates a new User.
     * @param name the user's name
     * @param age the user's age
     */
    public User(String name, int age) {
        this.name = name;
        this.age = age;
    }

    /**
     * Gets the user's age.
     * @return the age
     */
    public int getAge() {
        return age;
    }

    @Override
    public String toString() {
        return String.format("User(%s, %d)", name, age);
    }
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	// Parse the file
	javaParser := parser.NewJavaCodeParser()
	req := parser.ParseRequest{
		ID:      "test-java-code",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := javaParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify document metadata
	assert.Equal(t, "test-java-code", result.Document.ID)
	assert.Equal(t, "java_source", result.Document.DocType)
	assert.Equal(t, "com.example", result.Document.Metadata["package"])

	// Verify we have elements
	assert.NotEmpty(t, result.Elements)

	// Track what we found
	foundConstructor := false
	foundMethod := false
	foundClass := false
	foundField := false
	foundImport := false
	foundJavadoc := false
	foundAnnotation := false

	for _, elem := range result.Elements {
		t.Logf("Element: type=%s, category=%s, content_preview=%s",
			elem.ElementType, elem.ElementCategory, elem.ContentPreview)

		// Verify all code elements use universal types
		assert.Contains(t, []string{
			"document", "code_function", "code_grouping", "code_type",
			"code_data", "code_dependency", "code_documentation",
		}, elem.ElementType, "Element should use universal code model types")

		// Check for class
		if elem.ElementType == "code_grouping" && elem.ClassName != nil && *elem.ClassName == "User" {
			foundClass = true
			assert.Equal(t, "class", elem.Metadata["code_element_kind"])
			assert.Equal(t, "public", elem.Metadata["visibility"])
			assert.Equal(t, "java", elem.Metadata["language"])
		}

		// Check for constructor
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "User" {
			foundConstructor = true
			assert.NotNil(t, elem.Signature)
			assert.NotNil(t, elem.LineNumber)
		}

		// Check for method
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "getAge" {
			foundMethod = true
			assert.NotNil(t, elem.ClassName)
			assert.Equal(t, "User", *elem.ClassName)
			assert.Equal(t, "public", elem.Metadata["visibility"])
		}

		// Check for field
		if elem.ElementType == "code_data" && elem.Metadata["data_kind"] == "field" {
			foundField = true
			assert.Equal(t, "private", elem.Metadata["visibility"])
		}

		// Check for import
		if elem.ElementType == "code_dependency" && elem.Metadata["dependency_kind"] == "import" {
			foundImport = true
			assert.NotEmpty(t, elem.Metadata["target_namespace"])
		}

		// Check for Javadoc
		if elem.ElementType == "code_documentation" && elem.Metadata["doc_kind"] == "javadoc" {
			foundJavadoc = true
		}

		// Check for annotation
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "toString" {
			annotations, ok := elem.Metadata["annotations"].([]string)
			if ok && len(annotations) > 0 {
				foundAnnotation = true
			}
		}
	}

	// Verify we found all expected elements
	assert.True(t, foundClass, "Should find User class")
	assert.True(t, foundConstructor, "Should find constructor")
	assert.True(t, foundMethod, "Should find getAge method")
	assert.True(t, foundField, "Should find fields")
	assert.True(t, foundImport, "Should find imports")
	assert.True(t, foundJavadoc, "Should find Javadoc comments")
	assert.True(t, foundAnnotation, "Should find @Override annotation")
}

// TestJavaCodeParser_InterfaceAndEnum validates parsing of interfaces and enums
func TestJavaCodeParser_InterfaceAndEnum(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "Test.java")

	testCode := `package com.example;

/**
 * Runnable interface for tasks.
 */
public interface Runnable {
    void run();

    default void execute() {
        run();
    }
}

/**
 * Status enum.
 */
public enum Status {
    ACTIVE,
    INACTIVE,
    PENDING
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	javaParser := parser.NewJavaCodeParser()
	req := parser.ParseRequest{
		ID:      "test-interface-enum",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := javaParser.Parse(ctx, req)
	require.NoError(t, err)

	foundInterface := false
	foundEnum := false

	for _, elem := range result.Elements {
		if elem.ElementType == "code_type" && elem.ClassName != nil {
			if *elem.ClassName == "Runnable" {
				foundInterface = true
				assert.Equal(t, "interface", elem.Metadata["code_element_kind"])
			}
			if *elem.ClassName == "Status" {
				foundEnum = true
				assert.Equal(t, "enum", elem.Metadata["code_element_kind"])
			}
		}
	}

	assert.True(t, foundInterface, "Should find Runnable interface")
	assert.True(t, foundEnum, "Should find Status enum")
}

// TestJavaCodeParser_EntityReferences validates extraction of method calls and type usage
func TestJavaCodeParser_EntityReferences(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "Test.java")

	testCode := `package com.example;

import java.util.List;

public class Service {

    public void process(List<String> items) {
        validate(items);
        items.size();
    }

    private void validate(List<String> items) {
        // validation logic
    }
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	javaParser := parser.NewJavaCodeParser()
	req := parser.ParseRequest{
		ID:      "test-entity-references",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := javaParser.Parse(ctx, req)
	require.NoError(t, err)

	foundImport := false
	foundFunctionCall := false
	foundTypeUsage := false

	for _, elem := range result.Elements {
		if elem.ElementType != "code_dependency" {
			continue
		}

		depKind, ok := elem.Metadata["dependency_kind"].(string)
		if !ok {
			continue
		}

		t.Logf("Dependency: kind=%s, content=%s", depKind, elem.Content)

		switch depKind {
		case "import":
			foundImport = true

		case "function_call":
			foundFunctionCall = true
			t.Logf("  Function call: %s", elem.Metadata["target_function"])

		case "type_usage":
			foundTypeUsage = true
			t.Logf("  Type usage: %s", elem.Metadata["target_type"])
		}
	}

	assert.True(t, foundImport, "Should find imports")
	assert.True(t, foundFunctionCall, "Should find method calls")
	assert.True(t, foundTypeUsage, "Should find type usage")
}

// TestJavaCodeParser_InterfaceRegistration validates parser can be registered
func TestJavaCodeParser_InterfaceRegistration(t *testing.T) {
	registry := parser.NewParserRegistry()
	javaParser := parser.NewJavaCodeParser()

	registry.Register(javaParser)

	// Retrieve by name
	retrieved, err := registry.GetParser("java_code")
	require.NoError(t, err)
	assert.Equal(t, "java_code", retrieved.GetName())

	// Retrieve by file extension
	byExt, err := registry.GetParserForFile("Main.java")
	require.NoError(t, err)
	assert.Equal(t, "java_code", byExt.GetName())
}

// TestJavaCodeParser_InlineComments validates extraction of inline comments within method bodies
func TestJavaCodeParser_InlineComments(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "Test.java")

	testCode := `package com.example;

public class DataProcessor {
    /**
     * Process data with validation.
     * This is a Javadoc comment, should NOT be extracted as inline.
     */
    public int processData() {
        // TODO: Add validation logic here
        // Validate input parameters
        int x = 1;

        /* FIXME: This is a temporary workaround
           Need to refactor this section */
        int y = 2;

        // HACK: Quick fix for deadline
        int z = x + y;

        // NOTE: This calculation is simplified
        return z;
    }
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	javaParser := parser.NewJavaCodeParser()
	req := parser.ParseRequest{
		ID:      "test-inline-comments",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := javaParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Find the processData method
	var methodID string
	for _, elem := range result.Elements {
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "processData" {
			methodID = elem.ElementID
			break
		}
	}
	require.NotEmpty(t, methodID, "Should find processData method")

	// Track inline comments
	inlineComments := []parser.Element{}
	foundTODO := false
	foundFIXME := false
	foundHACK := false
	foundNOTE := false

	for _, elem := range result.Elements {
		if elem.ElementType == "code_documentation" &&
			elem.ParentID == methodID &&
			elem.Metadata["doc_kind"] == "inline_comment" {
			inlineComments = append(inlineComments, elem)

			t.Logf("Inline comment: %s (line %d)", elem.Content, *elem.LineNumber)

			// Check for markers
			if markers, ok := elem.Metadata["markers"].([]string); ok {
				for _, marker := range markers {
					switch marker {
					case "TODO":
						foundTODO = true
					case "FIXME":
						foundFIXME = true
					case "HACK":
						foundHACK = true
					case "NOTE":
						foundNOTE = true
					}
				}
			}

			// Verify metadata structure
			assert.Equal(t, "inline_comment", elem.Metadata["doc_kind"])
			assert.Equal(t, "java", elem.Metadata["language"])
			assert.NotNil(t, elem.LineNumber)
			assert.Contains(t, []string{"line_comment", "block_comment"}, elem.Metadata["comment_type"])
		}
	}

	// Verify we found inline comments (should NOT include Javadoc)
	assert.Greater(t, len(inlineComments), 0, "Should find inline comments")
	assert.True(t, foundTODO, "Should detect TODO marker")
	assert.True(t, foundFIXME, "Should detect FIXME marker")
	assert.True(t, foundHACK, "Should detect HACK marker")
	assert.True(t, foundNOTE, "Should detect NOTE marker")

	// Verify Javadoc was NOT extracted as inline comment
	for _, elem := range inlineComments {
		content := elem.Content
		assert.NotContains(t, content, "Javadoc", "Javadoc comments should not be extracted as inline comments")
	}

	t.Logf("Found %d inline comments in processData method", len(inlineComments))
}
