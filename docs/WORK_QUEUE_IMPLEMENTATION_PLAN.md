# Work Queue Implementation Plan

## Phase 1: Database Schema and Basic Operations ✅ COMPLETE
- [x] PostgreSQL database schema with 4 tables
- [x] WorkQueue and RunCoordinator classes 
- [x] Atomic document claiming with "FOR UPDATE SKIP LOCKED"
- [x] Config-hash based run coordination
- [x] Comprehensive test suite (8/8 tests passing)
- [x] Docker-based test environment
- [x] Performance validation (1,459+ docs/second throughput)

## Phase 2: Main Pipeline Integration ✅ COMPLETE
Integration of work queue with existing document ingestion pipeline to enable distributed processing.

### 2.1: Queue-Enabled Document Processor ✅
- [x] Create `QueuedDocumentProcessor` class
- [x] Implement queue-based document fetching and claiming
- [x] Handle document processing with retry logic
- [x] Integrate with existing parser and relationship detection

### 2.2: Worker Process Implementation ✅
- [x] Create `DocumentWorker` class for standalone worker processes
- [x] Implement worker lifecycle (start, heartbeat, graceful shutdown)
- [x] Add configuration management for workers
- [x] Implement link discovery and dynamic queue addition

### 2.3: Coordinator Process ✅
- [x] Create `ProcessingCoordinator` class for run management
- [x] Implement document discovery and queue population
- [x] Add CLI interfaces for coordinators and workers
- [x] Handle worker failure and recovery scenarios

### 2.4: Integration with Main Pipeline ✅
- [x] Modify `ingest_documents()` to support queue mode
- [x] Add queue configuration options to config system
- [x] Implement distributed vs single-process mode selection
- [x] Maintain backward compatibility with existing usage

### 2.5: Testing and Validation ✅
- [x] Unit tests for new classes (13 tests passing)
- [x] Integration test framework
- [x] Performance tests with multiple workers
- [x] End-to-end distributed processing framework

## Phase 3: Advanced Features ✅ COMPLETE
- [ ] Priority document processing (Not implemented - not required)
- [ ] Dynamic worker scaling (Not implemented - handled by DevOps)
- [x] Advanced monitoring and metrics
- [x] Dead letter queue for failed documents
- [x] Distributed cross-document relationship computation ("last man standing" post-processing)

### 3.1: Advanced Monitoring and Metrics ✅
- [x] Comprehensive metrics collection system (`monitoring.py`)
- [x] Worker metrics: processing rate, failure rate, last heartbeat
- [x] Run metrics: document counts, completion rate, duration 
- [x] Queue health metrics: depth analysis, stale documents, throughput
- [x] Alert system with configurable thresholds
- [x] CLI monitoring interface (`python -m go_doc_go.cli.monitor`)
- [x] Live monitoring dashboard with periodic refresh
- [x] JSON export capabilities for external systems

### 3.2: Dead Letter Queue System ✅ 
- [x] Dead letter queue implementation (`dead_letter.py`)
- [x] Automatic movement of permanently failed documents
- [x] Critical error detection (format errors, permissions, corruption)
- [x] Retry limit enforcement with configurable thresholds
- [x] Failure pattern analysis and reporting
- [x] CLI management interface (`python -m go_doc_go.cli.deadletter`)
- [x] Recovery operations: individual retry, bulk retry, purge
- [x] Integration with document processor for seamless operation

### 3.3: Distributed Cross-Document Relationships ✅
- [x] Fixed post-processing stub in coordinator (`_get_processed_documents()`)
- [x] "Last man standing" coordination for relationship computation
- [x] Cross-document semantic similarity detection
- [x] Proper query to retrieve completed documents from work queue
- [x] Integration with existing embedding and relationship systems

## Implementation Strategy

### Queue-Enabled vs Traditional Processing
```
Traditional (Phase 1):
ingest_documents() -> process each source -> process each document -> parse -> store

Queue-Enabled (Phase 2):
Coordinator: discover documents -> add to queue
Worker 1-N: claim document -> parse -> store -> discover links -> add to queue
```

### Key Design Principles
1. **Backward Compatibility**: Existing usage continues to work
2. **Identical Workers**: All workers use same code, no special roles  
3. **Automatic Coordination**: Workers coordinate via config hash
4. **Pull-Based**: Workers pull work from queue (no push/scheduling)
5. **Link Discovery**: Workers can dynamically add linked documents to queue
6. **Atomic Operations**: No duplicate processing across workers

### Configuration Example
```yaml
# Enable distributed processing
processing:
  mode: "distributed"  # or "single" for traditional
  
# Work queue configuration  
work_queue:
  claim_timeout: 300  # 5 minutes - how long a worker can hold a document
  heartbeat_interval: 30  # 30 seconds - worker heartbeat frequency
  max_retries: 3  # Maximum retry attempts for failed documents
  stale_threshold: 600  # 10 minutes - when to reclaim stale work
  
# Database configuration (PostgreSQL required for queue)
storage:
  backend: "postgresql"
  # ... postgres config
```

### CLI Usage Examples

#### Monitoring Distributed Processing
```bash
# Monitor a specific run with live updates
python -m go_doc_go.cli.monitor --run-id abc123 --live

# Get detailed run and worker metrics 
python -m go_doc_go.cli.monitor --run-id abc123 --details

# Check alerts for a run
python -m go_doc_go.cli.monitor --run-id abc123 --alerts

# Export metrics to JSON for external analysis
python -m go_doc_go.cli.monitor --run-id abc123 --export metrics.json
```

#### Dead Letter Queue Management
```bash
# List failed documents with details
python -m go_doc_go.cli.deadletter --list --details

# Analyze failure patterns
python -m go_doc_go.cli.deadletter --analyze --run-id abc123

# Retry specific failed document
python -m go_doc_go.cli.deadletter --retry 12345

# Retry all failures for a run
python -m go_doc_go.cli.deadletter --retry-run abc123

# Purge old failures (older than 30 days)
python -m go_doc_go.cli.deadletter --purge 30
```

#### Distributed Processing Commands
```bash
# Start coordinator (discovers documents and manages run)
python -m go_doc_go.cli.coordinator --config config.yaml

# Start worker (processes documents from queue) 
python -m go_doc_go.cli.worker --config config.yaml --worker-id worker-01

# Start multiple workers in one process
python -m go_doc_go.cli.worker --config config.yaml --num-workers 4
```