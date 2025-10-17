# Code File Parsing - Universal Code Model

## Overview

Go-Doc-Go now supports parsing source code files with the **Universal Code Model (UCM)** - a language-agnostic abstraction layer that enables cross-language analysis, policy enforcement, requirements traceability, and documentation alignment.

## Status

✅ **Phase 1 Complete**: Universal Code Model & Go Parser
- Universal code element types defined
- Code-specific promoted fields added to Element struct
- Go language parser implemented using `go/ast`
- **Entity reference extraction** (function calls, type usage, imports)
- Integration tests passing
- Ready for use

✅ **Phase 2 Complete**: Python Parser
- Python language parser implemented using `tree-sitter`
- Full support for classes, methods, functions, decorators
- Entity reference extraction (imports, function calls, type usage)
- Decorator support (@property, @staticmethod, @classmethod)
- Integration tests passing
- Ready for use

✅ **Phase 3 Complete**: Java Parser
- Java language parser implemented using `tree-sitter`
- Full support for classes, interfaces, enums, methods, constructors
- Entity reference extraction (imports, method calls, type usage)
- Annotation support (@Override, @Deprecated, custom annotations)
- Javadoc documentation extraction
- Integration tests passing
- Ready for use

✅ **Phase 4 Complete**: JavaScript/TypeScript Parser
- JavaScript/TypeScript parser implemented using `tree-sitter`
- Unified parser supporting both languages with automatic detection
- Full support for functions, arrow functions, classes, methods, constructors
- Entity reference extraction (imports, function calls, exports)
- TypeScript-specific: interfaces, type aliases, enums, async/await
- JSX/TSX support (.jsx, .tsx file extensions)
- ES6 modules (import/export statements)
- Integration tests passing
- Ready for use

