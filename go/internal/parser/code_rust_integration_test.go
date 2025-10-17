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

// TestRustParser_BasicParsing validates the Rust parser with a simple Rust file
func TestRustParser_BasicParsing(t *testing.T) {
	ctx := context.Background()

	// Create a temporary Rust file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.rs")

	testCode := `use std::io;

/// A simple user struct
pub struct User {
    pub name: String,
    age: u32,
}

impl User {
    /// Creates a new user
    pub fn new(name: String, age: u32) -> User {
        User { name, age }
    }

    /// Gets the user's age
    pub fn get_age(&self) -> u32 {
        self.age
    }
}

pub fn main() {
    let user = User::new("Alice".to_string(), 30);
    println!("Age: {}", user.get_age());
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	// Parse the file
	rustParser := parser.NewRustCodeParser()
	req := parser.ParseRequest{
		ID:      "test-rust-code",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := rustParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify document metadata
	assert.Equal(t, "test-rust-code", result.Document.ID)
	assert.Equal(t, "rust_source", result.Document.DocType)

	// Verify we have elements
	assert.NotEmpty(t, result.Elements)

	// Track what we found
	foundStruct := false
	foundMethod := false
	foundFunction := false
	foundImport := false

	for _, elem := range result.Elements {
		t.Logf("Element: type=%s, category=%s, content_preview=%s",
			elem.ElementType, elem.ElementCategory, elem.ContentPreview)

		// Verify all code elements use universal types
		assert.Contains(t, []string{
			"document", "code_function", "code_grouping", "code_type",
			"code_data", "code_dependency", "code_documentation",
		}, elem.ElementType, "Element should use universal code model types")

		// Check for struct
		if elem.ElementType == "code_grouping" && elem.ClassName != nil && *elem.ClassName == "User" {
			foundStruct = true
			assert.Equal(t, "struct", elem.Metadata["code_element_kind"])
			assert.Equal(t, "rust", elem.Metadata["language"])
			assert.Equal(t, "public", elem.Metadata["visibility"])
		}

		// Check for method
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "get_age" {
			foundMethod = true
			assert.NotNil(t, elem.ClassName)
			assert.Equal(t, "User", *elem.ClassName)
			assert.Equal(t, "method", elem.Metadata["code_element_kind"])
		}

		// Check for function
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "main" {
			foundFunction = true
			assert.NotNil(t, elem.Signature)
			assert.NotNil(t, elem.LineNumber)
		}

		// Check for import
		if elem.ElementType == "code_dependency" && elem.Metadata["dependency_kind"] == "import" {
			foundImport = true
			assert.NotEmpty(t, elem.Metadata["target_namespace"])
		}
	}

	// Verify we found all expected elements
	assert.True(t, foundStruct, "Should find User struct")
	assert.True(t, foundMethod, "Should find get_age method")
	assert.True(t, foundFunction, "Should find main function")
	assert.True(t, foundImport, "Should find imports")
}

// TestRustParser_EnumAndTrait validates parsing of enums and traits
func TestRustParser_EnumAndTrait(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.rs")

	testCode := `pub enum Status {
    Active,
    Inactive,
    Pending,
}

pub trait Validate {
    fn is_valid(&self) -> bool;
    fn validate(&mut self) -> Result<(), String>;
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	rustParser := parser.NewRustCodeParser()
	req := parser.ParseRequest{
		ID:      "test-enum-trait",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := rustParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	foundEnum := false
	foundTrait := false

	for _, elem := range result.Elements {
		if elem.ElementType == "code_grouping" && elem.ClassName != nil && *elem.ClassName == "Status" {
			foundEnum = true
			assert.Equal(t, "enum", elem.Metadata["code_element_kind"])
		}

		if elem.ElementType == "code_type" && elem.ClassName != nil && *elem.ClassName == "Validate" {
			foundTrait = true
			assert.Equal(t, "trait", elem.Metadata["code_element_kind"])
		}
	}

	assert.True(t, foundEnum, "Should find Status enum")
	assert.True(t, foundTrait, "Should find Validate trait")
}

// TestRustParser_EntityReferences validates extraction of function calls
func TestRustParser_EntityReferences(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.rs")

	testCode := `use std::io;

fn validate(value: i32) -> bool {
    value > 0
}

fn process(value: i32) {
    if validate(value) {
        println!("Valid: {}", value);
    }
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	rustParser := parser.NewRustCodeParser()
	req := parser.ParseRequest{
		ID:      "test-entity-references",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := rustParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	foundImport := false
	foundFunctionCall := false
	foundMacro := false
	functionCallCount := 0

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
			assert.NotEmpty(t, elem.Metadata["target_namespace"])

		case "function_call":
			functionCallCount++
			if elem.Metadata["is_macro"] == true {
				foundMacro = true
				t.Logf("  Macro: %s", elem.Metadata["target_function"])
			} else {
				foundFunctionCall = true
				t.Logf("  Function call: %s", elem.Metadata["target_function"])
			}
		}
	}

	assert.True(t, foundImport, "Should find imports")
	assert.True(t, foundFunctionCall, "Should find function calls")
	assert.True(t, foundMacro, "Should find macro invocations")
	assert.Greater(t, functionCallCount, 0, "Should have function call dependencies")

	t.Logf("Total function calls/macros: %d", functionCallCount)
}

// TestRustParser_InlineComments validates extraction of inline comments
func TestRustParser_InlineComments(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.rs")

	testCode := `fn process_data() -> i32 {
    // TODO: Add validation logic here
    let x = 1;

    /* FIXME: This is a temporary workaround
       Need to refactor this section */
    let y = 2;

    // HACK: Quick fix for deadline
    let z = x + y;

    // NOTE: This calculation is simplified
    z
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	rustParser := parser.NewRustCodeParser()
	req := parser.ParseRequest{
		ID:      "test-inline-comments",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := rustParser.Parse(ctx, req)
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
			assert.Equal(t, "rust", elem.Metadata["language"])
			assert.NotNil(t, elem.LineNumber)
			assert.Contains(t, []string{"line_comment", "block_comment"}, elem.Metadata["comment_type"])
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

// TestRustParser_InterfaceRegistration validates parser can be registered
func TestRustParser_InterfaceRegistration(t *testing.T) {
	registry := parser.NewParserRegistry()
	rustParser := parser.NewRustCodeParser()

	registry.Register(rustParser)

	// Retrieve by name
	retrieved, err := registry.GetParser("rust_code")
	require.NoError(t, err)
	assert.Equal(t, "rust_code", retrieved.GetName())

	// Retrieve by file extension
	byExt, err := registry.GetParserForFile("main.rs")
	require.NoError(t, err)
	assert.Equal(t, "rust_code", byExt.GetName())
}

// TestRustParser_Module validates parsing of modules
func TestRustParser_Module(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.rs")

	testCode := `pub mod utils {
    pub fn helper() {
        println!("Helper function");
    }
}

mod internal {
    fn private_func() {
        // Internal implementation
    }
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	rustParser := parser.NewRustCodeParser()
	req := parser.ParseRequest{
		ID:      "test-modules",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := rustParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	foundPublicModule := false
	foundPrivateModule := false

	for _, elem := range result.Elements {
		if elem.ElementType == "code_grouping" && elem.Metadata["code_element_kind"] == "module" {
			moduleName := elem.Metadata["module_name"].(string)
			if moduleName == "utils" {
				foundPublicModule = true
				assert.Equal(t, "public", elem.Metadata["visibility"])
			}
			if moduleName == "internal" {
				foundPrivateModule = true
				assert.Equal(t, "private", elem.Metadata["visibility"])
			}
		}
	}

	assert.True(t, foundPublicModule, "Should find public module")
	assert.True(t, foundPrivateModule, "Should find private module")
}

// TestRustParser_Visibility validates parsing of visibility modifiers
func TestRustParser_Visibility(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.rs")

	testCode := `pub struct PublicStruct {
    pub public_field: i32,
    private_field: i32,
}

pub fn public_function() {
    // Public function
}

fn private_function() {
    // Private function
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	rustParser := parser.NewRustCodeParser()
	req := parser.ParseRequest{
		ID:      "test-visibility",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := rustParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	foundPublicStruct := false
	foundPublicFunction := false
	foundPrivateFunction := false

	for _, elem := range result.Elements {
		if elem.ElementType == "code_grouping" && elem.ClassName != nil && *elem.ClassName == "PublicStruct" {
			foundPublicStruct = true
			assert.Equal(t, "public", elem.Metadata["visibility"])
		}

		if elem.ElementType == "code_function" && elem.FunctionName != nil {
			if *elem.FunctionName == "public_function" {
				foundPublicFunction = true
				assert.Equal(t, "public", elem.Metadata["visibility"])
			}
			if *elem.FunctionName == "private_function" {
				foundPrivateFunction = true
				assert.Equal(t, "private", elem.Metadata["visibility"])
			}
		}
	}

	assert.True(t, foundPublicStruct, "Should find public struct")
	assert.True(t, foundPublicFunction, "Should find public function")
	assert.True(t, foundPrivateFunction, "Should find private function")
}
