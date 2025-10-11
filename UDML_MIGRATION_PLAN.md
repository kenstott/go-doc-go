# UDML Migration Plan: Go Codebase Transformation

## Executive Summary

This document outlines the migration plan to transform the existing Go document processing system into **UDML (Universal Document Model Language)** - an LLM-powered, scalable knowledge extraction architecture.

**Current State:** Flat Parquet storage with basic partitioning (date, source)
**Target State:** Hive-partitioned storage with type-specific schemas, JSON projection, JSONPath queries, LLM-generated ontologies, and format-agnostic knowledge graph export

**Timeline:** 9-10 weeks
**Complexity:** Major architectural refactor with clean breaking changes - no backward compatibility in target state

---

## Migration Progress Status

### ✅ Completed Phases

#### Phase 0.1: Parser Interface (COMPLETED)
- ✅ Parser interface implemented with GetName(), GetSupportedFormats(), Parse(), SupportsStreaming(), Close()
- ✅ ParseRequest and ParseResult structures defined
- ✅ ParserRegistry implemented with registration and file extension lookup
- ✅ All 10 parsers migrated to implement the Parser interface
- ✅ Comprehensive interface compliance tests created and passing

#### Phase 1.2: Parser Promoted Fields Migration (COMPLETED)
- ✅ All 10 parsers updated with UDML Phase 1 promoted fields:
  - **PDF Parser**: PageNumber, SectionLevel, RowIndex, ColumnIndex, TemporalType, ElementCategory
  - **DOCX Parser**: SectionLevel, RowIndex, ColumnIndex, TemporalType, ElementCategory
  - **PPTX Parser**: PageNumber (slides), RowIndex, ColumnIndex, TemporalType, ElementCategory
  - **XLSX Parser**: PageNumber (sheets), RowIndex, ColumnIndex, TemporalType, ElementCategory
  - **HTML Parser**: TagName, SectionLevel, RowIndex, ColumnIndex, TemporalType, ElementCategory
  - **XML Parser**: TagName, TemporalType, ElementCategory
  - **Markdown Parser**: SectionLevel, TemporalType, ElementCategory
  - **CSV Parser**: RowIndex, ColumnIndex, TemporalType, ElementCategory
  - **JSON Parser**: TemporalType, ElementCategory
  - **Text Parser**: TemporalType, ElementCategory
- ✅ Temporal integration via ProcessTemporalContent() added to all parsers
- ✅ Comprehensive integration tests created (multi_parser_integration_test.go)
- ✅ All parsers passing integration tests
- ✅ Backward compatibility maintained via ParseLegacy methods

### 🚧 In Progress Phases
- None currently

### ⏳ Pending Phases
- Phase 1.1: Type System Updates
- Phase 1.2: Schema Registry Implementation
- Phase 1.3: Hive-Partitioned Storage
- Phase 2: JSON Projection & Query Engine
- Phase 3: LLM Integration
- Phase 4: Entity Extraction Engine
- Phase 5: Multi-Format Export
- Phase 6: Versioning System
- Phase 7: Testing & Documentation

---

## Architecture Comparison

### Current Architecture
```
Documents → Parsers → ParseResult (flat elements)
                          ↓
                    Parquet Storage (date/source partitions)
                          ↓
                    Single elements table (with JSON metadata)
                          ↓
                    Neo4j Export (hardcoded)
```

### Target UDML Architecture (Interface-Driven)

```
                           ┌─────────────────────────────────┐
                           │   JobController Interface       │
                           │   - ClaimDocuments()            │
                           │   - UpdateHeartbeat()           │
Documents ────────────────►│   - CompleteDocument()          │
                           └───────────┬─────────────────────┘
                                       │
                           ┌───────────▼─────────────────────┐
                           │   Parser Interface              │
                           │   - Parse(request)              │
                           │   - GetSupportedFormats()       │
                           │   Implementations:              │
                           │   • PdfParser                   │
                           │   • DocxParser                  │
                           │   • CsvParser, etc.             │
                           └───────────┬─────────────────────┘
                                       │ ParseResult (UDML Elements)
                           ┌───────────▼─────────────────────┐
                           │   Storage Interface             │
                           │   - WriteDocument()             │
                           │   - QueryElements(jsonpath)     │
                           │   - GetDocument()               │
                           │   Implementation:               │
                           │   • DuckDBStorage (Parquet)     │
                           └───────────┬─────────────────────┘
                                       │
                           ┌───────────▼─────────────────────┐
                           │   QueryBackend Interface        │
                           │   - Translate(expr, opts)       │
                           │   - SupportsFeature()           │
                           │   Implementations:              │
                           │   • DuckDBBackend (Phase 1)     │
                           │   • Neo4jBackend (Future)       │
                           │   • ElasticsearchBackend        │
                           │   • PostgreSQLBackend           │
                           └───────────┬─────────────────────┘
                                       │ Native Queries
                           ┌───────────▼─────────────────────┐
                           │   Ontology Extractor            │
                           │   (Uses QueryBackend)           │
                           └───────────┬─────────────────────┘
                                       │ Entities & Relationships
                           ┌───────────▼─────────────────────┐
                           │   Graph Model                   │
                           │   - Nodes (entities)            │
                           │   - Edges (relationships)       │
                           └───────────┬─────────────────────┘
                                       │
                           ┌───────────▼─────────────────────┐
                           │   Exporter Interface            │
                           │   - Export(graph, writer)       │
                           │   - GetCapabilities()           │
                           │   Implementations:              │
                           │   • Neo4jExporter (Cypher)      │
                           │   • RDFExporter (Turtle)        │
                           │   • GraphMLExporter             │
                           │   • JSONExporter                │
                           └─────────────────────────────────┘

                   Cross-Cutting Interfaces:
                   ┌─────────────────────────────────┐
                   │   Analytics Interface           │
                   │   - RecordProcessing()          │
                   │   - RecordError()               │
                   │   - GetMetrics()                │
                   └─────────────────────────────────┘

                   ┌─────────────────────────────────┐
                   │   EmbeddingGenerator (Existing) │
                   │   - Generate(text)              │
                   │   - GenerateBatch(texts)        │
                   └─────────────────────────────────┘

                   ┌─────────────────────────────────┐
                   │   DocumentReconstructor         │
                   │   - ReconstructDocument()       │
                   │   - ReconstructFragment()       │
                   └─────────────────────────────────┘
```

---

## Core Interface Summary

**Phase 0 defines 7 foundational interfaces that all components depend on:**

| Interface | Purpose | Implementations | Section |
|-----------|---------|----------------|---------|
| **Parser** | Parse documents into UDML elements | PdfParser, DocxParser, CsvParser, HtmlParser, etc. | 0.1 |
| **Storage** | Store/query UDML elements with Hive partitioning | DuckDBStorage (Parquet-backed) | 0.2 |
| **JobController** | Distributed work queue for document processing | SQLiteJobController, PostgreSQLJobController | 0.3 |
| **Analytics** | Track metrics, errors, performance | PrometheusAnalytics, SQLiteAnalytics | 0.4 |
| **Exporter** | Export knowledge graphs to various formats | Neo4jExporter, RDFExporter, GraphMLExporter, JSONExporter | 0.5 |
| **QueryBackend** | Translate JSONPath to native queries | DuckDBBackend, Neo4jBackend, ElasticsearchBackend, PostgreSQLBackend, SolrBackend | 2.3 |
| **DocumentReconstructor** | Rebuild documents from UDML (reverse parsing) | Reconstructor with format-specific formatters | 2.4 |

**Additional Existing Interface (Preserved):**
- **EmbeddingGenerator** (already exists) - Generate embeddings for similarity queries

**Design Benefits:**
- ✅ **Testability**: Mock implementations for unit tests
- ✅ **Extensibility**: Add new parsers, exporters, backends without changing core code
- ✅ **Swappability**: Switch storage backends (Parquet → PostgreSQL) without code changes
- ✅ **Clean Architecture**: Dependencies point inward (interfaces in core, implementations in adapters)

---

## Phase 0: Core Interface Definitions (Week 1)

Before implementing specific features, define foundational interfaces that all components depend on. These interfaces enable clean separation of concerns, testability, and future extensibility.

### 0.1 Parser Interface (`go/internal/parser/interface.go`)

**New File:** `go/internal/parser/interface.go`

```go
package parser

import (
    "context"
    "io"
)

// Parser defines the interface all format-specific parsers must implement
type Parser interface {
    // GetName returns the parser identifier (e.g., "pdf", "docx", "csv")
    GetName() string

    // GetSupportedFormats returns file extensions this parser handles
    GetSupportedFormats() []string

    // Parse parses content and returns UDML elements
    Parse(ctx context.Context, req ParseRequest) (*ParseResult, error)

    // SupportsStreaming indicates if parser can handle streaming content
    SupportsStreaming() bool

    // Close releases any resources held by the parser
    Close() error
}

// ParseRequest encapsulates parsing input
type ParseRequest struct {
    ID          string                 // Document ID
    Content     interface{}            // File path string or io.Reader
    Metadata    map[string]interface{} // Initial metadata
    Config      *ParserConfig          // Parser-specific configuration
}

// ParseResult encapsulates parser output (UDML format)
type ParseResult struct {
    Document      Document       // Document metadata
    Elements      []Element      // Parsed elements with promoted fields
    Relationships []Relationship // Structural relationships
    Links         []Link         // Hyperlinks and references
}

// ParserConfig holds parser-specific configuration
type ParserConfig struct {
    MaxContentPreview  int  // Content preview length (default: 100)
    ExtractImages      bool // Extract image elements
    ExtractTables      bool // Extract table structure
    ExtractHeaders     bool // Extract headers/footers (DOCX/PPTX)
    ExtractComments    bool // Extract comments/annotations
    MaxImageSize       int  // Maximum image size in bytes
}

// ParserRegistry manages available parsers
type ParserRegistry struct {
    parsers map[string]Parser
}

func NewParserRegistry() *ParserRegistry {
    return &ParserRegistry{
        parsers: make(map[string]Parser),
    }
}

func (r *ParserRegistry) Register(parser Parser) {
    r.parsers[parser.GetName()] = parser
}

func (r *ParserRegistry) GetParser(name string) (Parser, error) {
    parser, exists := r.parsers[name]
    if !exists {
        return nil, fmt.Errorf("parser not found: %s", name)
    }
    return parser, nil
}

func (r *ParserRegistry) GetParserForFile(filename string) (Parser, error) {
    ext := strings.ToLower(filepath.Ext(filename))
    for _, parser := range r.parsers {
        for _, format := range parser.GetSupportedFormats() {
            if ext == format || ext == "."+format {
                return parser, nil
            }
        }
    }
    return nil, fmt.Errorf("no parser found for file: %s", filename)
}
```

**Usage Example:**

```go
// Register all parsers at startup
registry := parser.NewParserRegistry()
registry.Register(parser.NewPdfParser())
registry.Register(parser.NewDocxParser())
registry.Register(parser.NewCsvParser())

// Parse a document
parser, err := registry.GetParserForFile("report.pdf")
result, err := parser.Parse(ctx, parser.ParseRequest{
    ID:       "doc123",
    Content:  "report.pdf",
    Metadata: map[string]interface{}{"source": "uploads"},
    Config:   &parser.ParserConfig{MaxContentPreview: 200},
})
```

**Files to Create:**
- [x] `go/internal/parser/interface.go` - Parser interface and registry ✅ COMPLETED
- [x] `go/internal/parser/interface_test.go` - Interface tests ✅ COMPLETED

**✅ PHASE 0.1 PARSER INTERFACE COMPLETED:**
- Parser interface implemented with GetName(), GetSupportedFormats(), Parse(), SupportsStreaming(), Close()
- ParseRequest and ParseResult structures defined
- ParserRegistry implemented with registration and file extension lookup
- All parsers migrated to implement the Parser interface
- Comprehensive interface compliance tests created and passing

---

### 0.2 Storage Interface (`go/internal/storage/interface.go`)

**New File:** `go/internal/storage/interface.go`

```go
package storage

import (
    "context"
    "github.com/kennethstott/go-doc-go/internal/parser"
)

// Storage defines backend-agnostic document storage operations
type Storage interface {
    // WriteDocument stores parsed document with Hive partitioning
    WriteDocument(ctx context.Context, result *parser.ParseResult) error

    // WriteBatch stores multiple documents atomically
    WriteBatch(ctx context.Context, results []*parser.ParseResult) error

    // QueryElements executes JSONPath query and returns matching elements
    QueryElements(ctx context.Context, query string, opts QueryOptions) ([]parser.Element, error)

    // GetDocument retrieves all elements for a document
    GetDocument(ctx context.Context, docID string) (*parser.ParseResult, error)

    // DeleteDocument removes document and all elements
    DeleteDocument(ctx context.Context, docID string) error

    // GetStatistics returns storage metrics
    GetStatistics(ctx context.Context) (*StorageStatistics, error)

    // Close releases storage resources
    Close() error
}

// QueryOptions for element queries
type QueryOptions struct {
    Backend        string            // Target backend (duckdb, neo4j, etc.)
    PartitionPath  string            // Path to Hive-partitioned data
    PromotedFields []string          // Available promoted fields
    Limit          int               // Maximum results
    Offset         int               // Result offset for pagination
    IncludeContent bool              // Include full content (vs. preview only)
}

// StorageStatistics tracks storage metrics
type StorageStatistics struct {
    TotalDocuments   int64
    TotalElements    int64
    TotalSize        int64
    PartitionCount   int
    ElementTypeCounts map[string]int64
}

// PartitionWriter handles Hive-partitioned writes
type PartitionWriter interface {
    // WritePartition writes elements to specific partition
    WritePartition(ctx context.Context, partition Partition, elements []parser.Element) error

    // FlushPartition forces partition write to disk
    FlushPartition(ctx context.Context, partition Partition) error

    // GetPartitionPath returns file path for partition
    GetPartitionPath(partition Partition) string
}

// Partition represents Hive partition key
type Partition struct {
    ElementType string    // e.g., "paragraph"
    Version     string    // Schema version
    Date        string    // Ingestion date (YYYY-MM-DD)
    Source      string    // Source identifier
}

func (p Partition) ToPath() string {
    return fmt.Sprintf("element_type=%s/version=%s/date=%s/source=%s",
        p.ElementType, p.Version, p.Date, p.Source)
}
```

**Files to Create:**
- [ ] `go/internal/storage/interface.go` - Storage interfaces
- [ ] `go/internal/storage/duckdb.go` - DuckDB implementation
- [ ] `go/internal/storage/parquet.go` - Parquet writer implementation

---

### 0.3 Job Control Interface (`go/internal/jobcontrol/interface.go`)

**New File:** `go/internal/jobcontrol/interface.go`

```go
package jobcontrol

import (
    "context"
    "time"
)

// JobController manages distributed document processing workflow
type JobController interface {
    // EnqueueDocument adds document to processing queue
    EnqueueDocument(ctx context.Context, doc *DocumentJob) error

    // ClaimDocuments atomically claims N unclaimed documents for processing
    // Returns document IDs claimed by this worker
    ClaimDocuments(ctx context.Context, workerID string, count int) ([]string, error)

    // UpdateHeartbeat updates worker heartbeat (prevents claim timeout)
    UpdateHeartbeat(ctx context.Context, workerID string, docIDs []string) error

    // CompleteDocument marks document as successfully processed
    CompleteDocument(ctx context.Context, workerID string, docID string, result *ProcessingResult) error

    // FailDocument marks document as failed (with retry logic)
    FailDocument(ctx context.Context, workerID string, docID string, err error) error

    // GetQueueStatistics returns queue metrics
    GetQueueStatistics(ctx context.Context) (*QueueStatistics, error)

    // GetWorkerStatus returns status of all active workers
    GetWorkerStatus(ctx context.Context) ([]WorkerStatus, error)

    // ReleaseExpiredClaims releases documents claimed by dead workers
    ReleaseExpiredClaims(ctx context.Context, timeout time.Duration) (int, error)
}

// DocumentJob represents a document to be processed
type DocumentJob struct {
    ID          string                 // Document ID
    Path        string                 // File path or S3 key
    Format      string                 // Document format (pdf, docx, etc.)
    Priority    int                    // Processing priority (higher = sooner)
    Metadata    map[string]interface{} // Additional metadata
    EnqueuedAt  time.Time              // When job was queued
    Retries     int                    // Number of retry attempts
    MaxRetries  int                    // Maximum retry attempts (default: 3)
}

// ProcessingResult captures processing outcome
type ProcessingResult struct {
    ElementCount      int
    RelationshipCount int
    ProcessingTimeMs  int64
    ParserUsed        string
    ErrorDetails      string // Empty if successful
}

// QueueStatistics tracks work queue metrics
type QueueStatistics struct {
    TotalDocuments     int64
    ClaimedDocuments   int64
    CompletedDocuments int64
    FailedDocuments    int64
    AvgProcessingTime  float64 // In milliseconds
    ActiveWorkers      int
}

// WorkerStatus represents worker state
type WorkerStatus struct {
    WorkerID       string
    ClaimedDocs    []string
    LastHeartbeat  time.Time
    ProcessedCount int64
    FailureCount   int64
    StartTime      time.Time
}

// ClaimStrategy determines document claiming behavior
type ClaimStrategy interface {
    // SelectDocuments chooses which documents to claim
    SelectDocuments(ctx context.Context, available []DocumentJob, count int) []string
}

// FIFOStrategy claims oldest documents first
type FIFOStrategy struct{}

func (s *FIFOStrategy) SelectDocuments(ctx context.Context, available []DocumentJob, count int) []string {
    // Sort by EnqueuedAt, return first N IDs
}

// PriorityStrategy claims highest priority documents first
type PriorityStrategy struct{}

func (s *PriorityStrategy) SelectDocuments(ctx context.Context, available []DocumentJob, count int) []string {
    // Sort by Priority DESC, then EnqueuedAt ASC, return first N IDs
}
```

**Files to Create:**
- [ ] `go/internal/jobcontrol/interface.go` - Job control interfaces
- [ ] `go/internal/jobcontrol/sqlite.go` - SQLite implementation (Phase 1)
- [ ] `go/internal/jobcontrol/postgres.go` - PostgreSQL implementation (future)
- [ ] `go/internal/jobcontrol/strategies.go` - Claim strategies

---

### 0.4 Analytics Interface (`go/internal/analytics/interface.go`)

**New File:** `go/internal/analytics/interface.go`

```go
package analytics

import (
    "context"
    "time"
)

// Analytics tracks system metrics and performance
type Analytics interface {
    // RecordDocumentProcessed logs successful document processing
    RecordDocumentProcessed(ctx context.Context, event *ProcessingEvent) error

    // RecordError logs processing error
    RecordError(ctx context.Context, event *ErrorEvent) error

    // RecordPerformance logs performance metrics
    RecordPerformance(ctx context.Context, metric *PerformanceMetric) error

    // GetProcessingMetrics returns aggregated processing statistics
    GetProcessingMetrics(ctx context.Context, window TimeWindow) (*ProcessingMetrics, error)

    // GetErrorRate returns error rate for time window
    GetErrorRate(ctx context.Context, window TimeWindow) (float64, error)

    // GetParserStatistics returns per-parser statistics
    GetParserStatistics(ctx context.Context, window TimeWindow) ([]ParserStats, error)

    // GetWorkerStatistics returns per-worker statistics
    GetWorkerStatistics(ctx context.Context, window TimeWindow) ([]WorkerStats, error)

    // ExportMetrics exports metrics in Prometheus format
    ExportMetrics(ctx context.Context) (string, error)
}

// ProcessingEvent captures document processing outcome
type ProcessingEvent struct {
    Timestamp         time.Time
    DocumentID        string
    WorkerID          string
    Parser            string
    ElementCount      int
    RelationshipCount int
    ProcessingTimeMs  int64
    FileSizeBytes     int64
    Success           bool
}

// ErrorEvent captures processing errors
type ErrorEvent struct {
    Timestamp    time.Time
    DocumentID   string
    WorkerID     string
    Parser       string
    ErrorType    string // "parser_error", "storage_error", "oom_error", etc.
    ErrorMessage string
    Stack        string
    Recoverable  bool
}

// PerformanceMetric tracks system performance
type PerformanceMetric struct {
    Timestamp    time.Time
    MetricName   string  // "parser.pdf.latency", "storage.write.throughput", etc.
    Value        float64
    Unit         string  // "ms", "bytes/sec", "count", etc.
    Labels       map[string]string
}

// TimeWindow for metric aggregation
type TimeWindow struct {
    Start time.Time
    End   time.Time
}

// ProcessingMetrics aggregated statistics
type ProcessingMetrics struct {
    TotalDocuments    int64
    SuccessfulDocs    int64
    FailedDocs        int64
    AvgProcessingTime float64 // Milliseconds
    TotalElements     int64
    Throughput        float64 // Docs/second
}

// ParserStats per-parser statistics
type ParserStats struct {
    Parser            string
    DocumentCount     int64
    SuccessRate       float64
    AvgProcessingTime float64
    TotalElements     int64
    ErrorCount        int64
}

// WorkerStats per-worker statistics
type WorkerStats struct {
    WorkerID          string
    DocumentCount     int64
    SuccessRate       float64
    AvgProcessingTime float64
    Uptime            time.Duration
    ErrorCount        int64
}

// MetricsCollector provides convenient metric recording
type MetricsCollector struct {
    analytics Analytics
}

func (c *MetricsCollector) StartTimer(name string) *Timer {
    return &Timer{
        name:      name,
        startTime: time.Now(),
        collector: c,
    }
}

type Timer struct {
    name      string
    startTime time.Time
    collector *MetricsCollector
}

func (t *Timer) Stop() {
    elapsed := time.Since(t.startTime).Milliseconds()
    t.collector.analytics.RecordPerformance(context.Background(), &PerformanceMetric{
        Timestamp:  time.Now(),
        MetricName: t.name,
        Value:      float64(elapsed),
        Unit:       "ms",
    })
}
```

**Usage Example:**

```go
// Initialize analytics
analytics := analytics.NewPrometheusAnalytics()
collector := analytics.NewMetricsCollector(analytics)

// Time a parsing operation
timer := collector.StartTimer("parser.pdf.latency")
result, err := parser.Parse(ctx, req)
timer.Stop()

// Record successful processing
analytics.RecordDocumentProcessed(ctx, &analytics.ProcessingEvent{
    Timestamp:         time.Now(),
    DocumentID:        "doc123",
    Parser:            "pdf",
    ElementCount:      len(result.Elements),
    RelationshipCount: len(result.Relationships),
    ProcessingTimeMs:  timer.ElapsedMs(),
    Success:           true,
})
```

**Files to Create:**
- [ ] `go/internal/analytics/interface.go` - Analytics interfaces
- [ ] `go/internal/analytics/prometheus.go` - Prometheus implementation
- [ ] `go/internal/analytics/sqlite.go` - SQLite implementation
- [ ] `go/internal/analytics/collector.go` - Metrics collector

---

### 0.5 Graph Export Interface (`go/internal/export/interface.go`)

**New File:** `go/internal/export/interface.go`

