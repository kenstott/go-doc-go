# Go-Doc-Go Development Guide

**Version 1.0** - Development guide for contributing to the Go-Doc-Go document processing engine.

## Development Setup

### Prerequisites

- Go 1.24 or later
- Git
- Optional: Docker for containerized testing
- Optional: PostgreSQL for distributed worker testing
- Optional: Neo4j for graph export testing

### Local Development Environment

```bash
# 1. Clone the repository
git clone https://github.com/kenstott/go-doc-go.git
cd go-doc-go

# 2. Build the worker
cd go
go build -o ../bin/goworker ./cmd/worker

# 3. Run tests
go test ./...

# 4. Build all parsers (for testing individual parsers)
cd cmd
for dir in */; do
    go build -o ../../bin/$(basename $dir) ./$dir
done
```

## Project Structure

```
go-doc-go/
├── go/                          # Go implementation
│   ├── cmd/                     # Command-line binaries
│   │   ├── worker/             # Main worker binary
│   │   ├── csvparser/          # Standalone CSV parser
│   │   ├── docxparser/         # Standalone DOCX parser
│   │   ├── pdfparser/          # Standalone PDF parser
│   │   ├── xlsxparser/         # Standalone XLSX parser
│   │   ├── pptxparser/         # Standalone PPTX parser
│   │   ├── jsonparser/         # Standalone JSON parser
│   │   ├── htmlparser/         # Standalone HTML parser
│   │   ├── markdownparser/     # Standalone Markdown parser
│   │   ├── xmlparser/          # Standalone XML parser
│   │   ├── ontology/           # Ontology CLI tool
│   │   └── parquetparser/      # Standalone Parquet parser
│   ├── internal/               # Private packages
│   │   ├── analytics/          # Analytics storage (Parquet, Neo4j)
│   │   ├── cache/              # LRU and memory-mapped caching
│   │   ├── contentsource/      # Content source implementations
│   │   ├── detector/           # Document type detection
│   │   ├── embeddings/         # ONNX embedding generation
│   │   ├── export/             # Neo4j graph export
│   │   ├── jobcontrol/         # Job control (SQLite, PostgreSQL)
│   │   ├── ontology/           # Ontology-based extraction
│   │   ├── parser/             # All document parsers
│   │   ├── resolver/           # Content resolution
│   │   ├── temporal/           # Temporal analysis
│   │   ├── udml/               # UDML query and builder
│   │   └── worker/             # Worker orchestration
│   ├── go.mod                  # Go module definition
│   └── go.sum                  # Dependency checksums
├── bin/                         # Compiled binaries (gitignored)
├── docs/                        # Documentation
├── schemas/                     # UDML JSON schemas
├── tests/                       # Test fixtures and assets
└── config.toml                  # Example configuration
```

## Development Workflow

### 1. Making Changes

```bash
# Create a feature branch
git checkout -b feature/your-feature-name

# Make your changes in go/internal/ or go/cmd/

# Test your changes
cd go
go test ./internal/parser  # Test specific package
go test ./...              # Test all packages

# Build and test the worker
go build -o ../bin/goworker ./cmd/worker
../bin/goworker --config ../config.toml --max-documents 5
```

### 2. Running Tests

```bash
cd go

# Run all tests
go test ./...

# Run specific package tests
go test ./internal/parser
go test ./internal/worker
go test ./internal/analytics

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run with race detector
go test -race ./...

# Run specific test
go test -run TestPDFParser ./internal/parser

# Verbose output
go test -v ./internal/parser
```

### 3. Code Quality Checks

```bash
# Format code
go fmt ./...

# Vet code
go vet ./...

# Run linter (install golangci-lint first)
golangci-lint run

# Check for common issues
go vet ./...

# Static analysis
staticcheck ./...
```

### 4. Testing Worker Locally

