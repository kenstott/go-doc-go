# MCP Tool Calling - Usage Guide

## Overview

The MCP (Model Context Protocol) integration enables Claude to interactively explore your UDML corpus while generating ontology schemas. This results in higher-quality schemas that are better aligned with your actual data.

## Enabling MCP

### Configuration

Enable MCP by setting `EnableMCP: true` in your `BuilderConfig`:

```go
config := ontology.BuilderConfig{
    // Standard config
    ParquetPath:    "/path/to/udml/data",
    SampleSize:     100,
    LLMProvider:    "anthropic",
    LLMModel:       "claude-3-5-sonnet-20241022",
    LLMAPIKey:      os.Getenv("ANTHROPIC_API_KEY"),

    // MCP configuration
    EnableMCP:      true,                    // Enable corpus exploration tools
    EmbeddingModel: "all-MiniLM-L6-v2",     // For semantic search

    // Output
    SchemaName:    "my_ontology",
    SchemaVersion: "1.0.0",
}

builder, err := ontology.NewOntologyBuilder(config)
if err != nil {
    log.Fatal(err)
}
defer builder.Close()

// Run ontology building process
result, err := builder.Build(context.Background())
if err != nil {
    log.Fatal(err)
}
```

### What Happens When MCP is Enabled

1. **Query Backend Creation**: DuckDB backend is initialized with your Parquet data
2. **Embedding Generator**: ONNX model loaded for semantic search
3. **MCP Server**: 6 corpus exploration tools become available
4. **LLM Connection**: Tools are automatically registered with Anthropic client
5. **Tool Calling**: During schema generation, Claude can call tools to explore your corpus

## Available Tools

### 1. search_corpus
Search the entire corpus using semantic similarity, keywords, or regex.

**When Claude Uses It:**
- Needs more examples beyond the 20 samples provided
- Wants to verify domain prevalence
- Looking for specific terminology or patterns

**Example Tool Call:**
```json
{
  "name": "search_corpus",
  "arguments": {
    "query": "revenue recognition",
    "search_type": "semantic",
    "limit": 20
  }
}
```

### 2. compute_frequencies
Count how often specific terms appear in the corpus.

**When Claude Uses It:**
- Validating domain decisions (e.g., "Is this a financial corpus?")
- Comparing term frequencies to prioritize entity types

**Example Tool Call:**
```json
{
  "name": "compute_frequencies",
  "arguments": {
    "terms": ["revenue", "EBITDA", "gross profit", "net income"],
    "case_sensitive": false
  }
}
```

### 3. analyze_patterns
Analyze how a regex pattern matches across the corpus.

**When Claude Uses It:**
- Testing extraction patterns before committing to them
- Understanding pattern distribution across element types

**Example Tool Call:**
```json
{
  "name": "analyze_patterns",
  "arguments": {
    "pattern": "\\b[A-Z]{2,5}\\b",
    "max_examples": 10
  }
}
```

### 4. find_cooccurrences
Find how often two entities appear together.

**When Claude Uses It:**
- Discovering relationship patterns
- Validating that entities co-occur in meaningful contexts

**Example Tool Call:**
```json
{
  "name": "find_cooccurrences",
  "arguments": {
    "entity1": "CEO",
    "entity2": "company",
    "context_window": "element"
  }
}
```

### 5. aggregate_statistics
Get corpus statistics like element type distribution and document counts.

**When Claude Uses It:**
- Understanding corpus composition
- Making decisions about element type priorities

**Example Tool Call:**
```json
{
  "name": "aggregate_statistics",
  "arguments": {
    "metrics": ["element_type_distribution", "document_count"]
  }
}
```

### 6. get_element_context
Retrieve surrounding context for specific elements.

**When Claude Uses It:**
- Understanding document structure around interesting elements
- Verifying element relationships in hierarchy

## Benefits

### Without MCP
- Claude sees only 20 representative samples
- Cannot validate hypotheses against full corpus
- May miss important patterns in underrepresented data
- Limited ability to refine extraction rules

### With MCP
- Claude explores entire corpus interactively
- Validates domain decisions with frequency analysis
- Tests extraction patterns before committing
- Discovers entity relationships through co-occurrence
- Adapts schema based on corpus-wide patterns
- **Result**: Higher quality ontologies with better coverage

## Performance Considerations

### Latency Impact
Each tool call adds latency:
- **Semantic search**: ~100-500ms (depends on corpus size)
- **Frequency counting**: ~50-200ms
- **Pattern analysis**: ~100-300ms

**Total impact**: Expect 2-5 extra seconds per LLM method (identifyDomains, defineEntityTypes, defineRelationshipTypes) if Claude uses tools.

### Resource Usage
- **Memory**: +~500MB for DuckDB backend + embeddings
- **CPU**: Moderate during tool execution
- **Disk**: Temporary DuckDB working files

### Optimization Tips
1. **Parquet Partitioning**: Use Hive-partitioned Parquet for faster queries
2. **Embedding Cache**: Embeddings are cached after first semantic search
3. **Sample Size**: Smaller sample sizes in config mean fewer tool calls needed

## Debugging

### Logging
Tool usage is logged automatically:

```
✓ MCP enabled - creating corpus exploration server...
✓ MCP server connected to Anthropic client
✓ MCP server ready with 6 corpus exploration tools
✓ Calling LLM with 6 MCP tools for domain identification
```

### Troubleshooting

**"MCP enabled but LLM client does not support tool calling"**
- Cause: Using a non-Anthropic LLM client
- Solution: MCP tool calling currently requires Anthropic API

**"Failed to create query backend for MCP"**
- Cause: Invalid Parquet path or insufficient permissions
- Solution: Verify `ParquetPath` exists and is readable

**"Failed to create embedding generator for MCP"**
- Cause: ONNX model not found or incompatible platform
- Solution: Ensure `assets/` directory contains ONNX models

## Testing

Unit tests are available in `builder_test.go`:

```bash
go test -v -run "TestGetMCPToolDefinitions|TestAnthropicClient_SetMCPServer|TestMCPCapableLLMClient" ./internal/udml/ontology
```

## Migration Guide

### Upgrading from Non-MCP to MCP

1. **Update Config**:
```go
// Before
config := ontology.BuilderConfig{
    ParquetPath: "/data",
    // ...
}

// After
config := ontology.BuilderConfig{
    ParquetPath:    "/data",
    EnableMCP:      true,                  // NEW
    EmbeddingModel: "all-MiniLM-L6-v2",   // NEW
    // ...
}
```

2. **No Code Changes Needed**: MCP gracefully degrades if disabled

3. **Test Both Modes**: Verify schemas are similar with/without MCP

## Best Practices

1. **Start Simple**: Try MCP on a small corpus first
2. **Monitor Latency**: Check if extra seconds are acceptable for your use case
3. **Compare Results**: Generate schemas with and without MCP to see the difference
4. **Iterate**: MCP is most valuable for complex, heterogeneous corpora

## Future Enhancements

- **OpenAI Support**: Tool calling with GPT-4
- **Custom Tools**: Add domain-specific exploration tools
- **Caching**: Cache tool results to reduce latency
- **Parallel Execution**: Execute multiple tool calls concurrently
- **Analytics**: Track which tools are most useful

## References

- MCP Server Implementation: `internal/udml/ontology/mcp/server.go`
- Tool Calling Integration: `internal/udml/ontology/builder.go`
- Anthropic Tool Use Docs: https://docs.anthropic.com/en/docs/build-with-claude/tool-use