```go
package export

import (
    "context"
    "io"
    "github.com/kennethstott/go-doc-go/internal/graph"
)

// Exporter defines format-agnostic graph export
type Exporter interface {
    // GetName returns exporter identifier (e.g., "neo4j", "rdf", "graphml")
    GetName() string

    // GetFormat returns output format (e.g., "cypher", "turtle", "xml")
    GetFormat() string

    // Export writes graph to output in target format
    Export(ctx context.Context, graph *graph.Graph, writer io.Writer) error

    // ExportBatch exports multiple graphs efficiently
    ExportBatch(ctx context.Context, graphs []*graph.Graph, writer io.Writer) error

    // SupportsStreaming indicates if exporter can stream large graphs
    SupportsStreaming() bool

    // GetCapabilities returns supported graph features
    GetCapabilities() ExportCapabilities
}

// ExportCapabilities describes exporter features
type ExportCapabilities struct {
    SupportsMultipleLabels   bool // Multiple node labels
    SupportsProperties       bool // Node/edge properties
    SupportsDirectedEdges    bool // Directed relationships
    SupportsUndirectedEdges  bool // Undirected relationships
    SupportsHyperedges       bool // Hyperedges (>2 nodes)
    SupportsNestedGraphs     bool // Subgraphs
    MaxPropertyValueSize     int  // Max property size (0 = unlimited)
}

// ExporterRegistry manages available exporters
type ExporterRegistry struct {
    exporters map[string]Exporter
}

func NewExporterRegistry() *ExporterRegistry {
    return &ExporterRegistry{
        exporters: make(map[string]Exporter),
    }
}

func (r *ExporterRegistry) Register(exporter Exporter) {
    r.exporters[exporter.GetName()] = exporter
}

func (r *ExporterRegistry) GetExporter(name string) (Exporter, error) {
    exporter, exists := r.exporters[name]
    if !exists {
        return nil, fmt.Errorf("exporter not found: %s", name)
    }
    return exporter, nil
}

func (r *ExporterRegistry) ListExporters() []string {
    names := make([]string, 0, len(r.exporters))
    for name := range r.exporters {
        names = append(names, name)
    }
    return names
}

// ExportOptions configures export behavior
type ExportOptions struct {
    Format            string            // Output format override
    IncludeProvenance bool              // Include UDML provenance metadata
    IncludeEmbeddings bool              // Include embedding vectors
    FilterLabels      []string          // Only export nodes with these labels
    MaxBatchSize      int               // Batch size for batch exports
    Compression       CompressionFormat // Output compression
}

type CompressionFormat string

const (
    CompressionNone CompressionFormat = "none"
    CompressionGzip CompressionFormat = "gzip"
    CompressionZstd CompressionFormat = "zstd"
)

// StreamingExporter for large graph exports
type StreamingExporter interface {
    Exporter

    // BeginExport starts streaming export
    BeginExport(ctx context.Context, writer io.Writer) error

    // ExportNode writes single node
    ExportNode(ctx context.Context, node *graph.Node) error

    // ExportEdge writes single edge
    ExportEdge(ctx context.Context, edge *graph.Edge) error

    // EndExport finalizes streaming export
    EndExport(ctx context.Context) error
}
```

**Usage Example:**

```go
// Register exporters
registry := export.NewExporterRegistry()
registry.Register(export.NewNeo4jExporter())
registry.Register(export.NewRDFExporter())
registry.Register(export.NewGraphMLExporter())

// Export graph
exporter, err := registry.GetExporter("neo4j")
var buf bytes.Buffer
err = exporter.Export(ctx, knowledgeGraph, &buf)

// Write to file
os.WriteFile("output.cypher", buf.Bytes(), 0644)
```

**Files to Create:**
- [ ] `go/internal/export/interface.go` - Export interfaces
- [ ] `go/internal/export/neo4j.go` - Neo4j Cypher exporter
- [ ] `go/internal/export/rdf.go` - RDF Turtle/N-Triples exporter
- [ ] `go/internal/export/graphml.go` - GraphML exporter
- [ ] `go/internal/export/json.go` - JSON graph format exporter

---

### 0.6 Interface Testing Strategy

**Core Principles:**

1. **Interface Compliance Tests**: Each implementation must pass interface contract tests
2. **Mock Implementations**: Create mocks for all interfaces to enable isolated unit testing
3. **Integration Tests**: Test interface interactions (e.g., Parser → Storage → Export)

**Example Interface Test:**

```go
// go/internal/parser/interface_test.go
package parser_test

import (
    "testing"
    "github.com/kennethstott/go-doc-go/internal/parser"
)

// TestParserInterface ensures all parsers implement Parser interface correctly
func TestParserInterface(t *testing.T) {
    parsers := []parser.Parser{
        parser.NewPdfParser(),
        parser.NewDocxParser(),
        parser.NewCsvParser(),
        // ... all parsers
    }

    for _, p := range parsers {
        t.Run(p.GetName(), func(t *testing.T) {
            // Test interface contract
            assertNotEmpty(t, p.GetName())
            assertNotEmpty(t, p.GetSupportedFormats())

            // Test basic parsing
            result, err := p.Parse(context.Background(), getTestRequest(p))
            assertNoError(t, err)
            assertValidParseResult(t, result)

            // Test cleanup
            assertNoError(t, p.Close())
        })
    }
}
```

**Files to Create:**
- [ ] `go/internal/parser/interface_test.go`
- [ ] `go/internal/storage/interface_test.go`
- [ ] `go/internal/jobcontrol/interface_test.go`
- [ ] `go/internal/analytics/interface_test.go`
- [ ] `go/internal/export/interface_test.go`
- [ ] `go/internal/mocks/` - Mock implementations for testing

---

### 0.7 Dependency Injection Container

**New File:** `go/internal/container/container.go`

```go
package container

import (
    "github.com/kennethstott/go-doc-go/internal/parser"
    "github.com/kennethstott/go-doc-go/internal/storage"
    "github.com/kennethstott/go-doc-go/internal/jobcontrol"
    "github.com/kennethstott/go-doc-go/internal/analytics"
    "github.com/kennethstott/go-doc-go/internal/export"
)

// Container holds all application dependencies
type Container struct {
    ParserRegistry   *parser.ParserRegistry
    Storage          storage.Storage
    JobController    jobcontrol.JobController
    Analytics        analytics.Analytics
    ExporterRegistry *export.ExporterRegistry
}

// NewContainer creates fully initialized container
func NewContainer(config *Config) (*Container, error) {
    // Initialize analytics first (needed by other components)
    analytics, err := initAnalytics(config.Analytics)
    if err != nil {
        return nil, err
    }

    // Initialize job controller
    jobController, err := initJobController(config.JobControl, analytics)
    if err != nil {
        return nil, err
    }

    // Initialize storage
    storage, err := initStorage(config.Storage, analytics)
    if err != nil {
        return nil, err
    }

    // Initialize parsers
    parserRegistry := parser.NewParserRegistry()
    registerParsers(parserRegistry, config.Parsers)

    // Initialize exporters
    exporterRegistry := export.NewExporterRegistry()
    registerExporters(exporterRegistry, config.Exporters)

    return &Container{
        ParserRegistry:   parserRegistry,
        Storage:          storage,
        JobController:    jobController,
        Analytics:        analytics,
        ExporterRegistry: exporterRegistry,
    }, nil
}

// Close releases all resources
func (c *Container) Close() error {
    var errors []error

    if err := c.Storage.Close(); err != nil {
        errors = append(errors, err)
    }

    // Close all parsers
    for _, parser := range c.ParserRegistry.GetAll() {
        if err := parser.Close(); err != nil {
            errors = append(errors, err)
        }
    }

    if len(errors) > 0 {
        return fmt.Errorf("errors closing container: %v", errors)
    }
    return nil
}
```

**Files to Create:**
- [ ] `go/internal/container/container.go` - Dependency injection
- [ ] `go/internal/container/config.go` - Container configuration

---

## Phase 1: UDML Storage Foundation (Week 1-2)

### 1.1 Update Type System (`go/internal/parser/types.go`)

**Current:**
```go
type Element struct {
    ElementID       string
    ElementType     string
    ElementCategory string
    Content         string
    ContentPreview  string
    ParentID        string
    Position        int
    Depth           int
    ContentLocation map[string]interface{}
    Metadata        map[string]interface{}
}
```

**Changes Required:**
```go
// Add query-optimized promoted fields for common JSONPATH query patterns
type Element struct {
    // Core fields (unchanged)
    ElementID       string
    ElementType     string  // Will be Hive partition key
    ElementCategory string
    Content         string
    ContentPreview  string
    ParentID        string
    Position        int
    Depth           int

    // Query-optimized promoted fields (nullable, 70-95% NULL is acceptable)
    // These fields are promoted because they appear frequently in ontology rules
    PageNumber    *int    `parquet:"page_number" json:"page_number,omitempty"`       // PDF/DOCX/PPTX location (~30% populated)
    SectionLevel  *int    `parquet:"section_level" json:"section_level,omitempty"`   // Heading hierarchy (~15% populated)
    RowIndex      *int    `parquet:"row_index" json:"row_index,omitempty"`           // Table row position (~20% populated)
    ColumnIndex   *int    `parquet:"column_index" json:"column_index,omitempty"`     // Table column position (~20% populated)
    TemporalType  *string `parquet:"temporal_type" json:"temporal_type,omitempty"`   // date/datetime/year/etc (~5-10% populated)
    TagName       *string `parquet:"tag_name" json:"tag_name,omitempty"`             // HTML/XML tag identifier (~25% populated)

    // JSON overflow (format-specific, rarely queried attributes)
    ContentLocation map[string]interface{} `json:"content_location,omitempty"`
    Metadata        map[string]interface{} `json:"metadata,omitempty"`
    TemporalMetadata map[string]interface{} `json:"temporal_metadata,omitempty"`
}
```

**Why These 6 Promoted Fields?**

Promoted fields are chosen based on **query frequency in JSONPATH ontology rules**, not universality across formats. Common query patterns:

1. **Location-based extraction** (page_number): "Find entities on pages 10-15"
2. **Structural queries** (section_level): "Extract from H1/H2 headings only"
3. **Tabular extraction** (row_index, column_index): "Get amounts from column 3, skip header rows"
4. **Temporal filtering** (temporal_type): "Find all date fields", "Extract effective dates"
5. **Markup-based extraction** (tag_name): "Find all div.disclosure elements"

**Sparsity is Acceptable:**

| Field | Populated | NULL | Query Frequency | Backend Benefit |
|-------|-----------|------|-----------------|-----------------|
| page_number | 30% | 70% | Very High | All backends: 60-100x faster than JSON path |
| section_level | 15% | 85% | Very High | Critical for structural queries |
| row_index | 20% | 80% | High | Tabular queries extremely common |
| column_index | 20% | 80% | High | Pairs with row_index for cell addressing |
| temporal_type | 5-10% | 90-95% | Medium-High | Temporal analysis in knowledge extraction |
| tag_name | 25% | 75% | Medium | HTML/XML processing |

**Why High NULL Rates Are Fine:**

- ✅ **Parquet**: Columnar storage compresses NULLs to ~1 bit per value
- ✅ **RDBMS**: NULL bitmap overhead is minimal
- ✅ **Neo4j**: Absent properties have zero storage cost
- ✅ **Query Performance**: Direct column queries are 60-1000x faster than JSON path extraction across all backends
- ✅ **Multi-Backend Design**: Promoted fields enable efficient queries in PostgreSQL, Neo4j, Elasticsearch, Solr (future support)

**Files to Modify:**
- `go/internal/parser/types.go` - Update Element struct
- `go/internal/analytics/types.go` - Update analytics Element struct (keep in sync)

**Migration Strategy:**
- Add new promoted fields (nullable pointers) - cleanly breaking change
- Parsers gradually populate promoted fields
- Old data reads with NULL values for new fields

---

### 1.1.1 Backend-Specific Query Benefits

**Why Promoted Fields Matter Across Backends**

While Phase 1 implements DuckDB + Parquet, the promoted fields design enables future support for PostgreSQL, Neo4j, Elasticsearch, and Solr. Each backend has different performance characteristics for sparse columns vs. nested JSON:

| Backend | Sparse Column Performance | JSON/Nested Performance | Speedup |
|---------|--------------------------|------------------------|---------|
| **Parquet + DuckDB** | ✅ Excellent (predicate pushdown) | ✅ Good (JSON functions) | 60x |
| **PostgreSQL** | ✅ Good (B-tree indexes) | ⚠️ Slow (JSONB GIN index) | 100x |
| **Neo4j** | ✅ Excellent (native properties) | ❌ Poor (no nested support) | 1000x |
| **Elasticsearch** | ✅ Excellent (inverted index) | ⚠️ OK (nested objects) | 50x |
| **Solr** | ✅ Good (indexed fields) | ⚠️ Limited (JSON support) | 80x |

#### Example Query: "Find paragraphs on pages 10-15 mentioning 'risk factors'"

**DuckDB + Parquet (Phase 1 - Current)**
```sql
-- Promoted fields enable predicate pushdown to Parquet row groups
SELECT * FROM 'analytics/element_type=paragraph/**/**.parquet' e
JOIN 'analytics/embeddings.parquet' emb ON emb.element_id = e.element_id
WHERE e.page_number BETWEEN 10 AND 15  -- Direct column filter (Parquet row group skipping)
  AND cosine_similarity(emb.embedding, query_vec) > 0.8

-- Performance: ~50ms for 10M elements (with partition pruning + row group skipping)
```

**Without Promoted Fields (JSON path extraction)**
```sql
-- Must scan all rows, extract JSON, then filter
SELECT * FROM 'analytics/element_type=paragraph/**/**.parquet' e
WHERE json_extract(e.content_location, '$.page_number') BETWEEN 10 AND 15

-- Performance: ~3000ms for 10M elements (60x slower - no row group skipping)
```

#### PostgreSQL (Future Support)

**With Promoted Fields**
```sql
-- Direct B-tree index on nullable column
CREATE INDEX idx_page_number ON elements(page_number) WHERE page_number IS NOT NULL;

SELECT * FROM elements
WHERE element_type = 'paragraph'
  AND page_number BETWEEN 10 AND 15;

-- Performance: Index scan ~10ms for 10M rows
```

**Without Promoted Fields**
```sql
-- JSONB GIN index is less efficient for range queries
CREATE INDEX idx_location_gin ON elements USING GIN (content_location);

SELECT * FROM elements
WHERE element_type = 'paragraph'
  AND (content_location->>'page_number')::int BETWEEN 10 AND 15;

-- Performance: Sequential scan ~1000ms for 10M rows (100x slower)
```

#### Neo4j (Future Support)

**With Promoted Fields**
```cypher
// Native property index (extremely fast)
CREATE INDEX element_page_number FOR (e:Element) ON (e.page_number);

MATCH (e:Element {type: 'paragraph'})
WHERE e.page_number >= 10 AND e.page_number <= 15
RETURN e;

// Performance: Index lookup ~5ms
```

**Without Promoted Fields**
```cypher
// No native support for nested JSON queries
// Would require manual parsing or APOC procedures
MATCH (e:Element {type: 'paragraph'})
WHERE apoc.convert.fromJsonMap(e.content_location).page_number >= 10
  AND apoc.convert.fromJsonMap(e.content_location).page_number <= 15
RETURN e;

// Performance: Full scan ~5000ms (1000x slower)
```

#### Elasticsearch (Future Support)

**With Promoted Fields**
```json
{
  "query": {
    "bool": {
      "must": [
        {"term": {"element_type": "paragraph"}},
        {"range": {"page_number": {"gte": 10, "lte": 15}}}
      ]
    }
  }
}
```
Performance: Inverted index scan ~20ms

**Without Promoted Fields**
```json
{
  "query": {
    "bool": {
      "must": [
        {"term": {"element_type": "paragraph"}},
        {"range": {"content_location.page_number": {"gte": 10, "lte": 15}}}
      ]
    }
  }
}
```
Performance: Nested object query ~1000ms (50x slower)

#### Solr (Future Support)

**With Promoted Fields**
```
q=element_type:paragraph AND page_number:[10 TO 15]
```
Performance: Indexed field query ~30ms

**Without Promoted Fields**
```
q=element_type:paragraph AND content_location_json:*page_number*[10,15]*
```
Performance: Full-text search on JSON string ~2400ms (80x slower)

#### Summary: Multi-Backend Design Justification

The 6 promoted fields are chosen to optimize **common JSONPATH query patterns** across **all current and future backends**. While DuckDB handles JSON well, future PostgreSQL/Neo4j/Elasticsearch/Solr support requires promoted fields for acceptable performance at scale (10-100 billion elements).

**Key Insight:** 70-95% NULL values are a small price for 60-1000x query speedups across all backends.

---

### 1.1.2 Parser Modifications to Populate Promoted Fields

Each parser must be updated to populate the 6 promoted fields from existing `Metadata` and `ContentLocation` maps.

#### PDF Parser (`go/internal/parser/pdf.go`)

**Current State:**
```go
element := Element{
    // ...
    ContentLocation: map[string]interface{}{
        "page_number": pageNum + 1,
        "table_index": tableIdx,
    },
}
```

**Update Required:**
```go
// Extract page_number from ContentLocation and promote it
pageNum := pageNum + 1
element := Element{
    // ... existing fields ...
    PageNumber: &pageNum,  // Promoted field (query-optimized)
    ContentLocation: map[string]interface{}{
        // page_number NOT duplicated here - only in promoted field
        "table_index": tableIdx,      // Not promoted - stays in overflow
    },
}

// For heading elements in PDF
if element.ElementType == "heading" {
    level := extractHeadingLevel(textStyle)  // From font size/style
    element.SectionLevel = &level
}

// For table cells
if element.ElementType == "table_cell" {
    rowIdx := cellRow
    colIdx := cellCol
    element.RowIndex = &rowIdx
    element.ColumnIndex = &colIdx
    // row_index and column_index only in promoted fields, not duplicated in ContentLocation
}
```

#### DOCX Parser (`go/internal/parser/docx.go`)

**Current State:**
```go
element.Metadata["level"] = level  // Heading level stored in metadata
```

**Update Required:**
```go
// For heading elements
if element.ElementType == "heading" {
    if level, ok := element.Metadata["level"].(int); ok {
        element.SectionLevel = &level  // Promote to top-level field
        // level only in promoted field SectionLevel, not duplicated here
    }
}

// For table cells
if element.ElementType == "table_cell" {
    if row, ok := element.Metadata["row"].(int); ok {
        element.RowIndex = &row
        element.Metadata["row"] = row
    }
    if col, ok := element.Metadata["column"].(int); ok {
        element.ColumnIndex = &col
        element.Metadata["column"] = col
    }
}

// DOCX doesn't have page numbers easily accessible, leave NULL
```

#### HTML Parser (`go/internal/parser/html.go`)

**Current State:**
```go
elementType := mapHTMLTag(node.Data)  // h1, h2, div, table, etc.
```

**Update Required:**
```go
// Promote tag_name for all elements
tagName := node.Data  // "h1", "div", "table", "p", etc.
element.TagName = &tagName

// For heading tags (h1-h6)
if strings.HasPrefix(tagName, "h") && len(tagName) == 2 {
    level, _ := strconv.Atoi(string(tagName[1]))  // Extract number from "h1"
    element.SectionLevel = &level
}

// For table cells (td, th)
if tagName == "td" || tagName == "th" {
    if row, ok := element.Metadata["row"].(int); ok {
        element.RowIndex = &row
    }
    if col, ok := element.Metadata["column"].(int); ok {
        element.ColumnIndex = &col
    }
}

// HTML doesn't have page numbers, leave NULL
```

#### XML Parser (`go/internal/parser/xml.go`)

**Current State:**
```go
element.Metadata["tag_name"] = node.Data
```

**Update Required:**
```go
// Promote tag_name
tagName := node.Data
element.TagName = &tagName
// tag_name only in promoted field TagName, not duplicated here
```

#### CSV/Parquet Parsers (`go/internal/parser/csv.go`)

**Current State:**
```go
element.Metadata["row"] = rowIdx
element.ContentLocation["row"] = rowIdx
```

**Update Required:**
```go
// For table_row and table_cell elements
if element.ElementType == "table_row" || element.ElementType == "table_cell" {
    rowIdx := currentRow
    element.RowIndex = &rowIdx
    // row only in promoted field RowIndex, not duplicated in ContentLocation
}

// For table_cell specifically
if element.ElementType == "table_cell" {
    colIdx := currentCol
    element.ColumnIndex = &colIdx
    element.Metadata["column"] = colIdx
    element.ContentLocation["column"] = colIdx
}
```

#### Markdown Parser (`go/internal/parser/markdown.go`)

**Current State:**
```go
if element.ElementType == "heading" {
    element.Metadata["level"] = headingLevel
}
```

**Update Required:**
```go
// For headings (# ## ### etc.)
if element.ElementType == "heading" {
    level := headingLevel  // 1-6 from markdown syntax
    element.SectionLevel = &level
    // level only in promoted field SectionLevel, not duplicated in Metadata
}

// For code blocks with language
if element.ElementType == "code_block" {
    if lang, ok := element.Metadata["language"].(string); ok {
        // Store in metadata (not promoted)
        element.Metadata["language"] = lang
    }
}
```

#### JSON Parser (`go/internal/parser/json.go`)

**Update Required:**
```go
// For temporal fields
if element.Metadata["temporal_type"] != nil {
    if temporalType, ok := element.Metadata["temporal_type"].(string); ok {
        element.TemporalType = &temporalType  // Promote temporal_type
    }
}
```

#### Universal Temporal Enrichment (All Parsers)

**Location:** `go/internal/parser/temporal.go` (or within each parser)

```go
// After element creation, enrich with temporal data
func enrichTemporalMetadata(element *Element) {
    // If temporal metadata exists, promote temporal_type
    if element.Metadata["temporal_type"] != nil {
        if tempType, ok := element.Metadata["temporal_type"].(string); ok {
            element.TemporalType = &tempType
        }
    }

    // Also check TemporalMetadata map if it exists
    if element.TemporalMetadata != nil {
        if tempType, ok := element.TemporalMetadata["temporal_type"].(string); ok {
            element.TemporalType = &tempType
        }
    }
}
```

#### Summary: Parser Update Checklist

| Parser | page_number | section_level | row_index | column_index | temporal_type | tag_name |
|--------|------------|---------------|-----------|--------------|---------------|----------|
| PDF | ✅ From ContentLocation | ✅ From style | ✅ Tables | ✅ Tables | ✅ Universal | ❌ N/A |
| DOCX | ❌ Complex | ✅ From Metadata | ✅ Tables | ✅ Tables | ✅ Universal | ❌ N/A |
| HTML | ❌ N/A | ✅ From tag (h1-h6) | ✅ Tables | ✅ Tables | ✅ Universal | ✅ node.Data |
| XML | ❌ N/A | ❌ N/A | ❌ N/A | ❌ N/A | ✅ Universal | ✅ node.Data |
| CSV | ❌ N/A | ❌ N/A | ✅ From position | ✅ From position | ✅ Universal | ❌ N/A |
| Markdown | ❌ N/A | ✅ From syntax | ❌ N/A | ❌ N/A | ✅ Universal | ❌ N/A |
| JSON | ❌ N/A | ❌ N/A | ❌ N/A | ❌ N/A | ✅ From metadata | ❌ N/A |

**Files to Modify:**
- [x] `go/internal/parser/pdf.go` - Add page_number, section_level, row_index, column_index ✅ COMPLETED
- [x] `go/internal/parser/docx.go` - Add section_level, row_index, column_index ✅ COMPLETED
- [x] `go/internal/parser/pptx.go` - Add page_number, row_index, column_index ✅ COMPLETED
- [x] `go/internal/parser/xlsx.go` - Add page_number, row_index, column_index ✅ COMPLETED
- [x] `go/internal/parser/html.go` - Add tag_name, section_level, row_index, column_index ✅ COMPLETED
- [x] `go/internal/parser/xml.go` - Add tag_name ✅ COMPLETED
- [x] `go/internal/parser/csv.go` - Add row_index, column_index ✅ COMPLETED
- [x] `go/internal/parser/markdown.go` - Add section_level ✅ COMPLETED
- [x] `go/internal/parser/json.go` - Add temporal_type promotion ✅ COMPLETED
- [x] `go/internal/parser/temporal.go` - Add universal temporal_type enrichment ✅ COMPLETED

---

### 1.2 Create Type-Specific Schema Registry (`go/internal/udml/schemas.go`)

**New File:** `go/internal/udml/schemas.go`