```bash
# Build worker
cd go
go build -o ../bin/goworker ./cmd/worker

# Test with minimal config
../bin/goworker --config ../config.toml --max-documents 1

# Test with debug output
../bin/goworker --config ../config.toml --max-documents 5 --workers 1

# Test distributed workers (requires PostgreSQL)
# Terminal 1
../bin/goworker --config ../config_postgres.toml --worker-id worker-01 --workers 2

# Terminal 2
../bin/goworker --config ../config_postgres.toml --worker-id worker-02 --workers 2
```

## Adding New Features

### Adding a New Document Parser

1. **Create parser in `go/internal/parser/`**:

```go
// go/internal/parser/myformat.go
package parser

import (
    "fmt"
    "github.com/kennethstott/doculyzer-go-conversion/go/internal/udml"
)

type MyFormatParser struct {
    config ParserConfig
}

func NewMyFormatParser(config ParserConfig) *MyFormatParser {
    return &MyFormatParser{config: config}
}

func (p *MyFormatParser) Parse(content []byte, docID string) (*udml.ParseResult, error) {
    elements := []udml.Element{}

    // Your parsing logic here
    // Extract elements from the document

    return &udml.ParseResult{
        Elements: elements,
    }, nil
}

func (p *MyFormatParser) Name() string {
    return "myformat"
}

func (p *MyFormatParser) SupportedTypes() []string {
    return []string{".myformat", "application/x-myformat"}
}
```

2. **Register parser in factory**:

```go
// go/internal/parser/factory.go
func NewParser(docType string, config ParserConfig) (Parser, error) {
    switch docType {
    case "myformat":
        return NewMyFormatParser(config), nil
    // ... existing parsers ...
    default:
        return nil, fmt.Errorf("unknown document type: %s", docType)
    }
}
```

3. **Add tests**:

```go
// go/internal/parser/myformat_test.go
package parser

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestMyFormatParser(t *testing.T) {
    parser := NewMyFormatParser(ParserConfig{})
    content := []byte("test content")

    result, err := parser.Parse(content, "test-doc-id")

    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Greater(t, len(result.Elements), 0)
}
```

4. **Create standalone CLI (optional)**:

```go
// go/cmd/myformatparser/main.go
package main

import (
    "flag"
    "fmt"
    "os"
    "github.com/kennethstott/doculyzer-go-conversion/go/internal/parser"
)

func main() {
    flag.Parse()

    if flag.NArg() < 1 {
        fmt.Println("Usage: myformatparser <file>")
        os.Exit(1)
    }

    // Parser implementation
}
```

### Adding a New Content Source

1. **Create source in `go/internal/contentsource/`**:

```go
// go/internal/contentsource/mysource.go
package contentsource

import (
    "context"
)

type MySource struct {
    config map[string]interface{}
}

func NewMySource(config map[string]interface{}) (*MySource, error) {
    return &MySource{config: config}, nil
}

func (s *MySource) Discover(ctx context.Context) ([]Document, error) {
    documents := []Document{}

    // Your discovery logic here
    // Return list of documents to process

    return documents, nil
}

func (s *MySource) Fetch(ctx context.Context, doc Document) ([]byte, error) {
    // Your fetch logic here
    // Return document content as bytes

    return nil, nil
}
```

2. **Register in factory**:

```go
// go/internal/contentsource/factory.go
func NewContentSource(sourceType string, config map[string]interface{}) (ContentSource, error) {
    switch sourceType {
    case "mysource":
        return NewMySource(config)
    // ... existing sources ...
    }
}
```

### Adding a New Analytics Output

1. **Create output in `go/internal/analytics/`**:

```go
// go/internal/analytics/mybackend.go
package analytics

import (
    "github.com/kennethstott/doculyzer-go-conversion/go/internal/udml"
)

type MyBackendStorage struct {
    config map[string]interface{}
}

func NewMyBackendStorage(config map[string]interface{}) (*MyBackendStorage, error) {
    return &MyBackendStorage{config: config}, nil
}

func (s *MyBackendStorage) StoreDocument(doc *udml.Document) error {
    // Store document logic
    return nil
}

func (s *MyBackendStorage) StoreElements(elements []udml.Element) error {
    // Store elements logic
    return nil
}

func (s *MyBackendStorage) Close() error {
    // Cleanup logic
    return nil
}
```

