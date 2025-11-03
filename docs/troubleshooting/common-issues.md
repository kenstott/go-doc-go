# Common Issues and Solutions

## Quick Index

- [Installation and Setup](#installation-and-setup)
  - [Binary not found after build](#binary-not-found-after-build)
  - [Go dependencies failed to download](#go-dependencies-failed-to-download)
- [Configuration Errors](#configuration-errors)
  - [Configuration file not found](#configuration-file-not-found)
  - [Invalid configuration validation errors](#invalid-configuration-validation-errors)
- [Parsing Errors](#parsing-errors)
  - [Parser fails with "no parser found for file"](#parser-fails-with-no-parser-found-for-file)
  - [PDF parsing fails with invalid header](#pdf-parsing-fails-with-invalid-header)
  - [Out of memory during large document parsing](#out-of-memory-during-large-document-parsing)
- [Worker and Queue Issues](#worker-and-queue-issues)
  - [Worker not claiming documents from queue](#worker-not-claiming-documents-from-queue)
  - [Documents stuck in "processing" state](#documents-stuck-in-processing-state)
  - [Database connection errors](#database-connection-errors)
- [Storage and Output Issues](#storage-and-output-issues)
  - [Parquet files not created](#parquet-files-not-created)
  - [Neo4j export fails with connection timeout](#neo4j-export-fails-with-connection-timeout)
- [Embedding Generation Issues](#embedding-generation-issues)
  - [ONNX model not found](#onnx-model-not-found)
  - [Embedding generation is very slow](#embedding-generation-is-very-slow)
- [Discovery and Crawling Issues](#discovery-and-crawling-issues)
  - [Discovered links not being processed](#discovered-links-not-being-processed)
  - [Code dependencies not being followed](#code-dependencies-not-being-followed)

---

## Installation and Setup

### Binary not found after build

**Symptoms**:
```bash
go build -o goworker ./cmd/worker
./goworker
# bash: goworker: command not found
```

**Common Causes**:
1. Binary built to wrong directory
2. Binary not in PATH
3. Incorrect current working directory

**Solution**:

1. **Verify build location**:
   ```bash
   # Check if binary was created
   ls -la goworker
   # or
   ls -la bin/goworker
   ```

2. **Build to correct location** (per project standards):
   ```bash
   # CORRECT: Build to bin/ directory
   go build -o bin/goworker ./cmd/worker

   # Run from project root
   bin/goworker --config config.toml
   ```

3. **Verify current directory**:
   ```bash
   pwd
   # Should output: /path/to/doculyzer-go-conversion
   ```

**Prevention**:
- Always build binaries to `bin/` directory: `go build -o bin/<binary-name>`
- Run commands from project root directory
- Use absolute paths in scripts

**Related**: [Project Guidelines](../../CLAUDE.md#binary-output-location)

---

### Go dependencies failed to download

**Symptoms**:
```
go: github.com/some/package@v1.2.3: Get "https://proxy.golang.org/...": dial tcp: lookup proxy.golang.org: no such host
```

**Common Causes**:
1. No internet connection
2. Corporate firewall blocking proxy.golang.org
3. GOPROXY environment variable misconfigured

**Solution**:

1. **Check internet connectivity**:
   ```bash
   ping proxy.golang.org
   ```

2. **Configure Go proxy** (if behind firewall):
   ```bash
   # Use direct mode (no proxy)
   export GOPROXY=direct

   # Or use corporate proxy
   export GOPROXY=https://your-corp-proxy.com,direct
   ```

3. **Download dependencies**:
   ```bash
   go mod download
   go mod verify
   ```

4. **Clear module cache** (if corrupted):
   ```bash
   go clean -modcache
   go mod download
   ```

**Prevention**:
- Set GOPROXY in shell profile (.bashrc, .zshrc)
- Use vendor directory: `go mod vendor`

---

## Configuration Errors

### Configuration file not found

**Symptoms**:
```
Error: configuration file not found: config.toml
```

**Common Causes**:
1. File doesn't exist at specified path
2. Incorrect relative path
3. Wrong current working directory

**Solution**:

1. **Verify file exists**:
   ```bash
   ls -la config.toml
   # or
   ls -la tests/test_configs/my_config.toml
   ```

2. **Use absolute path**:
   ```bash
   bin/goworker --config /absolute/path/to/config.toml
   ```

3. **Use correct relative path** (from project root):
   ```bash
   # If config is in tests/test_configs/
   bin/goworker --config tests/test_configs/config.toml
   ```

4. **Verify current directory**:
   ```bash
   pwd
   # Should be project root
   ```

**Prevention**:
- Store configs in `tests/test_configs/` (version controlled)
- Use absolute paths in production
- Document expected working directory

**Related**: [Configuration Guide](../configuration/README.md)

---

### Invalid configuration validation errors

**Symptoms**:
```
Error: configuration validation failed: processing.job_control.backend is required
Error: invalid content source type: "files" (must be "file", "web", or "s3")
```

**Common Causes**:
1. Missing required fields
2. Typos in configuration keys
3. Invalid values for enums

**Solution**:

1. **Check error message** for specific field:
   ```
   Error: processing.job_control.backend is required
                ↑         ↑          ↑
              section  subsection  field
   ```

2. **Add missing field**:
   ```toml
   [processing.job_control]
   backend = "sqlite"  # or "postgres"
   path = "./queue.db"
   ```

3. **Verify enum values**:
   ```toml
   [[content_sources]]
   type = "file"  # NOT "files" - must be exact
   ```

4. **Validate against example**:
   ```bash
   # Compare with working example
   diff config.toml examples/distributed-workers/config.toml
   ```

**Common Required Fields**:
```toml
[processing]
workers = 4

[processing.job_control]
backend = "sqlite"  # Required
path = "./queue.db"  # Required

[analytics]
enabled = true

[[analytics.outputs]]
type = "parquet"  # Required
base_path = "./output"  # Required
```

**Prevention**:
- Start from example configs
- Use schema validation tools
- Read error messages carefully

**Related**: [Configuration Reference](../configuration/README.md)

---

## Parsing Errors

### Parser fails with "no parser found for file"

**Symptoms**:
```
Error: no parser found for file: document.xyz (extension: .xyz)
```

**Common Causes**:
1. Unsupported file format
2. File extension not recognized
3. Parser not registered

**Solution**:

1. **Check supported formats**:
   ```bash
   # Supported: .pdf, .docx, .xlsx, .pptx, .html, .md, .json, .csv, .xml,
   #            .parquet, .go, .java, .py, .js, .ts, .rb, .rs, .c, .cpp, etc.
   ```

2. **Verify file extension**:
   ```bash
   ls -la document.xyz
   # Rename if needed
   mv document.xyz document.pdf
   ```

3. **Check if parser is registered**:
   See [Parser Documentation](../../go/internal/parser/doc.go) for full list.

**Workaround**:
- Rename file to correct extension
- Convert to supported format
- Use generic text parser for unknown formats

**Prevention**:
- Use standard file extensions
- Validate file types before processing
- Check content source file patterns

**Related**: [Parser Documentation](../../go/internal/parser/doc.go)

---

### PDF parsing fails with invalid header

**Symptoms**:
```
Error: failed to parse: invalid PDF header
```

**Common Causes**:
1. Corrupted PDF file
2. Non-standard PDF variant
3. Password-protected PDF

**Solution**:

1. **Validate PDF file**:
   ```bash
   # Install pdfinfo (poppler-utils)
   pdfinfo document.pdf
   ```

2. **Repair corrupted PDF**:
   ```bash
   # Install ghostscript
   gs -o repaired.pdf -sDEVICE=pdfwrite broken.pdf
   ```

3. **Check for password protection**:
   ```bash
   pdfinfo document.pdf | grep Encrypted
   # Output: Encrypted: yes
   ```

4. **Remove password** (if you have it):
   ```bash
   # Install qpdf
   qpdf --password=PASSWORD --decrypt input.pdf output.pdf
   ```

**Prevention**:
- Validate PDFs before processing
- Use PDF repair tools in pipeline
- Handle encrypted PDFs separately

**Related**: [PDF Parser](../../go/internal/parser/doc_pdf.go)

---

### Out of memory during large document parsing

**Symptoms**:
```
fatal error: runtime: out of memory
```

**Common Causes**:
1. Document too large (> 100MB)
2. Too many workers for available memory
3. Memory leak in parser

**Solution**:

1. **Reduce worker count**:
   ```toml
   [processing]
   workers = 2  # Down from 10
   ```

2. **Increase system memory**:
   ```bash
   # Check current memory usage
   free -h
   # or on macOS
   vm_stat
   ```

3. **Process large files separately**:
   ```bash
   # Split large files
   # Process in smaller batches
   ```

4. **Monitor memory usage**:
   ```bash
   # While processing
   watch -n 1 'ps aux | grep goworker'
   ```

**Memory Guidelines**:
- Base memory: ~100MB per worker
- PDF parsing: ~5x file size
- DOCX parsing: ~3x file size
- Embedding model: ~500MB (shared)

**Prevention**:
- Set max file size limits
- Use streaming parsers for large files
- Monitor memory usage

**Related**: [Performance Tuning](../operations/scaling.md)

---

## Worker and Queue Issues

### Worker not claiming documents from queue

**Symptoms**:
```
[INFO] Worker started
[DEBUG] Checking queue...
[DEBUG] No documents to claim
# But documents exist in queue
```

**Common Causes**:
1. Documents in wrong state (already processing)
2. Database connection issue
3. Run ID mismatch
4. Stale heartbeat locks

**Solution**:

1. **Check queue status**:
   ```bash
   # SQLite
   sqlite3 queue.db "SELECT state, COUNT(*) FROM documents GROUP BY state;"

   # PostgreSQL
   psql -d jobqueue -c "SELECT state, COUNT(*) FROM documents GROUP BY state;"
   ```

2. **Check for stale locks**:
   ```sql
   -- Documents stuck in processing with old heartbeats
   SELECT doc_id, claimed_by, heartbeat_at
   FROM documents
   WHERE state = 'processing'
     AND heartbeat_at < NOW() - INTERVAL '10 minutes';
   ```

3. **Reset stale documents**:
   ```sql
   -- Return to pending state
   UPDATE documents
   SET state = 'pending', claimed_by = NULL, heartbeat_at = NULL
   WHERE state = 'processing'
     AND heartbeat_at < NOW() - INTERVAL '10 minutes';
   ```

4. **Verify run ID** (config hash):
   ```bash
   # All workers must use IDENTICAL config for same run_id
   md5sum config.toml
   ```

**Prevention**:
- Set reasonable claim_timeout (300 seconds)
- Monitor heartbeat intervals
- Use worker health checks
- Ensure config consistency

**Related**: [Job Control Documentation](../../go/internal/jobcontrol/doc.go)

---

### Documents stuck in "processing" state

**Symptoms**:
```
# Documents never complete
SELECT state, COUNT(*) FROM documents GROUP BY state;
# processing | 50  (never decreases)
```

**Common Causes**:
1. Worker crashed mid-processing
2. Heartbeat not being sent
3. claim_timeout too short

**Solution**:

1. **Check worker logs** for crashes:
   ```bash
   tail -f worker.log | grep -i "error\|panic\|fatal"
   ```

2. **Increase claim timeout**:
   ```toml
   [processing.job_control]
   claim_timeout = 600  # 10 minutes (up from 300)
   ```

3. **Manually reclaim stale documents**:
   ```bash
   # See "Worker not claiming documents" solution above
   ```

4. **Check heartbeat configuration**:
   ```toml
   [processing.job_control]
   heartbeat_interval = 30  # Send every 30 seconds
   claim_timeout = 300  # 5 minutes
   # heartbeat_interval MUST be < claim_timeout
   ```

**Prevention**:
- Set claim_timeout > 2x expected processing time
- Monitor worker health
- Implement graceful shutdown
- Log heartbeat failures

**Related**: [Worker Documentation](../../go/internal/worker/doc.go)

---

### Database connection errors

**Symptoms**:
```
Error: failed to connect to database: dial tcp [::1]:5432: connect: connection refused
```

**Common Causes**:
1. Database not running
2. Incorrect connection string
3. Firewall blocking connection
4. Wrong credentials

**Solution**:

1. **Verify database is running**:
   ```bash
   # PostgreSQL
   pg_isready -h localhost -p 5432

   # SQLite (check file exists)
   ls -la ./queue.db
   ```

2. **Check connection string**:
   ```toml
   # PostgreSQL format
   path = "postgresql://username:password@localhost:5432/dbname"

   # SQLite format
   path = "./queue.db"  # Relative to working directory
   ```

3. **Test connection manually**:
   ```bash
   # PostgreSQL
   psql "postgresql://username:password@localhost:5432/dbname"

   # SQLite
   sqlite3 ./queue.db "SELECT 1;"
   ```

4. **Check permissions**:
   ```bash
   # SQLite file permissions
   ls -la queue.db
   # Should be readable/writable by process user
   ```

**Prevention**:
- Test database connection before starting workers
- Use connection pooling
- Log connection errors with full context
- Document database setup requirements

**Related**: [Job Control Documentation](../../go/internal/jobcontrol/doc.go)

---

## Storage and Output Issues

### Parquet files not created

**Symptoms**:
```
[INFO] Processed 100 documents
# But no parquet files in output directory
ls -la output/
# Empty or missing
```

**Common Causes**:
1. Incorrect base_path (relative path from wrong directory)
2. No write permissions
3. Analytics not enabled
4. Storage errors silently ignored

**Solution**:

1. **Check analytics configuration**:
   ```toml
   [analytics]
   enabled = true  # Must be true!

   [[analytics.outputs]]
   type = "parquet"
   base_path = "./output"  # Relative to working directory
   hive_partition_by_source = true
   ```

2. **Verify working directory**:
   ```bash
   pwd
   # If in go/, but base_path = "./output"
   # Files created in go/output/ NOT output/

   # Fix: Use absolute path or correct relative path
   base_path = "../output"  # If running from go/
   ```

3. **Check permissions**:
   ```bash
   # Test write access
   touch output/test.txt
   # If fails, check permissions
   ls -ld output/
   chmod 755 output/
   ```

4. **Check logs for storage errors**:
   ```bash
   grep -i "storage\|parquet" worker.log
   ```

**Expected Directory Structure**:
```
output/
├── documents/
│   └── source_name=wikipedia/
│       └── data.parquet
├── elements/
│   └── source_name=wikipedia/
│       └── data.parquet
└── relationships/
    └── source_name=wikipedia/
        └── data.parquet
```

**Prevention**:
- Use absolute paths in configuration
- Verify output directory exists
- Enable verbose logging
- Monitor storage writes

**Related**: [Analytics Documentation](../../go/internal/analytics/doc.go)

---

### Neo4j export fails with connection timeout

**Symptoms**:
```
Error: Neo4j export failed: dial tcp 127.0.0.1:7687: connect: connection refused
```

**Common Causes**:
1. Neo4j not running
2. Incorrect connection URI
3. Wrong port (7687 for Bolt, 7474 for HTTP)
4. Authentication failure

**Solution**:

1. **Verify Neo4j is running**:
   ```bash
   # Check Neo4j status
   neo4j status

   # Start if needed
   neo4j start
   ```

2. **Test connection**:
   ```bash
   # Using cypher-shell
   cypher-shell -a bolt://localhost:7687 -u neo4j -p password
   ```

3. **Check configuration**:
   ```toml
   [processing.neo4j_export]
   enabled = true

   [processing.neo4j_export.connection]
   uri = "bolt://localhost:7687"  # NOT http://
   username = "neo4j"
   password = "your-password"
   database = "neo4j"
   ```

4. **Verify Neo4j settings** (neo4j.conf):
   ```
   dbms.connector.bolt.listen_address=0.0.0.0:7687
   dbms.connector.bolt.enabled=true
   ```

**Prevention**:
- Start Neo4j before enabling export
- Use health check before export
- Log connection errors with URI (sanitize password)
- Document Neo4j version requirements

**Related**: [Neo4j Export Configuration](../configuration/storage.md)

---

## Embedding Generation Issues

### ONNX model not found

**Symptoms**:
```
Error: failed to load ONNX model: open ./models/all-MiniLM-L6-v2/model.onnx: no such file or directory
```

**Common Causes**:
1. Model files not downloaded
2. Incorrect model_path
3. Missing model files (tokenizer.json, config.json)

**Solution**:

1. **Verify model directory**:
   ```bash
   ls -la models/all-MiniLM-L6-v2/
   # Should contain:
   # - model.onnx
   # - tokenizer.json
   # - config.json
   ```

2. **Download model** (if missing):
   ```bash
   # Create directory
   mkdir -p models/all-MiniLM-L6-v2

   # Download from Hugging Face or other source
   # (Exact download commands depend on model source)
   ```

3. **Use absolute path** in config:
   ```toml
   [embedding]
   model_path = "/absolute/path/to/models/all-MiniLM-L6-v2"
   ```

4. **Verify path from working directory**:
   ```bash
   pwd
   # If in go/, but model_path = "./models"
   # Looks for go/models/ NOT models/

   # Fix: Use correct relative path
   model_path = "../models/all-MiniLM-L6-v2"
   ```

**Prevention**:
- Document model download process
- Use absolute paths for model_path
- Include model in deployment package
- Validate model files before starting

**Related**: [Embeddings Documentation](../../go/internal/embeddings/doc.go)

---

### Embedding generation is very slow

**Symptoms**:
```
[INFO] Generated 100 embeddings in 30s (3.3 embeddings/sec)
# Expected: 100-1000 embeddings/sec
```

**Common Causes**:
1. Not using batch processing
2. batch_size too small
3. pool_size < worker count
4. CPU-bound system

**Solution**:

1. **Increase batch size**:
   ```toml
   [embedding]
   batch_size = 64  # Up from 16
   ```

2. **Match pool_size to workers**:
   ```toml
   [embedding]
   pool_size = 4  # Match processing.workers

   [processing]
   workers = 4
   ```

3. **Check CPU usage**:
   ```bash
   # While processing
   top
   # Should see high CPU usage for goworker
   ```

4. **Use batch processing** in code:
   ```go
   // GOOD: Batch processing
   vectors, _ := generator.GenerateBatch(texts)

   // BAD: Individual processing
   for _, text := range texts {
       vec, _ := generator.Generate(text)  // Slow!
   }
   ```

**Performance Tuning**:
```toml
[embedding]
batch_size = 32  # Optimal for most systems
pool_size = 4    # Match worker count
contextual = true
predecessor_count = 1  # Limit context size
successor_count = 1
```

**Prevention**:
- Benchmark with test documents
- Monitor embedding throughput
- Tune batch_size for hardware
- Use profiling tools

**Related**: [Embeddings Documentation](../../go/internal/embeddings/doc.go)

---

## Discovery and Crawling Issues

### Discovered links not being processed

**Symptoms**:
```
[DEBUG] Extracted hyperlink: https://example.com/page2
[DEBUG] Extracted hyperlink: https://example.com/page3
# But new pages never processed
```

**Common Causes**:
1. Discovery not enabled
2. max_depth reached
3. Patterns excluding discovered URLs
4. Links already processed

**Solution**:

1. **Enable discovery**:
   ```toml
   [[content_sources]]
   name = "website"
   type = "web"
   base_url = "https://example.com"
   url_list = ["https://example.com/start"]

   [content_sources.discovery]
   enabled = true  # Must be true!
   max_depth = 3

   [content_sources.discovery.hyperlinks]
   enabled = true  # Must be true!
   ```

2. **Check max_depth**:
   ```bash
   # Document at depth 2 won't discover new docs if max_depth = 2
   # Increase max_depth:
   ```
   ```toml
   [content_sources.discovery]
   max_depth = 5  # Allow deeper crawling
   ```

3. **Check exclude patterns**:
   ```toml
   [content_sources.discovery]
   exclude_patterns = [
       "/archive/",  # This might exclude too much
       "/old/"
   ]
   # Remove or adjust patterns
   ```

4. **Verify links are new**:
   ```sql
   -- Check if already processed
   SELECT doc_id, state FROM documents WHERE doc_id = 'https://example.com/page2';
   ```

**Prevention**:
- Test discovery with small max_depth first
- Monitor discovery logs
- Use specific include/exclude patterns
- Log when links are skipped (with reason)

**Related**: [Discovery Configuration](../features/discovery/)

---

### Code dependencies not being followed

**Symptoms**:
```
[DEBUG] Extracted import: github.com/user/package
# But package files never processed
```

**Common Causes**:
1. Code dependency discovery not enabled
2. Import path not resolved to file path
3. Package type filtered out (stdlib/external)
4. Go module not initialized

**Solution**:

1. **Enable code dependency discovery**:
   ```toml
   [content_sources.discovery]
   enabled = true

   [content_sources.discovery.code_dependencies]
   enabled = true  # Must be true!
   follow_stdlib = false  # Don't follow standard library
   follow_local = true    # Follow same-project imports
   follow_external = false  # Don't follow third-party
   ```

2. **Verify Go module** (for Go imports):
   ```bash
   # Must have go.mod in project
   ls -la go.mod

   # Module path must match imports
   grep "^module" go.mod
   ```

3. **Test import resolution**:
   ```bash
   # Check if import can be resolved
   go list -json github.com/user/package
   ```

4. **Check filters**:
   ```toml
   [content_sources.discovery.code_dependencies]
   follow_local = true  # Make sure local imports allowed
   ```

**Common Issues by Language**:

**Go**:
- Requires go.mod in project
- Requires `go list` command available
- Module path must match import paths

**Python**:
- Requires modules in sys.path
- May need virtual environment activated

**JavaScript/TypeScript**:
- Requires node_modules installed
- Follows Node.js resolution algorithm

**Prevention**:
- Initialize language-specific build system
- Test import resolution manually
- Enable detailed discovery logging
- Document required setup per language

**Related**: [Discovery Configuration](../features/discovery/)

---

## Additional Resources

- [Configuration Guide](../configuration/README.md)
- [Architecture Overview](../architecture/README.md)
- [Operations Guide](../operations/README.md)
- [GitHub Issues](https://github.com/yourusername/go-doc-go/issues) - Report bugs or get help

---

## Getting More Help

If your issue isn't covered here:

1. **Check logs** with verbose output:
   ```bash
   bin/goworker --config config.toml --log-level debug
   ```

2. **Search GitHub issues**:
   - Existing solutions: https://github.com/yourusername/go-doc-go/issues
   - Report new issues with logs and config

3. **Join community**:
   - Discussions forum
   - Slack/Discord channel

4. **Provide details** when asking for help:
   - Go-Doc-Go version: `bin/goworker --version`
   - Go version: `go version`
   - Operating system: `uname -a`
   - Full error message
   - Relevant config sections
   - Steps to reproduce