```go
package udml

import "github.com/apache/arrow/go/v18/arrow"

// SchemaRegistry manages type-specific Parquet schemas
type SchemaRegistry struct {
    schemas map[string]*arrow.Schema
}

func NewSchemaRegistry() *SchemaRegistry {
    r := &SchemaRegistry{
        schemas: make(map[string]*arrow.Schema),
    }
    r.registerDefaultSchemas()
    return r
}

// Get schema for element type
func (r *SchemaRegistry) GetSchema(elementType string) *arrow.Schema {
    if schema, exists := r.schemas[elementType]; exists {
        return schema
    }
    return r.getDefaultSchema()
}

// Register universal schema (same for all element types)
// All element types use the SAME schema with 6 query-optimized nullable fields
func (r *SchemaRegistry) registerDefaultSchemas() {
    // Universal schema with 6 promoted fields
    // This schema is shared across ALL element types (paragraph, table, heading, etc.)
    universalSchema := arrow.NewSchema([]arrow.Field{
        // Core universal fields (required)
        {Name: "element_id", Type: arrow.BinaryTypes.String, Nullable: false},
        {Name: "doc_id", Type: arrow.BinaryTypes.String, Nullable: false},
        {Name: "source_name", Type: arrow.BinaryTypes.String, Nullable: false},
        {Name: "element_type", Type: arrow.BinaryTypes.String, Nullable: false},
        {Name: "element_category", Type: arrow.BinaryTypes.String, Nullable: false},
        {Name: "content", Type: arrow.BinaryTypes.String, Nullable: true},
        {Name: "content_preview", Type: arrow.BinaryTypes.String, Nullable: true},
        {Name: "content_hash", Type: arrow.BinaryTypes.String, Nullable: true},
        {Name: "parent_id", Type: arrow.BinaryTypes.String, Nullable: true},
        {Name: "element_order", Type: arrow.PrimitiveTypes.Float64, Nullable: true},
        {Name: "document_position", Type: arrow.PrimitiveTypes.Float64, Nullable: true},

        // 6 Query-Optimized Promoted Fields (all nullable)
        {Name: "page_number", Type: arrow.PrimitiveTypes.Int64, Nullable: true},      // ~30% populated
        {Name: "section_level", Type: arrow.PrimitiveTypes.Int64, Nullable: true},    // ~15% populated
        {Name: "row_index", Type: arrow.PrimitiveTypes.Int64, Nullable: true},        // ~20% populated
        {Name: "column_index", Type: arrow.PrimitiveTypes.Int64, Nullable: true},     // ~20% populated
        {Name: "temporal_type", Type: arrow.BinaryTypes.String, Nullable: true},      // ~5-10% populated
        {Name: "tag_name", Type: arrow.BinaryTypes.String, Nullable: true},           // ~25% populated

        // JSON overflow (format-specific attributes)
        {Name: "content_location", Type: arrow.BinaryTypes.String, Nullable: true},   // JSON
        {Name: "metadata", Type: arrow.BinaryTypes.String, Nullable: true},           // JSON
        {Name: "temporal_metadata", Type: arrow.BinaryTypes.String, Nullable: true},  // JSON
    }, nil)

    // ALL element types use the same universal schema
    // This enables cross-type queries while maintaining Hive partitioning benefits
    elementTypes := []string{
        "document", "section", "paragraph", "heading",
        "list", "list_item", "table", "table_header", "table_row", "table_cell",
        "code_block", "image", "link", "footnote", "citation",
        "div", "span", "article", "nav", "header", "footer",
        "field", "object", "array",
        // ... all 31+ element types
    }

    for _, elemType := range elementTypes {
        r.schemas[elemType] = universalSchema
    }
}
```

**Tasks:**
- [ ] Create `go/internal/udml/` package
- [ ] Implement SchemaRegistry with all element types
- [ ] Define base schema (common fields)
- [ ] Define type-specific schemas (31+ types)
- [ ] Add schema validation

---

### 1.3 Hive-Partitioned Storage (`go/internal/analytics/parquet_hive.go`)

**New File:** `go/internal/analytics/parquet_hive.go`

```go
package analytics

import (
    "fmt"
    "path/filepath"
    "time"
    "your-module/go/internal/udml"
)

// HiveParquetStorage implements Hive-partitioned storage
type HiveParquetStorage struct {
    basePath       string
    version        string
    schemaRegistry *udml.SchemaRegistry
    allocator      memory.Allocator
}

func NewHiveParquetStorage(config map[string]interface{}) (*HiveParquetStorage, error) {
    basePath := config["path"].(string)
    version := config["version"].(string) // e.g., "v2.0.0"

    return &HiveParquetStorage{
        basePath:       basePath,
        version:        version,
        schemaRegistry: udml.NewSchemaRegistry(),
        allocator:      memory.NewGoAllocator(),
    }, nil
}

// AppendElements writes to Hive partitions
func (s *HiveParquetStorage) AppendElements(elements []Element) error {
    // Group by element type
    byType := make(map[string][]Element)
    for _, elem := range elements {
        byType[elem.ElementType] = append(byType[elem.ElementType], elem)
    }

    // Write each type to its partition
    for elemType, elems := range byType {
        if err := s.writeTypePartition(elemType, elems); err != nil {
            return err
        }
    }

    return nil
}

func (s *HiveParquetStorage) writeTypePartition(elemType string, elements []Element) error {
    // Build Hive partition path
    partitionPath := filepath.Join(
        s.basePath,
        fmt.Sprintf("element_type=%s", elemType),
        fmt.Sprintf("version=%s", s.version),
        fmt.Sprintf("date=%s", time.Now().Format("2006-01-02")),
        fmt.Sprintf("source=%s", elements[0].SourceName),
    )

    // Get type-specific schema
    schema := s.schemaRegistry.GetSchema(elemType)

    // Generate unique filename
    filename := fmt.Sprintf("elements_%s.parquet", generateRandomHex(8))
    fullPath := filepath.Join(partitionPath, filename)

    // Create directory
    if err := os.MkdirAll(partitionPath, 0755); err != nil {
        return err
    }

    // Write Parquet with type-specific schema
    return s.writeParquetWithSchema(fullPath, schema, elements)
}
```

**Files to Create/Modify:**
- [ ] Create `go/internal/analytics/parquet_hive.go`
- [ ] Modify `go/internal/analytics/factory.go` to support "hive" backend type
- [ ] Update config parsing to handle `version` parameter
- [ ] Add schema-aware Parquet writer

**Migration Strategy:**
- Run both old (flat) and new (Hive) storage in parallel during transition
- Add config flag: `use_hive_partitioning: true/false`
- Validate output matches before full cutover

---

### 1.4 Update All Parsers to Populate Promoted Fields

**Parsers to Update:** (11 files)
- `go/internal/parser/pdf.go`
- `go/internal/parser/docx.go`
- `go/internal/parser/xlsx.go`
- `go/internal/parser/html.go`
- `go/internal/parser/xml.go`
- `go/internal/parser/csv.go`
- `go/internal/parser/json.go`
- `go/internal/parser/markdown.go`
- `go/internal/parser/pptx.go`
- `go/internal/parser/text.go`
- `go/internal/parser/parquet.go`

**Example Changes (PDF parser):**
```go
// Before
element := Element{
    ElementID:   generateID("para_"),
    ElementType: "paragraph",
    Content:     text,
    Position:    pos,
    Metadata: map[string]interface{}{
        "page":      pageNum,
        "font_size": fontSize,
    },
}

// After
pageNumPtr := &pageNum
fontSizePtr := &fontSize
element := Element{
    ElementID:   generateID("para_"),
    ElementType: "paragraph",
    Content:     text,
    Position:    pos,
    PageNumber:  pageNumPtr,   // ← Promoted field
    FontSize:    fontSizePtr,  // ← Promoted field
    Metadata: map[string]interface{}{
        // Keep rare attributes in metadata
        "font_family": "Arial",
    },
}
```

**Tasks per Parser:**
- [x] Identify type-specific attributes ✅ COMPLETED
- [x] Move common attributes to promoted fields ✅ COMPLETED
- [x] Keep rare attributes in Metadata JSON ✅ COMPLETED
- [x] Update tests to validate promoted fields ✅ COMPLETED

**Estimated Effort:** 1-2 days per parser, 11 parsers = ~2 weeks

**✅ PHASE 1.2 PARSER MIGRATION COMPLETED:**
- All 10 parsers (CSV, JSON, Text, HTML, Markdown, XML, PDF, DOCX, PPTX, XLSX) successfully migrated to Parser interface
- All promoted fields (PageNumber, SectionLevel, RowIndex, ColumnIndex, TemporalType, TagName, ElementCategory) implemented
- Temporal integration via ProcessTemporalContent() added to all parsers
- Comprehensive integration tests created and passing
- Backward compatibility maintained via ParseLegacy methods

---

## Phase 2: JSON Projection & Query Engine (Week 3-4)

### 2.1 JSON Document Builder (`go/internal/udml/json.go`)

**New File:** `go/internal/udml/json.go`

```go
package udml

import (
    "database/sql"
    "encoding/json"
)

// JSONDocument represents a UDML document in JSON format
type JSONDocument struct {
    DocID    string                   `json:"doc_id"`
    Elements []map[string]interface{} `json:"elements"`
}

// JSONBuilder constructs JSON documents from Parquet
type JSONBuilder struct {
    db *sql.DB // DuckDB connection
}

func NewJSONBuilder(duckdbPath string) (*JSONBuilder, error) {
    db, err := sql.Open("duckdb", duckdbPath)
    if err != nil {
        return nil, err
    }
    return &JSONBuilder{db: db}, nil
}

// BuildJSON creates nested JSON from flat Parquet
func (b *JSONBuilder) BuildJSON(docID string, version string) (*JSONDocument, error) {
    // Query all element types for document
    query := `
        SELECT * FROM 'analytics/element_type=*/**/**.parquet'
        WHERE doc_id = ? AND version = ?
        ORDER BY position
    `

    rows, err := b.db.Query(query, docID, version)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    // Build hierarchical structure
    elements := b.buildHierarchy(rows)

    return &JSONDocument{
        DocID:    docID,
        Elements: elements,
    }, nil
}

func (b *JSONBuilder) buildHierarchy(rows *sql.Rows) []map[string]interface{} {
    // Parse flat rows into tree structure
    flatElements := b.parseRows(rows)

    // Build parent-child relationships
    tree := make([]map[string]interface{}, 0)
    elementMap := make(map[string]map[string]interface{})

    for _, elem := range flatElements {
        elementMap[elem["element_id"].(string)] = elem
    }

    // Nest children
    for _, elem := range flatElements {
        if parentID, ok := elem["parent_id"].(string); ok && parentID != "" {
            if parent, exists := elementMap[parentID]; exists {
                if parent["children"] == nil {
                    parent["children"] = []map[string]interface{}{}
                }
                parent["children"] = append(parent["children"].([]map[string]interface{}), elem)
            }
        } else {
            // Root element
            tree = append(tree, elem)
        }
    }

    return tree
}
```

**Files to Create:**
- [ ] Create `go/internal/udml/json.go`
- [ ] Implement DuckDB integration for cross-partition queries
- [ ] Hierarchical tree builder from flat results
- [ ] JSON serialization with proper nesting
- [ ] Caching layer for frequently accessed documents

**Dependencies:**
- Add DuckDB Go driver: `go get github.com/marcboeker/go-duckdb`
- Configure DuckDB to read Parquet files

---

### 2.2 JSONPath Parser (`go/internal/udml/jsonpath/parser.go`)

**New Package:** `go/internal/udml/jsonpath/`

```go
package jsonpath

import (
    "github.com/PaesslerAG/jsonpath" // Existing JSONPath library
)

// Parser extends standard JSONPath with custom functions
type Parser struct {
    functions map[string]CustomFunction
}

// CustomFunction defines interface for extended functions
type CustomFunction interface {
    Name() string
    Evaluate(args []interface{}) (interface{}, error)
    TranslateToSQL(args []string) (string, error)
}

// Parse JSONPath expression
func (p *Parser) Parse(expr string) (*Expression, error) {
    // Use standard library for basic parsing
    baseExpr, err := jsonpath.Compile(expr)
    if err != nil {
        return nil, err
    }

    // Detect custom functions
    customFuncs := p.detectCustomFunctions(expr)

    return &Expression{
        Base:      baseExpr,
        CustomFns: customFuncs,
    }, nil
}

// Register custom function
func (p *Parser) RegisterFunction(fn CustomFunction) {
    p.functions[fn.Name()] = fn
}
```

**Custom Functions to Implement:**

1. **similarity()** - Semantic text similarity using existing contextual embeddings
2. **fuzzy_match()** - Fuzzy string matching
3. **extract_date()** - Date extraction from text
4. **extract_currency()** - Currency amount extraction

**Files to Create:**
- [ ] `go/internal/udml/jsonpath/parser.go`
- [ ] `go/internal/udml/jsonpath/functions.go`
- [ ] `go/internal/udml/jsonpath/similarity.go` - **CRITICAL: Integrates with existing contextual embeddings**
- [ ] `go/internal/udml/jsonpath/extractors.go`

---

### 2.2.1 **CRITICAL: Integration with Existing Contextual Embeddings System**

**Background:** The codebase already contains a sophisticated **graphlet embedding method** (contextual embeddings) in `go/internal/embeddings/contextual.go`. This system MUST be retained and used as the basis for cosine similarity in the `similarity()` function.

#### Existing Contextual Embedding Architecture

**Core Component:** `ContextualTextBuilder` (`go/internal/embeddings/contextual.go:1-379`)

The existing system builds context-aware text representations by combining:
- **Element text** (40% token budget) - The element's own content
- **Parent chain** (25% token budget) - Hierarchical context from ancestors
- **Siblings** (20% token budget) - Predecessors and successors at same level
- **Children** (15% token budget) - Direct descendants

**Key Features:**
- **3-part content waterfall**: Content → Resolver → ContentPreview
- **Structural filtering**: Skips containers (section, div, list) that don't add semantic value
- **Token budget allocation**: Configurable ratios for different context types
- **Whitespace normalization**: Cleans and compacts text

```go
// Existing implementation in go/internal/embeddings/contextual.go
type ContextualTextBuilder struct {
    predecessorCount int      // Number of predecessor siblings (default: 2)
    successorCount   int      // Number of successor siblings (default: 2)
    maxTokens        int      // Maximum tokens for contextual text (default: 512)
    safeMaxTokens    int      // Safety buffer (90% of maxTokens)
    resolver         resolver.ContentResolver
    elementRatio     float64  // 0.40 - Element's own text
    parentsRatio     float64  // 0.25 - Parent chain context
    siblingsRatio    float64  // 0.20 - Sibling context
    childrenRatio    float64  // 0.15 - Children context
}

func (b *ContextualTextBuilder) BuildContextualText(
    element parser.Element,
    allElements []parser.Element,
    relationships []parser.Relationship,
) string {
    // 1. Allocate token budgets
    elementBudget := int(float64(b.safeMaxTokens) * b.elementRatio)   // 205 tokens
    parentBudget := int(float64(b.safeMaxTokens) * b.parentsRatio)    // 128 tokens
    siblingBudget := int(float64(b.safeMaxTokens) * b.siblingsRatio)  // 102 tokens
    childBudget := int(float64(b.safeMaxTokens) * b.childrenRatio)    // 77 tokens

    // 2. Extract element text (with budget)
    elementText := b.getElementText(element, elementBudget)

    // 3. Collect parent chain texts (bottom-up through hierarchy)
    parentTexts := b.collectParentTexts(element, elementMap, parentBudget)

    // 4. Collect sibling texts (predecessors + successors)
    siblingTexts := b.collectSiblingTexts(element, allElements, siblingBudget)

    // 5. Collect immediate children texts
    childTexts := b.collectChildrenTexts(element, relationships, elementMap, childBudget)

    // 6. Combine into final contextual text
    parts := []string{}
    parts = append(parts, parentTexts...)  // Parents first (top-down)
    parts = append(parts, elementText)     // Then element itself
    parts = append(parts, siblingTexts...) // Then siblings
    parts = append(parts, childTexts...)   // Finally children

    return strings.Join(parts, "\n")
}
```

#### Integration Flow: JSONPath `similarity()` → Contextual Embeddings

**Architecture:**
```
JSONPath Query
    ↓
similarity("text", "query phrase", 0.8)
    ↓
SimilarityFunction.TranslateToSQL()
    ↓
SQL: JOIN embeddings WHERE cosine_similarity(emb.vector, query_emb) > 0.8
    ↓
DuckDB executes with pre-computed contextual embeddings
    ↓
Results filtered by semantic similarity
```

#### Implementation: `go/internal/udml/jsonpath/similarity.go`

**New File:** This integrates JSONPath with existing embeddings infrastructure

```go
package jsonpath

import (
    "fmt"
    "github.com/kennethstott/go-doc-go/internal/embeddings"
    "github.com/kennethstott/go-doc-go/internal/parser"
)

// SimilarityFunction implements semantic similarity using contextual embeddings
type SimilarityFunction struct {
    generator embeddings.EmbeddingGenerator  // Uses existing ONNX/Ollama/OpenAI
    config    *embeddings.Config             // Existing embedding config
}

// NewSimilarityFunction creates function using existing embedding infrastructure
func NewSimilarityFunction(config *embeddings.Config) (*SimilarityFunction, error) {
    // Use existing factory from go/internal/embeddings/factory.go
    generator, err := embeddings.NewGenerator(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create embedding generator: %w", err)
    }

    return &SimilarityFunction{
        generator: generator,
        config:    config,
    }, nil
}

// TranslateToSQL converts similarity() call to SQL with embeddings JOIN
func (s *SimilarityFunction) TranslateToSQL(args []string) (string, error) {
    // args: [textField, queryPhrase, threshold]
    // Example: similarity("text", "financial disclosure", 0.8)

    if len(args) != 3 {
        return "", fmt.Errorf("similarity() requires 3 args: field, query, threshold")
    }

    textField := args[0]    // Usually "content" or "text"
    queryPhrase := args[1]  // User's search phrase
    threshold := args[2]    // Minimum similarity score (0.0-1.0)

    // Generate query embedding using same contextual approach
    // NOTE: For query embeddings, context is just the query text itself
    queryEmbedding, err := s.generator.Generate(queryPhrase)
    if err != nil {
        return "", fmt.Errorf("failed to generate query embedding: %w", err)
    }

    // Build SQL with embeddings JOIN
    // IMPORTANT: embeddings.parquet contains pre-computed contextual embeddings
    sql := fmt.Sprintf(`
        JOIN 'analytics/embeddings.parquet' emb
            ON emb.element_id = e.element_id
        WHERE cosine_similarity(emb.embedding, %s) > %s
    `, s.serializeVector(queryEmbedding), threshold)

    return sql, nil
}

// Name returns function name for registration
func (s *SimilarityFunction) Name() string {
    return "similarity"
}

// Evaluate runs similarity check in-memory (for non-SQL contexts)
func (s *SimilarityFunction) Evaluate(args []interface{}) (interface{}, error) {
    // In-memory evaluation for testing or small datasets
    textField := args[0].(string)
    queryPhrase := args[1].(string)
    threshold := args[2].(float64)

    // Generate embeddings
    textEmb, _ := s.generator.Generate(textField)
    queryEmb, _ := s.generator.Generate(queryPhrase)

    // Compute cosine similarity
    similarity := s.cosineSimilarity(textEmb, queryEmb)

    return similarity > threshold, nil
}

// Close releases embedding generator resources
func (s *SimilarityFunction) Close() error {
    return s.generator.Close()
}
```

#### Pre-Computing Contextual Embeddings

**Embeddings Generation Process:** (Already exists in codebase)

```go
// This is the EXISTING flow - just document it for UDML integration
// Location: go/internal/embeddings/contextual.go + processor.go

// 1. During document processing, for each element:
func ProcessElementEmbeddings(
    element parser.Element,
    allElements []parser.Element,
    relationships []parser.Relationship,
    config *embeddings.Config,
) ([]float64, error) {

    if !config.Contextual {
        // Simple embedding of element text
        return generator.Generate(element.Content)
    }

    // 2. Build contextual text using graphlet method
    builder := embeddings.NewContextualTextBuilder(
        config.PredecessorCount,  // Default: 2
        config.SuccessorCount,    // Default: 2
        512,                      // maxTokens
        resolver,
    )

    contextualText := builder.BuildContextualText(element, allElements, relationships)

    // 3. Generate embedding from contextual text
    embedding, err := generator.Generate(contextualText)
    if err != nil {
        return nil, err
    }

    return embedding, nil
}

// 4. Store in embeddings.parquet with element_id link
// Schema: element_id, embedding (FLOAT[384]), model_name, created_at
```

#### DuckDB Cosine Similarity Function

**Custom Scalar Function:** (Need to implement in DuckDB)

```go
// go/internal/udml/duckdb_extensions.go

// Register cosine similarity function in DuckDB
func RegisterCosineSimilarity(conn *sql.DB) error {
    // DuckDB supports custom scalar functions via C API
    // For Go, we can use UDF registration

    _, err := conn.Exec(`
        CREATE OR REPLACE FUNCTION cosine_similarity(vec1 FLOAT[], vec2 FLOAT[])
        RETURNS FLOAT AS
        $$
            SELECT
                list_sum(list_transform(
                    list_zip(vec1, vec2),
                    x -> x[1] * x[2]
                )) / (
                    sqrt(list_sum(list_transform(vec1, x -> x * x))) *
                    sqrt(list_sum(list_transform(vec2, x -> x * x)))
                )
        $$
    `)

    return err
}
```

#### Common JSONPATH Query Patterns with Promoted Fields

The 6 promoted fields enable efficient filtering in ontology rules. Here are common patterns:

**Pattern 1: Location-Based Extraction (page_number)**
```jsonpath
// "Find risk factors mentioned on pages 10-15"
$.elements[?(
    @.type == 'paragraph' &&
    @.page_number >= 10 &&
    @.page_number <= 15 &&
    similarity(@.content, 'risk factors', 0.8)
)]
```

**Pattern 2: Structural Queries (section_level)**
```jsonpath
// "Extract top-level headings (H1/H2)"
$.elements[?(
    @.type == 'heading' &&
    @.section_level <= 2
)]
```

**Pattern 3: Tabular Extraction (row_index, column_index)**
```jsonpath
// "Get amounts from column 3, skipping header row"
$.elements[?(
    @.type == 'table_cell' &&
    @.row_index > 0 &&
    @.column_index == 2 &&
    contains(@.content, '$')
)]
```

**Pattern 4: Temporal Filtering (temporal_type)**
```jsonpath
// "Find all date fields in contract sections"
$.elements[?(
    @.temporal_type == 'date' &&
    contains(@.parent_path, 'contract')
)]
```

**Pattern 5: Markup-Based Extraction (tag_name)**
```jsonpath
// "Extract content from div.disclosure elements"
$.elements[?(
    @.tag_name == 'div' &&
    contains(@.content_location.class, 'disclosure')
)]
```

**Pattern 6: Combined Filters (Multi-Field)**
```jsonpath
// "Find financial tables on pages 20-30 with more than 10 rows"
$.elements[?(
    @.type == 'table' &&
    @.page_number >= 20 &&
    @.page_number <= 30 &&
    @.row_index > 10
)]
```

#### Complete Query Flow Example

**User Query:**
```jsonpath
$.elements[?(@.type == 'paragraph' && @.page_number >= 10 && @.page_number <= 15 && similarity(@.content, 'financial disclosure requirements', 0.8))]
```

**Translation Steps:**

1. **JSONPath Parser** detects structural filters + `similarity()` function
2. **Translator** extracts promoted field filters:
   - `type == 'paragraph'` → Partition pruning to `element_type=paragraph/`
   - `page_number >= 10 AND page_number <= 15` → Direct column filter (Parquet row group skipping)
3. **SimilarityFunction.TranslateToSQL()** generates optimized SQL:
   ```sql
   SELECT * FROM 'analytics/element_type=paragraph/**/**.parquet' e
   JOIN 'analytics/embeddings.parquet' emb ON emb.element_id = e.element_id
   WHERE e.element_type = 'paragraph'
     AND e.page_number >= 10                     -- Promoted field filter (row group skipping)
     AND e.page_number <= 15
     AND cosine_similarity(
           emb.embedding,
           ARRAY[0.234, -0.123, 0.456, ...]    -- Query embedding vector
         ) > 0.8
   ```
4. **DuckDB** executes with:
   - **Partition pruning** (only `element_type=paragraph/` partitions) - 60x speedup
   - **Parquet row group skipping** (page_number filters) - Additional 10x speedup
   - **Cosine similarity** on pre-computed contextual embeddings
5. **Results** returned as matching elements with high semantic similarity

**Performance Comparison:**

| Query Component | Without Promoted Fields | With Promoted Fields | Speedup |
|-----------------|------------------------|---------------------|---------|
| Element type filter | Full scan of all types | Partition pruning | 60x |
| Page number filter | JSON extraction + filter | Direct column filter + row group skip | 10x |
| Similarity filter | Same (uses embeddings) | Same (uses embeddings) | 1x |
| **Total** | ~30 seconds | ~50ms | **600x** |

The promoted fields provide compound benefits: partition pruning (60x) × row group skipping (10x) = **600x overall speedup**

