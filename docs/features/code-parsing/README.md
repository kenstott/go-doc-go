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

✅ **Entity References Supported**:
- Package imports (`dependency_kind: "import"`)
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
| **Python** | 🔄 Planned | Tree-sitter | functions, classes, decorators, docstrings, imports |
| **Java** | 🔄 Planned | JavaParser | classes, methods, annotations, Javadoc |
| **JavaScript/TypeScript** | 🔄 Planned | Tree-sitter | functions, classes, JSDoc, imports |
| **C/C++** | 🔄 Planned | Tree-sitter | functions, classes, structs, macros |

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

### 2. Register with Parser Registry

```go
registry := parser.NewParserRegistry()
registry.Register(parser.NewGoCodeParser())

// Auto-detect parser by file extension
parser, _ := registry.GetParserForFile("main.go")
```

### 3. Query with SQL

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

### 4. Build Knowledge Graphs

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

Run the Go parser tests:

```bash
cd go
go test ./internal/parser -run TestGoCodeParser -v
```

Example output:
```
=== RUN   TestGoCodeParser_BasicParsing
--- PASS: TestGoCodeParser_BasicParsing (0.01s)
=== RUN   TestGoCodeParser_PromotedFields
--- PASS: TestGoCodeParser_PromotedFields (0.00s)
=== RUN   TestGoCodeParser_ElementHierarchy
--- PASS: TestGoCodeParser_ElementHierarchy (0.00s)
=== RUN   TestGoCodeParser_InterfaceRegistration
--- PASS: TestGoCodeParser_InterfaceRegistration (0.00s)
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

1. **Add more languages**: Python, Java, JavaScript/TypeScript
2. **Create code ontologies**: Security patterns, architectural patterns, anti-patterns
3. **Build analysis tools**: Policy checkers, requirement tracers, doc validators
4. **Integrate with CI/CD**: Automated checks on every commit

## Related Documentation

- [Universal Code Model](./universal-code-model.md)
- [Ontology System](../ontology/README.md)
- [Configuration](../../configuration/README.md)
- [Quick Reference](../../../QUICK_REFERENCE.md)