## Testing Guidelines

### Unit Tests

Test individual functions and methods:

```go
func TestParseElement(t *testing.T) {
    parser := NewMyParser(ParserConfig{})
    element := parser.ParseElement([]byte("test"))

    assert.Equal(t, "paragraph", element.ElementType)
    assert.NotEmpty(t, element.Content)
}
```

### Integration Tests

Test complete workflows:

```go
func TestWorkerProcessing(t *testing.T) {
    // Setup test environment
    tmpDir := t.TempDir()

    // Create test configuration
    config := &WorkerConfig{
        JobControl: JobControlConfig{
            Backend: "sqlite",
            Path:    filepath.Join(tmpDir, "test.db"),
        },
    }

    // Run worker
    worker := NewWorker(config)
    err := worker.ProcessDocuments(context.Background(), 1)

    assert.NoError(t, err)
}
```

### Table-Driven Tests

Test multiple scenarios efficiently:

```go
func TestParserFormats(t *testing.T) {
    tests := []struct {
        name     string
        input    []byte
        expected int  // expected element count
    }{
        {"simple", []byte("test"), 1},
        {"complex", []byte("multiple\nparagraphs"), 2},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := parser.Parse(tt.input, "test-id")
            assert.NoError(t, err)
            assert.Len(t, result.Elements, tt.expected)
        })
    }
}
```

## Debugging

### Enable Verbose Logging

```go
// Add logging in your code
import "log"

log.Printf("Processing document: %s", docID)
log.Printf("Found %d elements", len(elements))
```

### Use Delve Debugger

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug worker
dlv debug ./cmd/worker -- --config ../config.toml

# Set breakpoint
(dlv) break parser.Parse
(dlv) continue
(dlv) print element
```

### Profile Performance

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=. ./internal/parser
go tool pprof cpu.prof

# Memory profiling
go test -memprofile=mem.prof -bench=. ./internal/parser
go tool pprof mem.prof

# Profile running worker
# Add to worker code:
import "runtime/pprof"

f, _ := os.Create("cpu.prof")
pprof.StartCPUProfile(f)
defer pprof.StopCPUProfile()
```

## Configuration for Development

### Test Configuration

```toml
# config_dev.toml
[processing.job_control]
backend = "sqlite"
path = "./test_data/jobs.db"

[[content_sources]]
name = "test_docs"
type = "file"
base_path = "./test_data/documents"
file_pattern = "*.{pdf,docx}"

[analytics]
enabled = true

[[analytics.outputs]]
type = "parquet"
path = "./test_data/analytics.parquet"

[embedding]
enabled = false  # Disable for faster testing
```

### Environment Variables

```bash
# .env
GO_DOC_GO_CONFIG_PATH=./config_dev.toml
ONNXRUNTIME_SHARED_LIBRARY_PATH=/path/to/libonnxruntime.so
```

## Common Development Tasks

### Update Dependencies

```bash
# Update all dependencies
go get -u ./...

# Update specific dependency
go get -u github.com/xuri/excelize/v2

# Tidy dependencies
go mod tidy

# Verify dependencies
go mod verify
```

### Build All Binaries

```bash
# Build script
#!/bin/bash
cd go/cmd
for dir in */; do
    echo "Building $(basename $dir)..."
    go build -o ../../bin/$(basename $dir) ./$dir
done
```

### Cross-Compile for Different Platforms

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o bin/worker-linux-amd64 ./cmd/worker

# Linux ARM64 (Raspberry Pi, AWS Graviton)
GOOS=linux GOARCH=arm64 go build -o bin/worker-linux-arm64 ./cmd/worker

# macOS Intel
GOOS=darwin GOARCH=amd64 go build -o bin/worker-darwin-amd64 ./cmd/worker

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o bin/worker-darwin-arm64 ./cmd/worker