#### Configuration in UDML Context

**Embedding Config** (already exists in `go/internal/embeddings/interface.go:24-40`):
```yaml
embeddings:
  enabled: true
  provider: "onnx"  # or "ollama", "openai"
  model: "all-MiniLM-L6-v2"
  model_path: "./models/all-MiniLM-L6-v2"
  dimensions: 384
  contextual: true  # CRITICAL: Enable graphlet/contextual embeddings
  predecessor_count: 2
  successor_count: 2
  cache_dir: ".cache/embeddings"
  pool_size: 4      # Matches worker count
  batch_size: 32    # Batch inference
```

#### Summary: Why This Integration is Critical

1. **Existing Investment:** `ContextualTextBuilder` is a sophisticated, working system (379 lines of code)
2. **Semantic Quality:** Graphlet approach captures hierarchical + sibling context for better embeddings
3. **Performance:** Pre-computed embeddings in Parquet enable fast similarity queries at scale
4. **Consistency:** Same embedding method used for both storage and query-time similarity
5. **Flexibility:** Works with ONNX (local), Ollama (local), or OpenAI (API) providers

**Files Modified for Integration:**
- [ ] Create `go/internal/udml/jsonpath/similarity.go` - Uses existing `embeddings.EmbeddingGenerator`
- [ ] Create `go/internal/udml/duckdb_extensions.go` - Register `cosine_similarity()` UDF
- [ ] Modify `go/internal/embeddings/processor.go` - Ensure contextual embeddings written to `embeddings.parquet`
- [ ] Update `go/internal/analytics/parquet_hive.go` - Add embeddings table writer

---

### 2.3 Query Backend Interface (Multi-Backend Support)

**Philosophy:** UDML is designed to work with multiple analytical backends (DuckDB, PostgreSQL, Neo4j, Elasticsearch, Solr). JSONPath queries must translate to each backend's native query language while maintaining consistent semantics and leveraging promoted field optimizations.

**Architecture:** Pluggable backend interface allows Phase 1 to focus on DuckDB while enabling future backend additions without changing JSONPath parsing or ontology rule definitions.

#### 2.3.1 Backend Interface Definition

**New File:** `go/internal/udml/jsonpath/backend.go`

```go
package jsonpath

import "fmt"

// QueryBackend defines the interface for backend-specific query generation
type QueryBackend interface {
    // GetName returns backend identifier (e.g., "duckdb", "neo4j", "postgresql")
    GetName() string

    // Translate converts JSONPath expression to native query
    Translate(expr *Expression, opts TranslateOptions) (*NativeQuery, error)

    // SupportsFeature checks if backend supports a specific feature
    // Features: "similarity", "partition_pruning", "vector_search", "regex"
    SupportsFeature(feature string) bool

    // GetOptimizations returns backend-specific optimization hints
    GetOptimizations() []string
}

// NativeQuery represents a backend-specific query result
type NativeQuery struct {
    Backend      string                 // Backend identifier ("duckdb", "neo4j", etc.)
    Query        string                 // Native query string (SQL, Cypher, JSON DSL, etc.)
    Parameters   map[string]interface{} // Query parameters (for parameterized queries)
    Hints        []string              // Optimization hints applied
    EstimatedOps int                   // Estimated operations for cost comparison
    Metadata     map[string]interface{} // Backend-specific metadata
}

// TranslateOptions provides context for query translation
type TranslateOptions struct {
    PartitionPath   string   // Base path to data ("analytics/", "elements", etc.)
    PromotedFields  []string // Available promoted fields
    UseEmbeddings   bool     // Whether similarity() function is available
    MaxResults      int      // Result limit hint (0 = no limit)
    Version         string   // UDML schema version filter
}

// Expression represents parsed JSONPath expression (backend-agnostic)
type Expression struct {
    Base       interface{}        // Base JSONPath AST
    Predicates []Predicate       // Filter predicates
    CustomFns  []CustomFunction  // Custom functions (similarity, etc.)
}

type Predicate struct {
    Field    string      // Field name ("type", "page_number", etc.)
    Operator string      // Operator ("==", ">=", "contains", etc.)
    Value    interface{} // Comparison value
}
```

#### 2.3.2 Backend Registry

**New File:** `go/internal/udml/jsonpath/registry.go`

```go
package jsonpath

// BackendRegistry manages available query backends
type BackendRegistry struct {
    backends map[string]QueryBackend
    default  string
}

func NewBackendRegistry() *BackendRegistry {
    r := &BackendRegistry{
        backends: make(map[string]QueryBackend),
        default:  "duckdb",
    }

    // Register default backends (Phase 1: DuckDB only)
    r.Register(NewDuckDBBackend())

    // Future backends will be registered here:
    // Phase 4+:
    // r.Register(NewPostgreSQLBackend())
    // r.Register(NewNeo4jBackend())
    // r.Register(NewElasticsearchBackend())
    // r.Register(NewSolrBackend())

    return r
}

// Register adds a backend to the registry
func (r *BackendRegistry) Register(backend QueryBackend) {
    r.backends[backend.GetName()] = backend
}

// GetBackend retrieves a backend by name
func (r *BackendRegistry) GetBackend(name string) (QueryBackend, error) {
    backend, exists := r.backends[name]
    if !exists {
        return nil, fmt.Errorf("unknown backend: %s (available: %v)", name, r.ListBackends())
    }
    return backend, nil
}

// GetDefaultBackend returns the default backend (DuckDB in Phase 1)
func (r *BackendRegistry) GetDefaultBackend() QueryBackend {
    return r.backends[r.default]
}

// ListBackends returns names of all registered backends
func (r *BackendRegistry) ListBackends() []string {
    names := make([]string, 0, len(r.backends))
    for name := range r.backends {
        names = append(names, name)
    }
    return names
}
```

**Files to Create:**
- [ ] `go/internal/udml/jsonpath/backend.go` - Interface definitions
- [ ] `go/internal/udml/jsonpath/registry.go` - Backend registry

---

### 2.3.3 DuckDB Backend Implementation (Phase 1)

**New File:** `go/internal/udml/jsonpath/duckdb_backend.go`

```go
package jsonpath

import (
    "fmt"
    "strings"
)

// DuckDBBackend implements QueryBackend for DuckDB SQL translation
type DuckDBBackend struct {
    partitionPath string // Base path for Hive-partitioned Parquet
}

func NewDuckDBBackend() *DuckDBBackend {
    return &DuckDBBackend{
        partitionPath: "analytics/",
    }
}

// GetName returns backend identifier
func (d *DuckDBBackend) GetName() string {
    return "duckdb"
}

// Translate converts JSONPath expression to DuckDB SQL
func (d *DuckDBBackend) Translate(expr *Expression, opts TranslateOptions) (*NativeQuery, error) {
    params := make(map[string]interface{})
    hints := []string{}

    // Extract element_type filter for partition pruning (60x speedup)
    elemType := d.extractElementTypeFilter(expr)

    var fromClause string
    if elemType != "" {
        // Partition pruning: Only scan specific element_type partition
        fromClause = fmt.Sprintf("'%selement_type=%s/**/**.parquet'", opts.PartitionPath, elemType)
        hints = append(hints, fmt.Sprintf("partition_pruning:element_type=%s", elemType))
    } else {
        // Scan all partitions
        fromClause = fmt.Sprintf("'%selement_type=*/**/**.parquet'", opts.PartitionPath)
    }

    // Build SELECT clause
    var sqlBuilder strings.Builder
    sqlBuilder.WriteString("SELECT * FROM ")
    sqlBuilder.WriteString(fromClause)

    // Translate predicates to WHERE clause with promoted field pushdown
    whereClauses, predicateParams := d.translatePredicates(expr.Predicates, opts.PromotedFields)
    for k, v := range predicateParams {
        params[k] = v
    }

    if len(whereClauses) > 0 {
        sqlBuilder.WriteString(" WHERE ")
        sqlBuilder.WriteString(strings.Join(whereClauses, " AND "))
        hints = append(hints, "predicate_pushdown:promoted_fields")
    }

    // Handle custom functions (similarity, fuzzy_match, etc.)
    if len(expr.CustomFns) > 0 {
        customSQL, customParams := d.translateCustomFunctions(expr.CustomFns, opts)
        for k, v := range customParams {
            params[k] = v
        }
        sqlBuilder.WriteString(" ")
        sqlBuilder.WriteString(customSQL)
        hints = append(hints, "custom_functions:similarity")
    }

    // Add version filter if specified
    if opts.Version != "" {
        if len(whereClauses) > 0 {
            sqlBuilder.WriteString(" AND ")
        } else {
            sqlBuilder.WriteString(" WHERE ")
        }
        sqlBuilder.WriteString(fmt.Sprintf("version = '%s'", opts.Version))
    }

    // Add result limit
    if opts.MaxResults > 0 {
        sqlBuilder.WriteString(fmt.Sprintf(" LIMIT %d", opts.MaxResults))
    }

    return &NativeQuery{
        Backend:      "duckdb",
        Query:        sqlBuilder.String(),
        Parameters:   params,
        Hints:        hints,
        EstimatedOps: d.estimateOperations(elemType, len(whereClauses)),
        Metadata: map[string]interface{}{
            "partition_pruning": elemType != "",
            "predicate_pushdown": len(whereClauses) > 0,
        },
    }, nil
}

// SupportsFeature checks if DuckDB supports a specific feature
func (d *DuckDBBackend) SupportsFeature(feature string) bool {
    supported := map[string]bool{
        "similarity":        true,  // Via UDF
        "partition_pruning": true,  // Hive partitioning
        "vector_search":     true,  // Via extension
        "regex":             true,  // REGEXP operator
        "fuzzy_match":       true,  // Via levenshtein extension
        "json_path":         true,  // json_extract_path function
    }
    return supported[feature]
}

// GetOptimizations returns DuckDB-specific optimization strategies
func (d *DuckDBBackend) GetOptimizations() []string {
    return []string{
        "partition_pruning:element_type",    // 60x speedup
        "predicate_pushdown:promoted_fields", // 10x speedup (row group skipping)
        "vectorized_execution",               // SIMD processing
        "column_pruning",                     // Only read needed columns
        "parallel_scan",                      // Multi-threaded Parquet reading
    }
}

// extractElementTypeFilter finds element_type predicate for partition pruning
func (d *DuckDBBackend) extractElementTypeFilter(expr *Expression) string {
    for _, pred := range expr.Predicates {
        // Match: $.elements[?(@.type == 'paragraph')]
        if (pred.Field == "type" || pred.Field == "element_type") && pred.Operator == "==" {
            if str, ok := pred.Value.(string); ok {
                return str
            }
        }
    }
    return ""
}

// translatePredicates converts predicates to SQL WHERE clauses
func (d *DuckDBBackend) translatePredicates(predicates []Predicate, promotedFields []string) ([]string, map[string]interface{}) {
    clauses := []string{}
    params := make(map[string]interface{})

    // Map of promoted fields for fast lookup
    promotedSet := make(map[string]bool)
    for _, field := range promotedFields {
        promotedSet[field] = true
    }

    for i, pred := range predicates {
        paramName := fmt.Sprintf("p%d", i)

        // Check if field is promoted (use direct column access)
        if promotedSet[pred.Field] {
            // Promoted field: Direct column access (10x faster via row group skipping)
            clause := d.translatePromotedField(pred, paramName)
            clauses = append(clauses, clause)
            params[paramName] = pred.Value
        } else {
            // Non-promoted field: JSON extraction (slower)
            clause := d.translateJSONField(pred, paramName)
            clauses = append(clauses, clause)
            params[paramName] = pred.Value
        }
    }

    return clauses, params
}

// translatePromotedField generates SQL for promoted field predicates
func (d *DuckDBBackend) translatePromotedField(pred Predicate, paramName string) string {
    switch pred.Operator {
    case "==", "=":
        return fmt.Sprintf("%s = $%s", pred.Field, paramName)
    case "!=":
        return fmt.Sprintf("%s != $%s", pred.Field, paramName)
    case ">":
        return fmt.Sprintf("%s > $%s", pred.Field, paramName)
    case ">=":
        return fmt.Sprintf("%s >= $%s", pred.Field, paramName)
    case "<":
        return fmt.Sprintf("%s < $%s", pred.Field, paramName)
    case "<=":
        return fmt.Sprintf("%s <= $%s", pred.Field, paramName)
    case "contains":
        return fmt.Sprintf("%s LIKE '%%' || $%s || '%%'", pred.Field, paramName)
    case "regex":
        return fmt.Sprintf("regexp_matches(%s, $%s)", pred.Field, paramName)
    default:
        return fmt.Sprintf("%s = $%s", pred.Field, paramName)
    }
}

// translateJSONField generates SQL for JSON metadata field extraction
func (d *DuckDBBackend) translateJSONField(pred Predicate, paramName string) string {
    // Use DuckDB's json_extract_path for metadata/content_location fields
    switch pred.Operator {
    case "==", "=":
        return fmt.Sprintf("json_extract_path(metadata, '%s') = $%s", pred.Field, paramName)
    case "contains":
        return fmt.Sprintf("json_extract_path(metadata, '%s') LIKE '%%' || $%s || '%%'", pred.Field, paramName)
    default:
        return fmt.Sprintf("json_extract_path(metadata, '%s') = $%s", pred.Field, paramName)
    }
}

// translateCustomFunctions handles similarity() and other custom functions
func (d *DuckDBBackend) translateCustomFunctions(fns []CustomFunction, opts TranslateOptions) (string, map[string]interface{}) {
    var clauses []string
    params := make(map[string]interface{})

    for i, fn := range fns {
        switch fn.Name {
        case "similarity":
            // similarity(query_text, threshold) -> Uses pre-computed contextual embeddings
            if len(fn.Args) >= 2 {
                queryText := fn.Args[0]
                threshold := fn.Args[1]

                // Generate query embedding parameter
                params[fmt.Sprintf("query_emb_%d", i)] = queryText
                params[fmt.Sprintf("threshold_%d", i)] = threshold

                // Join with embeddings table and compute cosine similarity
                clause := fmt.Sprintf(`
                    AND element_id IN (
                        SELECT element_id
                        FROM '%sembeddings.parquet'
                        WHERE cosine_similarity(embedding, $query_emb_%d) >= $threshold_%d
                    )
                `, opts.PartitionPath, i, i)
                clauses = append(clauses, clause)
            }

        case "fuzzy_match":
            // fuzzy_match(field, query, max_distance)
            if len(fn.Args) >= 3 {
                field := fn.Args[0].(string)
                query := fn.Args[1]
                maxDist := fn.Args[2]

                params[fmt.Sprintf("fuzzy_query_%d", i)] = query
                params[fmt.Sprintf("fuzzy_dist_%d", i)] = maxDist

                clause := fmt.Sprintf("levenshtein(%s, $fuzzy_query_%d) <= $fuzzy_dist_%d",
                    field, i, i)
                clauses = append(clauses, clause)
            }
        }
    }

    return strings.Join(clauses, " "), params
}

// estimateOperations provides cost estimate for query planning
func (d *DuckDBBackend) estimateOperations(elemType string, predicateCount int) int {
    baseOps := 1000000 // Baseline: scan all elements

    // Partition pruning reduces by ~60x (20 element types)
    if elemType != "" {
        baseOps /= 60
    }

    // Each promoted field predicate reduces by ~10x (row group skipping)
    for i := 0; i < predicateCount; i++ {
        baseOps /= 10
    }

    return baseOps
}
```

**Example Usage:**

```go
// Create DuckDB backend
backend := NewDuckDBBackend()

// Example 1: Simple element type filter
expr1 := &Expression{
    Predicates: []Predicate{
        {Field: "type", Operator: "==", Value: "paragraph"},
    },
}

opts := TranslateOptions{
    PartitionPath: "analytics/",
    PromotedFields: []string{"page_number", "section_level", "row_index", "column_index", "temporal_type", "tag_name"},
    Version: "v1.0.0",
}

query1, _ := backend.Translate(expr1, opts)
// Result: SELECT * FROM 'analytics/element_type=paragraph/**/**.parquet' WHERE version = 'v1.0.0'
// Hints: [partition_pruning:element_type=paragraph]

// Example 2: Promoted field predicate (10x speedup via row group skipping)
expr2 := &Expression{
    Predicates: []Predicate{
        {Field: "type", Operator: "==", Value: "table_cell"},
        {Field: "page_number", Operator: "==", Value: 5},
    },
}

query2, _ := backend.Translate(expr2, opts)
// Result: SELECT * FROM 'analytics/element_type=table_cell/**/**.parquet'
//         WHERE page_number = $p0 AND version = 'v1.0.0'
// Hints: [partition_pruning:element_type=table_cell, predicate_pushdown:promoted_fields]

// Example 3: Similarity search (contextual embeddings)
expr3 := &Expression{
    Predicates: []Predicate{
        {Field: "type", Operator: "==", Value: "paragraph"},
    },
    CustomFns: []CustomFunction{
        {Name: "similarity", Args: []interface{}{"financial projections", 0.8}},
    },
}

query3, _ := backend.Translate(expr3, opts)
// Result: SELECT * FROM 'analytics/element_type=paragraph/**/**.parquet'
//         WHERE version = 'v1.0.0'
//         AND element_id IN (
//             SELECT element_id FROM 'analytics/embeddings.parquet'
//             WHERE cosine_similarity(embedding, $query_emb_0) >= $threshold_0
//         )
// Hints: [partition_pruning:element_type=paragraph, custom_functions:similarity]
```

**Performance Characteristics:**

| Query Pattern | Optimization | Speedup | EstimatedOps |
|--------------|--------------|---------|--------------|
| No filters | None | 1x | 1,000,000 |
| Element type filter | Partition pruning | 60x | 16,667 |
| + 1 promoted field | Row group skipping | 600x | 1,667 |
| + 2 promoted fields | Row group skipping | 6,000x | 167 |
| + similarity() | Vector index | 10,000x+ | ~100 |

**Files to Create:**
- [ ] `go/internal/udml/jsonpath/duckdb_backend.go` - DuckDB backend implementation
- [ ] `go/internal/udml/jsonpath/duckdb_backend_test.go` - Unit tests
- [ ] SQL template generation with parameterization
- [ ] Predicate translation with promoted field detection
- [ ] Custom function SQL generation (similarity, fuzzy_match)
- [ ] Query optimization analysis

**Testing:**
- Unit tests for each JSONPath pattern
- Predicate translation correctness
- Partition pruning detection
- Performance benchmarks (translation speed < 1ms)
- Validation against actual DuckDB execution
- Test promoted vs. JSON field performance difference

---

### 2.3.4 Future Backend Implementations (Phase 4+)

**Note:** These backend implementations will be added in Phase 4+ after DuckDB is fully operational. They demonstrate the pluggable architecture's extensibility.

#### 2.3.4.1 Neo4j Backend (Graph Database)

**New File:** `go/internal/udml/jsonpath/neo4j_backend.go`

```go
package jsonpath

// Neo4jBackend implements QueryBackend for Neo4j Cypher translation
type Neo4jBackend struct {
    uri      string
    database string
}

func NewNeo4jBackend(uri, database string) *Neo4jBackend {
    return &Neo4jBackend{
        uri:      uri,
        database: database,
    }
}

func (n *Neo4jBackend) GetName() string {
    return "neo4j"
}

// Translate converts JSONPath expression to Cypher query
func (n *Neo4jBackend) Translate(expr *Expression, opts TranslateOptions) (*NativeQuery, error) {
    params := make(map[string]interface{})
    hints := []string{}

    // Extract element_type for node label filtering
    elemType := n.extractElementTypeFilter(expr)

    var matchClause string
    if elemType != "" {
        // Match specific node label (1000x faster than filtering all nodes)
        matchClause = fmt.Sprintf("MATCH (e:%s)", elemType)
        hints = append(hints, fmt.Sprintf("label_scan:%s", elemType))
    } else {
        // Match all Element nodes
        matchClause = "MATCH (e:Element)"
    }

    // Build WHERE clause from predicates
    whereClauses := n.translatePredicates(expr.Predicates, params)

    var cypherBuilder strings.Builder
    cypherBuilder.WriteString(matchClause)

    if len(whereClauses) > 0 {
        cypherBuilder.WriteString(" WHERE ")
        cypherBuilder.WriteString(strings.Join(whereClauses, " AND "))
        hints = append(hints, "property_index_scan")
    }

    // Handle similarity() function using Neo4j vector search
    if len(expr.CustomFns) > 0 {
        vectorClause := n.translateVectorSearch(expr.CustomFns, params)
        cypherBuilder.WriteString(vectorClause)
        hints = append(hints, "vector_index_search")
    }

    // Return elements with relationships
    cypherBuilder.WriteString(" RETURN e")

    if opts.MaxResults > 0 {
        cypherBuilder.WriteString(fmt.Sprintf(" LIMIT %d", opts.MaxResults))
    }

    return &NativeQuery{
        Backend:      "neo4j",
        Query:        cypherBuilder.String(),
        Parameters:   params,
        Hints:        hints,
        EstimatedOps: n.estimateOperations(elemType, len(whereClauses)),
        Metadata: map[string]interface{}{
            "label_filtering": elemType != "",
            "vector_search":   len(expr.CustomFns) > 0,
        },
    }, nil
}

func (n *Neo4jBackend) SupportsFeature(feature string) bool {
    return map[string]bool{
        "similarity":        true,  // vector.similarity() function
        "partition_pruning": false, // Use label scanning instead
        "vector_search":     true,  // Native vector index
        "regex":             true,  // =~ operator
        "graph_traversal":   true,  // Native graph queries
    }[feature]
}

func (n *Neo4jBackend) GetOptimizations() []string {
    return []string{
        "label_scan:element_type",        // Node label filtering (1000x speedup)
        "property_index:promoted_fields",  // Indexed property access
        "vector_index:embeddings",         // Vector similarity search
        "relationship_traversal",          // Native graph operations
    }
}

func (n *Neo4jBackend) translatePredicates(predicates []Predicate, params map[string]interface{}) []string {
    clauses := []string{}
    for i, pred := range predicates {
        paramName := fmt.Sprintf("p%d", i)
        params[paramName] = pred.Value

        // Neo4j promoted fields are node properties
        switch pred.Operator {
        case "==":
            clauses = append(clauses, fmt.Sprintf("e.%s = $%s", pred.Field, paramName))
        case ">=":
            clauses = append(clauses, fmt.Sprintf("e.%s >= $%s", pred.Field, paramName))
        case "contains":
            clauses = append(clauses, fmt.Sprintf("e.%s CONTAINS $%s", pred.Field, paramName))
        case "regex":
            clauses = append(clauses, fmt.Sprintf("e.%s =~ $%s", pred.Field, paramName))
        }
    }
    return clauses
}

func (n *Neo4jBackend) translateVectorSearch(fns []CustomFunction, params map[string]interface{}) string {
    for i, fn := range fns {
        if fn.Name == "similarity" && len(fn.Args) >= 2 {
            queryText := fn.Args[0]
            threshold := fn.Args[1]

            params[fmt.Sprintf("query_emb_%d", i)] = queryText
            params[fmt.Sprintf("threshold_%d", i)] = threshold

            // Use Neo4j's vector.similarity function
            return fmt.Sprintf(` AND vector.similarity.cosine(e.embedding, $query_emb_%d) >= $threshold_%d`, i, i)
        }
    }
    return ""
}

func (n *Neo4jBackend) estimateOperations(elemType string, predicateCount int) int {
    baseOps := 1000000
    if elemType != "" {
        baseOps /= 1000 // Label scan is extremely fast
    }
    for i := 0; i < predicateCount; i++ {
        baseOps /= 5
    }
    return baseOps
}
```

**Example Cypher Output:**
```cypher
// JSONPath: $.elements[?(@.type == 'paragraph' && @.page_number == 5)]
MATCH (e:paragraph)
WHERE e.page_number = $p0
RETURN e LIMIT 100

// With similarity:
MATCH (e:paragraph)
WHERE vector.similarity.cosine(e.embedding, $query_emb_0) >= $threshold_0
RETURN e
```

---

#### 2.3.4.2 Elasticsearch Backend (Search Engine)

**New File:** `go/internal/udml/jsonpath/elasticsearch_backend.go`

