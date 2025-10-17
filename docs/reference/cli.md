# CLI Reference

Complete command-line interface reference for the Go-Doc-Go worker (`goworker` binary).

---

## Table of Contents

- [Overview](#overview)
- [Basic Usage](#basic-usage)
- [Command-Line Flags](#command-line-flags)
  - [Required Flags](#required-flags)
  - [Optional Flags](#optional-flags)
- [Environment Variables](#environment-variables)
- [Configuration Priority](#configuration-priority)
- [Usage Examples](#usage-examples)
- [Exit Codes](#exit-codes)
- [Shutdown Methods](#shutdown-methods)

---

## Overview

The `goworker` binary is the main worker process for Go-Doc-Go. It discovers, claims, and processes documents according to the configuration file.

**Binary Location**: `bin/goworker` (build from `go/cmd/worker`)

**Build Command**:
```bash
go build -o bin/goworker ./go/cmd/worker
```bash

---

## Basic Usage

```bash
./bin/goworker --config path/to/config.toml [OPTIONS]
```bash

**Minimal Usage**:
```bash
## Use default config location (./config.toml)
./bin/goworker --config config.toml

## Process specific number of documents
./bin/goworker --config config.toml --max-documents 100

## Use multiple concurrent workers
./bin/goworker --config config.toml --workers 8
```bash

---

## Command-Line Flags

### Required Flags

#### `--config` (string)

**Description**: Path to TOML configuration file

**Default**: None (required, but can be set via `GO_DOC_GO_CONFIG_PATH` environment variable)

**Examples**:
```bash
## Relative path
./bin/goworker --config config.toml

## Absolute path
./bin/goworker --config /etc/go-doc-go/config.toml

## Using environment variable
export GO_DOC_GO_CONFIG_PATH=/etc/go-doc-go/config.toml
./bin/goworker
```toml

**Notes**:
- File must be in TOML format
- File must exist at specified path
- If not provided via flag, checks `GO_DOC_GO_CONFIG_PATH` environment variable
- If neither flag nor env var provided, defaults to `./config.toml`

---

### Optional Flags

#### `--worker-id` (string)

**Description**: Custom worker identifier for tracking and logging

**Default**: Auto-generated as `worker_<hostname>_<pid>`

**Examples**:
```bash
## Auto-generated ID (default)
./bin/goworker --config config.toml
## Creates ID like: worker_myhost_12345

## Custom ID
./bin/goworker --config config.toml --worker-id prod-worker-01

## Custom ID with instance number
./bin/goworker --config config.toml --worker-id node01-worker-1
```toml

**Notes**:
- Used for document claiming and tracking
- Visible in logs and database records
- Must be unique across all workers accessing same job control database
- Useful for identifying workers in distributed deployments

---

#### `--max-documents` (int)

**Description**: Maximum number of documents to process before shutting down

**Default**: `0` (unlimited - process continuously)

**Examples**:
```bash
## Process unlimited documents (default)
./bin/goworker --config config.toml

## Process exactly 100 documents then exit
./bin/goworker --config config.toml --max-documents 100

## Process just 1 document (useful for testing)
./bin/goworker --config config.toml --max-documents 1
```toml

**Notes**:
- Worker exits gracefully after processing specified number
- Useful for testing, debugging, and batch processing
- Value of `0` means unlimited (keep processing until stopped)
- Worker completes in-progress documents before exiting

---

#### `--workers` (int)

**Description**: Number of concurrent goroutine workers within single process

**Default**: `0` (uses `NUM_WORKERS` env var, defaults to `1` if not set)

**Examples**:
```bash
## Single worker (default)
./bin/goworker --config config.toml

## 4 concurrent workers
./bin/goworker --config config.toml --workers 4

## 16 concurrent workers (high concurrency)
./bin/goworker --config config.toml --workers 16

## Use environment variable
export NUM_WORKERS=8
./bin/goworker --config config.toml
```toml

**Notes**:
- Increases parallelism within single process
- Each worker claims and processes documents concurrently
- Recommended: 1-2 workers per CPU core
- More workers = higher throughput, but more memory usage
- Works with both SQLite (single process) and PostgreSQL (distributed)

**Priority Order**:
1. `--workers` flag (highest)
2. `NUM_WORKERS` environment variable
3. `max_workers` in config file (ignored - kept for compatibility)
4. Default: 1

---

#### `--instances` (int)

**Description**: Number of separate worker processes to spawn (multiprocessing)

**Default**: `1` (single process)

**Examples**:
```bash
## Single process (default)
./bin/goworker --config config.toml

## Spawn 4 separate worker processes
./bin/goworker --config config.toml --instances 4

## Combine with --workers for maximum parallelism
./bin/goworker --config config.toml --instances 4 --workers 4
## Result: 4 processes × 4 workers = 16 total concurrent workers

## Use environment variable
export NUM_INSTANCES=4
./bin/goworker --config config.toml
```toml

**Notes**:
- Creates multiple independent worker processes
- Each instance is a separate OS process
- Provides process-level fault isolation
- Better utilizes multiple CPU cores
- Each instance inherits all other flags (`--workers`, `--max-documents`, etc.)
- Parent process waits for all instances to complete

**When to Use**:
- Fault tolerance (one process crash doesn't affect others)
- Multi-core systems (better CPU utilization)
- Distributed workloads across single machine

**Priority Order**:
1. `--instances` flag (highest)
2. `NUM_INSTANCES` environment variable
3. Default: 1

---

#### `--shutdown-when-idle` (bool)

**Description**: Automatically shutdown worker when no work is available

**Default**: `false`

**Examples**:
```bash
## Keep running even when idle (default)
./bin/goworker --config config.toml

## Shutdown when idle (batch processing mode)
./bin/goworker --config config.toml --shutdown-when-idle

## Useful for cron jobs
*/5 * * * * /path/to/bin/goworker --config config.toml --shutdown-when-idle
```toml

**Notes**:
- Worker exits gracefully when queue is empty
- Useful for batch processing or scheduled jobs
- Not recommended for continuous processing
- Checks for work periodically before shutting down

**Use Cases**:
- **Batch processing**: Process a specific batch then exit
- **Scheduled jobs**: Run via cron/systemd timer
- **Development**: Quick test runs
- **Resource management**: Free resources when no work available

---

#### `--shutdown-file` (string)

**Description**: Path to shutdown control file - worker exits when file exists

**Default**: None (shutdown file monitoring disabled)

**Examples**:
```bash
## Enable shutdown file monitoring
./bin/goworker --config config.toml --shutdown-file /tmp/shutdown.signal

## In another terminal, trigger shutdown
touch /tmp/shutdown.signal

## Worker detects file and initiates graceful shutdown
```

**Notes**:
- Worker checks for file existence every 5 seconds
- Graceful shutdown when file is detected
- File is NOT deleted by worker
- Useful for external control of worker lifecycle

**Use Cases**:
- **Orchestration**: External scripts control worker shutdown
- **Maintenance windows**: Trigger shutdown before maintenance
- **Deployment**: Gracefully stop workers during updates
- **Testing**: Programmatic control in tests

---

#### `--idle-timeout` (duration)

**Description**: Automatically shutdown after being idle for specified duration

**Default**: `0` (disabled - no idle timeout)

**Format**: Go duration format (e.g., `5m`, `1h`, `30s`)

**Examples**:
```bash
## No idle timeout (default)
./bin/goworker --config config.toml

## Shutdown after 5 minutes of idle time
./bin/goworker --config config.toml --idle-timeout 5m

## Shutdown after 1 hour of idle time
./bin/goworker --config config.toml --idle-timeout 1h

## Shutdown after 30 seconds (testing)
./bin/goworker --config config.toml --idle-timeout 30s
```toml

**Notes**:
- Timer starts when worker becomes idle (no documents to process)
- Timer resets when work becomes available again
- Graceful shutdown when timeout reached
- Value of `0` disables idle timeout

**Use Cases**:
- **Dynamic workloads**: Free resources during idle periods
- **Cost optimization**: Reduce cloud compute costs
- **Auto-scaling**: Workers self-terminate when not needed
- **Development**: Automatic cleanup after testing

---

## Environment Variables

Environment variables provide default values for CLI flags and configuration.

### Configuration Environment Variables

| Variable | Type | Default | Description | Priority |
|----------|------|---------|-------------|----------|
| `GO_DOC_GO_CONFIG_PATH` | string | `./config.toml` | Default config file path | Lower than `--config` flag |
| `NUM_WORKERS` | int | `1` | Number of goroutine workers | Between `--workers` flag and config file |
| `NUM_INSTANCES` | int | `1` | Number of worker processes | Lower than `--instances` flag |
| `ANTHROPIC_API_KEY` | string | None | API key for ontology extraction | Required if ontology enabled |

### Setting Environment Variables

```bash
## Bash/Zsh
export GO_DOC_GO_CONFIG_PATH=/etc/go-doc-go/config.toml
export NUM_WORKERS=8
export NUM_INSTANCES=4
export ANTHROPIC_API_KEY=sk-ant-your-key-here

## Fish shell
set -x GO_DOC_GO_CONFIG_PATH /etc/go-doc-go/config.toml
set -x NUM_WORKERS 8

## Temporary (single command)
NUM_WORKERS=8 ./bin/goworker --config config.toml

## Systemd service
[Service]
Environment="NUM_WORKERS=8"
Environment="GO_DOC_GO_CONFIG_PATH=/etc/go-doc-go/config.toml"
```bash

### Using .env Files (with direnv or manual sourcing)

```bash
## .env file
GO_DOC_GO_CONFIG_PATH=/etc/go-doc-go/config.toml
NUM_WORKERS=8
NUM_INSTANCES=2
ANTHROPIC_API_KEY=sk-ant-your-key-here

## Load with direnv
direnv allow

## Or source manually
set -a
source .env
set +a
```

---

## Configuration Priority

Settings can come from multiple sources. Here's the priority order (highest to lowest):

### 1. Command-Line Flags (Highest Priority)
```bash
./bin/goworker --config config.toml --workers 16
```bash

### 2. Environment Variables
```bash
export NUM_WORKERS=8
./bin/goworker --config config.toml
## Uses NUM_WORKERS=8 from environment
```toml

### 3. Configuration File (Lowest Priority)
```toml
[processing]
max_workers = 4
```toml

**Note**: For `--workers` flag, config file `max_workers` is ignored to maintain predictable defaults.

### Example Priority Resolution

```bash
## Scenario: All three sources specified
## config.toml:
[processing]
max_workers = 4

## Environment:
export NUM_WORKERS=8

## Command line:
./bin/goworker --config config.toml --workers 16

## Result: Uses 16 workers (CLI flag wins)
```toml

---

## Usage Examples

### Development & Testing

```bash
## Test configuration with single document
./bin/goworker --config config.toml --max-documents 1

## Quick test with auto-shutdown
./bin/goworker --config config.toml --max-documents 5 --shutdown-when-idle

## Debug with single worker
./bin/goworker --config config.toml --workers 1 --max-documents 10
```bash

### Production Deployment

```bash
## Standard production worker (8 concurrent workers)
./bin/goworker --config /etc/go-doc-go/config.toml --workers 8

## High-throughput worker (16 workers)
./bin/goworker --config config.toml --workers 16

## Multi-process deployment (4 processes × 4 workers = 16 total)
./bin/goworker --config config.toml --instances 4 --workers 4

## Background worker with logging
nohup ./bin/goworker --config config.toml --workers 8 > worker.log 2>&1 &
```bash

### Distributed Workers

```bash
## Node 1
./bin/goworker --config cluster-config.toml --worker-id node-01 --workers 8

## Node 2
./bin/goworker --config cluster-config.toml --worker-id node-02 --workers 8

## Node 3
./bin/goworker --config cluster-config.toml --worker-id node-03 --workers 8

## All workers coordinate via PostgreSQL job control
```bash

### Batch Processing

```bash
## Process 1000 documents then exit
./bin/goworker --config config.toml --max-documents 1000

## Process until queue empty, then exit
./bin/goworker --config config.toml --shutdown-when-idle

## Process for 1 hour, then exit
./bin/goworker --config config.toml --idle-timeout 1h
```bash

### Graceful Shutdown

```bash
## Start worker with shutdown file
./bin/goworker --config config.toml --shutdown-file /tmp/shutdown.signal

## In another terminal, trigger shutdown
touch /tmp/shutdown.signal

## Worker completes current documents and exits gracefully
```bash

### Combining Multiple Options

```bash
## High-performance batch processing
./bin/goworker \
  --config config.toml \
  --worker-id batch-processor-01 \
  --max-documents 10000 \
  --workers 16 \
  --shutdown-when-idle

## Auto-scaling worker with timeout
./bin/goworker \
  --config config.toml \
  --workers 8 \
  --idle-timeout 10m \
  --shutdown-file /var/run/goworker/shutdown

## Development with custom ID
./bin/goworker \
  --config dev-config.toml \
  --worker-id dev-test-worker \
  --max-documents 5 \
  --workers 1
```

---

## Exit Codes

The `goworker` binary uses standard Unix exit codes:

| Exit Code | Meaning | Description |
|-----------|---------|-------------|
| `0` | Success | Worker completed successfully or shutdown gracefully |
| `1` | General Error | Configuration error, runtime error, or fatal failure |

### Success Scenarios (Exit Code 0)

- Processed all documents up to `--max-documents` limit
- Graceful shutdown via SIGTERM/SIGINT
- Graceful shutdown via shutdown file
- Graceful shutdown via idle timeout
- Queue empty with `--shutdown-when-idle`

### Error Scenarios (Exit Code 1)

- Configuration file not found
- Invalid TOML syntax in config
- Database connection failure
- Failed to create worker
- Runtime panic or fatal error
- Shutdown timeout (worker didn't stop within 30 seconds)

### Checking Exit Codes

```bash
## Bash
./bin/goworker --config config.toml
if [ $? -eq 0 ]; then
  echo "Worker completed successfully"
else
  echo "Worker failed with error"
fi

## Or inline
./bin/goworker --config config.toml && echo "Success" || echo "Failed"

## In scripts
#!/bin/bash
./bin/goworker --config config.toml --max-documents 100
EXIT_CODE=$?

if [ $EXIT_CODE -ne 0 ]; then
  echo "Worker failed with exit code $EXIT_CODE"
  # Send alert, log error, etc.
  exit 1
fi
```

---

## Shutdown Methods

Go-Doc-Go workers support multiple shutdown methods for different use cases:

### 1. Signal-Based Shutdown (Recommended)

```bash
## Start worker
./bin/goworker --config config.toml --workers 8 &
WORKER_PID=$!

## Graceful shutdown via SIGTERM
kill -SIGTERM $WORKER_PID

## Or via SIGINT (Ctrl+C in terminal)
## Press Ctrl+C

## Worker completes current documents and exits within 30 seconds
```

**Behavior**:
- Worker receives signal
- Stops claiming new documents
- Completes currently processing documents
- Closes database connections
- Exits gracefully within 30 seconds
- If timeout exceeded, forced exit

---

### 2. Shutdown File

```bash
## Start with shutdown file monitoring
./bin/goworker --config config.toml --shutdown-file /tmp/shutdown.signal &

## Trigger shutdown by creating file
touch /tmp/shutdown.signal

## Worker detects file (checks every 5 seconds) and shuts down
```

**Behavior**:
- Worker checks for file existence every 5 seconds
- When file detected, initiates graceful shutdown
- Same graceful shutdown process as signals
- File is NOT deleted by worker (you must clean up)

**Use Cases**:
- Orchestration scripts
- External monitoring systems
- Scheduled maintenance
- CI/CD pipelines

---

### 3. Idle Timeout

```bash
## Worker shuts down after 10 minutes of idle time
./bin/goworker --config config.toml --idle-timeout 10m
```bash

**Behavior**:
- Timer starts when worker becomes idle (no documents to process)
- Timer resets if work becomes available
- When timeout reached, graceful shutdown
- Useful for auto-scaling scenarios

---

### 4. Max Documents Limit

```bash
## Process exactly 1000 documents then exit
./bin/goworker --config config.toml --max-documents 1000
```bash

**Behavior**:
- Worker exits after processing specified number of documents
- Counts successful completions only (not failures)
- Completes in-progress documents
- Graceful exit

---

### 5. Shutdown When Idle

```bash
## Exit when queue is empty
./bin/goworker --config config.toml --shutdown-when-idle
```bash

**Behavior**:
- Worker monitors queue for available work
- When no work available, checks again after delay
- If still no work, initiates shutdown
- Useful for batch processing

---

### Combining Shutdown Methods

```bash
## Multiple shutdown triggers (first one wins)
./bin/goworker \
  --config config.toml \
  --max-documents 10000 \
  --idle-timeout 30m \
  --shutdown-file /tmp/shutdown.signal

## Worker will exit when:
## - 10000 documents processed, OR
## - Idle for 30 minutes, OR
## - Shutdown file created, OR
## - SIGTERM/SIGINT received
```

---

## Related Documentation

- [Configuration Reference](../configuration/README.md) - Complete configuration options
- [Quick Reference](../../QUICK_REFERENCE.md) - Common commands and quick start
- [Troubleshooting Guide](../operations/troubleshooting.md) - Diagnose and fix issues
- [Scaling Guide](../operations/scaling.md) - Performance optimization

---

**Last Updated**: 2025-01-16