# Windows
GOOS=windows GOARCH=amd64 go build -o bin/worker-windows-amd64.exe ./cmd/worker
```

## Git Workflow

### Commit Messages

Follow conventional commits:

```bash
feat(parser): add support for MyFormat documents
fix(worker): resolve race condition in job claiming
docs(readme): update installation instructions
test(parser): add tests for edge cases
refactor(analytics): simplify Parquet storage interface
perf(embeddings): optimize batch processing
```

### Pull Request Process

1. Create feature branch: `git checkout -b feature/my-feature`
2. Make changes with tests
3. Run full test suite: `go test ./...`
4. Run code quality checks: `go fmt ./... && go vet ./...`
5. Push branch: `git push origin feature/my-feature`
6. Create pull request on GitHub
7. Address review comments
8. Merge after approval

## Troubleshooting Development Issues

### Import Errors

```bash
# Verify Go module path
go mod edit -module=github.com/kennethstott/doculyzer-go-conversion

# Update imports
goimports -w .
```

### Build Errors

```bash
# Clean build cache
go clean -cache

# Rebuild with verbose output
go build -v ./cmd/worker

# Check for missing dependencies
go mod tidy
```

### Test Failures

```bash
# Run tests with verbose output
go test -v ./internal/parser

# Run single test
go test -run TestSpecificTest ./internal/parser

# Skip cache
go test -count=1 ./...
```

## Performance Optimization

### Benchmarking

```go
// parser_bench_test.go
func BenchmarkPDFParser(b *testing.B) {
    content, _ := os.ReadFile("testdata/sample.pdf")
    parser := NewPDFParser(ParserConfig{})

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        parser.Parse(content, "test-id")
    }
}
```

Run benchmarks:

```bash
# Run all benchmarks
go test -bench=. ./...

# Run specific benchmark
go test -bench=BenchmarkPDFParser ./internal/parser

# With memory stats
go test -bench=. -benchmem ./internal/parser

# Save results for comparison
go test -bench=. ./... > bench_old.txt
# Make changes
go test -bench=. ./... > bench_new.txt
benchcmp bench_old.txt bench_new.txt
```

### Memory Profiling

```bash
# Profile memory usage
go test -memprofile=mem.prof -bench=. ./internal/parser
go tool pprof -http=:8080 mem.prof
```

## Release Process

1. Update version in code (if applicable)
2. Update CHANGELOG.md
3. Run full test suite: `go test ./...`
4. Build binaries for all platforms
5. Create git tag: `git tag v1.0.0`
6. Push tag: `git push origin v1.0.0`
7. Create GitHub release with binaries

## Contributing Guidelines

1. Fork the repository
2. Create feature branch
3. Write tests for new features
4. Ensure all tests pass: `go test ./...`
5. Format code: `go fmt ./...`
6. Update documentation
7. Submit pull request

## Resources

- [Go Documentation](https://golang.org/doc/) - Official Go docs
- [Effective Go](https://golang.org/doc/effective_go) - Go best practices
- [Go by Example](https://gobyexample.com/) - Practical examples
- [Apache Arrow Go](https://pkg.go.dev/github.com/apache/arrow/go/v14) - Parquet support
- [excelize](https://pkg.go.dev/github.com/xuri/excelize/v2) - Excel parsing
- [goquery](https://pkg.go.dev/github.com/PuerkitoBio/goquery) - HTML parsing
- [ONNX Runtime Go](https://github.com/yalue/onnxruntime_go) - ML inference

## Development Tips

1. **Start small**: Test with 1-5 documents first
2. **Use race detector**: `go test -race ./...` catches concurrency bugs
3. **Profile early**: Don't guess at performance issues
4. **Write tests first**: TDD helps design better APIs
5. **Keep binaries in bin/**: Consistent with project structure
6. **Use table-driven tests**: More maintainable test code
7. **Leverage Go tools**: `go fmt`, `go vet`, `golangci-lint`
8. **Document public APIs**: Good godoc comments help users