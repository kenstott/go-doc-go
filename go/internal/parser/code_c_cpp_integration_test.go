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

// TestCParser_BasicParsing validates the C parser with a simple C file
func TestCParser_BasicParsing(t *testing.T) {
	ctx := context.Background()

	// Create a temporary C file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.c")

	testCode := `#include <stdio.h>
#include <stdlib.h>

// Maximum buffer size
#define MAX_SIZE 1024

/**
 * User struct for managing user data.
 */
struct User {
    char *name;
    int age;
};

/**
 * Creates a new user.
 */
struct User* create_user(const char *name, int age) {
    struct User *user = malloc(sizeof(struct User));
    user->name = strdup(name);
    user->age = age;
    return user;
}

/**
 * Gets the user's age.
 */
int get_age(struct User *user) {
    return user->age;
}

/**
 * Frees a user.
 */
void free_user(struct User *user) {
    free(user->name);
    free(user);
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	// Parse the file
	cParser := parser.NewCCppCodeParser()
	req := parser.ParseRequest{
		ID:      "test-c-code",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := cParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify document metadata
	assert.Equal(t, "test-c-code", result.Document.ID)
	assert.Equal(t, "c_source", result.Document.DocType)

	// Verify we have elements
	assert.NotEmpty(t, result.Elements)

	// Track what we found
	foundStruct := false
	foundFunction := false
	foundInclude := false

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
			assert.Equal(t, "c", elem.Metadata["language"])
		}

		// Check for function
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "create_user" {
			foundFunction = true
			assert.NotNil(t, elem.Signature)
			assert.NotNil(t, elem.LineNumber)
			assert.Equal(t, "function", elem.Metadata["code_element_kind"])
		}

		// Check for include
		if elem.ElementType == "code_dependency" && elem.Metadata["dependency_kind"] == "include" {
			foundInclude = true
			assert.NotEmpty(t, elem.Metadata["target_file"])
		}
	}

	// Verify we found all expected elements
	assert.True(t, foundStruct, "Should find User struct")
	assert.True(t, foundFunction, "Should find create_user function")
	assert.True(t, foundInclude, "Should find includes")
}

// TestCppParser_BasicParsing validates the C++ parser with a simple C++ file
func TestCppParser_BasicParsing(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.cpp")

	testCode := `#include <iostream>
#include <string>
#include <vector>

namespace myapp {

/**
 * User class for managing user data.
 */
class User {
private:
    std::string name;
    int age;

public:
    /**
     * Constructor.
     */
    User(const std::string& name, int age) : name(name), age(age) {}

    /**
     * Gets the user's age.
     */
    int getAge() const {
        return age;
    }

    /**
     * Sets the user's age.
     */
    void setAge(int newAge) {
        age = newAge;
    }

    /**
     * Prints user info.
     */
    void print() const {
        std::cout << "User: " << name << ", Age: " << age << std::endl;
    }
};

/**
 * Creates and returns a new user.
 */
User createUser(const std::string& name, int age) {
    return User(name, age);
}

} // namespace myapp
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	cppParser := parser.NewCCppCodeParser()
	req := parser.ParseRequest{
		ID:      "test-cpp-code",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := cppParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify document metadata
	assert.Equal(t, "test-cpp-code", result.Document.ID)
	assert.Equal(t, "cpp_source", result.Document.DocType)

	// Track what we found
	foundClass := false
	foundMethod := false
	foundFunction := false
	foundNamespace := false
	foundInclude := false

	for _, elem := range result.Elements {
		t.Logf("Element: type=%s, category=%s, content_preview=%s",
			elem.ElementType, elem.ElementCategory, elem.ContentPreview)

		// Check for class
		if elem.ElementType == "code_grouping" && elem.ClassName != nil && *elem.ClassName == "User" {
			foundClass = true
			assert.Equal(t, "class", elem.Metadata["code_element_kind"])
			assert.Equal(t, "cpp", elem.Metadata["language"])
		}

		// Check for method
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "getAge" {
			foundMethod = true
			assert.NotNil(t, elem.ClassName)
			assert.Equal(t, "User", *elem.ClassName)
			assert.Equal(t, "method", elem.Metadata["code_element_kind"])
		}

		// Check for standalone function
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "createUser" {
			foundFunction = true
			assert.Equal(t, "function", elem.Metadata["code_element_kind"])
		}

		// Check for namespace
		if elem.ElementType == "code_grouping" && elem.Metadata["code_element_kind"] == "namespace" {
			foundNamespace = true
			assert.Equal(t, "myapp", elem.Metadata["namespace_name"])
		}

		// Check for include
		if elem.ElementType == "code_dependency" && elem.Metadata["dependency_kind"] == "include" {
			foundInclude = true
		}
	}

	// Verify we found all C++-specific elements
	assert.True(t, foundClass, "Should find User class")
	assert.True(t, foundMethod, "Should find getAge method")
	assert.True(t, foundFunction, "Should find createUser function")
	assert.True(t, foundNamespace, "Should find namespace")
	assert.True(t, foundInclude, "Should find includes")
}

// TestCCppParser_EntityReferences validates extraction of function calls
func TestCCppParser_EntityReferences(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.c")

	testCode := `#include <stdio.h>

void validate(int value) {
    // validation logic
}

void save(int value) {
    // save logic
}

void process(int value) {
    validate(value);
    printf("Processing: %d\n", value);
    save(value);
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	cParser := parser.NewCCppCodeParser()
	req := parser.ParseRequest{
		ID:      "test-entity-references",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := cParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	foundInclude := false
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
		case "include":
			foundInclude = true
			assert.NotEmpty(t, elem.Metadata["target_file"])

		case "function_call":
			foundFunctionCall = true
			functionCallCount++
			t.Logf("  Function call: %s", elem.Metadata["target_function"])
		}
	}

	assert.True(t, foundInclude, "Should find includes")
	assert.True(t, foundFunctionCall, "Should find function calls")
	assert.Greater(t, functionCallCount, 0, "Should have function call dependencies")

	t.Logf("Total function calls: %d", functionCallCount)
}

// TestCCppParser_InlineComments validates extraction of inline comments
func TestCCppParser_InlineComments(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.c")

	testCode := `/**
 * Data processor function.
 */
int process_data() {
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
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	cParser := parser.NewCCppCodeParser()
	req := parser.ParseRequest{
		ID:      "test-inline-comments",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := cParser.Parse(ctx, req)
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
			assert.Equal(t, "c", elem.Metadata["language"])
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

// TestCCppParser_InterfaceRegistration validates parser can be registered
func TestCCppParser_InterfaceRegistration(t *testing.T) {
	registry := parser.NewParserRegistry()
	cParser := parser.NewCCppCodeParser()

	registry.Register(cParser)

	// Retrieve by name
	retrieved, err := registry.GetParser("c_cpp_code")
	require.NoError(t, err)
	assert.Equal(t, "c_cpp_code", retrieved.GetName())

	// Retrieve by C file extension
	byExtC, err := registry.GetParserForFile("main.c")
	require.NoError(t, err)
	assert.Equal(t, "c_cpp_code", byExtC.GetName())

	// Retrieve by C++ file extension
	byExtCpp, err := registry.GetParserForFile("main.cpp")
	require.NoError(t, err)
	assert.Equal(t, "c_cpp_code", byExtCpp.GetName())

	// Retrieve by header file extension
	byExtH, err := registry.GetParserForFile("header.h")
	require.NoError(t, err)
	assert.Equal(t, "c_cpp_code", byExtH.GetName())

	// Retrieve by C++ header file extension
	byExtHpp, err := registry.GetParserForFile("header.hpp")
	require.NoError(t, err)
	assert.Equal(t, "c_cpp_code", byExtHpp.GetName())
}

// TestCppParser_Templates validates parsing of C++ templates
func TestCppParser_Templates(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.cpp")

	testCode := `#include <iostream>

template<typename T>
class Container {
private:
    T value;

public:
    Container(T val) : value(val) {}

    T getValue() const {
        return value;
    }

    void setValue(T val) {
        value = val;
    }
};

template<typename T>
T max(T a, T b) {
    return (a > b) ? a : b;
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	cppParser := parser.NewCCppCodeParser()
	req := parser.ParseRequest{
		ID:      "test-templates",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := cppParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	foundClass := false
	foundMethod := false
	foundFunction := false

	for _, elem := range result.Elements {
		if elem.ElementType == "code_grouping" && elem.ClassName != nil && *elem.ClassName == "Container" {
			foundClass = true
		}

		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "getValue" {
			foundMethod = true
		}

		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "max" {
			foundFunction = true
		}
	}

	assert.True(t, foundClass, "Should find Container template class")
	assert.True(t, foundMethod, "Should find getValue method")
	assert.True(t, foundFunction, "Should find max template function")
}

// TestCParser_Structs validates parsing of C structs and unions
func TestCParser_Structs(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.c")

	testCode := `#include <stdint.h>

struct Point {
    int x;
    int y;
};

union Data {
    int i;
    float f;
    char str[20];
};

enum Color {
    RED,
    GREEN,
    BLUE
};

struct Point create_point(int x, int y) {
    struct Point p = {x, y};
    return p;
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	cParser := parser.NewCCppCodeParser()
	req := parser.ParseRequest{
		ID:      "test-structs",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := cParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	foundStruct := false
	foundUnion := false
	foundEnum := false
	foundFunction := false

	for _, elem := range result.Elements {
		if elem.ElementType == "code_grouping" && elem.ClassName != nil {
			if *elem.ClassName == "Point" {
				foundStruct = true
				assert.Equal(t, "struct", elem.Metadata["code_element_kind"])
			}
			if *elem.ClassName == "Data" {
				foundUnion = true
				assert.Equal(t, "union", elem.Metadata["code_element_kind"])
			}
			if *elem.ClassName == "Color" {
				foundEnum = true
				assert.Equal(t, "enum", elem.Metadata["code_element_kind"])
			}
		}

		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "create_point" {
			foundFunction = true
		}
	}

	assert.True(t, foundStruct, "Should find Point struct")
	assert.True(t, foundUnion, "Should find Data union")
	assert.True(t, foundEnum, "Should find Color enum")
	assert.True(t, foundFunction, "Should find create_point function")
}

// TestCCppParser_Pointers validates parsing of pointer declarations
func TestCCppParser_Pointers(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.c")

	testCode := `#include <stdlib.h>

int* allocate_array(int size) {
    return (int*)malloc(size * sizeof(int));
}

void free_array(int *arr) {
    free(arr);
}

char* get_string(void) {
    return "Hello";
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	cParser := parser.NewCCppCodeParser()
	req := parser.ParseRequest{
		ID:      "test-pointers",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := cParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	foundAllocate := false
	foundFree := false
	foundGetString := false

	for _, elem := range result.Elements {
		if elem.ElementType == "code_function" && elem.FunctionName != nil {
			if *elem.FunctionName == "allocate_array" {
				foundAllocate = true
			}
			if *elem.FunctionName == "free_array" {
				foundFree = true
			}
			if *elem.FunctionName == "get_string" {
				foundGetString = true
			}
		}
	}

	assert.True(t, foundAllocate, "Should find allocate_array function")
	assert.True(t, foundFree, "Should find free_array function")
	assert.True(t, foundGetString, "Should find get_string function")
}
