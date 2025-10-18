# MCP Server Storage Interface Refactoring

## Overview

This document describes the architectural refactoring of the MCP (Model Context Protocol) server to use the unified `analytics.Storage` interface instead of direct DuckDB backend access. This change ensures multi-backend support and proper temporal filtering across all corpus exploration operations.

## Motivation

### Previous Architecture (Problems)

**Direct Backend Coupling:**
```
MCP Server → query.RawQueryBackend → DuckDB SQL (direct)
```

**Issues:**
1. **Violated multi-backend principle**: MCP server was hardcoded to DuckDB via `query.RawQueryBackend`
2. **No abstraction layer**: Raw SQL queries in MCP implementation files
3. **Temporal filtering inconsistency**: Custom VIEW mechanism specific to DuckDB
4. **Code duplication risk**: Each backend would need separate MCP implementations
5. **Architectural violation**: User requirement: "everything must go through Storage. Nothing can be duckdb specific."

### New Architecture (Solution)

**Unified Storage Interface:**
```
MCP Server → analytics.Storage → Backend-specific implementations
                                 ├─ HiveParquetStorage (DuckDB)
                                 ├─ ParquetStorage (DuckDB)
                                 ├─ Neo4jStorage (future)
                                 └─ PostgresStorage (future)
```

**Benefits:**
1. ✅ **Multi-backend support**: MCP server works with any Storage implementation
2. ✅ **Consistent temporal filtering**: Standard `filters` parameter across all operations
3. ✅ **Clean abstraction**: Storage interface hides backend-specific details
4. ✅ **Code reuse**: Shared implementations for similar backends (DuckDB)
5. ✅ **Architectural compliance**: All data access goes through Storage interface

## Changes Made

### 1. Extended Storage Interface

**File**: `go/internal/analytics/types.go`

Added 8 new methods for corpus exploration:

```go
// Corpus Exploration Methods - for MCP server and interactive tools
// All methods support temporal filtering through the filters parameter

// SearchSemanticSimilarity performs semantic similarity search using vector embeddings
// filters: standard filters (source_name, doc_id, element_type, latest_only, as_of_date)
SearchSemanticSimilarity(queryVector []float64, filters map[string]interface{}, threshold float64, limit int) ([]SearchResult, error)

// SearchByRegex performs regex pattern matching on element content
SearchByRegex(pattern string, filters map[string]interface{}, limit int) ([]SearchResult, error)

// SearchByKeyword performs keyword search on element content
SearchByKeyword(keyword string, filters map[string]interface{}, limit int) ([]SearchResult, error)

// AnalyzePattern analyzes a regex pattern and returns statistics
AnalyzePattern(pattern string, filters map[string]interface{}, maxExamples int) (*PatternStats, error)

// ComputeTermFrequencies computes frequency statistics for given terms
ComputeTermFrequencies(terms []string, caseSensitive bool, filters map[string]interface{}) ([]TermFrequency, error)

// FindCooccurrences finds co-occurrences of two entities
FindCooccurrences(entity1, entity2 string, contextWindow string, filters map[string]interface{}, maxExamples int) (*CooccurrenceResult, error)

// GetElementContext retrieves an element with hierarchical context
GetElementContext(elementID string, filters map[string]interface{}, contextDepth int, includeSiblings, includeChildren bool) (*ElementContext, error)

// AggregateStatistics computes aggregate statistics about the corpus
AggregateStatistics(metrics []string, filters map[string]interface{}) (*CorpusStats, error)
```

**Key Design Decision**: All methods accept `filters map[string]interface{}` parameter for:
- `latest_only: true` - Temporal deduplication by doc_id
- `as_of_date: "YYYY-MM-DD"` - Point-in-time queries
- `source_name`, `doc_id`, `element_type` - Standard filtering

### 2. Implemented Methods in Storage Backends

**Files**:
- `go/internal/analytics/parquet_hive_mcp.go` - HiveParquetStorage implementations
- `go/internal/analytics/parquet_mcp.go` - ParquetStorage implementations
- `go/internal/analytics/mcp_queries.go` - Shared DuckDB implementations