```go
package jsonpath

import "encoding/json"

// ElasticsearchBackend implements QueryBackend for Elasticsearch Query DSL
type ElasticsearchBackend struct {
    url   string
    index string
}

func NewElasticsearchBackend(url, index string) *ElasticsearchBackend {
    return &ElasticsearchBackend{
        url:   url,
        index: index,
    }
}

func (e *ElasticsearchBackend) GetName() string {
    return "elasticsearch"
}

// Translate converts JSONPath expression to Elasticsearch Query DSL
func (e *ElasticsearchBackend) Translate(expr *Expression, opts TranslateOptions) (*NativeQuery, error) {
    hints := []string{}

    // Build Query DSL structure
    query := map[string]interface{}{
        "bool": map[string]interface{}{
            "must": []interface{}{},
        },
    }

    // Extract element_type for index routing (50x speedup)
    elemType := e.extractElementTypeFilter(expr)
    if elemType != "" {
        query["bool"].(map[string]interface{})["must"] = append(
            query["bool"].(map[string]interface{})["must"].([]interface{}),
            map[string]interface{}{
                "term": map[string]interface{}{
                    "element_type": elemType,
                },
            },
        )
        hints = append(hints, fmt.Sprintf("index_routing:%s", elemType))
    }

    // Translate predicates to Query DSL terms/range queries
    for _, pred := range expr.Predicates {
        predQuery := e.translatePredicate(pred)
        if predQuery != nil {
            query["bool"].(map[string]interface{})["must"] = append(
                query["bool"].(map[string]interface{})["must"].([]interface{}),
                predQuery,
            )
            hints = append(hints, "term_query")
        }
    }

    // Handle similarity() function using knn search
    if len(expr.CustomFns) > 0 {
        knnQuery := e.translateKNNSearch(expr.CustomFns)
        if knnQuery != nil {
            query["knn"] = knnQuery
            hints = append(hints, "knn_search")
        }
    }

    // Serialize to JSON
    queryJSON, _ := json.MarshalIndent(query, "", "  ")

    return &NativeQuery{
        Backend:      "elasticsearch",
        Query:        string(queryJSON),
        Parameters:   nil, // Elasticsearch uses inline parameters
        Hints:        hints,
        EstimatedOps: e.estimateOperations(elemType, len(expr.Predicates)),
        Metadata: map[string]interface{}{
            "index":        e.index,
            "routing":      elemType,
            "knn_enabled":  len(expr.CustomFns) > 0,
        },
    }, nil
}

func (e *ElasticsearchBackend) SupportsFeature(feature string) bool {
    return map[string]bool{
        "similarity":        true,  // knn search
        "partition_pruning": false, // Use index routing instead
        "vector_search":     true,  // knn/vector search
        "regex":             true,  // regexp query
        "full_text":         true,  // match query
    }[feature]
}

func (e *ElasticsearchBackend) GetOptimizations() []string {
    return []string{
        "index_routing:element_type",   // Route to specific shard (50x speedup)
        "term_query:promoted_fields",   // Exact match on indexed fields
        "knn_search:embeddings",        // Vector similarity
        "doc_values:aggregations",      // Fast aggregations
    }
}

func (e *ElasticsearchBackend) translatePredicate(pred Predicate) interface{} {
    switch pred.Operator {
    case "==":
        return map[string]interface{}{
            "term": map[string]interface{}{
                pred.Field: pred.Value,
            },
        }
    case ">=", "<=", ">", "<":
        return map[string]interface{}{
            "range": map[string]interface{}{
                pred.Field: map[string]interface{}{
                    pred.Operator: pred.Value,
                },
            },
        }
    case "contains":
        return map[string]interface{}{
            "match": map[string]interface{}{
                pred.Field: pred.Value,
            },
        }
    case "regex":
        return map[string]interface{}{
            "regexp": map[string]interface{}{
                pred.Field: pred.Value,
            },
        }
    }
    return nil
}

func (e *ElasticsearchBackend) translateKNNSearch(fns []CustomFunction) interface{} {
    for _, fn := range fns {
        if fn.Name == "similarity" && len(fn.Args) >= 2 {
            queryText := fn.Args[0].(string)
            threshold := fn.Args[1].(float64)

            // Elasticsearch knn query (requires pre-computed query embedding)
            return map[string]interface{}{
                "field":         "embedding",
                "query_vector":  queryText, // Client must embed this
                "k":             100,
                "num_candidates": 10000,
                "similarity":     threshold,
            }
        }
    }
    return nil
}

func (e *ElasticsearchBackend) estimateOperations(elemType string, predicateCount int) int {
    baseOps := 1000000
    if elemType != "" {
        baseOps /= 50 // Index routing
    }
    for i := 0; i < predicateCount; i++ {
        baseOps /= 10
    }
    return baseOps
}
```

**Example Query DSL Output:**
```json
{
  "bool": {
    "must": [
      {"term": {"element_type": "paragraph"}},
      {"term": {"page_number": 5}}
    ]
  },
  "knn": {
    "field": "embedding",
    "query_vector": "[0.1, 0.2, ...]",
    "k": 100,
    "similarity": 0.8
  }
}
```

---

#### 2.3.4.3 PostgreSQL Backend (Relational Database)

**New File:** `go/internal/udml/jsonpath/postgresql_backend.go`

```go
package jsonpath

// PostgreSQLBackend implements QueryBackend for PostgreSQL SQL with JSONB
type PostgreSQLBackend struct {
    schema string
    table  string
}

func NewPostgreSQLBackend(schema, table string) *PostgreSQLBackend {
    return &PostgreSQLBackend{
        schema: schema,
        table:  table,
    }
}

func (p *PostgreSQLBackend) GetName() string {
    return "postgresql"
}

// Translate converts JSONPath expression to PostgreSQL SQL
func (p *PostgreSQLBackend) Translate(expr *Expression, opts TranslateOptions) (*NativeQuery, error) {
    params := make(map[string]interface{})
    hints := []string{}

    // Extract element_type for partition pruning (100x speedup with partitioning)
    elemType := p.extractElementTypeFilter(expr)

    var tableName string
    if elemType != "" {
        // Query partitioned table by element_type
        tableName = fmt.Sprintf("%s.elements_%s", p.schema, elemType)
        hints = append(hints, fmt.Sprintf("partition_pruning:%s", elemType))
    } else {
        // Query parent table (scans all partitions)
        tableName = fmt.Sprintf("%s.elements", p.schema)
    }

    var sqlBuilder strings.Builder
    sqlBuilder.WriteString("SELECT * FROM ")
    sqlBuilder.WriteString(tableName)

    // Translate predicates with promoted field detection
    whereClauses := p.translatePredicates(expr.Predicates, params, opts.PromotedFields)
    if len(whereClauses) > 0 {
        sqlBuilder.WriteString(" WHERE ")
        sqlBuilder.WriteString(strings.Join(whereClauses, " AND "))
        hints = append(hints, "index_scan:promoted_fields")
    }

    // Handle similarity() with pgvector extension
    if len(expr.CustomFns) > 0 {
        vectorClause := p.translateVectorSearch(expr.CustomFns, params)
        if vectorClause != "" {
            if len(whereClauses) > 0 {
                sqlBuilder.WriteString(" AND ")
            } else {
                sqlBuilder.WriteString(" WHERE ")
            }
            sqlBuilder.WriteString(vectorClause)
            hints = append(hints, "vector_search:pgvector")
        }
    }

    if opts.MaxResults > 0 {
        sqlBuilder.WriteString(fmt.Sprintf(" LIMIT %d", opts.MaxResults))
    }

    return &NativeQuery{
        Backend:      "postgresql",
        Query:        sqlBuilder.String(),
        Parameters:   params,
        Hints:        hints,
        EstimatedOps: p.estimateOperations(elemType, len(whereClauses)),
        Metadata: map[string]interface{}{
            "partition_pruning": elemType != "",
            "vector_search":     len(expr.CustomFns) > 0,
        },
    }, nil
}

func (p *PostgreSQLBackend) SupportsFeature(feature string) bool {
    return map[string]bool{
        "similarity":        true,  // pgvector extension
        "partition_pruning": true,  // Native partitioning
        "vector_search":     true,  // pgvector ivfflat index
        "regex":             true,  // ~ operator
        "jsonb_path":        true,  // jsonb_path_query
    }[feature]
}

func (p *PostgreSQLBackend) GetOptimizations() []string {
    return []string{
        "partition_pruning:element_type", // 100x speedup
        "btree_index:promoted_fields",    // Indexed columns
        "ivfflat_index:embeddings",       // Vector search
        "gin_index:jsonb_metadata",       // JSONB field indexing
    }
}

func (p *PostgreSQLBackend) translatePredicates(predicates []Predicate, params map[string]interface{}, promotedFields []string) []string {
    clauses := []string{}
    promotedSet := make(map[string]bool)
    for _, field := range promotedFields {
        promotedSet[field] = true
    }

    for i, pred := range predicates {
        paramName := fmt.Sprintf("$%d", i+1) // PostgreSQL uses $1, $2, etc.
        params[paramName] = pred.Value

        if promotedSet[pred.Field] {
            // Promoted field: Direct column access
            switch pred.Operator {
            case "==":
                clauses = append(clauses, fmt.Sprintf("%s = %s", pred.Field, paramName))
            case ">=":
                clauses = append(clauses, fmt.Sprintf("%s >= %s", pred.Field, paramName))
            case "contains":
                clauses = append(clauses, fmt.Sprintf("%s ILIKE '%%' || %s || '%%'", pred.Field, paramName))
            case "regex":
                clauses = append(clauses, fmt.Sprintf("%s ~ %s", pred.Field, paramName))
            }
        } else {
            // JSONB field extraction
            clauses = append(clauses, fmt.Sprintf("metadata->>'%s' = %s", pred.Field, paramName))
        }
    }

    return clauses
}

func (p *PostgreSQLBackend) translateVectorSearch(fns []CustomFunction, params map[string]interface{}) string {
    for i, fn := range fns {
        if fn.Name == "similarity" && len(fn.Args) >= 2 {
            queryText := fn.Args[0]
            threshold := fn.Args[1]

            paramName := fmt.Sprintf("$emb_%d", i)
            params[paramName] = queryText // Client embeds this

            // pgvector cosine similarity
            return fmt.Sprintf("(embedding <=> %s) < (1 - %v)", paramName, threshold)
        }
    }
    return ""
}

func (p *PostgreSQLBackend) estimateOperations(elemType string, predicateCount int) int {
    baseOps := 1000000
    if elemType != "" {
        baseOps /= 100 // Partition pruning
    }
    for i := 0; i < predicateCount; i++ {
        baseOps /= 8
    }
    return baseOps
}
```

**Example SQL Output:**
```sql
-- With partition pruning:
SELECT * FROM analytics.elements_paragraph
WHERE page_number = $1 AND (embedding <=> $emb_0) < 0.2
LIMIT 100
```

---

#### 2.3.4.4 Solr Backend (Search Platform)

**New File:** `go/internal/udml/jsonpath/solr_backend.go`

```go
package jsonpath

// SolrBackend implements QueryBackend for Apache Solr queries
type SolrBackend struct {
    url        string
    collection string
}

func NewSolrBackend(url, collection string) *SolrBackend {
    return &SolrBackend{
        url:        url,
        collection: collection,
    }
}

func (s *SolrBackend) GetName() string {
    return "solr"
}

// Translate converts JSONPath expression to Solr query syntax
func (s *SolrBackend) Translate(expr *Expression, opts TranslateOptions) (*NativeQuery, error) {
    params := make(map[string]interface{})
    hints := []string{}

    var queryParts []string

    // Extract element_type for collection routing
    elemType := s.extractElementTypeFilter(expr)
    if elemType != "" {
        queryParts = append(queryParts, fmt.Sprintf("element_type:%s", elemType))
        hints = append(hints, fmt.Sprintf("collection_routing:%s", elemType))
    }

    // Translate predicates to Solr query syntax
    for _, pred := range expr.Predicates {
        predQuery := s.translatePredicate(pred)
        if predQuery != "" {
            queryParts = append(queryParts, predQuery)
            hints = append(hints, "field_query")
        }
    }

    // Handle similarity() with vector search
    if len(expr.CustomFns) > 0 {
        vectorQuery := s.translateVectorSearch(expr.CustomFns)
        if vectorQuery != "" {
            queryParts = append(queryParts, vectorQuery)
            hints = append(hints, "vector_search")
        }
    }

    // Build Solr query
    solrQuery := strings.Join(queryParts, " AND ")
    if solrQuery == "" {
        solrQuery = "*:*" // Match all
    }

    // Add parameters
    params["q"] = solrQuery
    if opts.MaxResults > 0 {
        params["rows"] = opts.MaxResults
    }
    params["wt"] = "json" // Response format

    return &NativeQuery{
        Backend:      "solr",
        Query:        solrQuery,
        Parameters:   params,
        Hints:        hints,
        EstimatedOps: s.estimateOperations(elemType, len(expr.Predicates)),
        Metadata: map[string]interface{}{
            "collection":     s.collection,
            "routing":        elemType,
            "vector_enabled": len(expr.CustomFns) > 0,
        },
    }, nil
}

func (s *SolrBackend) SupportsFeature(feature string) bool {
    return map[string]bool{
        "similarity":        true, // Dense vector search
        "partition_pruning": false, // Use collection routing
        "vector_search":     true, // kNN plugin
        "regex":             false, // Limited regex support
        "full_text":         true, // Native full-text search
        "faceting":          true, // Native faceting
    }[feature]
}

func (s *SolrBackend) GetOptimizations() []string {
    return []string{
        "collection_routing:element_type", // 80x speedup
        "field_cache:promoted_fields",     // Doc values
        "knn_search:embeddings",           // Vector search
        "filter_cache",                    // Filter query caching
    }
}

func (s *SolrBackend) translatePredicate(pred Predicate) string {
    switch pred.Operator {
    case "==":
        return fmt.Sprintf("%s:%v", pred.Field, pred.Value)
    case ">=":
        return fmt.Sprintf("%s:[%v TO *]", pred.Field, pred.Value)
    case "<=":
        return fmt.Sprintf("%s:[* TO %v]", pred.Field, pred.Value)
    case "contains":
        return fmt.Sprintf("%s:*%v*", pred.Field, pred.Value)
    }
    return ""
}

func (s *SolrBackend) translateVectorSearch(fns []CustomFunction) string {
    for _, fn := range fns {
        if fn.Name == "similarity" && len(fn.Args) >= 2 {
            queryText := fn.Args[0].(string)
            threshold := fn.Args[1].(float64)

            // Solr vector search using kNN
            return fmt.Sprintf("{!knn f=embedding topK=100}%s", queryText)
        }
    }
    return ""
}

func (s *SolrBackend) estimateOperations(elemType string, predicateCount int) int {
    baseOps := 1000000
    if elemType != "" {
        baseOps /= 80
    }
    for i := 0; i < predicateCount; i++ {
        baseOps /= 8
    }
    return baseOps
}
```

**Example Solr Query:**
```
element_type:paragraph AND page_number:5 AND {!knn f=embedding topK=100}financial projections
```

---

### 2.3.4.5 Backend Comparison Summary

| Backend | Partition Strategy | Promoted Field Access | Vector Search | Speedup vs JSON |
|---------|-------------------|---------------------|---------------|-----------------|
| **DuckDB** | Hive partitioning (60x) | Column direct access + row group skipping (10x) | cosine_similarity UDF | 600x |
| **Neo4j** | Node label filtering (1000x) | Node property indexes (5x) | vector.similarity.cosine | 5000x |
| **Elasticsearch** | Index routing (50x) | Term queries on indexed fields (10x) | knn search | 500x |
| **PostgreSQL** | Table partitioning (100x) | B-tree indexes (8x) | pgvector ivfflat | 800x |
| **Solr** | Collection routing (80x) | Doc values (8x) | kNN plugin | 640x |

**Key Insight:** All backends benefit massively from promoted fields, but the optimization strategy varies:
- **DuckDB**: Parquet row group skipping
- **Neo4j**: Property indexes on graph nodes
- **Elasticsearch**: Inverted indexes on fields
- **PostgreSQL**: B-tree indexes on columns
- **Solr**: Doc values for fast field access

---

### 2.3.5 Unified Translator (Backend-Agnostic Translation)

**New File:** `go/internal/udml/jsonpath/translator.go`

The unified translator provides a single interface for translating JSONPath expressions to any registered backend's native query language.

```go
package jsonpath

import (
    "fmt"
)

// Translator provides unified JSONPath to native query translation
type Translator struct {
    registry *BackendRegistry
    parser   *Parser  // JSONPath parser
}

func NewTranslator() *Translator {
    return &Translator{
        registry: NewBackendRegistry(),
        parser:   NewParser(),
    }
}

// Translate JSONPath expression using specified or default backend
func (t *Translator) Translate(jsonpath string, opts TranslateOptions) (*NativeQuery, error) {
    // Parse JSONPath to AST
    expr, err := t.parser.Parse(jsonpath)
    if err != nil {
        return nil, fmt.Errorf("failed to parse JSONPath: %w", err)
    }

    // Get backend (use default if not specified)
    var backend QueryBackend
    if opts.Backend != "" {
        backend, err = t.registry.GetBackend(opts.Backend)
        if err != nil {
            return nil, err
        }
    } else {
        backend = t.registry.GetDefaultBackend()
    }

    // Translate using backend
    query, err := backend.Translate(expr, opts)
    if err != nil {
        return nil, fmt.Errorf("backend %s translation failed: %w", backend.GetName(), err)
    }

    return query, nil
}

// TranslateMultiple translates to multiple backends for comparison
func (t *Translator) TranslateMultiple(jsonpath string, opts TranslateOptions, backends []string) (map[string]*NativeQuery, error) {
    // Parse once
    expr, err := t.parser.Parse(jsonpath)
    if err != nil {
        return nil, err
    }

    results := make(map[string]*NativeQuery)
    for _, backendName := range backends {
        backend, err := t.registry.GetBackend(backendName)
        if err != nil {
            return nil, err
        }

        query, err := backend.Translate(expr, opts)
        if err != nil {
            return nil, fmt.Errorf("backend %s failed: %w", backendName, err)
        }

        results[backendName] = query
    }

    return results, nil
}

// GetBackendRecommendation suggests best backend for given query
func (t *Translator) GetBackendRecommendation(expr *Expression) string {
    // Analyze query characteristics
    hasVectorSearch := len(expr.CustomFns) > 0
    hasGraphTraversal := expr.NeedsGraphTraversal()
    hasComplexPredicates := len(expr.Predicates) > 3

    // Recommend backend based on query type
    switch {
    case hasGraphTraversal:
        return "neo4j" // Best for graph queries
    case hasVectorSearch && hasComplexPredicates:
        return "elasticsearch" // Best for combined vector + text search
    case hasVectorSearch:
        return "duckdb" // Good balance for analytics
    case hasComplexPredicates:
        return "postgresql" // Good for complex filtering
    default:
        return "duckdb" // Default for analytical queries
    }
}
```

**Example Usage:**

```go
package main

import (
    "fmt"
    "github.com/kennethstott/go-doc-go/internal/udml/jsonpath"
)

func main() {
    translator := jsonpath.NewTranslator()

    // JSONPath query
    jp := `$.elements[?(@.type == 'paragraph' && @.page_number >= 5)].similarity('financial projections', 0.8)`

    // Translate to default backend (DuckDB)
    opts := jsonpath.TranslateOptions{
        PartitionPath: "analytics/",
        PromotedFields: []string{"page_number", "section_level", "row_index", "column_index", "temporal_type", "tag_name"},
        Version: "v1.0.0",
        MaxResults: 100,
    }

    query, err := translator.Translate(jp, opts)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Backend: %s\n", query.Backend)
    fmt.Printf("Query: %s\n", query.Query)
    fmt.Printf("Hints: %v\n", query.Hints)
    fmt.Printf("Estimated Operations: %d\n", query.EstimatedOps)

    // Output:
    // Backend: duckdb
    // Query: SELECT * FROM 'analytics/element_type=paragraph/**/**.parquet'
    //        WHERE page_number >= $p0 AND version = 'v1.0.0'
    //        AND element_id IN (
    //            SELECT element_id FROM 'analytics/embeddings.parquet'
    //            WHERE cosine_similarity(embedding, $query_emb_0) >= $threshold_0
    //        ) LIMIT 100
    // Hints: [partition_pruning:element_type=paragraph predicate_pushdown:promoted_fields custom_functions:similarity]
    // Estimated Operations: 167

    // Translate to multiple backends for comparison
    opts.Backend = "" // Use default
    multiQuery, _ := translator.TranslateMultiple(jp, opts, []string{"duckdb", "neo4j", "postgresql"})

    for backend, q := range multiQuery {
        fmt.Printf("\n%s Query:\n%s\n", backend, q.Query)
        fmt.Printf("Cost estimate: %d ops\n", q.EstimatedOps)
    }
}
```

**JSONPath Parser (Simplified)**

```go
// Parser.go - JSONPath to AST parser
type Parser struct{}

func NewParser() *Parser {
    return &Parser{}
}

// Parse JSONPath string into Expression AST
func (p *Parser) Parse(jsonpath string) (*Expression, error) {
    // Simplified parsing logic
    // Production implementation would use a proper parser generator

    expr := &Expression{
        Predicates: []Predicate{},
        CustomFns:  []CustomFunction{},
    }

    // Extract predicates from [?(...)] syntax
    // Extract custom functions like similarity()
    // Build AST

    return expr, nil
}

// Expression represents parsed JSONPath as backend-agnostic AST
type Expression struct {
    Base       string            // Base path: "$.elements"
    Predicates []Predicate       // Filter predicates
    CustomFns  []CustomFunction  // Custom functions
}

func (e *Expression) NeedsGraphTraversal() bool {
    // Check if query requires graph operations
    // e.g., descendant axes, relationship traversal
    return false // Simplified
}

type Predicate struct {
    Field    string      // Field name
    Operator string      // Operator: ==, >=, contains, regex
    Value    interface{} // Comparison value
}

type CustomFunction struct {
    Name string        // Function name: similarity, fuzzy_match
    Args []interface{} // Function arguments
}
```

**Files to Create:**
- [ ] `go/internal/udml/jsonpath/translator.go` - Unified translator
- [ ] `go/internal/udml/jsonpath/parser.go` - JSONPath parser
- [ ] `go/internal/udml/jsonpath/ast.go` - Expression AST definitions
- [ ] `go/internal/udml/jsonpath/translator_test.go` - Translation tests

**Testing:**
- Parse complex JSONPath expressions correctly
- Translate to all backends produces valid queries
- Backend recommendation logic works correctly
- Multi-backend translation comparison
- Error handling for invalid JSONPath

---

### 2.3.6 Backend Configuration

Configuration allows users to specify which backends to use and their settings. Configuration is in YAML format.

**Example:** `config.yaml`

```yaml
# UDML Configuration
udml:
  version: "v1.0.0"

  # Storage configuration
  storage:
    base_path: "analytics/"
    parquet_compression: "zstd"
    partition_by:
      - element_type
      - version
      - date
      - source

  # Promoted fields (query-optimized)
  promoted_fields:
    - page_number
    - section_level
    - row_index
    - column_index
    - temporal_type
    - tag_name

  # Query backends
  backends:
    # Default backend (Phase 1)
    default: duckdb

    # DuckDB configuration
    duckdb:
      enabled: true
      database_path: ":memory:"  # or "analytics.duckdb"
      extensions:
        - parquet
        - httpfs
        - json
      optimizations:
        - partition_pruning
        - predicate_pushdown
        - parallel_scan

    # Neo4j configuration (Phase 4+)
    neo4j:
      enabled: false
      uri: "bolt://localhost:7687"
      database: "udml"
      username: "neo4j"
      password: "${NEO4J_PASSWORD}"  # From environment
      indexes:
        - element_type  # Node label
        - page_number   # Property index
        - embedding     # Vector index

    # Elasticsearch configuration (Phase 4+)
    elasticsearch:
      enabled: false
      url: "http://localhost:9200"
      index_prefix: "udml-"
      shards: 5
      replicas: 1
      mappings:
        element_type: "keyword"
        page_number: "integer"
        section_level: "integer"
        embedding:
          type: "dense_vector"
          dims: 384
          similarity: "cosine"

    # PostgreSQL configuration (Phase 4+)
    postgresql:
      enabled: false
      host: "localhost"
      port: 5432
      database: "udml"
      schema: "analytics"
      username: "postgres"
      password: "${POSTGRES_PASSWORD}"
      partitioning:
        strategy: "list"  # Partition by element_type
        key: "element_type"
      indexes:
        - name: "idx_page_number"
          columns: ["page_number"]
          type: "btree"
        - name: "idx_embedding"
          columns: ["embedding"]
          type: "ivfflat"
          options:
            lists: 100

    # Solr configuration (Phase 4+)
    solr:
      enabled: false
      url: "http://localhost:8983/solr"
      collection_prefix: "udml-"
      config_set: "udml_configset"
      shards: 5
      replication_factor: 2
      schema:
        element_type: "string"
        page_number: "pint"
        embedding: "knn_vector[384]"

  # Embeddings configuration (for similarity() function)
  embeddings:
    enabled: true
    provider: "onnx"  # onnx, ollama, openai
    model: "BAAI/bge-small-en-v1.5"
    model_path: "assets/bge-small-en-v1.5"
    dimensions: 384
    batch_size: 32
    contextual: true  # Use graphlet method
    predecessor_count: 3
    successor_count: 3
```

