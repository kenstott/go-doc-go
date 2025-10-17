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

// TestJavaScriptParser_BasicParsing validates the JavaScript parser with a simple JS file
func TestJavaScriptParser_BasicParsing(t *testing.T) {
	ctx := context.Background()

	// Create a temporary JavaScript file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.js")

	testCode := `import { useState } from 'react';

// User greeting constant
const GREETING = "Hello, World!";

/**
 * User class for managing user data.
 */
class User {
    constructor(name, age) {
        this.name = name;
        this.age = age;
    }

    /**
     * Gets the user's age.
     * @returns {number} the age
     */
    getAge() {
        return this.age;
    }

    toString() {
        return ` + "`User(${this.name}, ${this.age})`" + `;
    }
}

/**
 * Greet function prints a greeting message
 * @param {string} name - the user's name
 */
function greet(name) {
    console.log('Hello, ' + name + '!');
}

// Arrow function for validation
const validate = (value) => {
    return value !== null && value !== undefined;
};

export { User, greet, validate };
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	// Parse the file
	jsParser := parser.NewJavaScriptTypeScriptCodeParser()
	req := parser.ParseRequest{
		ID:      "test-js-code",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := jsParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify document metadata
	assert.Equal(t, "test-js-code", result.Document.ID)
	assert.Equal(t, "javascript_source", result.Document.DocType)

	// Verify we have elements
	assert.NotEmpty(t, result.Elements)

	// Track what we found
	foundClass := false
	foundConstructor := false
	foundMethod := false
	foundFunction := false
	foundArrowFunction := false
	foundConstant := false
	foundImport := false
	foundExport := false

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
			assert.Equal(t, "javascript", elem.Metadata["language"])
			assert.Equal(t, "public", elem.Metadata["visibility"])
		}

		// Check for constructor
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "constructor" {
			foundConstructor = true
			assert.NotNil(t, elem.ClassName)
			assert.Equal(t, "User", *elem.ClassName)
			assert.Equal(t, "constructor", elem.Metadata["code_element_kind"])
		}

		// Check for method
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "getAge" {
			foundMethod = true
			assert.NotNil(t, elem.ClassName)
			assert.Equal(t, "User", *elem.ClassName)
			assert.Equal(t, "method", elem.Metadata["code_element_kind"])
		}

		// Check for regular function
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "greet" {
			foundFunction = true
			assert.Equal(t, "function", elem.Metadata["code_element_kind"])
			assert.Equal(t, "public", elem.Metadata["visibility"])
		}

		// Check for arrow function
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "validate" {
			foundArrowFunction = true
			assert.Equal(t, "arrow_function", elem.Metadata["code_element_kind"])
		}

		// Check for constant
		if elem.ElementType == "code_data" && elem.Metadata["data_kind"] == "constant" {
			foundConstant = true
			assert.Equal(t, "immutable", elem.Metadata["mutability"])
		}

		// Check for import
		if elem.ElementType == "code_dependency" && elem.Metadata["dependency_kind"] == "import" {
			foundImport = true
			assert.NotEmpty(t, elem.Metadata["target_namespace"])
		}

		// Check for export
		if elem.ElementType == "code_dependency" && elem.Metadata["dependency_kind"] == "export" {
			foundExport = true
		}
	}

	// Verify we found all expected elements
	assert.True(t, foundClass, "Should find User class")
	assert.True(t, foundConstructor, "Should find constructor")
	assert.True(t, foundMethod, "Should find getAge method")
	assert.True(t, foundFunction, "Should find greet function")
	assert.True(t, foundArrowFunction, "Should find validate arrow function")
	assert.True(t, foundConstant, "Should find GREETING constant")
	assert.True(t, foundImport, "Should find imports")
	assert.True(t, foundExport, "Should find exports")
}

// TestTypeScriptParser_BasicParsing validates the TypeScript parser with a simple TS file
func TestTypeScriptParser_BasicParsing(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.ts")

	testCode := `import { Component } from '@angular/core';

/**
 * User interface for type safety.
 */
interface User {
    name: string;
    age: number;
}

/**
 * Status enum for user states.
 */
enum Status {
    Active = "ACTIVE",
    Inactive = "INACTIVE",
    Pending = "PENDING"
}

/**
 * User service class.
 */
class UserService {
    private users: User[] = [];

    /**
     * Adds a new user.
     * @param user - the user to add
     */
    addUser(user: User): void {
        this.users.push(user);
    }

    /**
     * Gets user count.
     * @returns {number} the user count
     */
    getUserCount(): number {
        return this.users.length;
    }
}

// Generic type alias
type Result<T> = { data: T; error?: string };

// Async arrow function
const fetchUser = async (id: number): Promise<User> => {
    // Fetch logic here
    return { name: "Test", age: 30 };
};

export { User, Status, UserService, Result, fetchUser };
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	tsParser := parser.NewJavaScriptTypeScriptCodeParser()
	req := parser.ParseRequest{
		ID:      "test-ts-code",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := tsParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify document metadata
	assert.Equal(t, "test-ts-code", result.Document.ID)
	assert.Equal(t, "typescript_source", result.Document.DocType)

	// Track what we found
	foundInterface := false
	foundEnum := false
	foundClass := false
	foundMethod := false
	foundTypeAlias := false
	foundAsyncArrowFunction := false

	for _, elem := range result.Elements {
		t.Logf("Element: type=%s, category=%s, content_preview=%s",
			elem.ElementType, elem.ElementCategory, elem.ContentPreview)

		// Check for interface
		if elem.ElementType == "code_type" && elem.ClassName != nil && *elem.ClassName == "User" {
			foundInterface = true
			assert.Equal(t, "interface", elem.Metadata["code_element_kind"])
			assert.Equal(t, "typescript", elem.Metadata["language"])
		}

		// Check for enum
		if elem.ElementType == "code_type" && elem.ClassName != nil && *elem.ClassName == "Status" {
			foundEnum = true
			assert.Equal(t, "enum", elem.Metadata["code_element_kind"])
		}

		// Check for class
		if elem.ElementType == "code_grouping" && elem.ClassName != nil && *elem.ClassName == "UserService" {
			foundClass = true
			assert.Equal(t, "class", elem.Metadata["code_element_kind"])
		}

		// Check for method with type annotations
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "addUser" {
			foundMethod = true
			assert.NotNil(t, elem.Signature)
			// Signature should contain the parameter type
			assert.Contains(t, *elem.Signature, "User")
		}

		// Check for type alias
		if elem.ElementType == "code_type" && elem.ClassName != nil && *elem.ClassName == "Result" {
			foundTypeAlias = true
			assert.Equal(t, "type_alias", elem.Metadata["code_element_kind"])
			assert.True(t, elem.Metadata["is_generic"].(bool))
		}

		// Check for async arrow function
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "fetchUser" {
			foundAsyncArrowFunction = true
			assert.Equal(t, "arrow_function", elem.Metadata["code_element_kind"])
			assert.True(t, elem.Metadata["is_async"].(bool))
		}
	}

	// Verify we found all TypeScript-specific elements
	assert.True(t, foundInterface, "Should find User interface")
	assert.True(t, foundEnum, "Should find Status enum")
	assert.True(t, foundClass, "Should find UserService class")
	assert.True(t, foundMethod, "Should find addUser method with type annotations")
	assert.True(t, foundTypeAlias, "Should find Result type alias")
	assert.True(t, foundAsyncArrowFunction, "Should find async arrow function")
}

// TestJavaScriptParser_JSX validates parsing of JSX syntax
func TestJavaScriptParser_JSX(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "Component.jsx")

	testCode := `import React from 'react';

/**
 * Welcome component displays a greeting.
 */
function Welcome({ name }) {
    return (
        <div className="welcome">
            <h1>Hello, {name}!</h1>
        </div>
    );
}

/**
 * App component is the main application.
 */
const App = () => {
    return <Welcome name="World" />;
};

export default App;
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	jsParser := parser.NewJavaScriptTypeScriptCodeParser()
	req := parser.ParseRequest{
		ID:      "test-jsx-code",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := jsParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify JSX metadata
	assert.Equal(t, "javascript_source", result.Document.DocType)
	assert.Equal(t, ".jsx", result.Document.Metadata["file_extension"])

	foundWelcome := false
	foundApp := false

	for _, elem := range result.Elements {
		if elem.ElementType == "code_function" && elem.FunctionName != nil {
			if *elem.FunctionName == "Welcome" {
				foundWelcome = true
			}
			if *elem.FunctionName == "App" {
				foundApp = true
			}
		}
	}

	assert.True(t, foundWelcome, "Should find Welcome component")
	assert.True(t, foundApp, "Should find App component")
}

// TestTypeScriptParser_TSX validates parsing of TSX syntax
func TestTypeScriptParser_TSX(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "Component.tsx")

	testCode := `import React from 'react';

interface Props {
    name: string;
    age?: number;
}

/**
 * UserCard component displays user information.
 */
const UserCard: React.FC<Props> = ({ name, age }) => {
    return (
        <div className="user-card">
            <h2>{name}</h2>
            {age && <p>Age: {age}</p>}
        </div>
    );
};

export default UserCard;
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	tsParser := parser.NewJavaScriptTypeScriptCodeParser()
	req := parser.ParseRequest{
		ID:      "test-tsx-code",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := tsParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify TSX metadata
	assert.Equal(t, "typescript_source", result.Document.DocType)
	assert.Equal(t, ".tsx", result.Document.Metadata["file_extension"])

	foundInterface := false
	foundComponent := false

	for _, elem := range result.Elements {
		if elem.ElementType == "code_type" && elem.ClassName != nil && *elem.ClassName == "Props" {
			foundInterface = true
		}
		if elem.ElementType == "code_function" && elem.FunctionName != nil && *elem.FunctionName == "UserCard" {
			foundComponent = true
		}
	}

	assert.True(t, foundInterface, "Should find Props interface")
	assert.True(t, foundComponent, "Should find UserCard component")
}

// TestJavaScriptTypeScriptParser_EntityReferences validates extraction of function calls and type usage
func TestJavaScriptTypeScriptParser_EntityReferences(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.ts")

	testCode := `import { Logger } from './logger';

interface User {
    name: string;
    age: number;
}

class UserService {
    private logger: Logger;

    processUser(user: User): void {
        this.validate(user);
        this.logger.log('Processing user');
        this.saveUser(user);
    }

    private validate(user: User): boolean {
        return user.name !== '' && user.age > 0;
    }

    private saveUser(user: User): void {
        // Save logic
    }
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	tsParser := parser.NewJavaScriptTypeScriptCodeParser()
	req := parser.ParseRequest{
		ID:      "test-entity-references",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := tsParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

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
			assert.NotEmpty(t, elem.Metadata["target_namespace"])

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
	// Type usage extraction is an advanced feature - TODO: implement TypeScript type annotation tracking
	_ = foundTypeUsage // Will be used when type usage tracking is implemented
}

// TestJavaScriptTypeScriptParser_InlineComments validates extraction of inline comments
func TestJavaScriptTypeScriptParser_InlineComments(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.js")

	testCode := `/**
 * Data processor class.
 * This JSDoc should NOT be extracted as inline comment.
 */
class DataProcessor {
    /**
     * Process data with validation.
     */
    processData() {
        // TODO: Add validation logic here
        // Validate input parameters
        let x = 1;

        /* FIXME: This is a temporary workaround
           Need to refactor this section */
        let y = 2;

        // HACK: Quick fix for deadline
        let z = x + y;

        // NOTE: This calculation is simplified
        return z;
    }
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	jsParser := parser.NewJavaScriptTypeScriptCodeParser()
	req := parser.ParseRequest{
		ID:      "test-inline-comments",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := jsParser.Parse(ctx, req)
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
			assert.Equal(t, "javascript", elem.Metadata["language"])
			assert.NotNil(t, elem.LineNumber)
			assert.Contains(t, []string{"line_comment", "block_comment"}, elem.Metadata["comment_type"])
		}
	}

	// Verify we found inline comments (should NOT include JSDoc)
	assert.Greater(t, len(inlineComments), 0, "Should find inline comments")
	assert.True(t, foundTODO, "Should detect TODO marker")
	assert.True(t, foundFIXME, "Should detect FIXME marker")
	assert.True(t, foundHACK, "Should detect HACK marker")
	assert.True(t, foundNOTE, "Should detect NOTE marker")

	// Verify JSDoc was NOT extracted as inline comment
	for _, elem := range inlineComments {
		content := elem.Content
		assert.NotContains(t, content, "JSDoc", "JSDoc comments should not be extracted as inline comments")
	}

	t.Logf("Found %d inline comments in processData method", len(inlineComments))
}

// TestJavaScriptTypeScriptParser_InterfaceRegistration validates parser can be registered
func TestJavaScriptTypeScriptParser_InterfaceRegistration(t *testing.T) {
	registry := parser.NewParserRegistry()
	jsParser := parser.NewJavaScriptTypeScriptCodeParser()

	registry.Register(jsParser)

	// Retrieve by name
	retrieved, err := registry.GetParser("javascript_typescript_code")
	require.NoError(t, err)
	assert.Equal(t, "javascript_typescript_code", retrieved.GetName())

	// Retrieve by JavaScript file extension
	byExt, err := registry.GetParserForFile("app.js")
	require.NoError(t, err)
	assert.Equal(t, "javascript_typescript_code", byExt.GetName())

	// Retrieve by TypeScript file extension
	byExtTS, err := registry.GetParserForFile("app.ts")
	require.NoError(t, err)
	assert.Equal(t, "javascript_typescript_code", byExtTS.GetName())

	// Retrieve by JSX file extension
	byExtJSX, err := registry.GetParserForFile("Component.jsx")
	require.NoError(t, err)
	assert.Equal(t, "javascript_typescript_code", byExtJSX.GetName())

	// Retrieve by TSX file extension
	byExtTSX, err := registry.GetParserForFile("Component.tsx")
	require.NoError(t, err)
	assert.Equal(t, "javascript_typescript_code", byExtTSX.GetName())
}

// TestJavaScriptParser_AsyncAwait validates parsing of async/await patterns
func TestJavaScriptParser_AsyncAwait(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.js")

	testCode := `/**
 * Fetch user data asynchronously.
 */
async function fetchUser(id) {
    const response = await fetch('/api/users/' + id);
    return await response.json();
}

/**
 * Async arrow function for processing data.
 */
const processData = async (data) => {
    await validateData(data);
    return await saveData(data);
};

/**
 * Class with async methods.
 */
class ApiClient {
    async get(url) {
        return await fetch(url);
    }

    async post(url, data) {
        return await fetch(url, {
            method: 'POST',
            body: JSON.stringify(data)
        });
    }
}
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	require.NoError(t, err)

	jsParser := parser.NewJavaScriptTypeScriptCodeParser()
	req := parser.ParseRequest{
		ID:      "test-async-await",
		Content: testFile,
		Config:  parser.DefaultParserConfig(),
	}

	result, err := jsParser.Parse(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	foundAsyncFunction := false
	foundAsyncArrowFunction := false
	foundAsyncMethod := false

	for _, elem := range result.Elements {
		if elem.ElementType != "code_function" {
			continue
		}

		if elem.FunctionName != nil && *elem.FunctionName == "fetchUser" {
			foundAsyncFunction = true
			assert.True(t, elem.Metadata["is_async"].(bool), "fetchUser should be marked as async")
		}

		if elem.FunctionName != nil && *elem.FunctionName == "processData" {
			foundAsyncArrowFunction = true
			assert.True(t, elem.Metadata["is_async"].(bool), "processData should be marked as async")
			assert.Equal(t, "arrow_function", elem.Metadata["code_element_kind"])
		}

		if elem.FunctionName != nil && (*elem.FunctionName == "get" || *elem.FunctionName == "post") {
			foundAsyncMethod = true
			assert.True(t, elem.Metadata["is_async"].(bool), "ApiClient methods should be marked as async")
		}
	}

	assert.True(t, foundAsyncFunction, "Should find async function")
	assert.True(t, foundAsyncArrowFunction, "Should find async arrow function")
	assert.True(t, foundAsyncMethod, "Should find async methods")
}