✅ **Phase 5 Complete**: C/C++ Parser
- C/C++ parser implemented using `tree-sitter`
- Unified parser supporting both languages with automatic file extension detection
- Full support for functions, methods, structs, classes, unions, enums
- C++-specific: namespaces, templates, constructors, access specifiers
- Entity reference extraction (#include directives, function calls)
- Pointer and reference support
- Integration tests passing
- Ready for use

✅ **Phase 6 Complete**: Rust Parser
- Rust parser implemented using `tree-sitter`
- Full support for functions, methods, structs, enums, traits, modules
- Entity reference extraction (use declarations, function calls, macro invocations)
- Visibility modifier support (pub/private)
- impl blocks with associated methods
- Doc comments (///) extraction
- Integration tests passing
- Ready for use

✅ **Phase 7 Complete**: Ruby Parser
- Ruby parser implemented using `tree-sitter`
- Full support for classes, modules, instance methods, class methods (singleton methods)
- Attributes (attr_reader, attr_writer, attr_accessor)
- Entity reference extraction (require/require_relative, function calls)
- Constants (top-level and class-level)
- Instance variables (@var) and class variables (@@var)
- Inline comment extraction with technical debt markers
- Integration tests passing
- Ready for use

✅ **Entity References Supported**:
- Package imports (`dependency_kind: "import"` or `"from_import"`)
- Function/method calls (`dependency_kind: "function_call"`)
- Type usage in parameters, returns, and fields (`dependency_kind: "type_usage"`)

## What is the Universal Code Model?

The UCM abstracts language-specific constructs into universal concepts:

| Universal Type | Maps To | Examples |
|----------------|---------|----------|
| **code_function** | Executable logic units | Go functions/methods, Java methods, Python functions, COBOL paragraphs, Prolog predicates |
| **code_grouping** | Organizational containers | Go structs/packages, Java classes, Python modules, COBOL divisions |
| **code_data** | Data definitions | Go variables/constants/fields, Java fields, Python variables, Perl scalars |
| **code_type** | Type definitions | Go type aliases, Java interfaces, TypeScript types |
| **code_dependency** | Relationships between code | Go imports, Java imports, Python imports, COBOL COPY statements |
| **code_documentation** | Code documentation | Go doc comments, Java Javadoc, Python docstrings |

## Powerful Use Cases

### 1. Policy Compliance
```sql
-- Find all public functions without documentation
SELECT
    function_name,
    namespace,
    line_number
FROM elements
WHERE element_type = 'code_function'
  AND metadata->>'visibility' = 'public'
  AND NOT EXISTS (
      SELECT 1 FROM elements doc
      WHERE doc.element_type = 'code_documentation'
        AND doc.parent_id = elements.element_id
  )
```

### 2. Requirements Traceability
```cypher
// Find code implementing specific requirements
MATCH (req:requirement {id: 'REQ-1234'})
      -[:implemented_by]->(func:code_function)
RETURN func.namespace, func.function_name, func.line_number
```

###  3. Cross-Language Queries
```sql
-- Count functions by language and visibility
SELECT
    metadata->>'language' as language,
    metadata->>'visibility' as visibility,
    COUNT(*) as function_count
FROM elements
WHERE element_type = 'code_function'
GROUP BY language, visibility
ORDER BY function_count DESC
```

### 4. Documentation Drift Detection
```sql
-- Find API functions where docs don't match signature
SELECT
    f.function_name,
    f.signature,
    d.content as doc_content
FROM elements f
LEFT JOIN elements d ON d.parent_id = f.element_id
    AND d.element_type = 'code_documentation'
WHERE f.element_type = 'code_function'
  AND f.metadata->>'visibility' = 'public'
  AND (d.content IS NULL OR d.content NOT LIKE '%' || f.function_name || '%')
```

### 5. Dependency Analysis
```sql
-- Find all code using a specific package
SELECT DISTINCT
    e.namespace as using_module,
    d.metadata->>'target_namespace' as dependency
FROM elements d
JOIN elements e ON e.element_id = d.parent_id
WHERE d.element_type = 'code_dependency'
  AND d.metadata->>'target_namespace' LIKE '%database%'
ORDER BY using_module
```

###  6. Call Graph Analysis
```sql
-- Find all functions called by a specific function
SELECT
    f.function_name as caller,
    d.metadata->>'target_function' as callee,
    d.line_number
FROM elements f
JOIN elements d ON d.parent_id = f.element_id
WHERE f.element_type = 'code_function'
  AND f.function_name = 'HandleRequest'
  AND d.element_type = 'code_dependency'
  AND d.metadata->>'dependency_kind' = 'function_call'
ORDER BY d.line_number;
```

### 7. Type Usage Analysis
```sql
-- Find all functions using a specific type
SELECT
    f.function_name,
    f.namespace,
    d.metadata->>'usage_context' as how_used
FROM elements f
JOIN elements d ON d.parent_id = f.element_id
WHERE f.element_type = 'code_function'
  AND d.element_type = 'code_dependency'
  AND d.metadata->>'dependency_kind' = 'type_usage'
  AND d.metadata->>'target_type' LIKE '%User%';
```

## Entity References

The parser captures **three types of entity references** as `code_dependency` elements:

### 1. Package Imports
```go
import "database/sql"  // Creates code_dependency with kind="import"
```

**Metadata:**
- `dependency_kind`: "import"
- `target_namespace`: "database/sql"
- `dependency_alias`: (if aliased)

### 2. Function Calls
```go
func Process() {
    Validate()      // Creates code_dependency with kind="function_call"
    fmt.Println()   // Creates code_dependency with kind="function_call"
}
```

**Metadata:**
- `dependency_kind`: "function_call"
- `target_function`: "Validate" or "fmt.Println"
- `call_type`: "direct" or "indirect"
- `line_number`: Source line where call occurs

### 3. Type Usage
```go
func Auth(u *User) error {  // Creates code_dependency with kind="type_usage"
    // ...
}

type Profile struct {
    User *User  // Creates code_dependency with kind="type_usage"
}
```

**Metadata:**
- `dependency_kind`: "type_usage"
- `usage_context`: "parameter", "return", or "field"
- `target_type`: "*User"
- `target_package`: Package containing the type

### Query Examples

**Find all callers of a function:**
```sql
SELECT f.function_name, d.line_number
FROM elements f
JOIN elements d ON d.parent_id = f.element_id
WHERE d.element_type = 'code_dependency'
  AND d.metadata->>'dependency_kind' = 'function_call'
  AND d.metadata->>'target_function' = 'ValidateInput';
```

**Build call graph:**
```cypher
MATCH (caller:code_function)-[call:code_dependency {dependency_kind: "function_call"}]->(callee:code_function)
RETURN caller.function_name, callee.function_name
```

**Find type dependencies:**
```sql
SELECT DISTINCT
    f.function_name,
    d.metadata->>'target_type' as type_used
FROM elements f
JOIN elements d ON d.parent_id = f.element_id
WHERE f.element_type = 'code_function'
  AND d.element_type = 'code_dependency'
  AND d.metadata->>'dependency_kind' = 'type_usage'
ORDER BY f.function_name;
```

## Inline Comment Extraction

The parser extracts **inline comments** within function/method bodies as separate `code_documentation` elements with `doc_kind: "inline_comment"`. This enables querying for technical debt markers, code quality analysis, and documentation completeness checks.

### Supported Comment Types

**Go:**
- Line comments: `// comment`
- Block comments: `/* comment */`

**Python:**
- Line comments: `# comment`

**Java:**
- Line comments: `// comment`
- Block comments: `/* comment */`
- **Note:** Javadoc (`/** ... */`) is filtered out to avoid duplication with top-level documentation

**JavaScript/TypeScript:**
- Line comments: `// comment`
- Block comments: `/* comment */`
- **Note:** JSDoc (`/** ... */`) is filtered out to avoid duplication with top-level documentation

**C/C++:**
- Line comments: `// comment`
- Block comments: `/* comment */`

**Rust:**
- Line comments: `// comment`
- Block comments: `/* comment */`
- Doc comments: `/// comment` and `//! comment`

**Ruby:**
- Line comments: `# comment`

### Technical Debt Markers

The parser automatically detects these markers in inline comments:
- `TODO` - Planned work or unfinished features
- `FIXME` - Known bugs or issues that need fixing
- `HACK` - Quick fixes or temporary workarounds
- `XXX` - Warning or important note
- `NOTE` - General note or explanation
- `BUG` - Known bug
- `OPTIMIZE` - Performance improvement opportunity
- `TEMP` - Temporary code

### Metadata Structure

```go
{
    "doc_kind":      "inline_comment",
    "comment_type":  "line_comment",  // or "block_comment"
    "language":      "go",             // or "python", "java"
    "line_number":   42,
    "markers":       ["TODO", "FIXME"] // Detected markers (if any)
}
```

### Use Cases

#### 1. Technical Debt Tracking

```sql
-- Find all TODO/FIXME comments
SELECT
    f.function_name,
    f.namespace,
    c.content,
    c.line_number,
    c.metadata->>'markers' as markers
FROM elements c
JOIN elements f ON f.element_id = c.parent_id
WHERE c.element_type = 'code_documentation'
  AND c.metadata->>'doc_kind' = 'inline_comment'
  AND c.metadata->'markers' ?| ARRAY['TODO', 'FIXME', 'HACK']
ORDER BY f.namespace, c.line_number;
```

#### 2. Code Quality Metrics

```sql
-- Calculate comment density per function
SELECT
    f.function_name,
    f.namespace,
    COUNT(*) as inline_comment_count,
    COUNT(*) FILTER (WHERE c.metadata->'markers' ?| ARRAY['TODO', 'FIXME']) as tech_debt_count
FROM elements f
LEFT JOIN elements c ON c.parent_id = f.element_id
    AND c.element_type = 'code_documentation'
    AND c.metadata->>'doc_kind' = 'inline_comment'
WHERE f.element_type = 'code_function'
GROUP BY f.function_name, f.namespace
ORDER BY tech_debt_count DESC, inline_comment_count DESC;
```

#### 3. Find HACK/FIXME Code Smells

```sql
-- Find functions with HACK or FIXME markers (code smells)
SELECT DISTINCT
    f.function_name,
    f.namespace,
    c.content,
    c.metadata->>'markers' as markers
FROM elements c
JOIN elements f ON f.element_id = c.parent_id
WHERE c.element_type = 'code_documentation'
  AND c.metadata->>'doc_kind' = 'inline_comment'
  AND c.metadata->'markers' ?| ARRAY['HACK', 'FIXME']
ORDER BY f.namespace, f.function_name;
```

#### 4. Documentation Completeness

```sql
-- Find functions with many inline comments but no docstrings/Javadoc
SELECT
    f.function_name,
    f.namespace,
    COUNT(DISTINCT c.element_id) as inline_comments,
    COUNT(DISTINCT d.element_id) as docstrings
FROM elements f
LEFT JOIN elements c ON c.parent_id = f.element_id
    AND c.element_type = 'code_documentation'
    AND c.metadata->>'doc_kind' = 'inline_comment'
LEFT JOIN elements d ON d.parent_id = f.element_id
    AND c.element_type = 'code_documentation'
    AND c.metadata->>'doc_kind' IN ('docstring', 'javadoc', 'doc_comment')
WHERE f.element_type = 'code_function'
GROUP BY f.function_name, f.namespace
HAVING COUNT(DISTINCT c.element_id) > 5 AND COUNT(DISTINCT d.element_id) = 0
ORDER BY inline_comments DESC;
```

### Example Code

**Go:**
```go
func ProcessData() {
    // TODO: Add validation logic here
    x := 1

    /* FIXME: This is a temporary workaround
       Need to refactor this section */
    y := 2

    // HACK: Quick fix for deadline
    z := x + y

    return z
}
```

**Python:**
```python
def process_data():
    # TODO: Add validation logic here
    x = 1

    # FIXME: This is a temporary workaround
    # Need to refactor this section
    y = 2

    # HACK: Quick fix for deadline
    z = x + y

    return z
```

**Java:**
```java
public int processData() {
    // TODO: Add validation logic here
    int x = 1;

    /* FIXME: This is a temporary workaround
       Need to refactor this section */
    int y = 2;

    // HACK: Quick fix for deadline
    int z = x + y;

    return z;
}
```

**JavaScript/TypeScript:**
```javascript
function processData() {
    // TODO: Add validation logic here
    let x = 1;

    /* FIXME: This is a temporary workaround
       Need to refactor this section */
    let y = 2;

    // HACK: Quick fix for deadline
    let z = x + y;

    // NOTE: This calculation is simplified
    return z;
}
```

**C/C++:**
```cpp
int process_data() {
    // TODO: Add validation logic here
    int x = 1;

    /* FIXME: This is a temporary workaround
       Need to refactor this section */
    int y = 2;

    // HACK: Quick fix for deadline
    int z = x + y;

    // NOTE: This calculation is simplified
    return z;
}
```

**Rust:**
```rust
fn process_data() -> i32 {
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
```

**Ruby:**
```ruby
def process_data
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
```

### Storage Overhead

Inline comment extraction adds approximately **50% storage** per function with comments:
- Average function: ~10 inline comment elements
- Storage per element: ~500 bytes (including metadata)
- Overhead: ~5KB per function with inline comments
- **For 100K LOC codebase**: ~7.5MB additional storage

This minimal overhead enables powerful technical debt tracking and code quality analysis.

## Promoted Fields

Code elements have **5 new promoted fields** for fast queries:

```go
type Element struct {
    // ... existing fields ...

    // Code-specific promoted fields (UDML Phase 2)
    FunctionName  *string  // Function/method/procedure name
    ClassName     *string  // Class/struct/interface name
    Namespace     *string  // Package/module path
    LineNumber    *int     // Source line number
    Signature     *string  // Complete signature
}
```

### Why Promoted Fields?

Promoted fields enable **60-1000x faster queries** by avoiding JSON parsing:

```sql
-- FAST: Uses promoted field index
SELECT * FROM elements
WHERE function_name = 'Calculate'
  AND line_number BETWEEN 10 AND 50;

-- SLOW: Requires JSON parsing
SELECT * FROM elements
WHERE metadata->>'function_name' = 'Calculate'
  AND CAST(metadata->>'line_number' AS INT) BETWEEN 10 AND 50;
```

## Universal Metadata

All code elements have **universal metadata** plus **language-specific metadata**:

```go
Metadata: map[string]interface{}{
    // Universal metadata (works across all languages)
    "code_element_kind": "function",      // function | method | class | etc.
    "visibility":        "public",        // public | private | protected | etc.
    "language":          "go",            // go | java | python | etc.
    "line_number":       42,

    // Language-specific metadata (Go example)
    "is_exported":       true,            // Go-specific
    "receiver_type":     "*MyStruct",     // Go-specific
    "package":           "mypackage",     // Go-specific
}
```

## Supported Languages

| Language | Status | Parser | Element Types |
|----------|--------|--------|---------------|
| **Go** | ✅ Complete | `go/ast` | functions, methods, structs, interfaces, imports, fields, constants, vars, doc comments |
| **Python** | ✅ Complete | `tree-sitter` | functions, methods, classes, decorators, docstrings, imports, type hints, properties |
| **Java** | ✅ Complete | `tree-sitter` | classes, interfaces, enums, methods, constructors, fields, annotations, Javadoc, imports |
| **JavaScript** | ✅ Complete | `tree-sitter` | functions, arrow functions, classes, methods, ES6 imports/exports, JSDoc, JSX |
| **TypeScript** | ✅ Complete | `tree-sitter` | all JavaScript features + interfaces, type aliases, enums, type annotations, TSX |
| **C/C++** | ✅ Complete | `tree-sitter` | functions, methods, classes, structs, unions, enums, namespaces, templates, #include directives |
| **Rust** | ✅ Complete | `tree-sitter` | functions, methods, structs, enums, traits, modules, impl blocks, use declarations, macro invocations |
| **Ruby** | ✅ Complete | `tree-sitter` | classes, modules, instance methods, class methods, attributes, constants, require/require_relative, inline comments |

## Quick Start

### 1. Parse Go Code

```go
import "github.com/kennethstott/doculyzer-go-conversion/internal/parser"

// Create parser
goParser := parser.NewGoCodeParser()

// Parse file
result, err := goParser.Parse(context.Background(), parser.ParseRequest{
    ID:      "my-go-file",
    Content: "/path/to/file.go",
    Config:  parser.DefaultParserConfig(),
})

// Access elements
for _, elem := range result.Elements {
    if elem.ElementType == "code_function" {
        fmt.Printf("Found function: %s at line %d\n",
            *elem.FunctionName, *elem.LineNumber)
    }
}
```

### 2. Parse Python Code

```go
import "github.com/kennethstott/doculyzer-go-conversion/internal/parser"

// Create parser
pyParser := parser.NewPythonCodeParser()

// Parse file
result, err := pyParser.Parse(context.Background(), parser.ParseRequest{
    ID:      "my-python-file",
    Content: "/path/to/file.py",
    Config:  parser.DefaultParserConfig(),
})

// Access elements
for _, elem := range result.Elements {
    if elem.ElementType == "code_function" {
        fmt.Printf("Found function: %s at line %d\n",
            *elem.FunctionName, *elem.LineNumber)

        // Check for decorators
        if decorators, ok := elem.Metadata["decorators"].([]string); ok {
            fmt.Printf("  Decorators: %v\n", decorators)
        }
    }
}
```

### 3. Parse Java Code

```go
import "github.com/kennethstott/doculyzer-go-conversion/internal/parser"

// Create parser
javaParser := parser.NewJavaCodeParser()

// Parse file
result, err := javaParser.Parse(context.Background(), parser.ParseRequest{
    ID:      "my-java-file",
    Content: "/path/to/MyClass.java",
    Config:  parser.DefaultParserConfig(),
})

// Access elements
for _, elem := range result.Elements {
    if elem.ElementType == "code_function" {
        fmt.Printf("Found method: %s at line %d\n",
            *elem.FunctionName, *elem.LineNumber)

        // Check for annotations
        if annotations, ok := elem.Metadata["annotations"].([]string); ok {
            fmt.Printf("  Annotations: %v\n", annotations)
        }
    }
}
```

### 4. Parse JavaScript/TypeScript Code

```go
import "github.com/kennethstott/doculyzer-go-conversion/internal/parser"

// Create parser (works for both .js and .ts files)
jsParser := parser.NewJavaScriptTypeScriptCodeParser()

// Parse JavaScript file
result, err := jsParser.Parse(context.Background(), parser.ParseRequest{
    ID:      "my-js-file",
    Content: "/path/to/app.js",
    Config:  parser.DefaultParserConfig(),
})

// Access elements
for _, elem := range result.Elements {
    if elem.ElementType == "code_function" {
        fmt.Printf("Found function: %s at line %d\n",
            *elem.FunctionName, *elem.LineNumber)

        // Check for async
        if isAsync, ok := elem.Metadata["is_async"].(bool); ok && isAsync {
            fmt.Printf("  (async function)\n")
        }

        // Check for arrow function
        if isArrow, ok := elem.Metadata["is_arrow_function"].(bool); ok && isArrow {
            fmt.Printf("  (arrow function)\n")
        }
    }

    if elem.ElementType == "code_type" && elem.Metadata["language"] == "typescript" {
        fmt.Printf("Found TypeScript type: %s\n", *elem.ClassName)
    }
}
```

### 5. Parse C/C++ Code

```go
import "github.com/kennethstott/doculyzer-go-conversion/internal/parser"

// Create parser (works for both C and C++ files)
cppParser := parser.NewCCppCodeParser()

// Parse C++ file
result, err := cppParser.Parse(context.Background(), parser.ParseRequest{
    ID:      "my-cpp-file",
    Content: "/path/to/MyClass.cpp",
    Config:  parser.DefaultParserConfig(),
})

// Access elements
for _, elem := range result.Elements {
    if elem.ElementType == "code_function" {
        fmt.Printf("Found function: %s at line %d\n",
            *elem.FunctionName, *elem.LineNumber)

        // Check if it's a method
        if elem.ClassName != nil {
            fmt.Printf("  Class: %s\n", *elem.ClassName)
        }
    }

    if elem.ElementType == "code_grouping" && elem.ClassName != nil {
        kind := elem.Metadata["code_element_kind"].(string)
        fmt.Printf("Found %s: %s\n", kind, *elem.ClassName)
    }

    if elem.ElementType == "code_dependency" && elem.Metadata["dependency_kind"] == "include" {
        targetFile := elem.Metadata["target_file"].(string)
        fmt.Printf("Includes: %s\n", targetFile)
    }
}
```

### 6. Parse Rust Code

```go
import "github.com/kennethstott/doculyzer-go-conversion/internal/parser"

// Create parser
rustParser := parser.NewRustCodeParser()

// Parse Rust file
result, err := rustParser.Parse(context.Background(), parser.ParseRequest{
    ID:      "my-rust-file",
    Content: "/path/to/main.rs",
    Config:  parser.DefaultParserConfig(),
})

// Access elements
for _, elem := range result.Elements {
    if elem.ElementType == "code_function" {
        fmt.Printf("Found function: %s at line %d\n",
            *elem.FunctionName, *elem.LineNumber)

        // Check if it's a method
        if elem.ClassName != nil {
            fmt.Printf("  impl for: %s\n", *elem.ClassName)
        }

        // Check visibility
        if vis, ok := elem.Metadata["visibility"].(string); ok {
            fmt.Printf("  Visibility: %s\n", vis)
        }
    }

    if elem.ElementType == "code_grouping" && elem.ClassName != nil {
        kind := elem.Metadata["code_element_kind"].(string)
        fmt.Printf("Found %s: %s\n", kind, *elem.ClassName)
    }

    if elem.ElementType == "code_type" && elem.Metadata["code_element_kind"] == "trait" {
        fmt.Printf("Found trait: %s\n", *elem.ClassName)
    }

    if elem.ElementType == "code_dependency" {
        depKind := elem.Metadata["dependency_kind"].(string)
        if depKind == "import" {
            targetNs := elem.Metadata["target_namespace"].(string)
            fmt.Printf("Uses: %s\n", targetNs)
        } else if depKind == "function_call" {
            if isMacro, ok := elem.Metadata["is_macro"].(bool); ok && isMacro {
                fmt.Printf("Macro call: %s\n", elem.Metadata["target_function"])
            }
        }
    }
}
```

### 7. Parse Ruby Code

```go
import "github.com/kennethstott/doculyzer-go-conversion/internal/parser"

// Create parser
rubyParser := parser.NewRubyCodeParser()

// Parse Ruby file
result, err := rubyParser.Parse(context.Background(), parser.ParseRequest{
    ID:      "my-ruby-file",
    Content: "/path/to/app.rb",
    Config:  parser.DefaultParserConfig(),
})

// Access elements
for _, elem := range result.Elements {
    if elem.ElementType == "code_function" {
        fmt.Printf("Found method: %s at line %d\n",
            *elem.FunctionName, *elem.LineNumber)

        // Check if it's a class method (singleton method)
        if isSingleton, ok := elem.Metadata["is_singleton"].(bool); ok && isSingleton {
            fmt.Printf("  (class method)\n")
        }

        // Check if it's inside a class
        if elem.ClassName != nil {
            fmt.Printf("  Class: %s\n", *elem.ClassName)
        }
    }

    if elem.ElementType == "code_grouping" && elem.ClassName != nil {
        kind := elem.Metadata["code_element_kind"].(string)
        if kind == "class" {
            fmt.Printf("Found class: %s\n", *elem.ClassName)
        } else if kind == "module" {
            moduleName := elem.Metadata["module_name"].(string)
            fmt.Printf("Found module: %s\n", moduleName)
        }
    }

    if elem.ElementType == "code_data" && elem.Metadata["code_element_kind"] == "attribute" {
        attrName := elem.Metadata["attribute_name"].(string)
        attrType := elem.Metadata["attribute_type"].(string)
        fmt.Printf("Found attribute: %s (%s)\n", attrName, attrType)
    }

    if elem.ElementType == "code_dependency" {
        depKind := elem.Metadata["dependency_kind"].(string)
        if depKind == "import" {
            importPath := elem.Metadata["target_namespace"].(string)
            importType := elem.Metadata["import_type"].(string)
            fmt.Printf("%s '%s'\n", importType, importPath)
        }
    }
}
```

### 8. Register with Parser Registry

```go
registry := parser.NewParserRegistry()
registry.Register(parser.NewGoCodeParser())
registry.Register(parser.NewPythonCodeParser())
registry.Register(parser.NewJavaCodeParser())
registry.Register(parser.NewJavaScriptTypeScriptCodeParser())
registry.Register(parser.NewCCppCodeParser())
registry.Register(parser.NewRustCodeParser())
registry.Register(parser.NewRubyCodeParser())

// Auto-detect parser by file extension
goParser, _ := registry.GetParserForFile("main.go")
pyParser, _ := registry.GetParserForFile("app.py")
javaParser, _ := registry.GetParserForFile("MyClass.java")
jsParser, _ := registry.GetParserForFile("app.js")
tsParser, _ := registry.GetParserForFile("index.ts")
jsxParser, _ := registry.GetParserForFile("Component.jsx")
tsxParser, _ := registry.GetParserForFile("Component.tsx")
cParser, _ := registry.GetParserForFile("main.c")
cppParser, _ := registry.GetParserForFile("MyClass.cpp")
rustParser, _ := registry.GetParserForFile("main.rs")
rubyParser, _ := registry.GetParserForFile("app.rb")
```

### 8. Query with SQL

```sql
-- Find all exported functions
SELECT
    function_name,
    namespace,
    signature,
    line_number
FROM elements
WHERE element_type = 'code_function'
  AND metadata->>'visibility' = 'public'
ORDER BY namespace, function_name;
```

### 8. Build Knowledge Graphs

```cypher
// Create relationships between code and docs
MATCH (func:code_function)
MATCH (req:requirement)
WHERE req.description CONTAINS func.function_name
CREATE (func)-[:implements]->(req)
```

## Element Taxonomy

The universal code elements are integrated into the existing taxonomy:

```json
{
  "categories": {
    "container": ["code_grouping", "code_type", ...],
    "content": ["code_function", "code_data", ...],
    "component": ["code_dependency", ...],
    "metadata": ["code_documentation", ...]
  },
  "code_elements": {
    "code_function": {
      "maps_to": {
        "go": ["function", "method"],
        "java": ["method", "constructor"],
        "python": ["function", "method", "lambda"]
      }
    }
  }
}
```

## Architecture

```
┌─────────────────────────────────────────┐
│   Source Code Files                     │
│   (.go, .py, .java, .js, .ts, etc.)   │
└──────────────┬──────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────┐
│   Language-Specific Parsers             │
│   • Go: go/ast                          │
│   • Python: Tree-sitter                 │
│   • Java: JavaParser                    │
└──────────────┬──────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────┐
│   Universal Code Model (UCM)            │
│   • code_function                       │
│   • code_grouping                       │
│   • code_data                           │
│   • code_dependency                     │
│   • code_documentation                  │
└──────────────┬──────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────┐
│   UDML Elements with Promoted Fields    │
│   • FunctionName, ClassName             │
│   • Namespace, LineNumber, Signature    │
│   • Universal + Language-specific meta  │
└──────────────┬──────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────┐
│   Storage & Analysis                    │
│   • PostgreSQL (relational queries)     │
│   • Neo4j (graph traversal)             │
│   • Parquet (analytics)                 │
│   • Elasticsearch (full-text search)    │
└─────────────────────────────────────────┘
```

## Testing

Run the code parser tests:

```bash
cd go

# Test Go parser
go test ./internal/parser -run TestGoCodeParser -v

# Test Python parser
go test ./internal/parser -run TestPythonCodeParser -v

# Test Java parser
go test ./internal/parser -run TestJavaCodeParser -v

# Test JavaScript/TypeScript parser
go test ./internal/parser -run "TestJavaScript|TestTypeScript" -v

# Test C/C++ parser
go test ./internal/parser -run "TestC" -v

# Test Rust parser
go test ./internal/parser -run TestRustParser -v

# Test Ruby parser
go test ./internal/parser -run TestRubyParser -v

# Test all code parsers
go test ./internal/parser -run "Test(Go|Python|Java|JavaScript|TypeScript|C|Rust|Ruby)" -v
```

Example output:
```
=== RUN   TestGoCodeParser_BasicParsing
--- PASS: TestGoCodeParser_BasicParsing (0.01s)
=== RUN   TestGoCodeParser_EntityReferences
--- PASS: TestGoCodeParser_EntityReferences (0.00s)
=== RUN   TestPythonCodeParser_BasicParsing
--- PASS: TestPythonCodeParser_BasicParsing (0.00s)
=== RUN   TestPythonCodeParser_ClassMethods
--- PASS: TestPythonCodeParser_ClassMethods (0.02s)
=== RUN   TestJavaCodeParser_BasicParsing
--- PASS: TestJavaCodeParser_BasicParsing (0.01s)
=== RUN   TestJavaCodeParser_InterfaceAndEnum
--- PASS: TestJavaCodeParser_InterfaceAndEnum (0.00s)
=== RUN   TestJavaCodeParser_EntityReferences
--- PASS: TestJavaCodeParser_EntityReferences (0.00s)
=== RUN   TestJavaScriptParser_BasicParsing
--- PASS: TestJavaScriptParser_BasicParsing (0.01s)
=== RUN   TestTypeScriptParser_BasicParsing
--- PASS: TestTypeScriptParser_BasicParsing (0.01s)
=== RUN   TestJavaScriptParser_AsyncAwait
--- PASS: TestJavaScriptParser_AsyncAwait (0.00s)
=== RUN   TestRustParser_BasicParsing
--- PASS: TestRustParser_BasicParsing (0.01s)
=== RUN   TestRustParser_EnumAndTrait
--- PASS: TestRustParser_EnumAndTrait (0.00s)
=== RUN   TestRustParser_EntityReferences
--- PASS: TestRustParser_EntityReferences (0.01s)
=== RUN   TestRubyParser_BasicParsing
--- PASS: TestRubyParser_BasicParsing (0.00s)
=== RUN   TestRubyParser_ModuleAndMixins
--- PASS: TestRubyParser_ModuleAndMixins (0.00s)
=== RUN   TestRubyParser_InlineComments
--- PASS: TestRubyParser_InlineComments (0.01s)
PASS
```

## Documentation

- **[Universal Code Model](./universal-code-model.md)** - Complete UCM specification
- **[Element Taxonomy](../../../element_taxonomy.json)** - Full taxonomy with code elements
- **[UDML Specification](../udml/specification.md)** - Universal Document Model spec

## Benefits

### For Developers
- **Cross-language analysis**: Same queries work for Go, Python, Java, etc.
- **Fast queries**: Promoted fields avoid JSON parsing overhead
- **Rich metadata**: Both universal and language-specific details

### For Organizations
- **Policy enforcement**: Automated compliance checking at scale
- **Requirements traceability**: Direct links between requirements and code
- **Documentation quality**: Detect drift between code and docs
- **Impact analysis**: Understand change ripple effects
- **Knowledge graphs**: Query relationships across code, policies, requirements, docs

### For Tools
- **Extensible**: Add new languages by implementing Parser interface
- **Standard format**: All parsers output Universal Document Model
- **Interoperable**: Works with existing UDML infrastructure

## Next Steps

1. **Add more languages**: PHP, Kotlin, Swift, Scala
2. **Create code ontologies**: Security patterns, architectural patterns, anti-patterns
3. **Build analysis tools**: Policy checkers, requirement tracers, doc validators
4. **Integrate with CI/CD**: Automated checks on every commit
5. **Enhanced type resolution**: Track type definitions across modules for better dependency graphs

## Related Documentation

- [Universal Code Model](./universal-code-model.md)
- [Ontology System](../ontology/README.md)
- [Configuration](../../configuration/README.md)
- [Quick Reference](../../../QUICK_REFERENCE.md)