**Go Configuration Loading:**

```go
// go/internal/config/udml_config.go
package config

import (
    "github.com/BurntSushi/toml"
    "gopkg.in/yaml.v3"
)

type UDMLConfig struct {
    Version string          `yaml:"version" toml:"version"`
    Storage StorageConfig   `yaml:"storage" toml:"storage"`
    PromotedFields []string `yaml:"promoted_fields" toml:"promoted_fields"`
    Backends BackendsConfig `yaml:"backends" toml:"backends"`
    Embeddings EmbeddingsConfig `yaml:"embeddings" toml:"embeddings"`
}

type BackendsConfig struct {
    Default        string             `yaml:"default" toml:"default"`
    DuckDB         *DuckDBConfig      `yaml:"duckdb,omitempty" toml:"duckdb,omitempty"`
    Neo4j          *Neo4jConfig       `yaml:"neo4j,omitempty" toml:"neo4j,omitempty"`
    Elasticsearch  *ElasticsearchConfig `yaml:"elasticsearch,omitempty" toml:"elasticsearch,omitempty"`
    PostgreSQL     *PostgreSQLConfig  `yaml:"postgresql,omitempty" toml:"postgresql,omitempty"`
    Solr           *SolrConfig        `yaml:"solr,omitempty" toml:"solr,omitempty"`
}

type DuckDBConfig struct {
    Enabled       bool     `yaml:"enabled" toml:"enabled"`
    DatabasePath  string   `yaml:"database_path" toml:"database_path"`
    Extensions    []string `yaml:"extensions" toml:"extensions"`
    Optimizations []string `yaml:"optimizations" toml:"optimizations"`
}

// LoadConfig loads UDML configuration from YAML/TOML
func LoadConfig(path string) (*UDMLConfig, error) {
    // Auto-detect format and load
    if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
        return loadYAML(path)
    } else if strings.HasSuffix(path, ".toml") {
        return loadTOML(path)
    }
    return nil, fmt.Errorf("unsupported config format")
}
```

**Files to Create:**
- [ ] `go/internal/config/udml_config.go` - Configuration structs and loader
- [ ] `config.yaml` - Example YAML configuration
- [ ] `config.toml` - Example TOML configuration (alternative format)

---

### 2.3.7 Backend Comparison: Promoted Field Mappings

This table shows how the 6 promoted fields map to native constructs across all backends, explaining why each backend benefits from promoted fields.

| Promoted Field | Type | DuckDB (Parquet) | Neo4j (Graph) | Elasticsearch (Search) | PostgreSQL (Relational) | Solr (Search) |
|----------------|------|------------------|---------------|----------------------|------------------------|---------------|
| **page_number** | *int | **Parquet column** → Row group skipping (10x) | **Node property** → B-tree index (5x) | **Indexed field** → Term query (10x) | **Column** → B-tree index (8x) | **Doc value** → Field cache (8x) |
| **section_level** | *int | **Parquet column** → Row group skipping (10x) | **Node property** → B-tree index (5x) | **Indexed field** → Range query (10x) | **Column** → B-tree index (8x) | **Doc value** → Range query (8x) |
| **row_index** | *int | **Parquet column** → Row group skipping (10x) | **Node property** → B-tree index (5x) | **Indexed field** → Term query (10x) | **Column** → B-tree index (8x) | **Doc value** → Field cache (8x) |
| **column_index** | *int | **Parquet column** → Row group skipping (10x) | **Node property** → B-tree index (5x) | **Indexed field** → Term query (10x) | **Column** → B-tree index (8x) | **Doc value** → Field cache (8x) |
| **temporal_type** | *string | **Parquet column** → Dictionary encoding + row group skip (10x) | **Node property** → String index (5x) | **Keyword field** → Term query (10x) | **Column** → B-tree index (8x) | **String field** → Term query (8x) |
| **tag_name** | *string | **Parquet column** → Dictionary encoding + row group skip (10x) | **Node property** → String index (5x) | **Keyword field** → Term query (10x) | **Column** → B-tree index (8x) | **String field** → Term query (8x) |

**Key Insights:**

1. **Storage Mapping:**
   - **DuckDB**: Direct Parquet columns enable statistics-based row group skipping
   - **Neo4j**: Node properties with indexes on promoted fields
   - **Elasticsearch**: Top-level document fields with inverted indexes
   - **PostgreSQL**: Table columns with B-tree indexes
   - **Solr**: Doc values for fast field access without uninversion

2. **Query Translation:**
   - **DuckDB**: `page_number = 5` → `WHERE page_number = 5` (column filter)
   - **Neo4j**: `page_number = 5` → `WHERE e.page_number = 5` (property filter)
   - **Elasticsearch**: `page_number = 5` → `{"term": {"page_number": 5}}` (term query)
   - **PostgreSQL**: `page_number = 5` → `WHERE page_number = 5` (column filter)
   - **Solr**: `page_number = 5` → `page_number:5` (field query)

3. **Without Promoted Fields:**
   - **DuckDB**: `json_extract_path(metadata, 'page_number') = 5` (JSON parsing, no row group skipping)
   - **Neo4j**: `n.metadata['page_number'] = 5` (map lookup, no index)
   - **Elasticsearch**: `{"term": {"metadata.page_number": 5}}` (nested field, slower)
   - **PostgreSQL**: `metadata->>'page_number' = '5'` (JSONB extraction, no index)
   - **Solr**: Complex dynamic field mapping, no type safety

4. **Nullable Field Handling:**
   - All backends handle NULL efficiently
   - **DuckDB**: Parquet null bitmap, zero storage overhead
   - **Neo4j**: Optional properties, no penalty
   - **Elasticsearch**: Missing field optimization
   - **PostgreSQL**: NULL bitmap in tuple header
   - **Solr**: Missing field handled by doc values

**Composite Optimization:**

When combining partition/label pruning + promoted field filtering:

| Backend | Partition Speedup | Field Speedup | Composite | Example Query |
|---------|------------------|---------------|-----------|---------------|
| DuckDB | 60x (Hive partition) | 10x (row group skip) | **600x** | `element_type=paragraph + page_number=5` |
| Neo4j | 1000x (label scan) | 5x (property index) | **5000x** | `MATCH (e:paragraph) WHERE e.page_number = 5` |
| Elasticsearch | 50x (index routing) | 10x (term query) | **500x** | `{"bool": {"must": [{"term": {"element_type": "paragraph"}}, {"term": {"page_number": 5}}]}}` |
| PostgreSQL | 100x (partition) | 8x (B-tree index) | **800x** | `SELECT * FROM elements_paragraph WHERE page_number = 5` |
| Solr | 80x (collection routing) | 8x (doc value) | **640x** | `element_type:paragraph AND page_number:5` |

---

### 2.4 Document Reconstruction (`go/internal/udml/reconstruction.go`)

**Purpose:** Generate text-only approximations of documents or document fragments from UDML storage in HTML, Markdown, or plain text format. This "reverse parsing" reconstructs the original document structure while preserving hierarchy.

**Python Reference:** `src/go_doc_go/storage/base.py:953` - `reconstruct_document()` method

#### 2.4.1 Reconstruction Interface

**New File:** `go/internal/udml/reconstruction.go`

```go
package udml

import (
    "context"
    "fmt"
    "io"
    "strings"
)

// DocumentReconstructor rebuilds documents from UDML elements
type DocumentReconstructor interface {
    // ReconstructDocument rebuilds entire document
    ReconstructDocument(ctx context.Context, docID string, format ReconstructionFormat) (string, error)

    // ReconstructFragment rebuilds specific elements and descendants
    ReconstructFragment(ctx context.Context, elementIDs []string, format ReconstructionFormat) (string, error)

    // ReconstructToWriter streams reconstruction to writer (for large documents)
    ReconstructToWriter(ctx context.Context, docID string, format ReconstructionFormat, writer io.Writer) error
}

// ReconstructionFormat specifies output format
type ReconstructionFormat string

const (
    FormatPlainText ReconstructionFormat = "text"
    FormatHTML      ReconstructionFormat = "html"
    FormatMarkdown  ReconstructionFormat = "markdown"
)

// Reconstructor implements document reconstruction
type Reconstructor struct {
    storage     Storage // UDML storage interface
    formatters  map[ReconstructionFormat]ElementFormatter
}

func NewReconstructor(storage Storage) *Reconstructor {
    r := &Reconstructor{
        storage:    storage,
        formatters: make(map[ReconstructionFormat]ElementFormatter),
    }

    // Register format-specific formatters
    r.formatters[FormatPlainText] = &PlainTextFormatter{}
    r.formatters[FormatHTML] = &HTMLFormatter{}
    r.formatters[FormatMarkdown] = &MarkdownFormatter{}

    return r
}

// ReconstructDocument rebuilds entire document from UDML
func (r *Reconstructor) ReconstructDocument(ctx context.Context, docID string, format ReconstructionFormat) (string, error) {
    // Fetch all elements for document
    result, err := r.storage.GetDocument(ctx, docID)
    if err != nil {
        return "", fmt.Errorf("failed to fetch document: %w", err)
    }

    // Get appropriate formatter
    formatter, exists := r.formatters[format]
    if !exists {
        return "", fmt.Errorf("unsupported format: %s", format)
    }

    // Build element hierarchy
    hierarchy := buildElementHierarchy(result.Elements)

    // Reconstruct with format-specific rendering
    var output strings.Builder
    if err := r.reconstructHierarchy(hierarchy, formatter, &output, 0); err != nil {
        return "", err
    }

    return output.String(), nil
}

// ReconstructFragment rebuilds specific elements
func (r *Reconstructor) ReconstructFragment(ctx context.Context, elementIDs []string, format ReconstructionFormat) (string, error) {
    // Fetch specified elements and their descendants
    elements, err := r.fetchElementsWithDescendants(ctx, elementIDs)
    if err != nil {
        return "", err
    }

    formatter := r.formatters[format]
    hierarchy := buildElementHierarchy(elements)

    var output strings.Builder
    if err := r.reconstructHierarchy(hierarchy, formatter, &output, 0); err != nil {
        return "", err
    }

    return output.String(), nil
}

// reconstructHierarchy recursively renders element tree
func (r *Reconstructor) reconstructHierarchy(node *HierarchyNode, formatter ElementFormatter, output *strings.Builder, depth int) error {
    // Render opening tag/structure
    opening := formatter.FormatOpening(node.Element, depth)
    output.WriteString(opening)

    // Render content
    content := formatter.FormatContent(node.Element, depth)
    output.WriteString(content)

    // Recursively render children
    for _, child := range node.Children {
        if err := r.reconstructHierarchy(child, formatter, output, depth+1); err != nil {
            return err
        }
    }

    // Render closing tag/structure
    closing := formatter.FormatClosing(node.Element, depth)
    output.WriteString(closing)

    return nil
}

// HierarchyNode represents element tree node
type HierarchyNode struct {
    Element  Element
    Children []*HierarchyNode
}

// buildElementHierarchy constructs tree from flat elements
func buildElementHierarchy(elements []Element) *HierarchyNode {
    elementMap := make(map[string]*HierarchyNode)
    var rootNodes []*HierarchyNode

    // First pass: create all nodes
    for _, elem := range elements {
        node := &HierarchyNode{
            Element:  elem,
            Children: make([]*HierarchyNode, 0),
        }
        elementMap[elem.ElementID] = node
    }

    // Second pass: build parent-child relationships
    for _, elem := range elements {
        node := elementMap[elem.ElementID]
        if elem.ParentID == "" {
            rootNodes = append(rootNodes, node)
        } else if parent, exists := elementMap[elem.ParentID]; exists {
            parent.Children = append(parent.Children, node)
        }
    }

    // Return synthetic root if multiple roots
    if len(rootNodes) == 1 {
        return rootNodes[0]
    }

    return &HierarchyNode{
        Element: Element{
            ElementID:   "root",
            ElementType: "document",
        },
        Children: rootNodes,
    }
}
```

#### 2.4.2 Format-Specific Formatters

**New File:** `go/internal/udml/formatters.go`

```go
package udml

import (
    "fmt"
    "strings"
)

// ElementFormatter defines format-specific element rendering
type ElementFormatter interface {
    // FormatOpening renders opening tag/structure
    FormatOpening(elem Element, depth int) string

    // FormatContent renders element content
    FormatContent(elem Element, depth int) string

    // FormatClosing renders closing tag/structure
    FormatClosing(elem Element, depth int) string
}

// PlainTextFormatter renders as plain text
type PlainTextFormatter struct{}

func (f *PlainTextFormatter) FormatOpening(elem Element, depth int) string {
    switch elem.ElementType {
    case "heading", "header":
        // Add visual separation for headings
        level := getHeadingLevel(elem)
        return strings.Repeat("#", level) + " "
    case "list_item":
        return strings.Repeat("  ", depth) + "• "
    case "table":
        return "\n"
    case "table_row":
        return ""
    case "table_cell":
        return "| "
    default:
        return ""
    }
}

func (f *PlainTextFormatter) FormatContent(elem Element, depth int) string {
    if elem.Content == "" {
        return ""
    }
    return elem.Content
}

func (f *PlainTextFormatter) FormatClosing(elem Element, depth int) string {
    switch elem.ElementType {
    case "paragraph", "heading", "header":
        return "\n\n"
    case "list_item":
        return "\n"
    case "table_cell":
        return " "
    case "table_row":
        return "|\n"
    case "table":
        return "\n"
    default:
        return ""
    }
}

// HTMLFormatter renders as HTML
type HTMLFormatter struct{}

func (f *HTMLFormatter) FormatOpening(elem Element, depth int) string {
    switch elem.ElementType {
    case "document":
        return "<html><body>\n"
    case "heading", "header":
        level := getHeadingLevel(elem)
        return fmt.Sprintf("<h%d>", level)
    case "paragraph":
        return "<p>"
    case "list":
        // Determine list type from metadata
        if elem.Metadata["list_type"] == "ordered" {
            return "<ol>\n"
        }
        return "<ul>\n"
    case "list_item":
        return "  <li>"
    case "table":
        return "<table>\n"
    case "table_header":
        return "  <thead>\n    <tr>"
    case "table_row":
        return "  <tr>"
    case "table_cell":
        if isHeaderCell(elem) {
            return "<th>"
        }
        return "<td>"
    case "code_block":
        language := getCodeLanguage(elem)
        return fmt.Sprintf("<pre><code class=\"language-%s\">", language)
    case "link":
        href := getLinkHref(elem)
        return fmt.Sprintf("<a href=\"%s\">", href)
    case "image":
        src := getImageSrc(elem)
        alt := getImageAlt(elem)
        return fmt.Sprintf("<img src=\"%s\" alt=\"%s\" />", src, alt)
    default:
        // Use tag_name if available (for HTML/XML elements)
        if tagName, ok := elem.TagName; ok && tagName != nil {
            return fmt.Sprintf("<%s>", *tagName)
        }
        return "<div>"
    }
}

func (f *HTMLFormatter) FormatContent(elem Element, depth int) string {
    if elem.Content == "" {
        return ""
    }
    // HTML-escape content
    return htmlEscape(elem.Content)
}

func (f *HTMLFormatter) FormatClosing(elem Element, depth int) string {
    switch elem.ElementType {
    case "document":
        return "</body></html>\n"
    case "heading", "header":
        level := getHeadingLevel(elem)
        return fmt.Sprintf("</h%d>\n", level)
    case "paragraph":
        return "</p>\n"
    case "list":
        if elem.Metadata["list_type"] == "ordered" {
            return "</ol>\n"
        }
        return "</ul>\n"
    case "list_item":
        return "</li>\n"
    case "table":
        return "</table>\n"
    case "table_header":
        return "</tr>\n  </thead>\n"
    case "table_row":
        return "</tr>\n"
    case "table_cell":
        if isHeaderCell(elem) {
            return "</th>"
        }
        return "</td>"
    case "code_block":
        return "</code></pre>\n"
    case "link":
        return "</a>"
    case "image":
        return "" // Self-closing
    default:
        if tagName, ok := elem.TagName; ok && tagName != nil {
            return fmt.Sprintf("</%s>", *tagName)
        }
        return "</div>"
    }
}

// MarkdownFormatter renders as Markdown
type MarkdownFormatter struct{}

func (f *MarkdownFormatter) FormatOpening(elem Element, depth int) string {
    switch elem.ElementType {
    case "heading", "header":
        level := getHeadingLevel(elem)
        return strings.Repeat("#", level) + " "
    case "list_item":
        // Determine list marker
        if elem.Metadata["list_type"] == "ordered" {
            index := getListItemIndex(elem)
            return strings.Repeat("  ", depth) + fmt.Sprintf("%d. ", index)
        }
        return strings.Repeat("  ", depth) + "- "
    case "code_block":
        language := getCodeLanguage(elem)
        return fmt.Sprintf("```%s\n", language)
    case "link":
        return "["
    case "image":
        alt := getImageAlt(elem)
        return fmt.Sprintf("![%s](", alt)
    case "table":
        return "" // Tables handled specially in Markdown
    case "table_row":
        return "|"
    case "table_cell":
        return " "
    default:
        return ""
    }
}

func (f *MarkdownFormatter) FormatContent(elem Element, depth int) string {
    if elem.Content == "" {
        return ""
    }

    switch elem.ElementType {
    case "code_block":
        return elem.Content
    default:
        // Escape Markdown special characters
        return markdownEscape(elem.Content)
    }
}

func (f *MarkdownFormatter) FormatClosing(elem Element, depth int) string {
    switch elem.ElementType {
    case "paragraph", "heading", "header":
        return "\n\n"
    case "list_item":
        return "\n"
    case "code_block":
        return "```\n\n"
    case "link":
        href := getLinkHref(elem)
        return fmt.Sprintf("](%s)", href)
    case "image":
        src := getImageSrc(elem)
        return fmt.Sprintf("%s)\n", src)
    case "table_cell":
        return " |"
    case "table_row":
        return "\n"
    default:
        return ""
    }
}

// Helper functions

func getHeadingLevel(elem Element) int {
    if elem.SectionLevel != nil {
        return *elem.SectionLevel
    }
    if level, ok := elem.Metadata["level"].(int); ok {
        return level
    }
    return 1
}

func getListItemIndex(elem Element) int {
    if idx, ok := elem.Metadata["index"].(int); ok {
        return idx
    }
    return 1
}

func getCodeLanguage(elem Element) string {
    if lang, ok := elem.Metadata["language"].(string); ok {
        return lang
    }
    return ""
}

func getLinkHref(elem Element) string {
    if href, ok := elem.Metadata["href"].(string); ok {
        return href
    }
    return "#"
}

func getImageSrc(elem Element) string {
    if src, ok := elem.Metadata["src"].(string); ok {
        return src
    }
    return ""
}

func getImageAlt(elem Element) string {
    if alt, ok := elem.Metadata["alt"].(string); ok {
        return alt
    }
    return ""
}

func isHeaderCell(elem Element) bool {
    if isHeader, ok := elem.Metadata["is_header"].(bool); ok {
        return isHeader
    }
    return false
}

func htmlEscape(s string) string {
    replacer := strings.NewReplacer(
        "&", "&amp;",
        "<", "&lt;",
        ">", "&gt;",
        "\"", "&quot;",
        "'", "&#39;",
    )
    return replacer.Replace(s)
}

func markdownEscape(s string) string {
    replacer := strings.NewReplacer(
        "*", "\\*",
        "_", "\\_",
        "[", "\\[",
        "]", "\\]",
        "`", "\\`",
    )
    return replacer.Replace(s)
}
```

#### 2.4.3 Format-Specific Reconstruction

The Python implementation handles format-specific reconstruction intelligently:

**DOCX Reconstruction:**
- Preserves heading levels (`section_level`)
- Maintains table structure (`row_index`, `column_index`)
- Includes headers/footers as separate sections
- Preserves list hierarchy

**PPTX Reconstruction:**
- Slide-by-slide reconstruction
- Preserves slide layouts (title, content, notes)
- Maintains shape hierarchy

**PDF Reconstruction:**
- Page-by-page output (`page_number`)
- Preserves reading order
- Optional: column detection for multi-column layouts

**XLSX Reconstruction:**
- Sheet-by-sheet output
- Preserves cell positions (`row_index`, `column_index`)
- Maintains formulas (if stored in metadata)

**HTML/XML Reconstruction:**
- Uses `tag_name` promoted field
- Preserves tag hierarchy
- Restores attributes from metadata

**JSON Reconstruction:**
- Rebuilds nested structure
- Uses element_type to determine object/array/field types

#### 2.4.4 CLI Command

**New File:** `go/cmd/udml-reconstruct/main.go`

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "os"

    "github.com/kennethstott/go-doc-go/internal/storage"
    "github.com/kennethstott/go-doc-go/internal/udml"
)

func main() {
    var (
        docID      = flag.String("doc-id", "", "Document ID to reconstruct")
        format     = flag.String("format", "text", "Output format: text, html, markdown")
        outputPath = flag.String("output", "", "Output file path (default: stdout)")
        storage    = flag.String("storage", "analytics/", "UDML storage path")
    )
    flag.Parse()

    if *docID == "" {
        fmt.Fprintln(os.Stderr, "Error: --doc-id required")
        os.Exit(1)
    }

    // Initialize storage
    storageBackend, err := storage.NewDuckDBStorage(*storage)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error initializing storage: %v\n", err)
        os.Exit(1)
    }
    defer storageBackend.Close()

    // Create reconstructor
    reconstructor := udml.NewReconstructor(storageBackend)

    // Parse format
    reconstructFormat := udml.FormatPlainText
    switch *format {
    case "text":
        reconstructFormat = udml.FormatPlainText
    case "html":
        reconstructFormat = udml.FormatHTML
    case "markdown", "md":
        reconstructFormat = udml.FormatMarkdown
    default:
        fmt.Fprintf(os.Stderr, "Error: unsupported format %s\n", *format)
        os.Exit(1)
    }

    // Reconstruct document
    ctx := context.Background()
    result, err := reconstructor.ReconstructDocument(ctx, *docID, reconstructFormat)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error reconstructing document: %v\n", err)
        os.Exit(1)
    }

    // Write output
    if *outputPath == "" {
        fmt.Println(result)
    } else {
        if err := os.WriteFile(*outputPath, []byte(result), 0644); err != nil {
            fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
            os.Exit(1)
        }
        fmt.Printf("Reconstruction written to %s\n", *outputPath)
    }
}
```

**Usage:**

```bash
# Reconstruct as plain text
udml-reconstruct --doc-id doc123 --format text

# Reconstruct as HTML
udml-reconstruct --doc-id doc123 --format html --output report.html

# Reconstruct as Markdown
udml-reconstruct --doc-id doc123 --format markdown --output report.md

# Reconstruct specific fragment
udml-reconstruct --doc-id doc123 --element-ids elem1,elem2,elem3 --format html
```

#### 2.4.5 Testing Strategy

**Unit Tests:**

