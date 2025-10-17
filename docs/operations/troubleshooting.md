# Troubleshooting Guide

Comprehensive troubleshooting guide for Go-Doc-Go common issues and solutions.

---

## Table of Contents

- [Worker Issues](#worker-issues)
  - [Worker Not Starting](#worker-not-starting)
  - [Worker Exits Immediately](#worker-exits-immediately)
  - [Worker Hangs/Becomes Unresponsive](#worker-hangsbecom es-unresponsive)
- [Configuration Issues](#configuration-issues)
  - [Configuration File Not Found](#configuration-file-not-found)
  - [Invalid TOML Syntax](#invalid-toml-syntax)
  - [Missing Required Fields](#missing-required-fields)
- [Database Issues](#database-issues)
  - [PostgreSQL Connection Errors](#postgresql-connection-errors)
  - [SQLite Database Locked](#sqlite-database-locked)
  - [Document Claiming Failures](#document-claiming-failures)
- [Document Processing Issues](#document-processing-issues)
  - [No Documents Found](#no-documents-found)
  - [Documents Not Being Processed](#documents-not-being-processed)
  - [Processing Errors](#processing-errors)
- [Performance Issues](#performance-issues)
  - [Slow Processing](#slow-processing)
  - [High Memory Usage](#high-memory-usage)
  - [High CPU Usage](#high-cpu-usage)
- [ONNX Runtime Issues](#onnx-runtime-issues)
  - [Library Not Found](#library-not-found)
  - [Model Loading Errors](#model-loading-errors)
  - [Embedding Generation Failures](#embedding-generation-failures)
- [Ontology Issues](#ontology-issues)
  - [Ontology Extraction Not Working](#ontology-extraction-not-working)
  - [API Key Issues](#api-key-issues)
  - [Schema Validation Errors](#schema-validation-errors)
- [Common Error Messages](#common-error-messages)

---

## Worker Issues

### Worker Not Starting

**Symptoms**: Worker binary fails to start or exits with error immediately after launch.

**Common Causes**:
1. Configuration file not found or inaccessible
2. Invalid configuration syntax
3. Database connection failure
4. Missing dependencies

**Solutions**:

#### Check Configuration File
```bash
## Verify config file exists and is readable
ls -la config.toml
cat config.toml

## Try running with explicit config path
./bin/goworker --config /absolute/path/to/config.toml
```bash

#### Verify Database Connection
```bash
## For SQLite: Check directory exists and is writable
mkdir -p $(dirname ./data/jobs.db)
touch ./data/jobs.db

## For PostgreSQL: Test connection
psql "postgresql://user:pass@host:5432/go_doc_go" -c "SELECT 1;"
```bash

#### Check Dependencies
```bash
## Verify Go version
go version  # Should be 1.21 or later

## Check ONNX Runtime (if using embeddings)
echo $ONNXRUNTIME_SHARED_LIBRARY_PATH
ls -la $ONNXRUNTIME_SHARED_LIBRARY_PATH
```bash

#### Enable Debug Logging
```bash
## Run with full output to see detailed error messages
./bin/goworker --config config.toml 2>&1 | tee worker.log
```bash

---

### Worker Exits Immediately

**Symptoms**: Worker starts but exits within seconds without processing documents.

**Common Causes**:
1. `--max-documents 0` with `--shutdown-when-idle true`
2. Empty queue with `--shutdown-when-idle true`
3. Configuration validation failure
4. Content source path doesn't exist

**Solutions**:

#### Check CLI Flags
```bash
## Don't use --shutdown-when-idle for development
./bin/goworker --config config.toml --workers 4

## If using max-documents, ensure value > 0
./bin/goworker --config config.toml --max-documents 10
```toml

#### Verify Content Sources
```toml
## In config.toml, check that paths exist
[[content_sources]]
name = "documents"
type = "file"
base_path = "./docs"  # This directory must exist
file_pattern = "**/*.{pdf,docx,txt}"
```bash

```bash
## Create content source directory if missing
mkdir -p ./docs

## Verify files match pattern
ls ./docs/**/*.pdf
```bash

#### Check Queue Status
```bash
## For SQLite
sqlite3 ./data/jobs.db "SELECT COUNT(*) FROM documents WHERE status='pending';"

## For PostgreSQL
psql -d go_doc_go -c "SELECT status, COUNT(*) FROM documents GROUP BY status;"
```go

---

### Worker Hangs/Becomes Unresponsive

**Symptoms**: Worker appears to be running but doesn't process documents or respond to signals.

**Common Causes**:
1. Deadlock in document processing
2. Network timeout (waiting for remote resource)
3. Large file causing memory exhaustion
4. Database connection lost

**Solutions**:

#### Check Worker Status
```bash
## Check if process is actually running
ps aux | grep goworker

## Check CPU and memory usage
top -pid $(pgrep goworker)
```bash

#### Check Database Connection
```bash
## For PostgreSQL, check for long-running queries
psql -d go_doc_go -c "SELECT pid, state, query, now() - query_start AS duration FROM pg_stat_activity WHERE datname = 'go_doc_go' ORDER BY duration DESC;"
```bash

#### Check for Stuck Documents
```bash
## Find documents that have been "processing" for too long
## SQLite
sqlite3 ./data/jobs.db "SELECT doc_id, worker_id, claimed_at FROM documents WHERE status='processing' AND datetime(claimed_at) < datetime('now', '-10 minutes');"

## PostgreSQL
psql -d go_doc_go -c "SELECT doc_id, worker_id, claimed_at FROM documents WHERE status='processing' AND claimed_at < NOW() - INTERVAL '10 minutes';"
```bash

#### Force Release Stuck Documents
```bash
## SQLite
sqlite3 ./data/jobs.db "UPDATE documents SET status='pending', worker_id=NULL, claimed_at=NULL WHERE status='processing' AND datetime(claimed_at) < datetime('now', '-10 minutes');"

## PostgreSQL
psql -d go_doc_go -c "UPDATE documents SET status='pending', worker_id=NULL, claimed_at=NULL WHERE status='processing' AND claimed_at < NOW() - INTERVAL '10 minutes';"
```bash

#### Graceful Shutdown
```bash
## Send SIGTERM for graceful shutdown
kill -SIGTERM $(pgrep goworker)

## If unresponsive after 30 seconds, force kill
sleep 30 && kill -SIGKILL $(pgrep goworker)
```go

---

## Configuration Issues

### Configuration File Not Found

**Error Message**: `Configuration file not found: config.toml`

**Solutions**:

```bash
## Use absolute path
./bin/goworker --config /Users/you/project/config.toml

## Or set environment variable
export GO_DOC_GO_CONFIG_PATH=/path/to/config.toml
./bin/goworker

## Verify file exists
ls -la config.toml
```toml

---

### Invalid TOML Syntax

**Error Message**: `Failed to parse TOML: ...`

**Common TOML Mistakes**:

1. **Using YAML syntax** (colons instead of equals):
```toml
## ❌ WRONG (YAML syntax)
storage:
  backend: "sqlite"

## ✅ CORRECT (TOML syntax)
[storage]
backend = "sqlite"
```toml

2. **Missing quotes around strings**:
```toml
## ❌ WRONG
path = /path/to/file

## ✅ CORRECT
path = "/path/to/file"
```toml

3. **Incorrect array syntax**:
```toml
## ❌ WRONG
[[content_sources]]
element_types = [paragraph, table]

## ✅ CORRECT
[[content_sources]]
element_types = ["paragraph", "table"]
```toml

4. **Incorrect nested sections**:
```toml
## ❌ WRONG
[processing]
[job_control]
backend = "sqlite"

## ✅ CORRECT
[processing.job_control]
backend = "sqlite"
```bash

**Validation**:
```bash
## Use online TOML validator
## Or install tomlv
go install github.com/BurntSushi/toml/cmd/tomlv@latest
tomlv config.toml
```toml

---

### Missing Required Fields

**Error Message**: `Missing required configuration field: ...`

**Required Fields by Section**:

#### Job Control (Required)
```toml
[processing.job_control]
backend = "sqlite"  # or "postgresql"
path = "./data/jobs.db"  # or PostgreSQL connection string
```toml

#### Content Sources (At least one required)
```toml
[[content_sources]]
name = "documents"
type = "file"
base_path = "./docs"
```bash

#### Solution
```bash
## Start with minimal working config
cat > config.toml << 'EOF'
[processing.job_control]
backend = "sqlite"
path = "./data/jobs.db"

[[content_sources]]
name = "local_files"
type = "file"
base_path = "./documents"
file_pattern = "**/*.{pdf,docx,txt,md}"
EOF
```

---

## Database Issues

### PostgreSQL Connection Errors

**Error Message**: `Failed to connect to database: ...`

**Common Causes**:
1. PostgreSQL not running
2. Incorrect credentials
3. Network/firewall blocking connection
4. Database doesn't exist

**Solutions**:

#### Test Connection
```bash
## Test with psql
psql "postgresql://user:pass@localhost:5432/go_doc_go" -c "SELECT version();"
```bash

#### Check PostgreSQL Status
```bash
## Check if PostgreSQL is running
brew services list | grep postgresql  # macOS
systemctl status postgresql           # Linux

## Start if needed
brew services start postgresql         # macOS
sudo systemctl start postgresql        # Linux
```bash

#### Create Database
```bash
## Connect as superuser and create database
psql -U postgres -c "CREATE DATABASE go_doc_go;"

## Grant permissions
psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE go_doc_go TO your_user;"
```toml

#### Check Connection String Format
```toml
[processing.job_control]
backend = "postgresql"
## Format: postgresql://username:password@host:port/database
path = "postgresql://user:pass@localhost:5432/go_doc_go"

## Optional connection parameters
pool_size = 20
max_overflow = 10
```bash

#### Firewall/Network Issues
```bash
## Test network connectivity
telnet localhost 5432
## Or
nc -zv localhost 5432

## Check PostgreSQL config allows connections
## /etc/postgresql/.../postgresql.conf
listen_addresses = '*'  # or specific IP

## Check pg_hba.conf allows your user
## /etc/postgresql/.../pg_hba.conf
host all all 0.0.0.0/0 md5
```sql

---

### SQLite Database Locked

**Error Message**: `database is locked`

**Common Causes**:
1. Multiple workers accessing same SQLite database
2. Previous worker didn't release locks
3. File system locking issues

**Solutions**:

#### Use One Worker with SQLite
```bash
## SQLite is single-writer, use only 1 worker
./bin/goworker --config config.toml --workers 4
## (4 goroutine workers, 1 process - OK)

## ❌ DON'T do this with SQLite:
./bin/goworker --config config.toml --instances 4
## (4 separate processes competing for same SQLite file)
```toml

#### Switch to PostgreSQL for Multiple Workers
```toml
## For distributed/multiple workers, use PostgreSQL
[processing.job_control]
backend = "postgresql"
path = "postgresql://user:pass@localhost:5432/go_doc_go"
```bash

#### Fix Stuck Locks
```bash
## Close any open connections
killall goworker

## Remove WAL files if safe (database must not be in use)
rm -f ./data/jobs.db-wal ./data/jobs.db-shm

## Verify database integrity
sqlite3 ./data/jobs.db "PRAGMA integrity_check;"
```sql

---

### Document Claiming Failures

**Error Message**: `Failed to claim document: ...`

**Common Causes**:
1. Database contention (too many workers)
2. Claim timeout too short
3. Database locks not releasing

**Solutions**:

#### Adjust Claim Timeout
```toml
[processing.job_control]
backend = "postgresql"
path = "postgresql://..."
claim_timeout = 600  # Increase to 10 minutes (default: 300)
heartbeat_interval = 30  # Keep heartbeat frequency
```bash

#### Reduce Worker Count
```bash
## If seeing many claim failures, reduce workers
./bin/goworker --config config.toml --workers 4
## Instead of --workers 16
```bash

#### Check Database Performance
```bash
## PostgreSQL: Check for lock contention
psql -d go_doc_go -c "SELECT * FROM pg_locks WHERE NOT granted;"

## Check table size and indexes
psql -d go_doc_go -c "SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size FROM pg_tables WHERE schemaname NOT IN ('pg_catalog', 'information_schema') ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;"
```go

---

## Document Processing Issues

### No Documents Found

**Symptoms**: Worker runs but reports "No documents found" or exits immediately.

**Common Causes**:
1. Content source path doesn't exist
2. File pattern doesn't match files
3. Files already processed (all in "completed" status)

**Solutions**:

#### Verify Content Source
```bash
## Check directory exists
ls -la ./documents

## Check files match pattern
ls ./documents/**/*.pdf
ls ./documents/**/*.docx

## Count matching files
find ./documents -name "*.pdf" -o -name "*.docx" | wc -l
```toml

#### Update File Pattern
```toml
[[content_sources]]
name = "documents"
type = "file"
base_path = "./documents"
## Try broader pattern
file_pattern = "**/*"  # All files
## Or specific extensions
file_pattern = "**/*.{pdf,docx,xlsx,txt,md,html,json,csv,xml}"
```bash

#### Check Queue Status
```bash
## SQLite: Count documents by status
sqlite3 ./data/jobs.db "SELECT status, COUNT(*) FROM documents GROUP BY status;"

## PostgreSQL
psql -d go_doc_go -c "SELECT status, COUNT(*) FROM documents GROUP BY status;"
```bash

#### Reset Completed Documents
```bash
## To reprocess documents, reset their status
## SQLite
sqlite3 ./data/jobs.db "UPDATE documents SET status='pending', worker_id=NULL, claimed_at=NULL, completed_at=NULL WHERE status='completed';"

## PostgreSQL
psql -d go_doc_go -c "UPDATE documents SET status='pending', worker_id=NULL, claimed_at=NULL, completed_at=NULL WHERE status='completed';"
```go

---

### Documents Not Being Processed

**Symptoms**: Documents in queue but worker doesn't process them.

**Solutions**:

#### Check Document Status
```bash
## SQLite
sqlite3 ./data/jobs.db "SELECT doc_id, status, worker_id, claimed_at FROM documents LIMIT 10;"

## PostgreSQL
psql -d go_doc_go -c "SELECT doc_id, status, worker_id, claimed_at FROM documents LIMIT 10;"
```bash

#### Check for Stuck Documents
```bash
## Documents stuck in "processing" status
## SQLite
sqlite3 ./data/jobs.db "SELECT COUNT(*) FROM documents WHERE status='processing';"

## PostgreSQL
psql -d go_doc_go -c "SELECT COUNT(*) FROM documents WHERE status='processing';"
```bash

#### Release Stuck Documents
```bash
## Reset stuck documents to pending
## SQLite
sqlite3 ./data/jobs.db "UPDATE documents SET status='pending', worker_id=NULL, claimed_at=NULL WHERE status='processing';"

## PostgreSQL
psql -d go_doc_go -c "UPDATE documents SET status='pending', worker_id=NULL, claimed_at=NULL WHERE status='processing';"
```go

---

### Processing Errors

**Error Message**: `Failed to process document: ...`

**Common Causes**:
1. Corrupt document file
2. Unsupported format
3. Missing parser dependencies
4. Out of memory

**Solutions**:

#### Check Document Format
```bash
## Verify file type
file ./documents/problem.pdf

## Check file size
ls -lh ./documents/problem.pdf

## Try opening manually
open ./documents/problem.pdf  # macOS
xdg-open ./documents/problem.pdf  # Linux
```bash

#### Skip Problematic Documents
```bash
## Mark document as failed to skip it
## SQLite
sqlite3 ./data/jobs.db "UPDATE documents SET status='failed', error='Manual skip' WHERE doc_id='problem_doc_id';"

## PostgreSQL
psql -d go_doc_go -c "UPDATE documents SET status='failed', error='Manual skip' WHERE doc_id='problem_doc_id';"
```bash

#### Check Logs for Details
```bash
## Search logs for specific document
grep "problem_doc_id" worker.log

## Find all processing errors
grep "Failed to process" worker.log | tail -20
```

---

## Performance Issues

### Slow Processing

**Symptoms**: Documents take much longer than expected to process.

**Common Causes**:
1. Too few workers
2. Large documents
3. Embedding generation enabled
4. Database bottleneck
5. Network latency (remote storage)

**Solutions**:

#### Increase Worker Count
```bash
## Start with 4-8 workers
./bin/goworker --config config.toml --workers 8

## For multi-core systems, try up to 16
./bin/goworker --config config.toml --workers 16
```bash

#### Use Multiple Instances
```bash
## Spawn multiple worker processes
./bin/goworker --config config.toml --instances 4 --workers 4
## Total: 4 processes × 4 workers = 16 concurrent workers
```toml

#### Optimize Embedding Configuration
```toml
[embedding]
enabled = true
batch_size = 64  # Increase for better throughput
cache_embeddings = true  # Enable caching

## Reduce context window for faster embedding
contextual = false  # Or reduce predecessor/successor counts
```toml

#### Use Faster Storage
```toml
## Use local SSD instead of network storage
[[content_sources]]
type = "file"
base_path = "/local/ssd/documents"  # Instead of /network/share/documents
```bash

#### Monitor Resource Usage
```bash
## Check CPU usage
top -pid $(pgrep goworker)

## Check I/O wait
iostat -x 1

## Profile with pprof (if compiled with profiling)
go tool pprof http://localhost:6060/debug/pprof/profile
```go

---

### High Memory Usage

**Symptoms**: Worker consumes excessive RAM, possibly causing OOM kills.

**Common Causes**:
1. Too many concurrent workers
2. Large embedding batches
3. Memory leaks
4. Large documents in memory

**Solutions**:

#### Reduce Worker Count
```bash
## Fewer workers = less memory
./bin/goworker --config config.toml --workers 4
## Instead of --workers 16
```toml

#### Reduce Embedding Batch Size
```toml
[embedding]
enabled = true
batch_size = 16  # Reduce from 64
```bash

#### Process in Smaller Batches
```bash
## Process documents in chunks
./bin/goworker --config config.toml --max-documents 100

## Run multiple times instead of processing everything at once
for i in {1..10}; do
  ./bin/goworker --config config.toml --max-documents 100
done
```bash

#### Monitor Memory Usage
```bash
## Watch memory over time
while true; do
  ps aux | grep goworker | grep -v grep
  sleep 5
done

## Or use top
top -pid $(pgrep goworker)
```bash

#### Set Memory Limits
```bash
## Linux: Use cgroups to limit memory
systemd-run --scope -p MemoryMax=4G ./bin/goworker --config config.toml

## Or use ulimit
ulimit -v 4194304  # 4GB in KB
./bin/goworker --config config.toml
```toml

---

### High CPU Usage

**Symptoms**: Worker uses 100% CPU or slows down other processes.

**Common Causes**:
1. Too many workers for available cores
2. CPU-intensive embedding model
3. Inefficient parsing

**Solutions**:

#### Match Workers to CPU Cores
```bash
## Check CPU count
nproc  # Linux
sysctl -n hw.ncpu  # macOS

## Use 1-2 workers per core
./bin/goworker --config config.toml --workers 8  # For 4-core CPU
```toml

#### Use Lighter Embedding Model
```toml
[embedding]
enabled = true
## Switch from large to small model
model_path = "./models/all-MiniLM-L6-v2"  # Faster, smaller
## Instead of "./models/all-mpnet-base-v2"  # Slower, larger
```bash

#### Adjust Process Priority
```bash
## Run with lower priority (nice)
nice -n 10 ./bin/goworker --config config.toml
```bash

---

## ONNX Runtime Issues

### Library Not Found

**Error Message**: `onnxruntime shared library not found`

**Solutions**:

#### Install ONNX Runtime
```bash
## Python pip (easiest)
pip install onnxruntime

## Or download from https://github.com/microsoft/onnxruntime/releases
```bash

#### Set Library Path
```bash
## Find library location
find ~/.venv -name "libonnxruntime*.dylib" -o -name "libonnxruntime*.so"

## Set environment variable
export ONNXRUNTIME_SHARED_LIBRARY_PATH="/path/to/libonnxruntime.1.23.0.dylib"

## Run worker
./bin/goworker --config config.toml
```bash

#### macOS: Fix Library Linking
```bash
## Check library dependencies
otool -L bin/goworker | grep onnx

## If needed, update library path
install_name_tool -change @rpath/libonnxruntime.dylib \
  /absolute/path/to/libonnxruntime.1.23.0.dylib \
  bin/goworker
```go

---

### Model Loading Errors

**Error Message**: `Failed to load ONNX model: ...`

**Common Causes**:
1. Model file doesn't exist
2. Model format incompatible
3. Incorrect model path

**Solutions**:

#### Verify Model Files
```bash
## Check model directory
ls -la ./models/all-MiniLM-L6-v2/

## Should contain:
## - model.onnx
## - tokenizer.json (or vocab.txt)
## - config.json
```python

#### Re-export Model
```python
## Export model to ONNX format
from optimum.onnxruntime import ORTModelForFeatureExtraction
from transformers import AutoTokenizer

model_id = 'sentence-transformers/all-MiniLM-L6-v2'
ort_model = ORTModelForFeatureExtraction.from_pretrained(model_id, export=True)
tokenizer = AutoTokenizer.from_pretrained(model_id)

ort_model.save_pretrained('./models/all-MiniLM-L6-v2')
tokenizer.save_pretrained('./models/all-MiniLM-L6-v2')
```toml

#### Verify Configuration
```toml
[embedding]
enabled = true
provider = "onnx"
model_path = "./models/all-MiniLM-L6-v2"  # Directory containing model.onnx
```

---

### Embedding Generation Failures

**Error Message**: `Failed to generate embeddings: ...`

**Solutions**:

#### Disable Embeddings Temporarily
```toml
## Test without embeddings first
[embedding]
enabled = false
```toml

#### Reduce Batch Size
```toml
[embedding]
enabled = true
batch_size = 8  # Start small
```toml

#### Check Model Dimensions
```toml
[embedding]
enabled = true
dimensions = 384  # Must match model output
## all-MiniLM-L6-v2: 384 dimensions
## all-mpnet-base-v2: 768 dimensions
```

---

## Ontology Issues

### Ontology Extraction Not Working

**Symptoms**: Documents processed but no ontology entities extracted.

**Common Causes**:
1. API key not set
2. Ontology not enabled in config
3. Schema file not found
4. Insufficient document diversity

**Solutions**:

#### Check API Key
```bash
## Verify Anthropic API key is set
echo $ANTHROPIC_API_KEY

## Set if missing
export ANTHROPIC_API_KEY=your_key_here
```toml

#### Enable Ontology in Config
```toml
[ontology]
enabled = true
schema_path = "./ontologies/your-schema.yaml"
diversity_threshold = 0.85
queue_idle_trigger_minutes = 5
min_interval_minutes = 60
```bash

#### Verify Schema File
```bash
## Check schema exists
ls -la ./ontologies/your-schema.yaml

## Validate YAML syntax
python -c "import yaml; yaml.safe_load(open('./ontologies/your-schema.yaml'))"
```bash

#### Check Logs
```bash
## Search for ontology-related messages
grep -i "ontology" worker.log
grep -i "anthropic" worker.log
```

---

### API Key Issues

**Error Message**: `API key not found` or `Authentication failed`

**Solutions**:

```bash
## Set API key
export ANTHROPIC_API_KEY=sk-ant-your-key-here

## Verify it's set
env | grep ANTHROPIC

## Add to shell profile for persistence
echo 'export ANTHROPIC_API_KEY=sk-ant-your-key-here' >> ~/.bashrc
source ~/.bashrc
```bash

---

### Schema Validation Errors

**Error Message**: `Invalid ontology schema: ...`

**Common Causes**:
1. Invalid YAML syntax
2. Missing required fields
3. Incorrect schema structure

**Solutions**:

#### Validate YAML
```bash
## Check syntax
python -c "import yaml; print(yaml.safe_load(open('./ontologies/schema.yaml')))"
```bash

#### Check Required Fields
```yaml
## Minimal valid schema
name: my_ontology
domain: general
version: "1.0"

element_entity_mappings:
  - domain: "general"
    entity_type: "Person"
    element_types: ["paragraph"]
    extraction_rules:
      - type: "regex_pattern"
        pattern: '[A-Z][a-z]+ [A-Z][a-z]+'
```bash

#### Use Interview Tool
```bash
## Create schema interactively
bin/ontology_interview ./corpus.parquet ./schema.yaml
```bash

---

## Common Error Messages

| Error Message | Cause | Solution |
|--------------|-------|----------|
| `Configuration file not found` | Wrong path or file doesn't exist | Check --config path, use absolute path |
| `Failed to parse TOML` | Invalid TOML syntax | Validate config with tomlv, check for YAML syntax |
| `Failed to connect to database` | Database not accessible | Check connection string, verify database is running |
| `No documents found` | Empty queue or wrong path | Check content_sources paths, verify files exist |
| `Worker timeout` | Document taking too long | Increase claim_timeout in config |
| `ONNX Runtime error` | Missing ONNX Runtime library | Install ONNX Runtime, set ONNXRUNTIME_SHARED_LIBRARY_PATH |
| `database is locked` | Multiple SQLite writers | Use PostgreSQL for multiple workers |
| `Failed to claim document` | Database contention | Reduce worker count, increase claim_timeout |
| `Out of memory` | Too many workers/large batch | Reduce --workers, reduce embedding batch_size |
| `Failed to load model` | Model file not found | Verify model_path, re-export model |
| `API key not found` | ANTHROPIC_API_KEY not set | Export API key environment variable |

---

## Getting More Help

### Enable Debug Logging
```bash
## Capture all output for debugging
./bin/goworker --config config.toml 2>&1 | tee -a debug.log
```bash

### Check System Resources
```bash
## Disk space
df -h

## Memory
free -h  # Linux
vm_stat  # macOS

## CPU
top
htop  # If installed
```bash

### Collect Diagnostic Information
```bash
## System info
uname -a
go version

## Worker info
./bin/goworker --help

## Config summary
cat config.toml

## Database status (SQLite)
sqlite3 ./data/jobs.db "SELECT status, COUNT(*) FROM documents GROUP BY status;"

## Database status (PostgreSQL)
psql -d go_doc_go -c "SELECT status, COUNT(*) FROM documents GROUP BY status;"
```go

### Report Issues
When reporting issues, include:
1. Go-Doc-Go version (`go version` output)
2. Operating system and version
3. Complete error message
4. Configuration file (sanitized - remove passwords)
5. Steps to reproduce
6. Relevant log output

---

## Related Documentation

- [Quick Reference](../../QUICK_REFERENCE.md) - Common commands and configurations
- [Scaling Guide](scaling.md) - Performance optimization and distributed setup
- [Configuration Reference](../configuration/README.md) - All configuration options
- [Getting Started](../getting-started/README.md) - Initial setup guide

---

**Last Updated**: 2025-01-16
---

## Related Documentation

- **Previous**: [Monitoring](monitoring.md)
- **Up**: [Documentation Home](../README.md)

### Quick Links

- [Documentation Home](../README.md)
- [Quick Reference](../../QUICK_REFERENCE.md)
- [CLI Reference](../reference/cli.md)
- [Configuration Overview](../configuration/README.md)
- [Scaling Guide](scaling.md)
