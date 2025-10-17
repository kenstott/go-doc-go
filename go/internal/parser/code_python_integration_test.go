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

// TestPythonCodeParser_BasicParsing validates the Python code parser with a simple Python file
func TestPythonCodeParser_BasicParsing(t *testing.T) {
	ctx := context.Background()

	// Create a temporary Python file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.py")

	testCode := `"""Module docstring."""
import os
from typing import List

# Global constant
MAX_SIZE = 100

class User:
    """User class for managing user data."""

    def __init__(self, name: str, age: int):
        """Initialize a new User."""
        self.name = name
        self.age = age

    def get_age(self) -> int:
        """Return the user's age."""
        return self.age

    @property
    def display_name(self) -> str:
        """Return formatted display name."""
        return self.name.upper()

def greet(name: str) -> None:
    """Print a greeting message."""
    print(f"Hello, {name}!")
    validate(name)

def validate(name: str) -> bool:
    """Validate the name is not empty."""
    return len(name) > 0
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	// Parse the file
	pyParser := parser.NewPythonCodeParser()
	req := parser.ParseRequest{
		ID:      "test-python-code",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := pyParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify document metadata
	assert.Equal(t, "test-python-code", result.Document.ID)
	assert.Equal(t, "python_source", result.Document.DocType)
	assert.Equal(t, "test", result.Document.Metadata["module"])

	// Verify we have elements
	assert.NotEmpty(t, result.Elements)

	// Track what we found
	foundFunction := false
	foundMethod := false
	foundClass := false
	foundConstant := false
	foundImport := false
	foundDocstring := false
	foundProperty := false

	for _, elem := range result.Elements {
		t.Logf("Element: type=%s, category=%s, content_preview=%s",
			elem.ElementType, elem.ElementCategory, elem.ContentPreview)

		// Verify all code elements use universal types
		assert.Contains(t, []string{
			"document", "code_function", "code_grouping", "code_type",
			"code_data", "code_dependency", "code_documentation",
		}, elem.ElementType, "Element should use universal code model types")

		// Check for function
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "greet" {
			foundFunction = true
			assert.NotNil(t, elem.Signature)
			assert.NotNil(t, elem.LineNumber)
			assert.Equal(t, "function", elem.Metadata["code_element_kind"])
			assert.Equal(t, "python", elem.Metadata["language"])
			assert.Equal(t, "public", elem.Metadata["visibility"])
		}

		// Check for method
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "get_age" {
			foundMethod = true
			assert.NotNil(t, elem.ClassName)
			assert.Equal(t, "method", elem.Metadata["code_element_kind"])
		}

		// Check for property
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "display_name" {
			foundProperty = true
			assert.Equal(t, "property", elem.Metadata["code_element_kind"])
			decorators, ok := elem.Metadata["decorators"].([]string)
			if ok && len(decorators) > 0 {
				assert.Contains(t, decorators[0], "@property")
			}
		}

		// Check for class
		if elem.ElementType == "code_grouping" && elem.ClassName != nil && *elem.ClassName == "User" {
			foundClass = true
			assert.Equal(t, "class", elem.Metadata["code_element_kind"])
			assert.Equal(t, "public", elem.Metadata["visibility"])
		}

		// Check for constant
		if elem.ElementType == "code_data" && elem.Metadata["data_kind"] == "variable" {
			if content, ok := elem.Metadata["data_name"].(string); ok && content == "MAX_SIZE" {
				foundConstant = true
				assert.Equal(t, "module", elem.Metadata["scope"])
			}
		}

		// Check for import
		if elem.ElementType == "code_dependency" && elem.Metadata["dependency_kind"] == "import" {
			foundImport = true
			assert.NotEmpty(t, elem.Metadata["target_namespace"])
		}

		// Check for docstring
		if elem.ElementType == "code_documentation" && elem.Metadata["doc_kind"] == "docstring" {
			foundDocstring = true
		}
	}

	// Verify we found all expected elements
	assert.True(t, foundFunction, "Should find greet function")
	assert.True(t, foundMethod, "Should find get_age method")
	assert.True(t, foundClass, "Should find User class")
	assert.True(t, foundConstant, "Should find MAX_SIZE constant")
	assert.True(t, foundImport, "Should find imports")
	assert.True(t, foundDocstring, "Should find docstrings")
	assert.True(t, foundProperty, "Should find property decorator")
}

// TestPythonCodeParser_PromotedFields validates that code-specific promoted fields are populated
func TestPythonCodeParser_PromotedFields(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.py")

	testCode := `"""Example module."""

def calculate(a: int, b: int) -> int:
    """Add two numbers."""
    return a + b
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	pyParser := parser.NewPythonCodeParser()
	req := parser.ParseRequest{
		ID:      "test-promoted-fields",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := pyParser.Parse(ctx, req)
	require.NoError(t, err)

	// Find the function element
	foundFunction := false
	for _, elem := range result.Elements {
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "calculate" {
			foundFunction = true

			// Verify promoted fields
			assert.NotNil(t, elem.FunctionName, "FunctionName should be populated")
			assert.Equal(t, "calculate", *elem.FunctionName)

			assert.NotNil(t, elem.Signature, "Signature should be populated")
			assert.Contains(t, *elem.Signature, "calculate")

			assert.NotNil(t, elem.Namespace, "Namespace should be populated")
			assert.Contains(t, *elem.Namespace, "test")

			assert.NotNil(t, elem.LineNumber, "LineNumber should be populated")
			assert.Greater(t, *elem.LineNumber, 0)

			// Verify universal metadata
			assert.Equal(t, "function", elem.Metadata["code_element_kind"])
			assert.Equal(t, "public", elem.Metadata["visibility"])
			assert.Equal(t, "python", elem.Metadata["language"])
		}
	}

	assert.True(t, foundFunction, "Should find calculate function with promoted fields")
}

// TestPythonCodeParser_ElementHierarchy validates parent-child relationships
func TestPythonCodeParser_ElementHierarchy(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.py")

	testCode := `class Person:
    """Person class."""

    name: str
    age: int

    def __init__(self, name: str, age: int):
        self.name = name
        self.age = age
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	pyParser := parser.NewPythonCodeParser()
	req := parser.ParseRequest{
		ID:      "test-hierarchy",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := pyParser.Parse(ctx, req)
	require.NoError(t, err)

	// Build parent-child map
	parentChildMap := make(map[string][]string)
	elementMap := make(map[string]parser.Element)

	for _, elem := range result.Elements {
		elementMap[elem.ElementID] = elem
		if elem.ParentID != "" {
			parentChildMap[elem.ParentID] = append(parentChildMap[elem.ParentID], elem.ElementID)
		}
	}

	// Find the class
	var classElement parser.Element
	for _, elem := range result.Elements {
		if elem.ElementType == "code_grouping" && elem.ClassName != nil && *elem.ClassName == "Person" {
			classElement = elem
			break
		}
	}

	require.NotEmpty(t, classElement.ElementID, "Should find Person class")

	// Verify class has children (methods, docstrings)
	children := parentChildMap[classElement.ElementID]
	assert.Greater(t, len(children), 0, "Class should have child elements (methods, docstrings)")

	// Verify at least one method is a child
	methodCount := 0
	for _, childID := range children {
		child := elementMap[childID]
		if child.ElementType == "code_function" {
			methodCount++
		}
	}
	assert.Greater(t, methodCount, 0, "Should have at least one method element")
}

// TestPythonCodeParser_InterfaceRegistration validates parser can be registered
func TestPythonCodeParser_InterfaceRegistration(t *testing.T) {
	registry := parser.NewParserRegistry()
	pyParser := parser.NewPythonCodeParser()

	registry.Register(pyParser)

	// Retrieve by name
	retrieved, err := registry.GetParser("python_code")
	require.NoError(t, err)
	assert.Equal(t, "python_code", retrieved.GetName())

	// Retrieve by file extension
	byExt, err := registry.GetParserForFile("main.py")
	require.NoError(t, err)
	assert.Equal(t, "python_code", byExt.GetName())
}

// TestPythonCodeParser_EntityReferences validates extraction of function calls and type usage
func TestPythonCodeParser_EntityReferences(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.py")

	testCode := `from typing import Optional

class User:
    """User class."""

    def __init__(self, name: str):
        self.name = name

def greet(user: User) -> None:
    """Greet a user."""
    print(user.name)
    validate(user)

def validate(user: User) -> bool:
    """Validate user data."""
    return user.name != ""
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	pyParser := parser.NewPythonCodeParser()
	req := parser.ParseRequest{
		ID:      "test-entity-references",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := pyParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Track what we found
	foundImport := false
	foundFunctionCall := false
	foundTypeUsage := false

	functionCallCount := 0
	typeUsageCount := 0

	for _, elem := range result.Elements {
		if elem.ElementType != "code_dependency" {
			continue
		}

		depKind, ok := elem.Metadata["dependency_kind"].(string)
		if !ok {
			continue
		}

		t.Logf("Dependency: kind=%s, content=%s, context=%v",
			depKind, elem.Content, elem.Metadata["usage_context"])

		switch depKind {
		case "import", "from_import":
			if elem.Content != "" {
				foundImport = true
			}

		case "function_call":
			functionCallCount++
			targetFunc, ok := elem.Metadata["target_function"].(string)
			if ok {
				t.Logf("  Function call: %s", targetFunc)
				if targetFunc == "print" || targetFunc == "validate" {
					foundFunctionCall = true
				}
			}

		case "type_usage":
			typeUsageCount++
			context, _ := elem.Metadata["usage_context"].(string)
			targetType, ok := elem.Metadata["target_type"].(string)
			if ok {
				t.Logf("  Type usage: %s in %s", targetType, context)
				if targetType == "User" || targetType == "Optional" {
					foundTypeUsage = true
				}
			}
		}
	}

	// Verify we found all expected entity references
	assert.True(t, foundImport, "Should find imports")
	assert.True(t, foundFunctionCall, "Should find function calls (print, validate)")
	assert.True(t, foundTypeUsage, "Should find type usage (User in parameters)")

	// Verify counts
	assert.Greater(t, functionCallCount, 0, "Should have function call dependencies")
	assert.Greater(t, typeUsageCount, 0, "Should have type usage dependencies")

	t.Logf("Total function calls: %d", functionCallCount)
	t.Logf("Total type usages: %d", typeUsageCount)
}

// TestPythonCodeParser_CallGraphAnalysis validates call graph can be reconstructed
func TestPythonCodeParser_CallGraphAnalysis(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.py")

	testCode := `def handle_request():
    """Handle incoming request."""
    validate_input()
    process_data()
    save_results()

def validate_input():
    pass

def process_data():
    pass

def save_results():
    pass
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	pyParser := parser.NewPythonCodeParser()
	req := parser.ParseRequest{
		ID:      "test-call-graph",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := pyParser.Parse(ctx, req)
	require.NoError(t, err)

	// Build call graph: find all function calls within handle_request
	var handleRequestID string
	for _, elem := range result.Elements {
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "handle_request" {
			handleRequestID = elem.ElementID
			break
		}
	}

	require.NotEmpty(t, handleRequestID, "Should find handle_request function")

	// Find all calls within handle_request
	callTargets := []string{}
	for _, elem := range result.Elements {
		if elem.ElementType == "code_dependency" &&
			elem.ParentID == handleRequestID &&
			elem.Metadata["dependency_kind"] == "function_call" {
			if targetFunc, ok := elem.Metadata["target_function"].(string); ok {
				callTargets = append(callTargets, targetFunc)
			}
		}
	}

	// Verify all three functions are called
	assert.Contains(t, callTargets, "validate_input", "handle_request should call validate_input")
	assert.Contains(t, callTargets, "process_data", "handle_request should call process_data")
	assert.Contains(t, callTargets, "save_results", "handle_request should call save_results")
	assert.Equal(t, 3, len(callTargets), "Should have exactly 3 function calls")

	t.Logf("Call graph from handle_request: %v", callTargets)
}

// TestPythonCodeParser_ClassMethods validates parsing of class methods and properties
func TestPythonCodeParser_ClassMethods(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.py")

	testCode := `class Calculator:
    """Simple calculator class."""

    def __init__(self, value: int = 0):
        self._value = value

    @property
    def value(self) -> int:
        """Get current value."""
        return self._value

    @staticmethod
    def add(a: int, b: int) -> int:
        """Add two numbers."""
        return a + b

    @classmethod
    def from_string(cls, s: str):
        """Create calculator from string."""
        return cls(int(s))
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	pyParser := parser.NewPythonCodeParser()
	req := parser.ParseRequest{
		ID:      "test-class-methods",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := pyParser.Parse(ctx, req)
	require.NoError(t, err)

	// Track what we found
	foundInit := false
	foundProperty := false
	foundStaticMethod := false
	foundClassMethod := false

	for _, elem := range result.Elements {
		if elem.ElementType != "code_function" {
			continue
		}

		funcName := ""
		if elem.FunctionName != nil {
			funcName = *elem.FunctionName
		}

		funcKind, _ := elem.Metadata["code_element_kind"].(string)

		t.Logf("Function: %s, kind: %s", funcName, funcKind)

		switch funcName {
		case "__init__":
			foundInit = true
			assert.Equal(t, "method", funcKind)

		case "value":
			foundProperty = true
			assert.Equal(t, "property", funcKind)

		case "add":
			foundStaticMethod = true
			assert.Equal(t, "static_method", funcKind)

		case "from_string":
			foundClassMethod = true
			assert.Equal(t, "class_method", funcKind)
		}
	}

	assert.True(t, foundInit, "Should find __init__ method")
	assert.True(t, foundProperty, "Should find @property decorated method")
	assert.True(t, foundStaticMethod, "Should find @staticmethod decorated method")
	assert.True(t, foundClassMethod, "Should find @classmethod decorated method")
}

// TestPythonCodeParser_InlineComments validates extraction of inline comments within function bodies
func TestPythonCodeParser_InlineComments(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.py")

	testCode := `def process_data():
    """Process data with validation."""
    # TODO: Add validation logic here
    # Validate input parameters
    x = 1

    # FIXME: This is a temporary workaround
    # Need to refactor this section
    y = 2

    # HACK: Quick fix for deadline
    z = x + y

    # NOTE: This calculation is simplified
    return z
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	pyParser := parser.NewPythonCodeParser()
	req := parser.ParseRequest{
		ID:      "test-inline-comments",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := pyParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Find the process_data function
	var funcID string
	for _, elem := range result.Elements {
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "process_data" {
			funcID = elem.ElementID
			break
		}
	}
	require.NotEmpty(t, funcID, "Should find process_data function")

	// Track inline comments
	inlineComments := []parser.Element{}
	foundTODO := false
	foundFIXME := false
	foundHACK := false
	foundNOTE := false

	for _, elem := range result.Elements {
		if elem.ElementType == "code_documentation" &&
			elem.ParentID == funcID &&
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
			assert.Equal(t, "python", elem.Metadata["language"])
			assert.NotNil(t, elem.LineNumber)
			assert.Equal(t, "line_comment", elem.Metadata["comment_type"])
		}
	}

	// Verify we found inline comments
	assert.Greater(t, len(inlineComments), 0, "Should find inline comments")
	assert.True(t, foundTODO, "Should detect TODO marker")
	assert.True(t, foundFIXME, "Should detect FIXME marker")
	assert.True(t, foundHACK, "Should detect HACK marker")
	assert.True(t, foundNOTE, "Should detect NOTE marker")

	t.Logf("Found %d inline comments in process_data function", len(inlineComments))
}