**Architecture**: Both Parquet-based backends delegate to shared implementation functions in `mcp_queries.go` to avoid code duplication.

**Example - Semantic Search Implementation**:

```go
// parquet_hive_mcp.go
func (s *HiveParquetStorage) SearchSemanticSimilarity(queryVector []float64, filters map[string]interface{}, threshold float64, limit int) ([]SearchResult, error) {
    return searchSemanticSimilarityImpl(s.basePath, queryVector, filters, threshold, limit)
}

// mcp_queries.go (shared implementation)
func searchSemanticSimilarityImpl(basePath string, queryVector []float64, filters map[string]interface{}, threshold float64, limit int) ([]SearchResult, error) {
    db, err := sql.Open("duckdb", "")
    if err != nil {
        return nil, fmt.Errorf("failed to open DuckDB connection: %w", err)
    }
    defer db.Close()

    // Build elements CTE with temporal filtering
    elementsCTE := buildElementsCTE(basePath, filters)

    // Execute semantic similarity search
    query := fmt.Sprintf(`
        WITH elements AS (%s)
        SELECT e.element_id, e.doc_id, e.element_type, e.content_preview,
               list_cosine_similarity(emb.embedding, $1::FLOAT[]) as similarity
        FROM elements e
        JOIN '%s/embeddings/**/*.parquet' emb ON e.element_id = emb.element_id
        WHERE list_cosine_similarity(emb.embedding, $1::FLOAT[]) >= $2
        ORDER BY similarity DESC
        LIMIT $3
    `, elementsCTE, basePath)

    // Execute query and return results...
}
```

**Temporal Filtering Implementation**:

```go
func buildElementsCTE(basePath string, filters map[string]interface{}) string {
    var whereClauses []string

    // Apply standard filters
    if sourceName, ok := filters["source_name"].(string); ok {
        whereClauses = append(whereClauses, fmt.Sprintf("source = '%s'", sourceName))
    }
    if elementType, ok := filters["element_type"].(string); ok {
        whereClauses = append(whereClauses, fmt.Sprintf("element_type = '%s'", elementType))
    }

    // Build base query
    baseQuery := fmt.Sprintf(`
        SELECT *, ROW_NUMBER() OVER (PARTITION BY doc_id ORDER BY date DESC) as rn
        FROM read_parquet('%s/elements/**/*.parquet')
    `, basePath)

    if len(whereClauses) > 0 {
        baseQuery += " WHERE " + strings.Join(whereClauses, " AND ")
    }

    // Apply temporal filtering
    if latestOnly, ok := filters["latest_only"].(bool); ok && latestOnly {
        return fmt.Sprintf("SELECT * FROM (%s) WHERE rn = 1", baseQuery)
    }

    if asOfDate, ok := filters["as_of_date"].(string); ok {
        return fmt.Sprintf(`
            SELECT * FROM (%s)
            WHERE rn = 1 AND date <= '%s'
        `, baseQuery, asOfDate)
    }

    return baseQuery
}
```

### 3. Refactored MCP Server Implementation

**Files**:
- `go/internal/udml/ontology/mcp/server.go` - MCP server structure and tool registration
- `go/internal/udml/ontology/mcp/impl.go` - Tool implementations delegating to Storage

**Before**:
```go
type OntologyCorpusExplorer struct {
    backend      query.RawQueryBackend
    embGenerator embeddings.EmbeddingGenerator
}

func NewOntologyCorpusExplorer(backend query.QueryBackend, embGenerator embeddings.EmbeddingGenerator) (*OntologyCorpusExplorer, error) {
    rawBackend, ok := backend.(query.RawQueryBackend)
    if !ok {
        return nil, fmt.Errorf("MCP server requires RawQueryBackend")
    }
    return &OntologyCorpusExplorer{
        backend:      rawBackend,
        embGenerator: embGenerator,
    }, nil
}

// Tool implementation using raw SQL
func (e *OntologyCorpusExplorer) executeSemanticSearch(...) {
    sql := `SELECT ... FROM ... WHERE ...` // DuckDB-specific SQL
    results, err := e.backend.QueryRaw(sql, params)
    // Parse raw results...
}
```