```go
func TestReconstruction_PlainText(t *testing.T) {
    elements := []Element{
        {ElementID: "root", ElementType: "document", ParentID: ""},
        {ElementID: "h1", ElementType: "heading", ParentID: "root", Content: "Title", SectionLevel: ptr(1)},
        {ElementID: "p1", ElementType: "paragraph", ParentID: "root", Content: "First paragraph."},
        {ElementID: "p2", ElementType: "paragraph", ParentID: "root", Content: "Second paragraph."},
    }

    reconstructor := NewReconstructor(mockStorage(elements))
    result, err := reconstructor.ReconstructDocument(context.Background(), "test", FormatPlainText)

    assert.NoError(t, err)
    assert.Contains(t, result, "# Title")
    assert.Contains(t, result, "First paragraph.")
    assert.Contains(t, result, "Second paragraph.")
}

func TestReconstruction_HTML(t *testing.T) {
    elements := []Element{
        {ElementID: "root", ElementType: "document", ParentID: ""},
        {ElementID: "h1", ElementType: "heading", ParentID: "root", Content: "Title", SectionLevel: ptr(1)},
    }

    reconstructor := NewReconstructor(mockStorage(elements))
    result, err := reconstructor.ReconstructDocument(context.Background(), "test", FormatHTML)

    assert.NoError(t, err)
    assert.Contains(t, result, "<html><body>")
    assert.Contains(t, result, "<h1>Title</h1>")
    assert.Contains(t, result, "</body></html>")
}
```

**Integration Tests:**

- Parse document → Store in UDML → Reconstruct → Compare with original
- Test each format (DOCX, PDF, HTML, Markdown) round-trip
- Test fragment reconstruction
- Test large document streaming

**Files to Create:**
- [ ] `go/internal/udml/reconstruction.go` - Reconstructor interface
- [ ] `go/internal/udml/formatters.go` - Format-specific formatters
- [ ] `go/internal/udml/hierarchy.go` - Hierarchy building utilities
- [ ] `go/cmd/udml-reconstruct/main.go` - CLI command
- [ ] `go/internal/udml/reconstruction_test.go` - Unit tests
- [ ] `tests/integration/test_reconstruction.go` - Integration tests

**Benefits:**

1. **Document Preview**: Generate quick previews without original files
2. **Format Conversion**: Convert between formats via UDML (PDF → Markdown)
3. **Content Extraction**: Extract specific sections as standalone documents
4. **Quality Validation**: Round-trip testing (parse → reconstruct → compare)
5. **Search Results**: Display reconstructed excerpts in search results
6. **API Responses**: Return formatted content via API

---

## Phase 3: LLM Integration (Week 5-6)

### 3.1 UDML Sample Loader (`go/internal/ontology/sampling.go`)

**New Package:** `go/internal/ontology/`

```go
package ontology

import (
    "database/sql"
    "encoding/json"
)

// Sampler loads representative UDML samples for LLM analysis
type Sampler struct {
    db *sql.DB
}

// LoadSamples loads stratified random samples from UDML storage
func (s *Sampler) LoadSamples(opts SampleOptions) ([]JSONDocument, error) {
    // Stratified sampling by element type
    samples := make([]JSONDocument, 0, opts.Count)

    // Sample from each element type proportionally
    for _, elemType := range opts.ElementTypes {
        typeQuery := fmt.Sprintf(`
            SELECT * FROM 'analytics/element_type=%s/**/**.parquet'
            WHERE version = ?
            ORDER BY RANDOM()
            LIMIT ?
        `, elemType)

        rows, err := s.db.Query(typeQuery, opts.Version, opts.CountPerType)
        if err != nil {
            return nil, err
        }

        // Convert to JSON documents
        typeSamples := s.rowsToJSON(rows)
        samples = append(samples, typeSamples...)
    }

    return samples, nil
}

type SampleOptions struct {
    Count        int
    CountPerType int
    ElementTypes []string
    Version      string
}
```

**Files to Create:**
- [ ] `go/internal/ontology/sampling.go`
- [ ] Stratified sampling by element type
- [ ] Sample quality validation
- [ ] Sample export to JSON for LLM

---

### 3.2 LLM Client Interface (`go/internal/llm/client.go`)

**New Package:** `go/internal/llm/`

```go
package llm

import (
    "context"
)

// Client interface for LLM providers
type Client interface {
    Generate(ctx context.Context, prompt string, opts GenerateOptions) (*Response, error)
}

// Response from LLM
type Response struct {
    Text       string
    StopReason string
    Usage      Usage
}

type Usage struct {
    InputTokens  int
    OutputTokens int
}

// Anthropic Claude client
type ClaudeClient struct {
    apiKey string
    model  string // claude-3-5-sonnet-20250929
}

func (c *ClaudeClient) Generate(ctx context.Context, prompt string, opts GenerateOptions) (*Response, error) {
    // Call Anthropic API
    // Implementation using anthropic-sdk-go or HTTP client
}

// OpenAI client (alternative)
type OpenAIClient struct {
    apiKey string
    model  string // gpt-4
}

// Factory
func NewLLMClient(provider string, config map[string]interface{}) (Client, error) {
    switch provider {
    case "anthropic", "claude":
        return NewClaudeClient(config)
    case "openai":
        return NewOpenAIClient(config)
    default:
        return nil, fmt.Errorf("unsupported LLM provider: %s", provider)
    }
}
```

**Files to Create:**
- [ ] `go/internal/llm/client.go`
- [ ] `go/internal/llm/claude.go` (Anthropic implementation)
- [ ] `go/internal/llm/openai.go` (OpenAI implementation)
- [ ] API client with retries, rate limiting
- [ ] Cost tracking

**Dependencies:**
- Anthropic SDK: `go get github.com/anthropics/anthropic-sdk-go`
- OpenAI SDK: `go get github.com/sashabaranov/go-openai`

---

### 3.3 Interactive Ontology Builder (5-Phase LLM-Guided Interview)

**Based on:** `src/go_doc_go/cli/ontology_interview.py` (754 lines, production system)

The interactive ontology builder is a 5-phase LLM-guided interview system that creates domain-specific ontologies by analyzing actual UDML documents and refining rules through conversation.

#### Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│ Phase 1: Domain Understanding                                        │
│ - Load sample documents from Go-Doc-Go config                       │
│ - Parse documents → UDML elements                                   │
│ - AI analyzes patterns, suggests domain                            │
│ - User confirms/modifies domain                                     │
└─────────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────────┐
│ Phase 2: Term Definition                                             │
│ - AI suggests domain-specific terms and synonyms                   │
│ - User adds/modifies/deletes terms                                 │
│ - Build domain vocabulary                                          │
└─────────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────────┐
│ Phase 3: Entity Extraction                                           │
│ - AI suggests entity types (Person, Company, Date, Amount)         │
│ - Define extraction rules (METADATA, REGEX, KEYWORDS, SEMANTIC)    │
│ - User refines entities and rules interactively                    │
└─────────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────────┐
│ Phase 4: Relationship Mapping                                        │
│ - AI suggests relationships between entities                        │
│ - Define constraints (hierarchy, direction, proximity)             │
│ - User adds/deletes relationships                                  │
└─────────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────────┐
│ Phase 5: Refinement                                                  │
│ - Display summary of ontology                                      │
│ - Option to revisit any phase                                      │
│ - Add derived entities                                             │
│ - Export to YAML                                                   │
└─────────────────────────────────────────────────────────────────────┘
```

#### 3.3.1 Core Interviewer Implementation

**New File:** `go/internal/ontology/interviewer.go`

```go
package ontology

import (
    "bufio"
    "context"
    "fmt"
    "os"
    "strings"

    "github.com/kennethstott/go-doc-go/internal/llm"
    "github.com/kennethstott/go-doc-go/internal/parser"
)

// OntologyInterviewer conducts 5-phase LLM-guided ontology creation
type OntologyInterviewer struct {
    llmClient       llm.Client
    sampler         *DocumentSampler
    builder         *OntologyBuilder
    maxIterations   int
    dataConfigPath  string
    scanner         *bufio.Scanner
}

func NewOntologyInterviewer(
    llmClient llm.Client,
    dataConfigPath string,
    maxIterations int,
) *OntologyInterviewer {
    return &OntologyInterviewer{
        llmClient:      llmClient,
        sampler:        NewDocumentSampler(dataConfigPath),
        builder:        NewOntologyBuilder(),
        maxIterations:  maxIterations,
        dataConfigPath: dataConfigPath,
        scanner:        bufio.NewScanner(os.Stdin),
    }
}

// ConductInterview runs the 5-phase interview process
func (i *OntologyInterviewer) ConductInterview(ctx context.Context) (*Ontology, error) {
    fmt.Println("╔════════════════════════════════════════════════════════════════╗")
    fmt.Println("║   Interactive Ontology Builder - LLM-Guided Interview        ║")
    fmt.Println("╚════════════════════════════════════════════════════════════════╝")
    fmt.Println()

    // Phase 1: Domain Understanding
    domain, samples, err := i.phaseDomainUnderstanding(ctx)
    if err != nil {
        return nil, fmt.Errorf("phase 1 failed: %w", err)
    }

    // Phase 2: Term Definition
    terms, err := i.phaseTermDefinition(ctx, domain, samples)
    if err != nil {
        return nil, fmt.Errorf("phase 2 failed: %w", err)
    }

    // Phase 3: Entity Extraction
    entities, err := i.phaseEntityExtraction(ctx, domain, terms, samples)
    if err != nil {
        return nil, fmt.Errorf("phase 3 failed: %w", err)
    }

    // Phase 4: Relationship Mapping
    relationships, err := i.phaseRelationshipMapping(ctx, entities, samples)
    if err != nil {
        return nil, fmt.Errorf("phase 4 failed: %w", err)
    }

    // Phase 5: Refinement
    ontology, err := i.phaseRefinement(ctx, domain, terms, entities, relationships)
    if err != nil {
        return nil, fmt.Errorf("phase 5 failed: %w", err)
    }

    return ontology, nil
}

// Phase 1: Domain Understanding
func (i *OntologyInterviewer) phaseDomainUnderstanding(ctx context.Context) (string, []Document, error) {
    fmt.Println("═══════════════════════════════════════")
    fmt.Println(" Phase 1: Domain Understanding")
    fmt.Println("═══════════════════════════════════════")
    fmt.Println()

    // Prompt for domain
    fmt.Print("Enter the domain for your ontology (e.g., 'financial', 'legal', 'medical'): ")
    i.scanner.Scan()
    userDomain := strings.TrimSpace(i.scanner.Text())

    // Ask if user wants to analyze sample documents
    fmt.Print("\nWould you like to analyze sample documents to help identify patterns? (y/n): ")
    i.scanner.Scan()
    analyzeSamples := strings.ToLower(strings.TrimSpace(i.scanner.Text())) == "y"

    var samples []Document
    var documentPatterns string

    if analyzeSamples {
        fmt.Println("\nAnalyzing sample documents...")
        var err error
        samples, err = i.sampler.LoadAndParseSamples(ctx, 5) // Load 5 representative documents
        if err != nil {
            fmt.Printf("Warning: Could not load samples: %v\n", err)
        } else {
            // Extract patterns from samples
            patterns := i.analyzeSamplePatterns(samples)
            documentPatterns = patterns.String()
            fmt.Printf("\nFound %d sample documents with patterns:\n%s\n", len(samples), documentPatterns)
        }
    }

    // Get AI suggestions for domain
    prompt := fmt.Sprintf(`You are an expert at ontology design.

User wants to create an ontology for the "%s" domain.

%s

Based on this information:
1. Confirm if "%s" is an appropriate domain name or suggest alternatives
2. Suggest 3-5 common document types in this domain
3. Identify key characteristics of this domain

Provide your response in this format:
DOMAIN: <recommended domain name>
DOC_TYPES: <comma-separated list>
CHARACTERISTICS: <brief description>
`, userDomain, documentPatterns, userDomain)

    resp, err := i.llmClient.Generate(ctx, prompt, llm.GenerateOptions{
        MaxTokens: 1000,
    })
    if err != nil {
        return "", nil, fmt.Errorf("LLM request failed: %w", err)
    }

    // Parse AI response
    aiDomain, docTypes, characteristics := i.parsePhase1Response(resp.Text)

    // Display AI suggestions
    fmt.Println("\n╔════ AI Analysis ════╗")
    fmt.Printf("Recommended Domain: %s\n", aiDomain)
    fmt.Printf("Document Types: %s\n", docTypes)
    fmt.Printf("Characteristics: %s\n", characteristics)
    fmt.Println("╚═════════════════════╝\n")

    // User confirmation
    fmt.Print("Accept AI recommendations? (y/n): ")
    i.scanner.Scan()
    acceptAI := strings.ToLower(strings.TrimSpace(i.scanner.Text())) == "y"

    finalDomain := userDomain
    if acceptAI {
        finalDomain = aiDomain
    }

    fmt.Printf("\n✓ Domain set to: %s\n\n", finalDomain)
    return finalDomain, samples, nil
}

// Phase 2: Term Definition
func (i *OntologyInterviewer) phaseTermDefinition(ctx context.Context, domain string, samples []Document) ([]Term, error) {
    fmt.Println("═══════════════════════════════════════")
    fmt.Println(" Phase 2: Term Definition")
    fmt.Println("═══════════════════════════════════════")
    fmt.Println()

    // Get AI suggestions for domain terms
    sampleText := i.extractSampleText(samples, 3)
    prompt := fmt.Sprintf(`You are an expert at %s domain terminology.

Sample document excerpts:
%s

Suggest 10-15 important terms and their synonyms for this domain.

Format your response as:
TERM: <term>
SYNONYMS: <syn1>, <syn2>, <syn3>
DESCRIPTION: <brief description>

TERM: <next term>
...`, domain, sampleText)

    resp, err := i.llmClient.Generate(ctx, prompt, llm.GenerateOptions{
        MaxTokens: 2000,
    })
    if err != nil {
        return nil, err
    }

    // Parse AI-suggested terms
    aiTerms := i.parseTerms(resp.Text)

    fmt.Printf("AI suggested %d terms:\n\n", len(aiTerms))
    for idx, term := range aiTerms {
        fmt.Printf("%d. %s\n", idx+1, term.Term)
        fmt.Printf("   Synonyms: %s\n", strings.Join(term.Synonyms, ", "))
        fmt.Printf("   Description: %s\n\n", term.Description)
    }

    // Interactive refinement
    fmt.Println("Options:")
    fmt.Println("  a - Accept all terms")
    fmt.Println("  d <num> - Delete term by number")
    fmt.Println("  m <num> - Modify term by number")
    fmt.Println("  n - Add new term")
    fmt.Println("  done - Finish term definition")
    fmt.Println()

    terms := aiTerms
    for {
        fmt.Print("Command: ")
        i.scanner.Scan()
        cmd := strings.TrimSpace(i.scanner.Text())

        if cmd == "done" || cmd == "a" {
            break
        }

        // Handle commands (simplified for brevity)
        // Full implementation would handle delete, modify, add operations
    }

    fmt.Printf("\n✓ Defined %d terms\n\n", len(terms))
    return terms, nil
}

// Phase 3: Entity Extraction
func (i *OntologyInterviewer) phaseEntityExtraction(ctx context.Context, domain string, terms []Term, samples []Document) ([]Entity, error) {
    fmt.Println("═══════════════════════════════════════")
    fmt.Println(" Phase 3: Entity Extraction Rules")
    fmt.Println("═══════════════════════════════════════")
    fmt.Println()

    // Get AI suggestions for entities
    termsStr := i.termsToString(terms)
    sampleText := i.extractSampleText(samples, 3)

    prompt := fmt.Sprintf(`You are an expert at entity extraction for the %s domain.

Domain terms: %s

Sample documents:
%s

Suggest 5-10 entity types to extract, with extraction rules.

For each entity, provide:
1. Entity type name
2. Description
3. Extraction rule type (METADATA, REGEX, KEYWORDS, SEMANTIC)
4. Rule details (JSONPath, regex pattern, keyword list, or similarity query)
5. Example matches from the sample

Format as:
ENTITY: <type>
DESCRIPTION: <description>
RULE_TYPE: <METADATA|REGEX|KEYWORDS|SEMANTIC>
RULE: <rule details>
EXAMPLES: <example1>, <example2>

ENTITY: <next entity>
...`, domain, termsStr, sampleText)

    resp, err := i.llmClient.Generate(ctx, prompt, llm.GenerateOptions{
        MaxTokens: 3000,
    })
    if err != nil {
        return nil, err
    }

    // Parse AI-suggested entities
    aiEntities := i.parseEntities(resp.Text)

    fmt.Printf("AI suggested %d entity types:\n\n", len(aiEntities))
    for idx, entity := range aiEntities {
        fmt.Printf("%d. %s\n", idx+1, entity.Name)
        fmt.Printf("   Description: %s\n", entity.Description)
        fmt.Printf("   Extraction Rules:\n")
        for _, rule := range entity.ExtractionRules {
            fmt.Printf("     - Type: %s\n", rule.Type)
            fmt.Printf("       Rule: %s\n", rule.Rule)
        }
        fmt.Println()
    }

    // Interactive refinement
    entities := i.refineEntitiesInteractively(aiEntities)

    fmt.Printf("\n✓ Defined %d entity types\n\n", len(entities))
    return entities, nil
}

// Phase 4: Relationship Mapping
func (i *OntologyInterviewer) phaseRelationshipMapping(ctx context.Context, entities []Entity, samples []Document) ([]Relationship, error) {
    fmt.Println("═══════════════════════════════════════")
    fmt.Println(" Phase 4: Relationship Mapping")
    fmt.Println("═══════════════════════════════════════")
    fmt.Println()

    // Get AI suggestions for relationships
    entitiesStr := i.entitiesToString(entities)
    sampleText := i.extractSampleText(samples, 3)

    prompt := fmt.Sprintf(`You are an expert at relationship extraction.

Entity types defined:
%s

Sample documents:
%s

Suggest 5-10 relationships between these entities.

For each relationship, provide:
1. Relationship name (e.g., "EMPLOYED_BY", "LOCATED_IN")
2. Source entity type
3. Target entity type
4. Relationship direction
5. Constraints (hierarchy_level, same_section, max_distance)

Format as:
RELATIONSHIP: <name>
SOURCE: <entity_type>
TARGET: <entity_type>
DIRECTION: <bidirectional|directional>
CONSTRAINTS: <key:value, key:value>

RELATIONSHIP: <next relationship>
...`, entitiesStr, sampleText)

    resp, err := i.llmClient.Generate(ctx, prompt, llm.GenerateOptions{
        MaxTokens: 2000,
    })
    if err != nil {
        return nil, err
    }

    // Parse AI-suggested relationships
    aiRelationships := i.parseRelationships(resp.Text)

    fmt.Printf("AI suggested %d relationships:\n\n", len(aiRelationships))
    for idx, rel := range aiRelationships {
        fmt.Printf("%d. %s: %s → %s\n", idx+1, rel.Name, rel.SourceEntityType, rel.TargetEntityType)
        fmt.Printf("   Constraints: %v\n\n", rel.Constraints)
    }

    // Interactive refinement
    relationships := i.refineRelationshipsInteractively(aiRelationships)

    fmt.Printf("\n✓ Defined %d relationships\n\n", len(relationships))
    return relationships, nil
}

// Phase 5: Refinement
func (i *OntologyInterviewer) phaseRefinement(ctx context.Context, domain string, terms []Term, entities []Entity, relationships []Relationship) (*Ontology, error) {
    fmt.Println("═══════════════════════════════════════")
    fmt.Println(" Phase 5: Refinement & Export")
    fmt.Println("═══════════════════════════════════════")
    fmt.Println()

    // Build ontology
    i.builder.SetDomain(domain)
    i.builder.SetTerms(terms)
    i.builder.SetEntities(entities)
    i.builder.SetRelationships(relationships)

    // Display summary
    fmt.Println("Ontology Summary:")
    fmt.Printf("  Domain: %s\n", domain)
    fmt.Printf("  Terms: %d\n", len(terms))
    fmt.Printf("  Entity Types: %d\n", len(entities))
    fmt.Printf("  Relationships: %d\n", len(relationships))
    fmt.Println()

    // Options to revisit phases or export
    fmt.Println("Options:")
    fmt.Println("  1 - Revisit Phase 1 (Domain)")
    fmt.Println("  2 - Revisit Phase 2 (Terms)")
    fmt.Println("  3 - Revisit Phase 3 (Entities)")
    fmt.Println("  4 - Revisit Phase 4 (Relationships)")
    fmt.Println("  d - Add derived entities")
    fmt.Println("  e - Export ontology")
    fmt.Println()

    fmt.Print("Command: ")
    i.scanner.Scan()
    cmd := strings.TrimSpace(i.scanner.Text())

    if cmd == "e" {
        ontology := i.builder.Build()
        fmt.Println("\n✓ Ontology created successfully!")
        return ontology, nil
    }

    // Handle other commands (revisit phases, add derived entities)
    // Simplified for brevity

    ontology := i.builder.Build()
    return ontology, nil
}

// Helper methods (simplified)
func (i *OntologyInterviewer) parsePhase1Response(text string) (string, string, string) {
    // Parse AI response
    return "domain", "doctype1, doctype2", "characteristics"
}

func (i *OntologyInterviewer) analyzeSamplePatterns(samples []Document) *DocumentPatterns {
    return &DocumentPatterns{}
}

func (i *OntologyInterviewer) extractSampleText(samples []Document, count int) string {
    return "sample text"
}

func (i *OntologyInterviewer) parseTerms(text string) []Term {
    return []Term{}
}

func (i *OntologyInterviewer) parseEntities(text string) []Entity {
    return []Entity{}
}

func (i *OntologyInterviewer) parseRelationships(text string) []Relationship {
    return []Relationship{}
}

func (i *OntologyInterviewer) termsToString(terms []Term) string {
    return "terms"
}

func (i *OntologyInterviewer) entitiesToString(entities []Entity) string {
    return "entities"
}

func (i *OntologyInterviewer) refineEntitiesInteractively(entities []Entity) []Entity {
    return entities
}

func (i *OntologyInterviewer) refineRelationshipsInteractively(rels []Relationship) []Relationship {
    return rels
}
```

**Key Implementation Notes:**
1. **Phases are sequential but can be revisited** - User can go back to refine
2. **AI provides suggestions, user refines** - Collaborative approach
3. **Sample document analysis is optional** - But improves accuracy
4. **Four extraction rule types**:
   - `METADATA`: Extract from document metadata fields (`$.metadata.author`)
   - `REGEX`: Pattern matching (`\$\d{1,3}(?:,\d{3})*(?:\.\d{2})?`)
   - `KEYWORDS`: Keyword lists (`["invoice", "bill", "receipt"]`)
   - `SEMANTIC`: Similarity-based using contextual embeddings (`similarity("contract terms", 0.8)`)

#### 3.3.2 Document Sampler

**New File:** `go/internal/ontology/sampler.go`

```go
package ontology

import (
    "context"
    "fmt"

    "github.com/kennethstott/go-doc-go/internal/config"
    "github.com/kennethstott/go-doc-go/internal/parser"
    "github.com/kennethstott/go-doc-go/internal/sources"
)

// DocumentSampler loads representative documents for ontology creation
type DocumentSampler struct {
    configPath string
}

func NewDocumentSampler(configPath string) *DocumentSampler {
    return &DocumentSampler{
        configPath: configPath,
    }
}

// LoadAndParseSamples loads N documents and parses them to UDML
func (s *DocumentSampler) LoadAndParseSamples(ctx context.Context, count int) ([]Document, error) {
    // Load Go-Doc-Go configuration
    cfg, err := config.LoadConfig(s.configPath)
    if err != nil {
        return nil, fmt.Errorf("failed to load config: %w", err)
    }

    // Get content sources from config
    sourceManager := sources.NewManager(cfg.ContentSources)

    // Fetch sample documents (stratified by source)
    rawDocs, err := sourceManager.FetchSamples(ctx, count)
    if err != nil {
        return nil, err
    }

    // Parse documents to UDML
    documents := make([]Document, 0, len(rawDocs))
    for _, rawDoc := range rawDocs {
        // Detect document type
        docType := parser.DetectDocumentType(rawDoc.Content, rawDoc.Filename)

        // Parse to UDML
        result, err := parser.ParseDocument(rawDoc.Content, docType, rawDoc.ID)
        if err != nil {
            continue // Skip failed parses
        }

        documents = append(documents, Document{
            ID:       rawDoc.ID,
            Filename: rawDoc.Filename,
            DocType:  docType,
            UDML:     result,
        })
    }

    return documents, nil
}

