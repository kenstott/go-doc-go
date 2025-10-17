# Simple PDF Processing Example

This example demonstrates the simplest possible Go-Doc-Go setup to process PDF documents using SQLite for job control and Parquet for analytics output.

## What This Example Does

- Processes PDF files from a local directory
- Extracts text, tables, and document structure
- Stores results in Parquet format for querying
- Uses SQLite (no database server required)

## Prerequisites

```bash
# Go 1.21 or later
go version

# Build the worker
cd ../../go
go build -o ../bin/goworker ./cmd/worker
cd ../examples/simple-pdf-processing
```

## Quick Start

### 1. Add Sample PDFs

```bash
# Create a docs directory and add your PDF files
mkdir -p docs
# Copy some PDFs to ./docs/
cp /path/to/your/*.pdf docs/
```

### 2. Run the Worker

```bash
../../bin/goworker --config config.toml --max-documents 10
```

### 3. Query the Results

```bash
# Using DuckDB
duckdb

D SELECT element_type, COUNT(*) as count
  FROM read_parquet('./output/analytics.parquet/elements/*.parquet')
  GROUP BY element_type
  ORDER BY count DESC;

# View extracted text
D SELECT element_id, content_preview
  FROM read_parquet('./output/analytics.parquet/elements/*.parquet')
  WHERE element_type = 'paragraph'
  LIMIT 10;
```

## Configuration Explained

The `config.toml` file configures:

1. **Job Control** (`processing.job_control`) - SQLite database to track document processing
2. **Content Sources** - Where to find documents (local `./docs` directory)
3. **Analytics Output** - Parquet files for fast querying

## Expected Output

After running the worker, you'll have:

```
output/
├── jobs.db                    # SQLite job control database
└── analytics.parquet/         # Analytics output
    ├── documents/             # Document metadata
    │   └── *.parquet
    ├── elements/              # Extracted elements (paragraphs, tables, etc.)
    │   └── *.parquet
    └── relationships/         # Element relationships
        └── *.parquet
```

## Understanding the Output

### Documents Table

```bash
duckdb -c "DESCRIBE SELECT * FROM read_parquet('./output/analytics.parquet/documents/*.parquet')"
```

**Columns:**
- `doc_id` - Unique document identifier
- `source_name` - Content source name
- `file_path` - Original file path
- `file_size` - File size in bytes
- `created_at` - Processing timestamp
- `metadata` - Document metadata (JSON)

### Elements Table

```bash
duckdb -c "DESCRIBE SELECT * FROM read_parquet('./output/analytics.parquet/elements/*.parquet')"
```

**Columns:**
- `element_id` - Unique element identifier
- `doc_id` - Parent document ID
- `element_type` - Type (paragraph, table, header, etc.)
- `content_preview` - Text content (first 100 chars)
- `full_content` - Complete text content
- `page_number` - Page location
- `position` - Position in document
- `metadata` - Additional metadata (JSON)

### Relationships Table

```bash
duckdb -c "DESCRIBE SELECT * FROM read_parquet('./output/analytics.parquet/relationships/*.parquet')"
```

**Columns:**
- `relationship_id` - Unique relationship identifier
- `source_element_id` - Source element
- `target_element_id` - Target element
- `relationship_type` - Type (contains, next, references, etc.)

## Query Examples

### Count Documents by Type

```sql
SELECT
  SUBSTRING(file_path, -4) as extension,
  COUNT(*) as count
FROM read_parquet('./output/analytics.parquet/documents/*.parquet')
GROUP BY extension;
```

### Find All Tables

```sql
SELECT
  doc_id,
  element_id,
  content_preview
FROM read_parquet('./output/analytics.parquet/elements/*.parquet')
WHERE element_type = 'table'
ORDER BY doc_id, position;
```

### Reconstruct Document Structure

```sql
-- Get document with its elements in order
SELECT
  e.element_type,
  e.position,
  e.content_preview
FROM read_parquet('./output/analytics.parquet/elements/*.parquet') e
WHERE e.doc_id = 'your-doc-id'
ORDER BY e.position;
```

### Search Content

```sql
SELECT
  d.file_path,
  e.element_type,
  e.content_preview
FROM read_parquet('./output/analytics.parquet/elements/*.parquet') e
JOIN read_parquet('./output/analytics.parquet/documents/*.parquet') d
  ON e.doc_id = d.doc_id
WHERE e.content_preview LIKE '%search term%'
LIMIT 20;
```

## Troubleshooting

### No documents found

```bash
# Check if PDFs are in the correct location
ls -la docs/

# Verify file pattern matches
# The config uses: file_pattern = "**/*.pdf"
```

### Worker exits immediately

```bash
# Check if output directory can be created
mkdir -p output

# Run with just 1 document to test
../../bin/goworker --config config.toml --max-documents 1
```

### Can't query Parquet files

```bash
# Verify Parquet files were created
ls -la output/analytics.parquet/*/

# Check if DuckDB is installed
duckdb --version

# Install DuckDB if needed
brew install duckdb  # macOS
# or download from https://duckdb.org
```

## Next Steps

1. **Process more documents**: Remove `--max-documents` flag to process all PDFs
2. **Add embeddings**: See [../semantic-search/](../semantic-search/) example
3. **Scale horizontally**: See [../distributed-workers/](../distributed-workers/) example
4. **Export to Neo4j**: See [../neo4j-knowledge-graph/](../neo4j-knowledge-graph/) example

## Related Documentation

- [Getting Started Guide](../../docs/getting-started/README.md)
- [Configuration Reference](../../docs/configuration/README.md)
- [Troubleshooting](../../docs/operations/troubleshooting.md)