**After**:
```go
type OntologyCorpusExplorer struct {
    storage      analytics.Storage
    embGenerator embeddings.EmbeddingGenerator
}

func NewOntologyCorpusExplorer(storage analytics.Storage, embGenerator embeddings.EmbeddingGenerator) (*OntologyCorpusExplorer, error) {
    if storage == nil {
        return nil, fmt.Errorf("storage cannot be nil")
    }
    return &OntologyCorpusExplorer{
        storage:      storage,
        embGenerator: embGenerator,
    }, nil
}

// Tool implementation using Storage interface
func (e *OntologyCorpusExplorer) executeSemanticSearch(ctx context.Context, query string, elementTypes []string, limit int, threshold float64) ([]analytics.SearchResult, error) {
    // Generate embedding
    queryEmb, err := e.embGenerator.Generate(query)
    if err != nil {
        return nil, fmt.Errorf("failed to generate query embedding: %w", err)
    }

    // Build filters
    filters := make(map[string]interface{})
    filters["latest_only"] = true // Temporal filtering enabled by default

    // Delegate to Storage interface
    results, err := e.storage.SearchSemanticSimilarity(queryEmb, filters, threshold, limit)
    if err != nil {
        return nil, fmt.Errorf("semantic search failed: %w", err)
    }

    // Filter by element type (in-memory for now)
    if len(elementTypes) > 0 {
        filtered := []analytics.SearchResult{}
        typeMap := make(map[string]bool)
        for _, t := range elementTypes {
            typeMap[t] = true
        }
        for _, result := range results {
            if typeMap[result.Element.ElementType] {
                filtered = append(filtered, result)
            }
        }
        return filtered, nil
    }

    return results, nil
}
```

**Key Changes**:
1. Replaced `query.RawQueryBackend` with `analytics.Storage`
2. Removed all raw SQL from MCP implementation files
3. Delegate to Storage interface methods
4. Standard temporal filtering via `filters` parameter
5. Clean separation of concerns

### 4. Updated MCP Server Entrypoint

**File**: `go/cmd/ontology_mcp/main.go`

**Before**:
```go
// Create query backend
backend := query.NewDuckDBBackend(parquetPath)

// Create MCP server
explorer, err := mcp.NewOntologyCorpusExplorer(backend, embGen)
```

**After**:
```go
// Create analytics storage (supports temporal filtering)
storageConfig := map[string]interface{}{
    "path": parquetPath,
}
storage, err := analytics.NewHiveParquetStorage(storageConfig)
if err != nil {
    log.Fatalf("Failed to create storage: %v", err)
}
defer storage.Close()

// Create MCP server
explorer, err := mcp.NewOntologyCorpusExplorer(storage, embGen)
```

### 5. Updated OntologyBuilder

**File**: `go/internal/udml/ontology/builder.go`

**Before**:
```go
import (
    "github.com/kennethstott/doculyzer-go-conversion/internal/udml/query"
)

type OntologyBuilder struct {
    sampler          *sampler.Sampler
    llmClient        LLMClient
    config           BuilderConfig
    substantiveTypes []string
    mcpServer        *mcp.OntologyCorpusExplorer
    queryBackend     query.QueryBackend  // Old approach
}

// NewOntologyBuilder creates MCP server with query backend
func NewOntologyBuilder(config BuilderConfig, llmClient LLMClient) (*OntologyBuilder, error) {
    // ...

    if config.EnableMCP {
        // Create query backend for MCP
        backendConfig := query.BackendConfig{
            Backend:     "duckdb",
            ParquetPath: config.ParquetPath,
        }
        queryBackend := query.NewDuckDBBackend(backendConfig)

        // Create MCP server
        mcpServer, err = mcp.NewOntologyCorpusExplorer(queryBackend, embGenerator)
    }

    return &OntologyBuilder{
        sampler:      samp,
        llmClient:    llmClient,
        config:       config,
        mcpServer:    mcpServer,
        queryBackend: queryBackend,  // Old field
    }, nil
}

func (b *OntologyBuilder) Close() error {
    // Close query backend
    if b.queryBackend != nil {
        b.queryBackend.Close()
    }
    return nil
}
```

