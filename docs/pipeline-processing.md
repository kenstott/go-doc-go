start devi wh# Go-Doc-Go Pipeline Processing Architecture

## Overview

Go-Doc-Go implements a sophisticated distributed document processing pipeline with automatic worker coordination, dual storage architecture, and horizontal scalability. The system separates job coordination (OLTP) from analytical storage (OLAP) to optimize both transactional performance and analytical query capabilities.

## Core Architecture Principles

### Single Job Definition
- **One Configuration**: A single pipeline configuration defines the entire processing job
- **Deterministic Run ID**: Configuration hash generates a unique run ID for automatic worker coordination
- **No Manual Coordination**: Workers automatically discover and join processing runs based on configuration

### Dual Storage Architecture

#### OLTP Storage (Job Coordination)
- **Purpose**: Transactional storage for work queue and processing coordination
- **Characteristics**:
  - High-performance atomic operations for job claiming
  - Transient, mutable state during processing
  - Optimized for concurrent reads/writes
  - Automatic cleanup after job completion
- **Supported Backends**:
  - **PostgreSQL**: Enterprise-grade MVCC with row-level locking
  - **Redis**: Ultra-fast in-memory with Lua scripting for atomicity
  - **SQLite**: Single-machine fallback option

#### OLAP Storage (Analytics)
- **Purpose**: Permanent append-only storage for processed results
- **Characteristics**:
  - Immutable, append-only data model
  - Optimized for analytical queries
  - Supports complex aggregations and relationships
  - Long-term data retention
- **Supported Backends**:
  - **Parquet**: Columnar format with S3/local support, Hive-style partitioning
  - **MongoDB**: Document-oriented with flexible schema
  - **Elasticsearch**: Full-text search with vector embeddings
  - **Apache Solr**: Enterprise search platform
  - **Neo4j**: Graph database for relationship analytics
  - **SQLAlchemy**: Universal SQL adapter (PostgreSQL, MySQL, Oracle, etc.)

## Processing Pipeline

### Two-Pass Processing Model

#### Pass 1: Document Parsing
- **Purpose**: Extract structured content from documents
- **Operations**:
  - Parse document formats (PDF, DOCX, XLSX, JSON, CSV, etc.)
  - Extract text, metadata, and structure
  - Identify relationships between elements
  - Generate element hierarchy
- **Characteristics**:
  - CPU-intensive
  - Can run on any hardware
  - Stateless processing

#### Pass 2: Embedding Generation
- **Purpose**: Generate vector embeddings for semantic search
- **Operations**:
  - Generate embeddings for each element
  - Create contextual embeddings with relationships
  - Build cross-document semantic links
- **Characteristics**:
  - **Automatic GPU acceleration when available**
  - Memory-intensive
  - Can be distributed across specialized hardware
- **GPU Auto-Detection**:
  - Automatically detects CUDA-capable GPUs
  - Falls back to CPU if GPU unavailable
  - Uses ONNX Runtime for optimal performance on both CPU and GPU

## Worker Coordination

### Automatic Hardware Detection
Workers automatically detect and utilize available hardware capabilities:

```python
# Worker automatically detects GPU and uses it for embeddings
config = load_config("pipeline.yaml")
worker = TwoPassWorker(config, worker_id="worker-01")
worker.run()  
# Output: "GPU detected: NVIDIA RTX 4090, using CUDA for embeddings"
# Output: "CPU: 16 cores detected, using 8 threads for parsing"
```

### Automatic Coordination
Workers automatically coordinate through the job storage backend without manual intervention:

```python
# Worker 1 (Automatically uses available hardware)
config = load_config("pipeline.yaml")
worker = TwoPassWorker(config, worker_id="worker-01", mode="auto")  # Default
worker.run()  # Detects GPU, prioritizes embedding tasks

# Worker 2 (CPU-only machine - automatically detected)  
config = load_config("pipeline.yaml")
worker = TwoPassWorker(config, worker_id="worker-02", mode="auto")
worker.run()  # No GPU detected, focuses on parsing tasks

# Worker 3 (Override automatic detection if needed)
config = load_config("pipeline.yaml")
worker = TwoPassWorker(config, worker_id="worker-03", mode="parse")
worker.run()  # Force parsing-only mode even with GPU available
```

### Work Queue Management

#### Document Claiming
- **Atomic Operations**: Documents are claimed atomically to prevent double-processing
- **Lease Mechanism**: Claims expire after timeout for fault tolerance
- **Heartbeat System**: Active workers send heartbeats to maintain claims
- **Automatic Retry**: Failed documents are automatically retried

#### Load Balancing
- **Pull-Based**: Workers pull work when ready (no push overload)
- **Self-Regulating**: System automatically balances based on worker capacity
- **Heterogeneous Support**: Different workers can have different capabilities

