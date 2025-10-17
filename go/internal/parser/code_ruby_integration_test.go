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

// TestRubyParser_BasicParsing validates the Ruby parser with a simple Ruby file
func TestRubyParser_BasicParsing(t *testing.T) {
	ctx := context.Background()

	// Create a temporary Ruby file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.rb")

	testCode := `require 'json'

# User class for managing user data
class User
  attr_reader :name
  attr_accessor :age

  # Initialize a new user
  def initialize(name, age)
    @name = name
    @age = age
  end

  # Get the user's age
  def get_age
    @age
  end

  # Class method for creating user
  def self.create(name, age)
    new(name, age)
  end
end

# Greet function
def greet(name)
  puts "Hello, #{name}!"
end
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	// Parse the file
	rubyParser := parser.NewRubyCodeParser()
	req := parser.ParseRequest{
		ID:      "test-ruby-code",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := rubyParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify document metadata
	assert.Equal(t, "test-ruby-code", result.Document.ID)
	assert.Equal(t, "ruby_source", result.Document.DocType)

	// Verify we have elements
	assert.NotEmpty(t, result.Elements)

	// Track what we found
	foundClass := false
	foundInstanceMethod := false
	foundClassMethod := false
	foundFunction := false
	foundImport := false
	foundAttribute := false

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
			assert.Equal(t, "ruby", elem.Metadata["language"])
		}

		// Check for instance method
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "get_age" {
			foundInstanceMethod = true
			assert.NotNil(t, elem.ClassName)
			assert.Equal(t, "User", *elem.ClassName)
			assert.Equal(t, "instance_method", elem.Metadata["code_element_kind"])
		}

		// Check for class method
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "create" {
			foundClassMethod = true
			assert.NotNil(t, elem.ClassName)
			assert.Equal(t, "User", *elem.ClassName)
			assert.Equal(t, "class_method", elem.Metadata["code_element_kind"])
			assert.True(t, elem.Metadata["is_singleton"].(bool))
		}

		// Check for function
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "greet" {
			foundFunction = true
			assert.NotNil(t, elem.Signature)
			assert.NotNil(t, elem.LineNumber)
			assert.Equal(t, "function", elem.Metadata["code_element_kind"])
		}

		// Check for import
		if elem.ElementType == "code_dependency" && elem.Metadata["dependency_kind"] == "import" {
			foundImport = true
			assert.NotEmpty(t, elem.Metadata["target_namespace"])
			assert.Equal(t, "json", elem.Metadata["target_namespace"])
		}

		// Check for attribute
		if elem.ElementType == "code_data" && elem.Metadata["code_element_kind"] == "attribute" {
			foundAttribute = true
			assert.NotNil(t, elem.ClassName)
			assert.Equal(t, "User", *elem.ClassName)
		}
	}

	// Verify we found all expected elements
	assert.True(t, foundClass, "Should find User class")
	assert.True(t, foundInstanceMethod, "Should find get_age instance method")
	assert.True(t, foundClassMethod, "Should find create class method")
	assert.True(t, foundFunction, "Should find greet function")
	assert.True(t, foundImport, "Should find require statement")
	assert.True(t, foundAttribute, "Should find attributes")
}

// TestRubyParser_ModuleAndMixins validates parsing of modules
func TestRubyParser_ModuleAndMixins(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.rb")

	testCode := `# Utilities module
module Utils
  # Validate method
  def self.validate(value)
    !value.nil?
  end

  # Process method
  def self.process(data)
    data.strip
  end
end

# Mixin module
module Comparable
  def compare(other)
    self <=> other
  end
end
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	rubyParser := parser.NewRubyCodeParser()
	req := parser.ParseRequest{
		ID:      "test-modules",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := rubyParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	foundUtilsModule := false
	foundComparableModule := false
	foundModuleMethod := false

	for _, elem := range result.Elements {
		if elem.ElementType == "code_grouping" && elem.Metadata["code_element_kind"] == "module" {
			moduleName := elem.Metadata["module_name"].(string)
			if moduleName == "Utils" {
				foundUtilsModule = true
			}
			if moduleName == "Comparable" {
				foundComparableModule = true
			}
		}

		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "validate" {
			foundModuleMethod = true
			assert.Equal(t, "Utils", *elem.Namespace)
		}
	}

	assert.True(t, foundUtilsModule, "Should find Utils module")
	assert.True(t, foundComparableModule, "Should find Comparable module")
	assert.True(t, foundModuleMethod, "Should find module method")
}

// TestRubyParser_EntityReferences validates extraction of function calls
func TestRubyParser_EntityReferences(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.rb")

	testCode := `require 'logger'

def validate(value)
  !value.nil?
end

def process(value)
  if validate(value)
    puts "Valid: #{value}"
    save(value)
  end
end

def save(value)
  File.write("data.txt", value)
end
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	rubyParser := parser.NewRubyCodeParser()
	req := parser.ParseRequest{
		ID:      "test-entity-references",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := rubyParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	foundImport := false
	foundFunctionCall := false
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
			assert.Equal(t, "logger", elem.Metadata["target_namespace"])

		case "function_call":
			functionCallCount++
			foundFunctionCall = true
			t.Logf("  Function call: %s", elem.Metadata["target_function"])
		}
	}

	assert.True(t, foundImport, "Should find require statement")
	assert.True(t, foundFunctionCall, "Should find function calls")
	assert.Greater(t, functionCallCount, 0, "Should have function call dependencies")

	t.Logf("Total function calls: %d", functionCallCount)
}

// TestRubyParser_InlineComments validates extraction of inline comments
func TestRubyParser_InlineComments(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.rb")

	testCode := `def process_data
  # TODO: Add validation logic here
  x = 1

  # FIXME: This is a temporary workaround
  # Need to refactor this section
  y = 2

  # HACK: Quick fix for deadline
  z = x + y

  # NOTE: This calculation is simplified
  z
end
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	rubyParser := parser.NewRubyCodeParser()
	req := parser.ParseRequest{
		ID:      "test-inline-comments",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := rubyParser.Parse(ctx, req)
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
			assert.Equal(t, "ruby", elem.Metadata["language"])
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

// TestRubyParser_InterfaceRegistration validates parser can be registered
func TestRubyParser_InterfaceRegistration(t *testing.T) {
	registry := parser.NewParserRegistry()
	rubyParser := parser.NewRubyCodeParser()

	registry.Register(rubyParser)

	// Retrieve by name
	retrieved, err := registry.GetParser("ruby_code")
	require.NoError(t, err)
	assert.Equal(t, "ruby_code", retrieved.GetName())

	// Retrieve by file extension
	byExt, err := registry.GetParserForFile("app.rb")
	require.NoError(t, err)
	assert.Equal(t, "ruby_code", byExt.GetName())
}

// TestRubyParser_ClassInheritance validates parsing of class inheritance
func TestRubyParser_ClassInheritance(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.rb")

	testCode := `class Animal
  def speak
    "Some sound"
  end
end

class Dog < Animal
  def speak
    "Woof!"
  end

  def fetch
    "Fetching..."
  end
end
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	rubyParser := parser.NewRubyCodeParser()
	req := parser.ParseRequest{
		ID:      "test-inheritance",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := rubyParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	foundAnimal := false
	foundDog := false

	for _, elem := range result.Elements {
		if elem.ElementType == "code_grouping" && elem.Metadata["code_element_kind"] == "class" {
			className := elem.Metadata["class_name"].(string)
			if className == "Animal" {
				foundAnimal = true
			}
			if className == "Dog" {
				foundDog = true
			}
		}
	}

	assert.True(t, foundAnimal, "Should find Animal class")
	assert.True(t, foundDog, "Should find Dog class")
}

// TestRubyParser_BlocksAndIterators validates parsing of blocks
func TestRubyParser_BlocksAndIterators(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.rb")

	testCode := `def process_items(items)
  items.each do |item|
    puts item
  end

  items.map { |x| x * 2 }
end
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	rubyParser := parser.NewRubyCodeParser()
	req := parser.ParseRequest{
		ID:      "test-blocks",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := rubyParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	foundFunction := false
	foundFunctionCalls := false

	for _, elem := range result.Elements {
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "process_items" {
			foundFunction = true
		}

		if elem.ElementType == "code_dependency" && elem.Metadata["dependency_kind"] == "function_call" {
			foundFunctionCalls = true
		}
	}

	assert.True(t, foundFunction, "Should find process_items function")
	assert.True(t, foundFunctionCalls, "Should find function calls")
}

// TestRubyParser_Constants validates parsing of constants
func TestRubyParser_Constants(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.rb")

	testCode := `MAX_SIZE = 100
DEFAULT_NAME = "Unknown"

class Config
  VERSION = "1.0.0"
  DEBUG = true
end
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	rubyParser := parser.NewRubyCodeParser()
	req := parser.ParseRequest{
		ID:      "test-constants",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := rubyParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	foundTopLevelConstant := false
	foundClassConstant := false

	for _, elem := range result.Elements {
		if elem.ElementType == "code_data" && elem.Metadata["code_element_kind"] == "constant" {
			constName := elem.Metadata["constant_name"].(string)
			if constName == "MAX_SIZE" {
				foundTopLevelConstant = true
			}
			if constName == "VERSION" {
				foundClassConstant = true
			}
		}
	}

	assert.True(t, foundTopLevelConstant, "Should find top-level constant")
	assert.True(t, foundClassConstant, "Should find class constant")
}
