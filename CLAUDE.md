# Go-Doc-Go Project Guidelines

## Project Overview
Go-Doc-Go is a comprehensive document parsing and analysis system written in **Go** that extracts structured information from various document formats (PDF, DOCX, XLSX, JSON, CSV, HTML, Markdown, etc.) and stores it in a queryable format with relationship tracking.

**Language:** Go

---

## ⚠️ MANDATORY VERIFICATION CHECKLIST ⚠️

**READ THIS BEFORE ANY ACTION** - These checks prevent the most common errors.

### 🔴 CREATING Files or Directories

**STOP. Verify location rules:**

| File Type | MUST go in | NEVER in |
|-----------|-----------|----------|
| Compiled binaries | `<project-root>/bin/` | `go/bin/`, `cmd/*/`, anywhere else |
| Test configs | `<project-root>/tests/test_configs/` | `/tmp`, `go/tests/`, `go/` |
| Test outputs | `<project-root>/tests/test_output/` | `/tmp`, `go/tests/`, `go/` |
| Source code (.go files) | `<project-root>/go/` | Anywhere is fine if in `go/` |
| Anything else | **NOT in `go/`** | `go/` is CODE ONLY |

**Verification command:**
```bash
pwd  # Where am I running from?
# If in go/ directory and creating files outside go/, use ../ prefix
# Example: ../bin/goworker, ../tests/test_output/results.json
```

### 🔴 REPORTING File Locations

**STOP. Verify before claiming:**

```bash
# NEVER report assumed locations - ALWAYS verify
ls -la <path> || find . -name "<filename>" | head -5
```

**Rule:** If you haven't run `ls` or `find`, don't claim a file exists or is in a location.

### 🔴 MODIFYING Parser/Component Code

**STOP. Check downstream integration:**

```bash
# Who consumes this component's output?
grep -rn "ElementType\|ComponentName" internal/worker/ internal/*/

# Do the contracts match?
# Example: If parser outputs "hyperlink", worker must check for "hyperlink" (not "link")
```

**Rule:** When changing data formats, types, or field names, verify ALL consumers expect the new format.

### 🔴 CLEANUP Operations

**STOP. Search systematically:**

```bash
# Don't assume - search for ALL matching files
find <dir> -type f -name "*test*" -o -name "*.log" -o -name "*.db" | head -20
find <dir> -type d -name "*test*"
```

**Rule:** Use `find` to discover all targets. List what will be removed. Then remove. Then verify with `find` again.

### 🔴 RELATIVE Path Configurations

**STOP. Consider working directory:**

```bash
pwd  # What directory do programs run from?
# If config has path = "./tests/output" but program runs from go/
# Then path resolves to go/tests/output (WRONG)
# Use path = "../tests/output" instead
```

**Rule:** When setting paths in config files, consider where the binary will run from.

---

## Go Development Standards

