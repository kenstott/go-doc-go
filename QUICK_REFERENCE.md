# Go-Doc-Go Quick Reference

**Version**: 1.0
**Language**: Go
**Binary**: `bin/goworker`

This guide provides everything you need to get Go-Doc-Go running quickly.

---

## Installation

### Prerequisites
- Go 1.21 or later
- ONNX Runtime (for ML-based features)

### Build from Source

```bash
## Clone repository
git clone https://github.com/yourusername/go-doc-go.git
cd go-doc-go

## Build worker binary
go build -o bin/goworker ./go/cmd/worker

## Build ontology interview CLI (optional)
go build -o bin/ontology_interview ./go/cmd/ontology_interview

## Verify installation
./bin/goworker --help
```bash

### Quick Start

```bash
## 1. Create minimal config
cat > config.toml << 'EOF'
[processing.job_control]
backend = "sqlite"
path = "./queue.db"

[[content_sources]]
name = "local_files"
type = "file"
base_path = "./documents"
file_pattern = "**/*.{pdf,docx,txt,md}"
EOF

## 2. Run worker
./bin/goworker --config config.toml --workers 4
```toml

---

## CLI Flags

All flags for `bin/goworker`:

### Required

| Flag | Type | Description |
|------|------|-------------|
| `--config` | string | Path to configuration file (TOML format) |

### Optional

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--worker-id` | string | auto-generated | Custom worker ID (format: `worker_<hostname>_<pid>`) |
| `--max-documents` | int | 0 | Maximum number of documents to process (0 = unlimited) |
| `--workers` | int | 1 | Number of concurrent goroutine workers |
| `--instances` | int | 1 | Number of separate worker processes to spawn (multiprocessing) |
| `--shutdown-when-idle` | bool | false | Shutdown worker when no work is available |
| `--shutdown-file` | string | "" | Path to shutdown control file (worker exits when file exists) |
| `--idle-timeout` | duration | 0 | Shutdown after being idle for duration (0 = disabled, e.g., `5m`, `1h`) |

### Examples

```bash
## Basic usage
./bin/goworker --config config.toml

## Process maximum 100 documents with 8 workers
./bin/goworker --config config.toml --max-documents 100 --workers 8

## Run with custom worker ID
./bin/goworker --config config.toml --worker-id prod-worker-01

## Shutdown after 10 minutes of idle time
./bin/goworker --config config.toml --idle-timeout 10m

## Spawn 4 separate worker processes
./bin/goworker --config config.toml --instances 4 --workers 2

## Use shutdown file for graceful shutdown
./bin/goworker --config config.toml --shutdown-file /tmp/shutdown.signal
## (create file to trigger shutdown: touch /tmp/shutdown.signal)
```toml

---

## Environment Variables

Go-Doc-Go supports the following environment variables:

| Variable | Priority | Default | Description |
|----------|----------|---------|-------------|
| `GO_DOC_GO_CONFIG_PATH` | Low | `./config.toml` | Default config file path if `--config` not specified |
| `NUM_WORKERS` | Medium | 1 | Number of goroutine workers if `--workers` not specified |
| `NUM_INSTANCES` | Medium | 1 | Number of worker processes if `--instances` not specified |
| `ANTHROPIC_API_KEY` | - | - | API key for ontology extraction (if using Claude) |

**Priority Order** (highest to lowest):
1. CLI flags (`--workers`, `--instances`)
2. Environment variables (`NUM_WORKERS`, `NUM_INSTANCES`)
3. Config file settings (`max_workers`)
4. Built-in defaults (1 worker)

### Example

```bash
## Use environment variables
export GO_DOC_GO_CONFIG_PATH=/etc/go-doc-go/config.toml
export NUM_WORKERS=8
export ANTHROPIC_API_KEY=sk-ant-...

./bin/goworker  # Uses environment variables
```go

---

## Configuration

Go-Doc-Go uses TOML configuration files. Here are common configurations:

### Minimal Configuration (SQLite)

```toml
## Minimal config for local development
[processing.job_control]
backend = "sqlite"
path = "./queue.db"

[[content_sources]]
name = "local_docs"
type = "file"
base_path = "./documents"
file_pattern = "**/*.{pdf,docx,md,txt}"
```toml

### Production Configuration (PostgreSQL)

```toml
## Production config with PostgreSQL
[processing.job_control]
backend = "postgresql"
path = "postgresql://user:password@localhost:5432/go_doc_go"
claim_timeout = 300
heartbeat_interval = 30
max_retries = 3