**After**:
```go
// Removed query import - no longer needed

type OntologyBuilder struct {
    sampler          *sampler.Sampler
    llmClient        LLMClient
    config           BuilderConfig
    substantiveTypes []string
    mcpServer        *mcp.OntologyCorpusExplorer
    mcpStorage       analytics.Storage  // New approach
}

// NewOntologyBuilder creates MCP server with Storage interface
func NewOntologyBuilder(config BuilderConfig, llmClient LLMClient) (*OntologyBuilder, error) {
    // ...

    if config.EnableMCP {
        // Create analytics storage for MCP (supports temporal filtering)
        storageConfig := map[string]interface{}{
            "path": config.ParquetPath,
        }
        mcpStorage, err := analytics.NewHiveParquetStorage(storageConfig)
        if err != nil {
            return nil, fmt.Errorf("failed to create storage for MCP: %w", err)
        }

        // Create embedding generator
        embConfig := embeddings.Config{
            Enabled:  true,
            Provider: "onnx",
            Model:    config.EmbeddingModel,
        }
        embGenerator, err := embeddings.CreateEmbeddingGenerator(embConfig)
        if err != nil {
            mcpStorage.Close()
            return nil, fmt.Errorf("failed to create embedding generator: %w", err)
        }

        // Create MCP server
        mcpServer, err = mcp.NewOntologyCorpusExplorer(mcpStorage, embGenerator)
        if err != nil {
            mcpStorage.Close()
            return nil, fmt.Errorf("failed to create MCP server: %w", err)
        }
    }

    return &OntologyBuilder{
        sampler:    samp,
        llmClient:  llmClient,
        config:     config,
        mcpServer:  mcpServer,
        mcpStorage: mcpStorage,  // New field
    }, nil
}

func (b *OntologyBuilder) Close() error {
    var errs []error

    // Close sampler
    if b.sampler != nil {
        if err := b.sampler.Close(); err != nil {
            errs = append(errs, fmt.Errorf("sampler close error: %w", err))
        }
    }

    // Close MCP storage backend
    if b.mcpStorage != nil {
        if err := b.mcpStorage.Close(); err != nil {
            errs = append(errs, fmt.Errorf("MCP storage close error: %w", err))
        }
    }

    if len(errs) > 0 {
        return fmt.Errorf("close errors: %v", errs)
    }

    return nil
}
```

**Key Changes**:
1. Removed `internal/udml/query` import (no longer needed)
2. Changed struct field from `queryBackend query.QueryBackend` to `mcpStorage analytics.Storage`
3. Updated MCP server creation to use `analytics.NewHiveParquetStorage()`
4. Updated `Close()` method to close `mcpStorage`
5. Improved error handling with multiple error collection

## Temporal Filtering Support

### Default Behavior

All MCP queries now enable temporal filtering by default:

```go
filters := make(map[string]interface{})
filters["latest_only"] = true  // Always get latest version of documents
```

This ensures that when exploring the corpus during ontology generation, the LLM sees only the most recent version of each document by default.

### Point-in-Time Queries

For historical analysis, queries can specify a date:

```go
filters := make(map[string]interface{})
filters["as_of_date"] = "2024-01-15"  // Corpus state as of Jan 15, 2024
```

### Implementation

Temporal filtering is implemented using DuckDB window functions:

```sql
-- Deduplicate by doc_id, keeping latest version
WITH versioned_elements AS (
    SELECT *,
           ROW_NUMBER() OVER (PARTITION BY doc_id ORDER BY date DESC) as rn
    FROM read_parquet('path/to/elements/**/*.parquet')
)
SELECT * FROM versioned_elements WHERE rn = 1
```