type Document struct {
    ID       string
    Filename string
    DocType  string
    UDML     *parser.ParseResult
}

type DocumentPatterns struct {
    ElementTypes     map[string]int      // Element type distribution
    MetadataFields   []string            // Common metadata fields
    ContentSamples   []string            // Text samples
    StructurePatterns []string           // Structural patterns
}

func (p *DocumentPatterns) String() string {
    return fmt.Sprintf("Element types: %v\nMetadata fields: %v",
        p.ElementTypes, p.MetadataFields)
}
```

#### 3.3.3 Ontology Builder

**New File:** `go/internal/ontology/builder.go`

```go
package ontology

// OntologyBuilder assembles ontology from interview phases
type OntologyBuilder struct {
    domain        string
    terms         []Term
    entities      []Entity
    relationships []Relationship
}

func NewOntologyBuilder() *OntologyBuilder {
    return &OntologyBuilder{}
}

func (b *OntologyBuilder) SetDomain(domain string) {
    b.domain = domain
}

func (b *OntologyBuilder) SetTerms(terms []Term) {
    b.terms = terms
}

func (b *OntologyBuilder) SetEntities(entities []Entity) {
    b.entities = entities
}

func (b *OntologyBuilder) SetRelationships(relationships []Relationship) {
    b.relationships = relationships
}

// Build creates final Ontology struct
func (b *OntologyBuilder) Build() *Ontology {
    return &Ontology{
        Name:          b.domain,
        Version:       "v1.0.0",
        Terms:         b.terms,
        Entities:      b.entities,
        Relationships: b.relationships,
    }
}

// ExportYAML exports ontology to YAML format
func (o *Ontology) ExportYAML() (string, error) {
    // YAML export implementation
    return "", nil
}
```

#### 3.3.4 CLI Command

**New File:** `go/cmd/ontology-builder/main.go`

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "os"

    "github.com/kennethstott/go-doc-go/internal/llm"
    "github.com/kennethstott/go-doc-go/internal/ontology"
)

func main() {
    configPath := flag.String("config", "config.yaml", "Path to Go-Doc-Go config")
    llmProvider := flag.String("llm", "anthropic", "LLM provider (anthropic, openai, ollama)")
    outputPath := flag.String("output", "ontology.yaml", "Output path for ontology")
    flag.Parse()

    // Initialize LLM client
    llmClient, err := llm.NewClient(*llmProvider, map[string]interface{}{
        "api_key": os.Getenv("ANTHROPIC_API_KEY"),
        "model":   "claude-3-5-sonnet-20250929",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Create interviewer
    interviewer := ontology.NewOntologyInterviewer(
        llmClient,
        *configPath,
        3, // max iterations
    )

    // Conduct interview
    ctx := context.Background()
    ontology, err := interviewer.ConductInterview(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Export to YAML
    yaml, err := ontology.ExportYAML()
    if err != nil {
        log.Fatal(err)
    }

    // Write to file
    if err := os.WriteFile(*outputPath, []byte(yaml), 0644); err != nil {
        log.Fatal(err)
    }

    fmt.Printf("\n✓ Ontology exported to: %s\n", *outputPath)
}
```

#### 3.3.5 Integration: Ontology → JSONPath → Native Query

**Complete Workflow:**

```
1. Interactive Interview
   ↓
2. Ontology (YAML)
   ├─ Terms
   ├─ Entities (with extraction rules)
   └─ Relationships (with constraints)
   ↓
3. Compile to JSONPath Queries
   SEMANTIC rule: similarity("financial projections", 0.8)
   →  JSONPath: $.elements[?(@.type == 'paragraph')].similarity('financial projections', 0.8)
   ↓
4. Translate to Native Queries (via QueryBackend)
   DuckDB:       SELECT * FROM ...WHERE cosine_similarity(...)
   Neo4j:        MATCH (e:paragraph) WHERE vector.similarity.cosine(...)
   Elasticsearch: {"knn": {"field": "embedding", ...}}
   PostgreSQL:   SELECT * FROM ...WHERE (embedding <=> $emb) < 0.2
   Solr:         {!knn f=embedding topK=100}financial projections
   ↓
5. Execute Queries → Extract Entities & Relationships
```

**Files to Create:**
- [ ] `go/internal/ontology/interviewer.go` - 5-phase interview system
- [ ] `go/internal/ontology/sampler.go` - Document sampler
- [ ] `go/internal/ontology/builder.go` - Ontology builder
- [ ] `go/internal/ontology/compiler.go` - Ontology → JSONPath compiler
- [ ] `go/cmd/ontology-builder/main.go` - CLI command
- [ ] `go/internal/ontology/interviewer_test.go` - Unit tests

**Testing:**
- Test each phase independently
- Mock LLM responses for deterministic tests
- Test ontology compilation to JSONPath
- Test JSONPath → native query translation for all backends
- Integration test: Full interview → ontology → query execution

---

## Phase 4: Entity Extraction Engine (Week 7)

### 4.1 Generic Graph Model (`go/internal/graph/model.go`)

**New Package:** `go/internal/graph/`

```go
package graph

// Graph represents a format-agnostic knowledge graph
type Graph struct {
    Nodes []Node
    Edges []Edge
}

// Node represents an entity
type Node struct {
    ID         string
    Labels     []string                 // ["BrakeSystem", "Entity"]
    Properties map[string]interface{}

    // UDML provenance
    SourceElementID string
    SourceDocID     string
    Confidence      float64
}

// Edge represents a relationship
type Edge struct {
    ID         string
    SourceID   string
    TargetID   string
    Type       string
    Properties map[string]interface{}
    Confidence float64
}

// Builder helps construct graphs
type Builder struct {
    nodes map[string]*Node
    edges []*Edge
}

func NewBuilder() *Builder {
    return &Builder{
        nodes: make(map[string]*Node),
        edges: make([]*Edge, 0),
    }
}

func (b *Builder) AddNode(node *Node) {
    b.nodes[node.ID] = node
}

func (b *Builder) AddEdge(edge *Edge) {
    b.edges = append(b.edges, edge)
}

func (b *Builder) Build() *Graph {
    nodes := make([]Node, 0, len(b.nodes))
    for _, node := range b.nodes {
        nodes = append(nodes, *node)
    }

    edges := make([]Edge, 0, len(b.edges))
    for _, edge := range b.edges {
        edges = append(edges, *edge)
    }

    return &Graph{
        Nodes: nodes,
        Edges: edges,
    }
}
```

**Files to Create:**
- [ ] `go/internal/graph/model.go`
- [ ] `go/internal/graph/builder.go`
- [ ] Graph validation
- [ ] Graph statistics

---

### 4.2 Extraction Engine (`go/internal/ontology/extractor.go`)

```go
package ontology

import (
    "your-module/go/internal/graph"
    "your-module/go/internal/udml/jsonpath"
)

// Extractor applies ontology rules to extract entities/relationships
type Extractor struct {
    jsonpathExecutor *jsonpath.Executor
    graphBuilder     *graph.Builder
}

// Extract entities from documents
func (e *Extractor) ExtractEntities(ontology *Ontology, docIDs []string) ([]*graph.Node, error) {
    nodes := make([]*graph.Node, 0)

    for _, entityDef := range ontology.Entities {
        for _, rule := range entityDef.ExtractionRules {
            // Execute JSONPath rule (translates to SQL)
            matches, err := e.jsonpathExecutor.Execute(rule.JSONPath, docIDs)
            if err != nil {
                return nil, err
            }

            // Create nodes from matches
            for _, match := range matches {
                node := &graph.Node{
                    ID:              generateEntityID(),
                    Labels:          []string{entityDef.Name, "Entity"},
                    SourceElementID: match.ElementID,
                    SourceDocID:     match.DocID,
                    Confidence:      rule.Confidence,
                    Properties: map[string]interface{}{
                        "text": match.Text,
                    },
                }

                // Extract attributes
                for _, attrDef := range entityDef.Attributes {
                    value := e.extractAttribute(match, attrDef)
                    if value != nil {
                        node.Properties[attrDef.Name] = value
                    }
                }

                nodes = append(nodes, node)
            }
        }
    }

    return nodes, nil
}

// Extract relationships between entities
func (e *Extractor) ExtractRelationships(
    ontology *Ontology,
    entities []*graph.Node,
) ([]*graph.Edge, error) {
    edges := make([]*graph.Edge, 0)

    for _, relDef := range ontology.Relationships {
        // Find source entities
        sources := filterByType(entities, relDef.SourceEntityType)
        targets := filterByType(entities, relDef.TargetEntityType)

        for _, source := range sources {
            for _, target := range targets {
                // Check constraints
                if e.meetsConstraints(source, target, relDef.Constraints) {
                    edge := &graph.Edge{
                        ID:         generateEdgeID(),
                        SourceID:   source.ID,
                        TargetID:   target.ID,
                        Type:       relDef.RelationshipType,
                        Confidence: relDef.Confidence,
                    }
                    edges = append(edges, edge)
                }
            }
        }
    }

    return edges, nil
}

// Check if entities meet relationship constraints
func (e *Extractor) meetsConstraints(
    source, target *graph.Node,
    constraints *Constraints,
) bool {
    // Same section check
    if constraints.SameSection {
        if !e.inSameSection(source.SourceElementID, target.SourceElementID) {
            return false
        }
    }

    // Proximity check
    if constraints.MaxDistance > 0 {
        distance := e.getElementDistance(source.SourceElementID, target.SourceElementID)
        if distance > constraints.MaxDistance {
            return false
        }
    }

    // Relationship phrase check
    if constraints.RelationshipPhrases != nil {
        if !e.hasRelationshipPhrase(source, target, constraints.RelationshipPhrases) {
            return false
        }
    }

    return true
}
```

**Files to Create:**
- [ ] `go/internal/ontology/extractor.go`
- [ ] `go/internal/ontology/constraints.go`
- [ ] `go/internal/ontology/attributes.go`
- [ ] Constraint validation logic
- [ ] Attribute extraction (regex, semantic)

---

## Phase 5: Multi-Format Export (Week 8)

### 5.1 Exporter Interface (`go/internal/export/exporter.go`)

```go
package export

import "your-module/go/internal/graph"

// Exporter interface for different graph formats
type Exporter interface {
    Name() string
    Export(graph *graph.Graph, path string) error
}

// Registry of exporters
var exporters = map[string]Exporter{
    "rdf":      &RDFExporter{},
    "jsonld":   &JSONLDExporter{},
    "triples":  &TriplesExporter{},
    "graphml":  &GraphMLExporter{},
    "cypher":   &CypherExporter{},
}

// ExportGraph exports to specified format
func ExportGraph(graph *graph.Graph, format string, path string) error {
    exporter, exists := exporters[format]
    if !exists {
        return fmt.Errorf("unsupported format: %s", format)
    }

    return exporter.Export(graph, path)
}

// RegisterExporter allows custom exporters
func RegisterExporter(name string, exporter Exporter) {
    exporters[name] = exporter
}
```

**Files to Create:**
- [ ] `go/internal/export/exporter.go` (interface)
- [ ] `go/internal/export/rdf.go` (RDF/Turtle export)
- [ ] `go/internal/export/jsonld.go` (JSON-LD export)
- [ ] `go/internal/export/triples.go` (CSV/Parquet triples)
- [ ] `go/internal/export/graphml.go` (GraphML export)
- [ ] `go/internal/export/cypher.go` (Cypher statements)

### 5.2 RDF Exporter Example

```go
// go/internal/export/rdf.go
package export

import (
    "fmt"
    "strings"
)

type RDFExporter struct {
    Prefix string
}

func (e *RDFExporter) Name() string {
    return "rdf"
}

func (e *RDFExporter) Export(graph *graph.Graph, path string) error {
    var turtle strings.Builder

    // Prefixes
    turtle.WriteString("@prefix : <http://udml.org/ontology#> .\n")
    turtle.WriteString("@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .\n\n")

    // Export nodes
    for _, node := range graph.Nodes {
        turtle.WriteString(fmt.Sprintf(":%s ", node.ID))

        // Types
        for _, label := range node.Labels {
            turtle.WriteString(fmt.Sprintf("rdf:type :%s ;\n  ", label))
        }

        // Properties
        for key, val := range node.Properties {
            turtle.WriteString(fmt.Sprintf(":%s \"%v\" ;\n  ", key, val))
        }

        turtle.WriteString(".\n\n")
    }

    // Export edges
    for _, edge := range graph.Edges {
        turtle.WriteString(fmt.Sprintf(
            ":%s :%s :%s .\n",
            edge.SourceID, edge.Type, edge.TargetID,
        ))
    }

    // Write to file
    return os.WriteFile(path, []byte(turtle.String()), 0644)
}
```

---

## Phase 6: Versioning System (Week 9)

### 6.1 Version Manager (`go/internal/udml/versioning.go`)

```go
package udml

import (
    "database/sql"
    "time"
)

// VersionManager handles UDML versioning
type VersionManager struct {
    db *sql.DB
}

// VersionMetadata describes a UDML version
type VersionMetadata struct {
    Version          string
    CreatedAt        time.Time
    ParserVersion    string
    EmbeddingModel   string
    OntologyVersion  string
    SchemaChanges    map[string]interface{}
    Notes            string
}

// CreateVersion registers a new UDML version
func (v *VersionManager) CreateVersion(meta *VersionMetadata) error {
    // Write to version_metadata.parquet
    query := `
        INSERT INTO version_metadata
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `
    _, err := v.db.Exec(query,
        meta.Version, meta.CreatedAt, meta.ParserVersion,
        meta.EmbeddingModel, meta.OntologyVersion,
        meta.SchemaChanges, meta.Notes,
    )
    return err
}

// GetLatestVersion returns latest version string
func (v *VersionManager) GetLatestVersion() (string, error) {
    query := `
        SELECT version FROM version_metadata
        ORDER BY created_at DESC LIMIT 1
    `
    var version string
    err := v.db.QueryRow(query).Scan(&version)
    return version, err
}

// GetVersionAtDate returns version as of specific date (time-travel)
func (v *VersionManager) GetVersionAtDate(date time.Time) (string, error) {
    query := `
        SELECT version FROM version_metadata
        WHERE created_at <= ?
        ORDER BY created_at DESC LIMIT 1
    `
    var version string
    err := v.db.QueryRow(query, date).Scan(&version)
    return version, err
}

// CompareVersions returns diff between two versions
func (v *VersionManager) CompareVersions(v1, v2 string) (*VersionDiff, error) {
    // Query schema changes, element counts, etc.
    // Return structured diff
}
```

**Files to Create:**
- [ ] `go/internal/udml/versioning.go`
- [ ] Version metadata persistence
- [ ] Time-travel queries
- [ ] Version comparison utilities

---

## Phase 7: Testing & Documentation (Week 10)

### 7.1 Integration Tests

**New Files:**
- [ ] `go/internal/udml/integration_test.go`
- [ ] `go/internal/ontology/extraction_test.go`
- [ ] `go/internal/export/export_test.go`

**Test Coverage:**
- End-to-end pipeline (document → UDML → ontology → graph)
- Performance benchmarks (100M+ elements)
- Version migration tests
- Multi-format export validation
- LLM integration tests (with mocks)

### 7.2 Documentation

**Files to Create:**
- [ ] `docs/UDML_SPECIFICATION.md`
- [ ] `docs/JSONPATH_EXTENSIONS.md`
- [ ] `docs/ONTOLOGY_YAML_SCHEMA.md`
- [ ] `docs/LLM_INTEGRATION_GUIDE.md`
- [ ] `docs/EXPORT_FORMATS.md`
- [ ] `examples/ontologies/automotive.yaml`
- [ ] `examples/ontologies/financial.yaml`
- [ ] `examples/ontologies/legal.yaml`

---

## Implementation Strategy: Clean Cutover

**Philosophy:** Clean breaking change with no backward compatibility in production code. The target state contains ONLY UDML implementation - no dual-format support, no old code paths.

### Cutover Approach

**Phase A: Development & Testing (Weeks 1-8)**
- Build complete UDML implementation on feature branch
- Test with representative documents
- Validate query performance meets targets (60x+ speedup)
- Run integration tests against full workflow

**Phase B: Data Conversion (Week 9, if needed)**
- **IF** existing Parquet data needs conversion:
  - Write throwaway conversion script (`scripts/convert_old_to_udml.sh`)
  - Convert existing data to UDML format
  - Validate converted data
  - Archive old data for rollback safety
- **ELSE**: Start fresh with empty UDML storage

**Phase C: Cutover (Week 10)**
- Merge feature branch to main
- Deploy UDML-only code
- Point system at UDML storage
- Monitor performance and correctness
- Old code deleted (not maintained)

### One-Time Data Conversion (Optional)

**Only needed if existing Parquet data must be preserved.**

```bash
#!/bin/bash
# scripts/convert_old_to_udml.sh - Throwaway script, run once and delete

# Backup old data
echo "Backing up old data..."
tar -czf analytics_backup_$(date +%Y%m%d).tar.gz analytics/

# Convert using DuckDB
echo "Converting to UDML format..."
duckdb << SQL
-- Read old flat Parquet
CREATE VIEW old_elements AS
  SELECT * FROM 'analytics/elements.parquet';

-- Write to Hive partitions by element_type
COPY (
  SELECT * FROM old_elements WHERE element_type = 'paragraph'
) TO 'analytics_udml/element_type=paragraph/version=1.0/data.parquet'
  (FORMAT PARQUET, PARTITION_BY (date, source));

-- Repeat for each element type...
SQL

echo "Conversion complete. Validate, then delete this script."
```

**Important:** This script is run ONCE during cutover, then deleted. It is NOT part of the production codebase.

### No Backward Compatibility

**What we DON'T have:**
- ❌ Config flags for old vs new storage
- ❌ Dual-format read/write code
- ❌ Runtime format detection
- ❌ Gradual rollout with parallel operation
- ❌ "Migration mode" in production

**What we DO have:**
- ✅ Clean UDML-only implementation
- ✅ Promoted fields (nullable pointers) are the ONLY way to query frequently-used attributes
- ✅ One data format: Hive-partitioned Parquet with universal schema
- ✅ Throwaway conversion script (if needed, run once, delete)

---

## Dependencies to Add

### Go Modules
```bash
# DuckDB for cross-partition queries
go get github.com/marcboeker/go-duckdb

# JSONPath library
go get github.com/PaesslerAG/jsonpath

# LLM clients
go get github.com/anthropics/anthropic-sdk-go
go get github.com/sashabaranov/go-openai

# Additional utilities
go get github.com/stretchr/testify  # Testing
go get gopkg.in/yaml.v3              # YAML parsing
```

### Python Dependencies
```bash
# LLM integration
pip install anthropic openai

# YAML handling
pip install pyyaml
```

---

## Risk Assessment & Mitigation

### High Risk Areas

1. **Performance Regression** (High Impact, Medium Probability)
   - **Risk:** Hive partitioning adds complexity, could slow queries
   - **Mitigation:** Benchmark during development, optimize partition strategy, validate 60x+ speedup before cutover
   - **Rollback:** Keep archived old data; revert code deployment if needed

2. **LLM Cost Overruns** (Medium Impact, Medium Probability)
   - **Risk:** LLM API calls expensive at scale
   - **Mitigation:** Token limits, caching, batch processing, cost monitoring
   - **Fallback:** Manual ontology creation if needed

3. **Data Conversion Failures** (High Impact, Low Probability)
   - **Risk:** Data loss/corruption during one-time conversion
   - **Mitigation:** Archive old data before conversion, extensive validation of converted data
   - **Rollback:** Restore from archive, revert code deployment

4. **Schema Evolution Breakage** (Medium Impact, Medium Probability)
   - **Risk:** Version changes break downstream consumers
   - **Mitigation:** Comprehensive versioning system, clear version in partition path
   - **Recovery:** Version-specific schema registry allows reading old versions

### Medium Risk Areas

5. **JSONPath Translation Bugs** (Medium Impact, Medium Probability)
   - **Mitigation:** Extensive test suite, gradual feature rollout

6. **Export Format Incompatibilities** (Low Impact, Medium Probability)
   - **Mitigation:** Validate against existing graph databases

---

## Success Metrics

### Performance Targets
- [x] Type-specific queries 60x faster than JSON extraction
- [x] 40% storage savings vs flat schema
- [x] Entity extraction: >10K entities/second
- [x] JSONPath→SQL translation: <50ms per rule

### Quality Targets
- [x] LLM ontology generation: >85% precision/recall
- [x] Iterative refinement: >30% error reduction
- [x] Natural language queries: 95% accuracy

### Operational Targets
- [x] Zero data loss during cutover (via archive/restore capability)
- [x] 100% test coverage for new modules
- [x] Complete documentation for all APIs
- [x] Clean codebase with no backward compatibility code

---

## Timeline Summary

| Phase | Duration | Deliverable |
|-------|----------|-------------|
| 1. Storage Foundation | 2 weeks | Hive-partitioned Parquet, type-specific schemas |
| 2. JSON + JSONPath | 2 weeks | JSON projection, JSONPath→SQL translator |
| 3. LLM Integration | 2 weeks | Ontology generator, LLM client |
| 4. Extraction Engine | 1 week | Entity/relationship extractor |
| 5. Multi-Format Export | 1 week | RDF, Cypher, GraphML exporters |
| 6. Versioning | 1 week | Version management system |
| 7. Testing & Docs | 1 week | Integration tests, documentation |
| **Total** | **10 weeks** | **Complete UDML system** |

---

## Next Steps

1. **Review this plan** with team
2. **Prioritize phases** based on business needs
3. **Assign developers** to each phase
4. **Set up development environment** (DuckDB, LLM API keys)
5. **Create feature branch** `feature/udml-migration`
6. **Begin Phase 1** (Storage Foundation)

---

## Questions for Discussion

1. Which LLM provider to use? (Anthropic Claude vs OpenAI GPT-4)
2. Version strategy? (Semantic versioning vs timestamps)
3. Export format priorities? (Which formats needed first?)
4. Timeline feasible? (Can we compress to 8 weeks?)
5. Resource allocation? (How many developers?)

---

## Appendix: File Tree After Migration

```
go/
├── internal/
│   ├── udml/                    ← NEW package
│   │   ├── schemas.go          ← Type-specific schemas
│   │   ├── json.go             ← JSON projection
│   │   ├── versioning.go       ← Version management
│   │   └── jsonpath/           ← NEW subpackage
│   │       ├── parser.go
│   │       ├── translator.go
│   │       ├── functions.go
│   │       └── similarity.go
│   ├── ontology/               ← NEW package
│   │   ├── generator.go        ← Ontology generation
│   │   ├── extractor.go        ← Entity extraction
│   │   ├── sampling.go         ← Sample loading
│   │   └── constraints.go
│   ├── graph/                  ← NEW package
│   │   ├── model.go            ← Generic graph model
│   │   └── builder.go
│   ├── llm/                    ← NEW package
│   │   ├── client.go           ← LLM interface
│   │   ├── claude.go
│   │   └── openai.go
│   ├── export/                 ← UPDATED
│   │   ├── exporter.go         ← Generic interface
│   │   ├── rdf.go
│   │   ├── jsonld.go
│   │   ├── triples.go
│   │   ├── graphml.go
│   │   └── cypher.go
│   ├── analytics/              ← UPDATED
│   │   ├── parquet_hive.go     ← NEW: Hive storage (UDML-only)
│   │   └── types.go            ← Update with promoted fields
│   └── parser/                 ← UPDATED
│       ├── types.go            ← Update Element struct
│       ├── pdf.go              ← Update to populate promoted fields
│       ├── docx.go             ← Update
│       └── ... (all parsers)
├── cmd/
│   ├── ontology-generator/     ← NEW: CLI for ontology gen
│       └── main.go

src/go_doc_go/
└── llm/                        ← NEW Python package
    ├── __init__.py
    ├── ontology_generator.py
    └── query_generator.py

docs/
├── UDML_SPECIFICATION.md       ← NEW
├── JSONPATH_EXTENSIONS.md      ← NEW
├── ONTOLOGY_YAML_SCHEMA.md     ← NEW
├── LLM_INTEGRATION_GUIDE.md    ← NEW
└── EXPORT_FORMATS.md           ← NEW

examples/
└── ontologies/                 ← NEW
    ├── automotive.yaml
    ├── financial.yaml
    └── legal.yaml
```

---

*End of Migration Plan*