[[content_sources]]
name = "s3_documents"
type = "s3"
bucket = "company-docs"
prefix = "processed/"

[embedding]
enabled = true
provider = "fastembed"
model = "BAAI/bge-small-en-v1.5"
dimensions = 384
batch_size = 64

[analytics]
enabled = true

[[analytics.outputs]]
type = "parquet"
path = "./analytics-output"
compression = "snappy"
```toml

### Distributed Workers Configuration

```toml
## Config for horizontal scaling (same on all nodes)
[processing.job_control]
backend = "postgresql"
path = "postgresql://user:pass@postgres-cluster:5432/go_doc_go"
claim_timeout = 300
heartbeat_interval = 30

[[content_sources]]
name = "shared_docs"
type = "file"
base_path = "/mnt/shared/documents"

[embedding]
enabled = true
provider = "fastembed"
model = "BAAI/bge-small-en-v1.5"
batch_size = 64
```bash

Run on multiple machines:
```bash
## Node 1
./bin/goworker --config cluster.toml --workers 8 --worker-id node-01

## Node 2
./bin/goworker --config cluster.toml --workers 8 --worker-id node-02

## Node N
./bin/goworker --config cluster.toml --workers 8 --worker-id node-N
```toml

### Configuration with Ontology Extraction

```toml
[processing.job_control]
backend = "sqlite"
path = "./queue.db"

[[content_sources]]
name = "medical_docs"
type = "file"
base_path = "./medical-documents"

[ontology]
enabled = true
schema_path = "./ontologies/medical.yaml"
diversity_threshold = 0.85
queue_idle_trigger_minutes = 5
min_interval_minutes = 60
```bash

---

## Common Commands

### Development & Testing

```bash
## Build binary
go build -o bin/goworker ./go/cmd/worker

## Test with single document
./bin/goworker --config config.toml --max-documents 1 --workers 1

## Run with debug output (check logs)
./bin/goworker --config config.toml --workers 4 2>&1 | tee worker.log

## Run all Go tests
cd go && go test ./... -v

## Run specific test
cd go && go test ./internal/worker -run TestWorkerDocumentProcessing -v

## Generate test coverage
cd go && go test ./... -coverprofile=coverage.out
cd go && go tool cover -html=coverage.out
```bash

### Production Operations

```bash
## Start worker in background
nohup ./bin/goworker --config /etc/go-doc-go/config.toml --workers 16 > worker.log 2>&1 &

## Monitor worker logs
tail -f worker.log

## Graceful shutdown via signal
kill -SIGTERM <worker-pid>

## Graceful shutdown via file
touch /tmp/shutdown.signal  # If --shutdown-file was set

## Check worker status (via logs)
grep "WORKER COMPLETED" worker.log
grep "Failed to" worker.log  # Check for errors

## Multi-instance deployment
./bin/goworker --config config.toml --instances 4 --workers 8
## This spawns 4 separate processes, each with 8 goroutine workers (32 total workers)
```bash

### Database Operations

```bash
## SQLite: Inspect queue database
sqlite3 ./queue.db "SELECT COUNT(*) FROM documents WHERE status='pending';"
sqlite3 ./queue.db "SELECT COUNT(*) FROM documents WHERE status='completed';"

## PostgreSQL: Check queue status
psql -d go_doc_go -c "SELECT status, COUNT(*) FROM documents GROUP BY status;"
psql -d go_doc_go -c "SELECT worker_id, COUNT(*) FROM documents WHERE status='processing' GROUP BY worker_id;"
```bash

### Performance Tuning

```bash
## Test different worker counts
for workers in 1 2 4 8 16; do
  echo "Testing with $workers workers..."
  time ./bin/goworker --config config.toml --max-documents 100 --workers $workers
done

## Monitor resource usage
top -pid $(pgrep goworker)

## Profile CPU usage (if built with profiling)
go tool pprof http://localhost:6060/debug/pprof/profile
```go

---

## Troubleshooting Quick Tips

### Worker Not Starting

**Problem**: Worker fails to start or exits immediately

**Solutions**:
```bash
## 1. Check config file exists and is valid TOML
./bin/goworker --config config.toml
## Error: Configuration file not found: config.toml

## 2. Validate TOML syntax (use online validator or toml-test)
## Check for missing quotes, brackets, or incorrect nesting