This works because:
1. Documents maintain stable `doc_id` across versions
2. Each version has a `date` timestamp (partition date from Hive structure)
3. Window function assigns `rn=1` to the latest version per doc_id
4. Filter `WHERE rn = 1` returns only the latest version

## Impact on Ontology Generation

### Sampler

**Question**: "Do we need to update the ontology sampler for temporality?"

**Answer**: **No updates needed**. The sampler already uses the Storage interface internally, so temporal filtering is automatically supported through the standard `filters` parameter.

**Example** (from `sampler.go`):
```go
// Sampler already uses Storage interface
func (s *Sampler) SampleElements(filters map[string]interface{}, limit int) ([]analytics.Element, error) {
    // Storage interface handles temporal filtering automatically
    results, err := s.storage.Query(filters, limit)
    if err != nil {
        return nil, err
    }
    return results, nil
}
```

### MCP Server

The MCP server now properly supports temporal filtering for all corpus exploration operations used during ontology schema generation:

1. **Semantic Search**: Find similar concepts (with `latest_only=true`)
2. **Pattern Analysis**: Analyze regex patterns across latest corpus
3. **Term Frequencies**: Count term occurrences in latest versions
4. **Co-occurrence Analysis**: Find entity relationships in current corpus
5. **Element Context**: Retrieve hierarchical context from latest data
6. **Aggregate Statistics**: Compute stats on current corpus state

## Testing

### Verification Steps

1. **Build with new architecture**:
```bash
cd go
go build -o ../bin/goworker ./cmd/worker
go build -o ../bin/ontology_interview ./cmd/ontology
```

2. **Test MCP server with temporal filtering**:
```bash
./bin/ontology_interview interview \
    --diversity-threshold 0.85 \
    --non-interactive \
    --enable-mcp \
    --embedding-model all-MiniLM-L6-v2 \
    ./tests/test_output/wikipedia_medical_mining_analytics \
    ./tests/test_output/ontology_results/medical_mining_ontology_mcp.json
```

3. **Verify temporal filtering in logs**:
```
[MCP:search_corpus:semantic] query="medical diagnosis", element_types=[], limit=50, threshold=0.70
✓ Temporal filtering: latest_only=true (deduplicating by doc_id)
✓ Found 42 results after temporal filtering
```

4. **Test multi-document versioning**:
```bash
# Process same document multiple times with updates
USE_GO_MODULES=true ../bin/goworker \
    --config ../tests/test_configs/wikipedia_medical_mining.toml \
    --max-documents 5 \
    --workers 3

# Verify MCP queries return latest versions only
./bin/ontology_interview interview --enable-mcp ...
```

## Future Enhancements

### 1. Multi-Element-Type Filtering in Storage

**Current**: Element type filtering happens in-memory after query execution

```go
// Filter by element type (in-memory filtering for now)
if len(elementTypes) > 0 {
    filtered := []analytics.SearchResult{}
    typeMap := make(map[string]bool)
    for _, t := range elementTypes {
        typeMap[t] = true
    }
    for _, result := range results {
        if typeMap[result.Element.ElementType] {
            filtered = append(filtered, result)
        }
    }
    return filtered, nil
}
```

**Future**: Push element type filtering into Storage interface

```go
// TODO: Extend Storage interface to support element_type IN (...) queries
filters["element_types"] = []string{"paragraph", "heading", "table"}
results, err := e.storage.SearchSemanticSimilarity(queryEmb, filters, threshold, limit)
```

