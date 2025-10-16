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

// TestGoCodeParser_BasicParsing validates the Go code parser with a simple Go file
func TestGoCodeParser_BasicParsing(t *testing.T) {
	ctx := context.Background()

	// Create a temporary Go file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	testCode := `package main

import "fmt"

// Greeting constant
const Greeting = "Hello, World!"

// User represents a user in the system
type User struct {
	Name string
	Age  int
}

// Greet prints a greeting message
func Greet(name string) {
	fmt.Printf("Hello, %s!\n", name)
}

// GetAge returns the user's age
func (u *User) GetAge() int {
	return u.Age
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	// Parse the file
	goParser := parser.NewGoCodeParser()
	req := parser.ParseRequest{
		ID:      "test-go-code",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := goParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify document metadata
	assert.Equal(t, "test-go-code", result.Document.ID)
	assert.Equal(t, "go_source", result.Document.DocType)
	assert.Equal(t, "main", result.Document.Metadata["package"])

	// Verify we have elements
	assert.NotEmpty(t, result.Elements)

	// Track what we found
	foundFunction := false
	foundMethod := false
	foundStruct := false
	foundConstant := false
	foundImport := false

	for _, elem := range result.Elements {
		t.Logf("Element: type=%s, category=%s, content_preview=%s",
			elem.ElementType, elem.ElementCategory, elem.ContentPreview)

		// Verify all code elements use universal types
		assert.Contains(t, []string{
			"document", "code_function", "code_grouping", "code_type",
			"code_data", "code_dependency", "code_documentation",
			"return_type", // structural element (not universal code-specific)
		}, elem.ElementType, "Element should use universal code model types")

		// Check for function
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "Greet" {
			foundFunction = true
			assert.NotNil(t, elem.Signature)
			assert.NotNil(t, elem.LineNumber)
			assert.Equal(t, "function", elem.Metadata["code_element_kind"])
			assert.Equal(t, "go", elem.Metadata["language"])
			assert.Equal(t, "public", elem.Metadata["visibility"])
		}

		// Check for method
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "GetAge" {
			foundMethod = true
			assert.NotNil(t, elem.ClassName)
			assert.Equal(t, "User", *elem.ClassName)
			assert.Equal(t, "method", elem.Metadata["code_element_kind"])
			assert.NotEmpty(t, elem.Metadata["receiver_type"])
		}

		// Check for struct
		if elem.ElementType == "code_grouping" && elem.ClassName != nil && *elem.ClassName == "User" {
			foundStruct = true
			assert.Equal(t, "struct", elem.Metadata["code_element_kind"])
			assert.Equal(t, "public", elem.Metadata["visibility"])
		}

		// Check for constant
		if elem.ElementType == "code_data" && elem.Metadata["data_kind"] == "constant" {
			foundConstant = true
			assert.Equal(t, "immutable", elem.Metadata["mutability"])
			assert.Equal(t, "global", elem.Metadata["scope"])
		}

		// Check for import
		if elem.ElementType == "code_dependency" && elem.Metadata["dependency_kind"] == "import" {
			foundImport = true
			assert.Equal(t, "import", elem.Metadata["dependency_kind"])
			assert.NotEmpty(t, elem.Metadata["target_namespace"])
		}
	}

	// Verify we found all expected elements
	assert.True(t, foundFunction, "Should find Greet function")
	assert.True(t, foundMethod, "Should find GetAge method")
	assert.True(t, foundStruct, "Should find User struct")
	assert.True(t, foundConstant, "Should find Greeting constant")
	assert.True(t, foundImport, "Should find fmt import")
}

// TestGoCodeParser_PromotedFields validates that code-specific promoted fields are populated
func TestGoCodeParser_PromotedFields(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	testCode := `package example

// Calculate adds two numbers
func Calculate(a, b int) int {
	return a + b
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	goParser := parser.NewGoCodeParser()
	req := parser.ParseRequest{
		ID:      "test-promoted-fields",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := goParser.Parse(ctx, req)
	require.NoError(t, err)

	// Find the function element
	foundFunction := false
	for _, elem := range result.Elements {
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "Calculate" {
			foundFunction = true

			// Verify promoted fields
			assert.NotNil(t, elem.FunctionName, "FunctionName should be populated")
			assert.Equal(t, "Calculate", *elem.FunctionName)

			assert.NotNil(t, elem.Signature, "Signature should be populated")
			assert.Contains(t, *elem.Signature, "Calculate")

			assert.NotNil(t, elem.Namespace, "Namespace should be populated")
			assert.Contains(t, *elem.Namespace, "example")

			assert.NotNil(t, elem.LineNumber, "LineNumber should be populated")
			assert.Greater(t, *elem.LineNumber, 0)

			// Verify universal metadata
			assert.Equal(t, "function", elem.Metadata["code_element_kind"])
			assert.Equal(t, "public", elem.Metadata["visibility"])
			assert.Equal(t, "go", elem.Metadata["language"])
		}
	}

	assert.True(t, foundFunction, "Should find Calculate function with promoted fields")
}

// TestGoCodeParser_ElementHierarchy validates parent-child relationships
func TestGoCodeParser_ElementHierarchy(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	testCode := `package test

type Person struct {
	Name string
	Age  int
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	goParser := parser.NewGoCodeParser()
	req := parser.ParseRequest{
		ID:      "test-hierarchy",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := goParser.Parse(ctx, req)
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

	// Find the struct
	var structElement parser.Element
	for _, elem := range result.Elements {
		if elem.ElementType == "code_grouping" && elem.ClassName != nil && *elem.ClassName == "Person" {
			structElement = elem
			break
		}
	}

	require.NotEmpty(t, structElement.ElementID, "Should find Person struct")

	// Verify struct has children (fields)
	children := parentChildMap[structElement.ElementID]
	assert.Greater(t, len(children), 0, "Struct should have child elements (fields)")

	// Verify fields are code_data elements
	fieldCount := 0
	for _, childID := range children {
		child := elementMap[childID]
		if child.ElementType == "code_data" && child.Metadata["data_kind"] == "field" {
			fieldCount++
		}
	}
	assert.Equal(t, 2, fieldCount, "Should have 2 field elements")
}

// TestGoCodeParser_InterfaceRegistration validates parser can be registered
func TestGoCodeParser_InterfaceRegistration(t *testing.T) {
	registry := parser.NewParserRegistry()
	goParser := parser.NewGoCodeParser()

	registry.Register(goParser)

	// Retrieve by name
	retrieved, err := registry.GetParser("go_code")
	require.NoError(t, err)
	assert.Equal(t, "go_code", retrieved.GetName())

	// Retrieve by file extension
	byExt, err := registry.GetParserForFile("main.go")
	require.NoError(t, err)
	assert.Equal(t, "go_code", byExt.GetName())
}

// TestGoCodeParser_EntityReferences validates extraction of function calls and type usage
func TestGoCodeParser_EntityReferences(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	testCode := `package main

import "fmt"

type User struct {
	Name string
	Age  int
}

// Greet prints a greeting message
func Greet(u *User) {
	fmt.Println(u.Name)
	Validate(u)
}

// Validate checks if user is valid
func Validate(u *User) bool {
	return u.Name != ""
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	goParser := parser.NewGoCodeParser()
	req := parser.ParseRequest{
		ID:      "test-entity-references",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := goParser.Parse(ctx, req)
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
		case "import":
			if elem.Content == "fmt" {
				foundImport = true
			}

		case "function_call":
			functionCallCount++
			targetFunc, ok := elem.Metadata["target_function"].(string)
			if ok {
				t.Logf("  Function call: %s", targetFunc)
				if targetFunc == "fmt.Println" || targetFunc == "Validate" {
					foundFunctionCall = true
				}
			}

		case "type_usage":
			typeUsageCount++
			context, _ := elem.Metadata["usage_context"].(string)
			targetType, ok := elem.Metadata["target_type"].(string)
			if ok {
				t.Logf("  Type usage: %s in %s", targetType, context)
				if targetType == "*User" {
					foundTypeUsage = true
				}
			}
		}
	}

	// Verify we found all expected entity references
	assert.True(t, foundImport, "Should find fmt import")
	assert.True(t, foundFunctionCall, "Should find function calls (fmt.Println, Validate)")
	assert.True(t, foundTypeUsage, "Should find type usage (*User in parameters)")

	// Verify counts
	assert.Greater(t, functionCallCount, 0, "Should have function call dependencies")
	assert.Greater(t, typeUsageCount, 0, "Should have type usage dependencies")

	t.Logf("Total function calls: %d", functionCallCount)
	t.Logf("Total type usages: %d", typeUsageCount)
}

// TestGoCodeParser_CallGraphAnalysis validates call graph can be reconstructed
func TestGoCodeParser_CallGraphAnalysis(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	testCode := `package main

func HandleRequest() {
	ValidateInput()
	ProcessData()
	SaveResults()
}

func ValidateInput() {}
func ProcessData() {}
func SaveResults() {}
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	goParser := parser.NewGoCodeParser()
	req := parser.ParseRequest{
		ID:      "test-call-graph",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := goParser.Parse(ctx, req)
	require.NoError(t, err)

	// Build call graph: find all function calls within HandleRequest
	var handleRequestID string
	for _, elem := range result.Elements {
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "HandleRequest" {
			handleRequestID = elem.ElementID
			break
		}
	}

	require.NotEmpty(t, handleRequestID, "Should find HandleRequest function")

	// Find all calls within HandleRequest
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
	assert.Contains(t, callTargets, "ValidateInput", "HandleRequest should call ValidateInput")
	assert.Contains(t, callTargets, "ProcessData", "HandleRequest should call ProcessData")
	assert.Contains(t, callTargets, "SaveResults", "HandleRequest should call SaveResults")
	assert.Equal(t, 3, len(callTargets), "Should have exactly 3 function calls")

	t.Logf("Call graph from HandleRequest: %v", callTargets)
}