## 3. Check database connectivity
## SQLite: Ensure directory exists and is writable
mkdir -p $(dirname ./queue.db)
touch ./queue.db  # Test write permissions

## PostgreSQL: Test connection
psql "postgresql://user:pass@host:5432/go_doc_go" -c "SELECT 1;"
```go

### No Documents Being Processed

**Problem**: Worker runs but doesn't process any documents

**Solutions**:
```bash
## 1. Check content source path exists
ls -la ./documents  # Verify path from content_sources

## 2. Check file patterns match
ls ./documents/**/*.pdf  # Test glob pattern

## 3. Verify documents in queue
sqlite3 ./queue.db "SELECT COUNT(*) FROM documents WHERE status='pending';"

## 4. Check for stuck claims (expired but not released)
## SQLite:
sqlite3 ./queue.db "UPDATE documents SET status='pending', worker_id=NULL, claimed_at=NULL WHERE status='processing' AND claimed_at < datetime('now', '-10 minutes');"
```sql

### Performance Issues

**Problem**: Processing is slow

**Solutions**:
```bash
## 1. Increase worker count
./bin/goworker --config config.toml --workers 8  # or 16, 32

## 2. Use multiple instances (process-level parallelism)
./bin/goworker --config config.toml --instances 4 --workers 8

## 3. Check embedding configuration (if enabled)
## Reduce batch_size if out of memory
## Increase batch_size if CPU-bound

## 4. Enable Parquet analytics for better performance
## (Parquet writes are buffered, more efficient than per-document writes)

## 5. Use PostgreSQL instead of SQLite for better concurrency
```sql

### Memory Issues

**Problem**: Worker uses too much memory or crashes with OOM

**Solutions**:
```bash
## 1. Reduce number of workers
./bin/goworker --config config.toml --workers 4  # Instead of 16

## 2. Reduce embedding batch size (in config.toml)
[embedding]
batch_size = 16  # Instead of 64

## 3. Process in smaller batches
./bin/goworker --config config.toml --max-documents 100
## Run multiple times instead of processing everything at once

## 4. Monitor memory usage
ps aux | grep goworker
top -pid $(pgrep goworker)
```go

### Database Connection Issues

**Problem**: PostgreSQL connection errors

**Solutions**:
```bash
## 1. Test connection string
psql "postgresql://user:pass@host:5432/go_doc_go" -c "SELECT version();"

## 2. Check connection pool settings (in config.toml)
[processing.job_control]
## Add these if not present:
## pool_size = 20
## max_overflow = 10

## 3. Verify database exists
psql -U postgres -c "CREATE DATABASE go_doc_go;"

## 4. Check network connectivity
telnet postgres-host 5432
```

### Ontology Extraction Not Working

**Problem**: Ontology features not extracting entities

**Solutions**:
```bash
## 1. Verify API key is set
echo $ANTHROPIC_API_KEY

## 2. Check ontology schema file exists
ls -la ./ontologies/schema.yaml

## 3. Verify ontology config (in config.toml)
[ontology]
enabled = true
schema_path = "./ontologies/schema.yaml"

## 4. Check logs for API errors
grep "ontology" worker.log
grep "anthropic" worker.log
```

### Common Error Messages

| Error | Cause | Solution |
|-------|-------|----------|
| `Configuration file not found` | Wrong path or file doesn't exist | Check `--config` path |
| `Failed to parse TOML` | Invalid TOML syntax | Validate config with TOML validator |
| `Failed to connect to database` | Database not accessible | Check connection string and database status |
| `No documents found` | Empty queue or wrong path | Check content_sources paths |
| `Worker timeout` | Document taking too long | Increase `claim_timeout` in config |
| `ONNX Runtime error` | Missing ONNX Runtime library | Install ONNX Runtime dependencies |

---

## Getting Help

- **Documentation**: See `docs/` directory for detailed guides
- **Configuration Guide**: `docs/configuration/README.md`
- **Scaling Guide**: `docs/operations/scaling.md`
- **Ontology Guide**: `docs/features/ontology/README.md`
- **GitHub Issues**: Report bugs and request features

## Quick Links

- [Full Documentation](docs/)
- [Configuration Reference](docs/configuration/README.md)
- [Scaling Guide](docs/operations/scaling.md)
- [Embeddings Guide](docs/features/embeddings/README.md)
- [Ontology System](docs/features/ontology/README.md)