**Benefits**:
- More efficient (filter at query time, not after)
- Reduced memory usage (don't fetch filtered-out results)
- Better performance for large corpora

### 2. Additional Storage Backends

The architecture now supports adding new backends:

**Neo4j**: Graph-based storage with native relationship traversal
```go
type Neo4jStorage struct {
    driver neo4j.Driver
}

func (s *Neo4jStorage) SearchSemanticSimilarity(...) ([]SearchResult, error) {
    // Implement using Neo4j vector index
}
```

**PostgreSQL**: Relational storage with pgvector extension
```go
type PostgresStorage struct {
    db *sql.DB
}

func (s *PostgresStorage) SearchSemanticSimilarity(...) ([]SearchResult, error) {
    // Implement using pgvector extension
}
```

### 3. Caching Layer

Add caching for frequently accessed MCP queries:

```go
type CachedStorage struct {
    underlying analytics.Storage
    cache      *lru.Cache
}

func (s *CachedStorage) SearchSemanticSimilarity(...) ([]SearchResult, error) {
    cacheKey := computeCacheKey(queryVector, filters, threshold, limit)

    if cached, ok := s.cache.Get(cacheKey); ok {
        return cached.([]SearchResult), nil
    }

    results, err := s.underlying.SearchSemanticSimilarity(...)
    if err == nil {
        s.cache.Add(cacheKey, results)
    }

    return results, err
}
```

## Migration Guide

### For Developers Adding New MCP Tools

1. **Define method in Storage interface** (`internal/analytics/types.go`):
```go
// NewMCPMethod does something useful
// filters: standard filters (source_name, doc_id, element_type, latest_only, as_of_date)
NewMCPMethod(param1 string, filters map[string]interface{}, limit int) (*Result, error)
```

2. **Implement in both Parquet backends**:
```go
// parquet_hive_mcp.go
func (s *HiveParquetStorage) NewMCPMethod(...) (*Result, error) {
    return newMCPMethodImpl(s.basePath, param1, filters, limit)
}

// parquet_mcp.go
func (s *ParquetStorage) NewMCPMethod(...) (*Result, error) {
    return newMCPMethodImpl(s.basePath, param1, filters, limit)
}
```

3. **Create shared implementation** (`mcp_queries.go`):
```go
func newMCPMethodImpl(basePath string, param1 string, filters map[string]interface{}, limit int) (*Result, error) {
    db, err := sql.Open("duckdb", "")
    if err != nil {
        return nil, err
    }
    defer db.Close()

    // Build elements CTE with temporal filtering
    elementsCTE := buildElementsCTE(basePath, filters)

    // Execute DuckDB query
    query := fmt.Sprintf(`
        WITH elements AS (%s)
        SELECT ... FROM elements WHERE ...
    `, elementsCTE)

    // Execute and return results
}
```

4. **Use in MCP server** (`internal/udml/ontology/mcp/impl.go`):
```go
func (e *OntologyCorpusExplorer) executeNewTool(ctx context.Context, ...) (*Result, error) {
    filters := make(map[string]interface{})
    filters["latest_only"] = true  // Enable temporal filtering

    return e.storage.NewMCPMethod(param1, filters, limit)
}
```

### For Existing Code Using query.RawQueryBackend

**Old Pattern**:
```go
backend := query.NewDuckDBBackend(config)
results, err := backend.QueryRaw("SELECT ... WHERE ...", params)
```

**New Pattern**:
```go
storageConfig := map[string]interface{}{
    "path": parquetPath,
}
storage, err := analytics.NewHiveParquetStorage(storageConfig)
if err != nil {
    return err
}
defer storage.Close()

filters := map[string]interface{}{
    "latest_only": true,
    "element_type": "paragraph",
}
results, err := storage.SearchByKeyword("medical", filters, 100)
```

## Conclusion

This refactoring successfully decouples the MCP server from DuckDB-specific implementations while adding comprehensive temporal filtering support. All corpus exploration operations now:

1. ✅ Use the unified Storage interface
2. ✅ Support multiple storage backends
3. ✅ Enable temporal filtering by default
4. ✅ Maintain clean separation of concerns
5. ✅ Follow the architectural principle: "everything must go through Storage"

The ontology generation pipeline (sampler → MCP server → LLM → schema) now operates entirely through the Storage abstraction layer, ensuring consistent behavior across different backend implementations and proper support for document versioning.