### Leader Election (Optional)
For coordinated operations like run completion:
- **Automatic Election**: First worker to claim leadership becomes coordinator
- **Lease-Based**: Leadership expires if not renewed
- **Graceful Handoff**: New leader elected if current leader fails

## GPU Acceleration

### Automatic GPU Detection and Utilization

The system automatically detects and utilizes GPU acceleration without any configuration:

#### Detection Process
1. **CUDA Detection**: Checks for NVIDIA GPUs with CUDA support
2. **ONNX Runtime Selection**: Automatically selects appropriate execution provider
3. **Memory Allocation**: Dynamically adjusts batch sizes based on GPU memory
4. **Fallback Logic**: Seamlessly falls back to CPU if GPU becomes unavailable

#### Zero-Configuration GPU Usage
```python
# No GPU configuration needed - just run the worker
worker = TwoPassWorker(config, worker_id="worker-01")
worker.run()

# System automatically:
# - Detects NVIDIA RTX 4090 with 24GB VRAM
# - Loads ONNX Runtime with CUDAExecutionProvider
# - Sets optimal batch size (64 for 24GB VRAM)
# - Processes embeddings at 10x CPU speed
```

#### Performance Characteristics
- **CPU Only**: ~10-50 embeddings/second
- **GPU (RTX 3060)**: ~200-500 embeddings/second
- **GPU (RTX 4090)**: ~1000-2000 embeddings/second
- **GPU (A100)**: ~2000-5000 embeddings/second

#### Multi-GPU Support
```python
# Automatically uses all available GPUs
worker = TwoPassWorker(config, worker_id="gpu-worker")
# Detects: 4x NVIDIA A100 GPUs
# Distributes embedding generation across all GPUs
# Achieves near-linear scaling
```

#### Dynamic Resource Allocation
The system automatically adjusts based on available resources:
- **High VRAM**: Larger batch sizes for better throughput
- **Low VRAM**: Smaller batches to prevent OOM errors
- **Shared GPUs**: Adjusts memory usage to coexist with other processes

## Scaling Patterns

### Horizontal Scaling
Add more workers at any time to increase throughput:

```bash
# Start initial workers
python -m go_doc_go.worker --config pipeline.yaml --worker-id worker-01

# Add more workers as needed (even while job is running)
python -m go_doc_go.worker --config pipeline.yaml --worker-id worker-02
python -m go_doc_go.worker --config pipeline.yaml --worker-id worker-03
# ... add as many as needed
```

### Specialized Hardware Utilization
Workers automatically detect and use available hardware:

```yaml
# Configuration is the same for all workers - hardware is auto-detected
processing:
  # Mode can be auto (default), parse, embed, or both
  mode: auto  # Automatically choose based on hardware
  
  # GPU settings (automatically detected and used if available)
  embedding:
    # No need to specify device - automatically uses GPU if available
    batch_size: auto  # Automatically sized based on GPU memory
    
  # CPU settings (always available)
  parsing:
    threads: auto  # Automatically set based on CPU cores
    batch_size: 10
```

The system automatically:
- Detects GPUs and uses them for embedding generation
- Detects CPU cores and optimizes thread count
- Adjusts batch sizes based on available memory
- Balances workload based on hardware capabilities

### Multi-Region Deployment
Workers can be distributed across regions:
- **Job Storage**: Centralized or replicated
- **Analytics Storage**: Regional data lakes
- **Cross-region Coordination**: Automatic via job storage

## Configuration Example

```yaml
name: enterprise-pipeline
version: 1.0.0

# Dual Storage Configuration (REQUIRED - no backward compatibility)
storage:
  # OLTP - Job Coordination
  job:
    type: postgresql
    host: job-db.internal
    port: 5432
    database: go_doc_go_jobs
    username: jobs_user
    password: ${JOB_DB_PASSWORD}
    pool_size: 20
    
  # OLAP - Analytics Storage
  analytics:
    type: parquet
    base_path: s3://analytics-bucket/documents/
    partitioning: [year, month, day, run_id]
    s3:
      region: us-west-2
      endpoint_url: null  # Use AWS S3

# Processing Configuration
processing:
  # Two-pass processing (ONLY supported mode)
  mode: two_pass
  
  # Pass 1: Parsing
  parsing:
    batch_size: 10
    timeout: 300
    max_retries: 3
    
  # Pass 2: Embeddings  
  embedding:
    model: BAAI/bge-base-en-v1.5
    batch_size: auto  # Automatically sized based on available GPU/CPU memory
    # No device specification needed - automatically detects and uses GPU if available
    use_onnx: true  # Enables GPU acceleration via ONNX Runtime
    
  # Worker Configuration
  workers:
    heartbeat_interval: 30
    claim_timeout: 300
    max_documents_per_worker: 100

# Content Sources
content_sources:
  - name: s3-documents
    type: s3
    bucket: input-documents
    prefix: /corpus/2024/
    
  - name: sharepoint
    type: sharepoint
    site_url: https://company.sharepoint.com
    libraries: ["Documents", "Reports"]
```