### Code Organization
1. **Single Responsibility Principle**: Each package/function should have one clear purpose
2. **DRY (Don't Repeat Yourself)**: Extract common functionality into reusable packages
3. **Explicit is better than implicit**: Use clear, descriptive names
4. **Composition over inheritance**: Prefer interfaces and composition over embedding
5. **No default values without explicit design**: Required configuration must be explicit and fail fast if missing
6. **No fallback code paths without explicit approval**: Avoid silent fallbacks that hide errors
7. **Clean design over backward compatibility**: No technical debt accumulation (see Design Integrity Principle)

### Naming Conventions
- **Packages**: lowercase, single word (e.g., `parser`, `config`, `storage`)
- **Types/Structs**: PascalCase (e.g., `DocumentParser`, `PdfParser`)
- **Functions/Methods**: PascalCase for exported, camelCase for unexported (e.g., `ParseDocument`, `extractText`)
- **Constants**: PascalCase for exported, camelCase for unexported (e.g., `MaxFileSize`, `defaultTimeout`)
- **Interfaces**: PascalCase, often ending in -er (e.g., `Parser`, `Reader`, `Handler`)

### Project Structure
```bash
go-doc-go/
├── cmd/                    # Command-line applications
│   ├── worker/            # Document processing worker
│   └── ontology/          # Ontology extraction CLI
├── go/                    # Go source code root
│   ├── internal/          # Private application code
│   │   ├── parser/       # Document parsers
│   │   ├── storage/      # Database/storage layer
│   │   ├── config/       # Configuration management
│   │   ├── jobcontrol/   # Work queue system
│   │   └── udml/         # UDML and ontology extraction
│   └── pkg/              # Public libraries (if any)
├── bin/                   # Compiled binaries (gitignored)
├── tests/                 # Test files and fixtures
├── assets/                # Static assets (models, etc.)
├── examples/              # Example configurations and ontologies
└── docs/                  # Documentation
```go

### Binary Output Location
All compiled Go binaries MUST be output to `<project-dir>/bin/` directory.

**Requirements:**
- Use the `-o` flag to specify output location when building Go binaries
- Keep all project binaries centralized in the `bin/` directory
- This ensures binaries are properly gitignored and organized

**Examples:**
```bash
## Building the worker binary
go build -o bin/goworker ./cmd/worker

## Building the ontology CLI
go build -o bin/ontology_interview ./cmd/ontology

## Building any command
go build -o bin/<binary-name> ./cmd/<command>/
```bash

**Benefits:**
- Centralized location for all binaries (easier to find and manage)
- Consistent with standard Go project layout conventions
- Properly gitignored (bin/ directory should be in .gitignore)
- Prevents binaries from being scattered across cmd/ directories
- Simplifies cleanup (rm -rf bin/)

### Error Handling
- Return errors explicitly; use Go's idiomatic error handling
- Create custom error types for domain-specific errors
- Wrap errors with context using `fmt.Errorf` with `%w`
- Log errors with appropriate context using structured logging

```go
package parser

import (
    "fmt"
    "errors"
)

var (
    ErrInvalidDocument = errors.New("invalid document format")
    ErrParserNotFound  = errors.New("parser not found for document type")
)

type ParseError struct {
    DocID string
    Err   error
}

func (e *ParseError) Error() string {
    return fmt.Sprintf("failed to parse document %s: %v", e.DocID, e.Err)
}

func (e *ParseError) Unwrap() error {
    return e.Err
}

func ParseDocument(docID string, content []byte) error {
    if len(content) == 0 {
        return &ParseError{
            DocID: docID,
            Err:   ErrInvalidDocument,
        }
    }
    // Parse logic...
    return nil
}
```

### Interface Design
- Keep interfaces small and focused (often 1-3 methods)
- Define interfaces at the point of use, not implementation
- Use standard library interfaces where possible (io.Reader, io.Writer, etc.)

```go
// Good: Small, focused interface
type Parser interface {
    Parse(content []byte) (*ParseResult, error)
}

// Good: Using standard library interfaces
type DocumentReader interface {
    io.Reader
    io.Closer
}
```toml

### Configuration Management
- Use struct tags for configuration parsing (YAML, TOML, JSON)
- Validate configuration at startup, fail fast
- No silent defaults - require explicit configuration

```go
type Config struct {
    Database DatabaseConfig `yaml:"database" toml:"database"`
    Parser   ParserConfig   `yaml:"parser" toml:"parser"`
}

type ParserConfig struct {
    MaxContentPreview int  `yaml:"max_content_preview" toml:"max_content_preview"`
    ExtractMetadata   bool `yaml:"extract_metadata" toml:"extract_metadata"`
    MaxDepth          int  `yaml:"max_depth" toml:"max_depth"`
}

func (c *Config) Validate() error {
    if c.Parser.MaxDepth == 0 {
        return fmt.Errorf("parser.max_depth is required")
    }
    // More validation...
    return nil
}
```

## Testing Guidelines

### Test Organization
```go
tests/
├── unit/              # Unit tests - isolated component tests (Go)
├── integration/       # Integration tests - component interactions
├── fixtures/          # Static test data (version controlled)
├── test_configs/      # Test configuration files (version controlled)
├── test_assets/       # Generated test assets (gitignored, disposable)
└── test_output/       # Test results and output (gitignored, disposable)
```

### Test File Location - CRITICAL
**NEVER create test files in `/tmp` or `/private/tmp`** - these directories are cleared on system restart.

**Requirements:**
- All test files MUST be stored within `./tests/` directory structure
- Test files in the project directory survive restarts
- Organize by purpose to enable selective cleanup by humans

**Directory Purposes:**

1. **`./tests/test_configs/`** (Version Controlled)
   - Test configuration files (YAML, TOML, JSON)
   - These define test scenarios and parameters
   - Should be committed to version control
   - Example: `wikipedia_medical_mining.yaml`, `ontology_extraction_test.toml`

2. **`./tests/fixtures/`** (Version Controlled)
   - Static, curated test data used by tests
   - Small sample documents for consistent test behavior
   - Expected output files for validation
   - Should be committed to version control
   - Example: `sample.pdf`, `expected_parsed_output.json`

3. **`./tests/test_assets/`** (Gitignored, Disposable)
   - Large files downloaded/generated during testing
   - Wikipedia pages, external documents, generated PDFs
   - Should be in `.gitignore` - NOT version controlled
   - Can be safely deleted by humans to reclaim disk space
   - Tests should regenerate these as needed
   - Example: `wikipedia_medicine_downloaded.html`, `generated_large_document.pdf`

4. **`./tests/test_output/`** (Gitignored, Disposable)
   - Test execution results and artifacts
   - Parsed output, analysis results, logs
   - Should be in `.gitignore` - NOT version controlled
   - Can be safely deleted by humans to reclaim disk space
   - Tests regenerate on each run
   - Example: `parsed_results.json`, `test_run_2024_01_15.log`

**Example Structure:**
```toml
tests/
├── test_configs/                    # VERSION CONTROLLED
│   ├── wikipedia_medical_mining.yaml
│   ├── pdf_parsing_test.toml
│   └── ontology_extraction_test.yaml
├── fixtures/                        # VERSION CONTROLLED
│   ├── sample_documents/
│   │   ├── tiny_sample.pdf         # Small files only
│   │   └── minimal_test.docx       # < 100KB each
│   └── expected_outputs/
│       └── expected_parse.json
├── test_assets/                     # GITIGNORED - Safe to delete
│   ├── wikipedia/
│   │   ├── Medicine_page.html      # Downloaded content
│   │   └── Mining_page.html
│   ├── generated/
│   │   └── large_test_doc.pdf      # Generated during tests
│   └── downloads/
│       └── external_samples/       # Downloaded test files
└── test_output/                     # GITIGNORED - Safe to delete
    ├── parsed_results/
    │   ├── test_run_001.json
    │   └── test_run_002.json
    ├── logs/
    │   └── test_execution.log
    └── duckdb/
        └── test_database.db        # Test database files
```

**Gitignore Configuration:**
```gitignore
## Disposable test files - safe to delete
tests/test_assets/
tests/test_output/

## Keep the directories themselves but ignore contents
!tests/test_assets/.gitkeep
!tests/test_output/.gitkeep
```

**Benefits:**
- Test files persist across system restarts
- Version control tracks only essential test files
- Clear separation of disposable vs. permanent test data
- Humans can easily identify and delete large/temporary files
- Reproducible test environments
- Disk space management without breaking tests

### Go Testing Best Practices

**Test File Structure:**
```go
package parser_test

import (
    "testing"
    "github.com/yourusername/go-doc-go/go/internal/parser"
)

func TestPdfParser_Parse(t *testing.T) {
    tests := []struct {
        name    string
        input   []byte
        want    *parser.ParseResult
        wantErr bool
    }{
        {
            name:    "valid PDF",
            input:   loadFixture(t, "tests/fixtures/sample.pdf"),
            want:    &parser.ParseResult{/* ... */},
            wantErr: false,
        },
        {
            name:    "empty content",
            input:   []byte{},
            want:    nil,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            p := parser.NewPdfParser()
            got, err := p.Parse(tt.input)

            if (err != nil) != tt.wantErr {
                t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("Parse() = %v, want %v", got, tt.want)
            }
        })
    }
}
```go

**Test Helpers:**
```go
func loadFixture(t *testing.T, path string) []byte {
    t.Helper()
    data, err := os.ReadFile(path)
    if err != nil {
        t.Fatalf("failed to load fixture %s: %v", path, err)
    }
    return data
}

func assertParseResult(t *testing.T, got *parser.ParseResult) {
    t.Helper()
    if got == nil {
        t.Fatal("expected non-nil ParseResult")
    }
    if len(got.Elements) == 0 {
        t.Error("expected non-empty Elements")
    }
}
```

## Development Workflow

### Pre-Commit Verification Checklist - MANDATORY
Before any git commit, ALL of the following MUST pass:

```bash
#!/bin/bash
## pre-commit-checklist.sh - run before any git commit

echo "Pre-Commit Checklist - MANDATORY"
echo "=================================="

## 1. All Go code builds without errors
go build ./... || { echo "✗ Build errors found"; exit 1; }
echo "✓ Code builds without errors"

## 2. All tests pass
go test ./... || { echo "✗ Tests failed"; exit 1; }
echo "✓ All tests pass"

## 3. No debugging artifacts left in code
if git diff --cached --name-only | grep '\.go$' | xargs grep -n "fmt.Println\|log.Println" | grep -v "// OK:"; then
    echo "✗ Debug print statements found - remove before commit"
    exit 1
fi
echo "✓ No debugging artifacts"

## 4. Code formatting
if [ -n "$(gofmt -l .)" ]; then
    echo "✗ Code needs formatting. Run: gofmt -w ."
    exit 1
fi
echo "✓ Code formatting verified"

## 5. Lint checks
if command -v golangci-lint &> /dev/null; then
    golangci-lint run ./... || { echo "✗ Linting errors found"; exit 1; }
    echo "✓ Linting passed"
else
    echo "⚠ golangci-lint not installed - skipping lint checks"
fi

## 6. Go vet
go vet ./... || { echo "✗ Go vet found issues"; exit 1; }
echo "✓ Go vet passed"

echo "All checks passed - ready to commit"
```bash

### Running Tests
```bash
## Run all tests
go test ./...

## Run tests with coverage
go test -cover ./...

## Run tests with verbose output
go test -v ./...

## Run specific test
go test -run TestPdfParser_Parse ./go/internal/parser

## Run tests with race detection
go test -race ./...

## Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```go

### Coverage Goals
- **Overall**: Minimum 70% coverage
- **Critical parsers** (PDF, DOCX, XLSX): Minimum 80% coverage
- **Core packages**: Minimum 80% coverage
- **New code**: Must include tests before merging

### Performance Benchmarks and SLAs

#### Document Processing SLAs
- Standard document (< 10MB): Parse in < 1 second
- Large document (< 100MB): Parse in < 10 seconds
- Memory usage: < 5x document size
- Concurrent parsing: Support 10 simultaneous parsers

#### Work Queue System SLAs
- **Document claiming latency**: < 10ms per document
- **Sustained throughput**: > 1000 docs/second with 10 concurrent workers
- **Memory usage per worker**: < 100MB base memory
- **Maximum concurrent workers**: 50 workers supported
- **Claim timeout**: 5 minutes (300 seconds)
- **Heartbeat interval**: 30 seconds

## Debugging and Troubleshooting

### Logging Best Practices
Use structured logging with appropriate log levels:

```go
package main

import (
    "log/slog"
    "os"
)

func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    }))

    logger.Debug("Starting parse",
        "doc_id", docID,
        "doc_type", docType,
    )

    if err := parseDocument(docID); err != nil {
        logger.Error("Failed to parse document",
            "doc_id", docID,
            "error", err,
        )
        return
    }

    logger.Info("Successfully parsed document",
        "doc_id", docID,
        "element_count", elementCount,
    )
}
```go

### Debug Helpers
```go
// DumpElementHierarchy prints the element tree for debugging
func DumpElementHierarchy(elements []Element, maxDepth int) {
    roots := findRootElements(elements)
    for _, root := range roots {
        dumpElement(elements, root, 0, maxDepth)
    }
}

func dumpElement(allElements []Element, elem Element, depth, maxDepth int) {
    if depth > maxDepth {
        return
    }
    indent := strings.Repeat("  ", depth)
    fmt.Printf("%s%s: %s [%s...]\n",
        indent,
        elem.Type,
        elem.ID[:8],
        elem.ContentPreview[:30],
    )

    children := findChildElements(allElements, elem.ID)
    for _, child := range children {
        dumpElement(allElements, child, depth+1, maxDepth)
    }
}
```

## Key Design Decisions

1. **Element Types**: All parsers MUST use values from the `ElementType` enum
2. **Relationship Types**: All relationships MUST use values from the `RelationshipType` enum
3. **Content Previews**: Limited to 100 characters by default for performance
4. **ID Generation**: UUIDs with meaningful prefixes for debugging
5. **Error Handling**: Fail fast with clear error messages, log all errors
6. **Memory Management**: Stream large files, use io.Reader/io.Writer interfaces
7. **Extensibility**: New parsers implement the `Parser` interface
8. **Work Queue Coordination**: Use config hash as run_id for automatic worker coordination
9. **Atomic Operations**: Use PostgreSQL row-level locking for atomic document claiming
10. **Distributed Processing**: Pull-based job control pattern with identical workers
11. **Design Integrity Over Backward Compatibility**: Prefer breaking changes to maintain clean design

## Design Integrity Principle

### Breaking Changes Are Preferred When Design Is Wrong

**Philosophy**: Clean design and correct implementation take precedence over backward compatibility. Technical debt from compatibility hacks compounds over time and makes the codebase harder to maintain.

**When to Break Compatibility**:
- The current implementation violates design principles
- Field names are misleading or incorrect
- The API encourages incorrect usage
- Maintaining compatibility would require ugly hacks
- The correct fix is simpler than the compatibility layer

**Examples of Good Breaking Changes**:
```go
// BAD: Maintaining compatibility with poor design
func (c *Config) GetIDColumns() []string {
    // Supporting both old and new field names
    if len(c.IDColumns) > 0 {
        return c.IDColumns
    }
    if len(c.DocIDColumns) > 0 {
        return c.DocIDColumns // deprecated
    }
    return nil
}

// GOOD: Fix the design properly
func (c *Config) Validate() error {
    // Breaking change but cleaner
    if len(c.IDColumns) == 0 {
        return fmt.Errorf("id_columns is required")
    }
    return nil
}
```

**Migration Strategy**:
1. Make the breaking change cleanly
2. Document the change clearly in release notes
3. Provide a migration script if needed
4. Update all tests to use the new design
5. Bump version number appropriately (major version for breaking changes)

**Anti-Pattern to Avoid**:
```go
// DON'T DO THIS: Accumulating compatibility cruft
field := config.NewName
if field == "" {
    field = config.OldName
}
if field == "" {
    field = config.LegacyName
}
if field == "" {
    field = config.AncientName
}
```