## Monitoring and Operations

### Pipeline Status
Monitor pipeline progress through the API:

```bash
# Get pipeline status
GET /api/pipeline/{pipeline_id}/status

# Response
{
  "run_id": "abc123...",
  "status": "processing",
  "progress": {
    "total_documents": 10000,
    "parsed": 8500,
    "embedded": 6200,
    "completed": 6000,
    "failed": 12
  },
  "workers": {
    "active": 8,
    "worker_stats": [
      {"id": "worker-01", "mode": "parse", "processed": 1200},
      {"id": "worker-02", "mode": "embed", "processed": 800},
      // ...
    ]
  }
}
```

### Performance Metrics
- **Throughput**: Documents/second per worker
- **Latency**: Time from claim to completion
- **Queue Depth**: Pending documents
- **Worker Efficiency**: CPU/GPU utilization

### Fault Tolerance
- **Worker Failure**: Documents automatically reclaimed after timeout
- **Storage Failure**: Automatic retry with exponential backoff
- **Network Partition**: Workers continue processing cached work
- **Data Consistency**: ACID guarantees for job coordination

## Best Practices

### Deployment
1. **Separate Job and Analytics Storage**: Use different infrastructure optimized for each workload
2. **Co-locate Workers with Job Storage**: Minimize network latency for coordination
3. **Use Regional Analytics Storage**: Keep data close to consumers

### Configuration
1. **Set Appropriate Timeouts**: Balance between fault recovery and processing time
2. **Size Worker Pools**: Based on document complexity and hardware
3. **Configure Partitioning**: Optimize for query patterns

### Monitoring
1. **Track Queue Depth**: Indicates if more workers needed
2. **Monitor Claim Timeouts**: May indicate worker issues
3. **Watch Storage Latency**: Can bottleneck entire pipeline
4. **Measure Worker Efficiency**: Optimize hardware allocation

## Migration Guide

### Storage Configuration

All pipelines MUST use dual storage architecture:

```yaml
# OLD (NO LONGER SUPPORTED)
storage:
  backend: postgresql
  uri: postgresql://localhost/godocgo

# NEW (REQUIRED)
storage:
  job:
    type: postgresql
    host: localhost
    database: godocgo_jobs
  analytics:
    type: parquet
    base_path: ./analytics/
```

### From Single Worker to Multi-Worker

```python
# OLD: Single process
processor = DocumentProcessor(config)
processor.process_all()

# NEW: Distributed workers
worker = TwoPassWorker(config, worker_id="worker-01")
worker.run()  # Automatically coordinates with other workers
```

## Performance Characteristics

### Job Storage (OLTP)
- **PostgreSQL**: ~1000 claims/second with 10 workers
- **Redis**: ~5000 claims/second with 10 workers  
- **SQLite**: ~100 claims/second (single worker only)

### Analytics Storage (OLAP)
- **Parquet**: Excellent compression (10:1), fast columnar queries
- **MongoDB**: Flexible schema, good for varied documents
- **Elasticsearch**: Sub-second full-text search on millions of documents
- **Neo4j**: Graph traversals for relationship analysis

### Processing Throughput
- **Parsing**: 10-100 documents/second per CPU worker
- **Embedding**: 100-1000 elements/second per GPU worker
- **End-to-end**: Linear scaling with worker count

## Architecture Benefits

### Separation of Concerns
- **OLTP**: Optimized for transactional operations
- **OLAP**: Optimized for analytical queries
- **Workers**: Stateless and horizontally scalable

### Flexibility
- **Storage Choice**: Pick best tool for each workload
- **Hardware Utilization**: Deploy workers on appropriate hardware
- **Geographic Distribution**: Process globally, analyze locally

### Reliability
- **No Single Point of Failure**: Any worker can fail without data loss
- **Automatic Recovery**: Failed work automatically redistributed
- **Consistent State**: ACID guarantees prevent corruption

### Scalability  
- **Linear Scaling**: Add workers for proportional throughput increase
- **No Coordination Overhead**: Workers self-coordinate through storage
- **Elastic Capacity**: Scale up for batch, scale down when idle

## Conclusion

The Go-Doc-Go pipeline processing architecture provides enterprise-grade document processing with:
- **Automatic Coordination**: No manual orchestration required
- **Dual Storage**: Optimized for both operations and analytics
- **Horizontal Scalability**: Add workers anytime for linear scaling
- **Fault Tolerance**: Automatic recovery from failures
- **Flexibility**: Support for diverse storage backends and hardware

This architecture enables processing millions of documents efficiently while maintaining data consistency, providing real-time monitoring, and supporting complex analytical queries on the results.
