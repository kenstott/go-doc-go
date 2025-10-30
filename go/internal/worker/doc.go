// Package worker provides the document processing worker that orchestrates parsing, analysis, and storage.
//
// The worker package implements the core document processing pipeline that coordinates
// content ingestion, parsing, relationship detection, embedding generation, ontology
// extraction, and storage. Workers can run in distributed mode with multiple instances
// coordinating via a shared job queue.
//
// # Architecture
//
// The Worker is the main orchestrator that:
//
//   1. Claims documents from the job queue (distributed coordination)
//   2. Fetches content from content sources
//   3. Parses documents into Universal Elements
//   4. Generates embeddings for semantic search
//   5. Detects relationships between elements
//   6. Stores results to analytics backends (Parquet, Neo4j)
//   7. Queues discovered documents for crawling
//   8. Triggers batch operations when queue is idle (cross-document analysis, Neo4j export)
//
// # Usage
//
// Create and run a worker:
//
//	worker := worker.NewWorker(worker.WorkerConfig{
//		WorkerID:          "worker-1",
//		NumWorkers:        4,
//		JobControl:        jobControlBackend,
//		ContentSources:    contentSourceMap,
//		AnalyticsStorages: []analytics.Storage{parquetStorage},
//		EmbeddingGenerator: embeddingGen,
//		MaxDocuments:      1000,  // Process limit (0 = unlimited)
//	})
//
//	ctx := context.Background()
//	err := worker.Run(ctx)
//	if err != nil {
//		log.Fatal(err)
//	}
//
// # Distributed Processing
//
// Workers coordinate via the job queue for distributed processing:
//
//   - Multiple workers can run concurrently on different machines
//   - Document claiming uses database row-level locking for atomicity
//   - Each worker sends heartbeats to prevent document reclamation
//   - Failed documents are automatically retried by other workers
//   - One worker is elected "leader" for batch operations
//
// **Worker Coordination**:
//
//	Machine 1:  Worker-A (leader)  ──┐
//	Machine 2:  Worker-B            ├──> Shared Job Queue (PostgreSQL/SQLite)
//	Machine 3:  Worker-C            ──┘
//
// All workers are identical - the first worker to claim a document processes it.
//
// # Processing Pipeline
//
// For each document, the worker executes this pipeline:
//
//	1. Claim Document
//	   ↓
//	2. Fetch Content (from ContentSource)
//	   ↓
//	3. Detect Document Type (PDF, DOCX, HTML, etc.)
//	   ↓
//	4. Parse Document (via Parser interface)
//	   ↓
//	5. Generate Embeddings (batch processing)
//	   ↓
//	6. Detect Relationships (structural + semantic)
//	   ↓
//	7. Store Results (Parquet/Neo4j)
//	   ↓
//	8. Queue Discovered Documents (hyperlinks, code imports)
//	   ↓
//	9. Mark Complete or Failed
//
// # Embedding Generation
//
// Workers use cross-document batching for efficient embedding generation:
//
//	// Accumulate elements across multiple documents
//	worker.embeddingAggregator.Add(docID, elementID, text)
//
//	// Flush batch when size threshold reached
//	if batchSize >= maxBatchSize {
//		embeddings := worker.embeddingAggregator.Flush()
//		// Store embeddings to analytics
//	}
//
// This amortizes model loading and inference overhead across documents.
//
// **Embedding Strategy**:
//   - Contextual text: Includes parent and sibling context
//   - Batch size: 100-500 elements per batch (configurable)
//   - Model: ONNX-based (thread-safe, no external API)
//   - Dimensions: 384 (default, configurable)
//
// # Relationship Detection
//
// The worker detects three types of relationships:
//
// **Structural Relationships** (always enabled):
//   - Parent-child (contains)
//   - Section-content (part_of)
//   - Table-row-cell hierarchy
//
// **Semantic Relationships** (optional):
//   - Similar-to (via embedding similarity)
//   - References (via content analysis)
//   - Depends-on (code imports)
//
// **Cross-Document Relationships** (batch operation):
//   - Similar documents (embedding similarity)
//   - Cross-document references (hyperlinks)
//   - Code dependencies (import graphs)
//
// # Discovery and Crawling
//
// Workers queue discovered documents for processing:
//
// **Hyperlink Discovery**:
//   - Extract hyperlinks from HTML/Markdown elements
//   - Resolve relative URLs to absolute
//   - Apply include/exclude patterns
//   - Respect max crawl depth
//   - Queue for processing
//
// **Code Dependency Discovery**:
//   - Extract import statements from code files
//   - Resolve import paths to source files (language-specific)
//   - Filter by package type (stdlib, local, external)
//   - Respect max crawl depth
//   - Queue for processing
//
// See docs/features/discovery/ for details.
//
// # Leader Worker Responsibilities
//
// One worker is elected "leader" (first to start) for batch operations:
//
// **Neo4j Export** (when queue idle):
//   - Wait for queue to be empty for N seconds
//   - Export Parquet data to Neo4j graph database
//   - Create nodes and relationships
//   - Update last export timestamp
//
// **Cross-Document Semantic Analysis** (when queue idle):
//   - Detect similar documents via embedding similarity
//   - Create cross-document relationships
//   - Run periodically (minimum interval: N minutes)
//
// **Ontology Extraction** (when queue idle):
//   - Extract entities from documents using ontology schemas
//   - Detect entity relationships
//   - Store results to UDML-O layer
//   - Run periodically (minimum interval: N minutes)
//
// # Configuration
//
// Worker configuration from TOML:
//
//	[processing]
//	workers = 4  # Number of concurrent document processors
//
//	[processing.job_control]
//	backend = "postgres"  # or "sqlite"
//	path = "postgresql://..."
//	claim_timeout = 300  # 5 minutes
//	heartbeat_interval = 30  # 30 seconds
//	max_retries = 3
//
//	[processing.neo4j_export]
//	enabled = true
//	empty_queue_wait_time = 60  # Wait 60s after queue empty
//	source_analytics = "parquet"
//	batch_size = 1000
//
//	[relationship_detection]
//	enabled = true
//	structural = true
//	semantic = true
//
//	[relationship_detection.cross_document_semantic]
//	enabled = true
//	similarity_threshold = 0.7
//	queue_idle_trigger_minutes = 5
//
// # Error Handling
//
// The worker handles errors gracefully:
//
//   - **Transient errors**: Retry with exponential backoff (network timeouts, rate limits)
//   - **Permanent errors**: Mark document as failed (invalid format, unsupported type)
//   - **Max retries**: After N retries, mark as permanently failed
//   - **Crash recovery**: Documents with expired heartbeats are reclaimed by other workers
//
// Failed documents are logged with full error context for debugging.
//
// # Graceful Shutdown
//
// Workers handle shutdown signals gracefully:
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	// Listen for signals
//	sigChan := make(chan os.Signal, 1)
//	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
//
//	go func() {
//		<-sigChan
//		log.Println("Shutting down gracefully...")
//		cancel()  // Triggers worker shutdown
//	}()
//
//	worker.Run(ctx)
//
// On shutdown, the worker:
//   1. Stops claiming new documents
//   2. Finishes processing current documents
//   3. Flushes pending embeddings
//   4. Releases database locks
//   5. Closes storage connections
//   6. Returns
//
// # Performance Characteristics
//
// **Throughput**:
//   - Single worker: 10-100 documents/second (depends on document size)
//   - 10 workers: 100-1000 documents/second
//   - Bottleneck: Usually parsing or embedding generation
//
// **Memory Usage**:
//   - Base: ~100MB per worker
//   - Active processing: ~500MB per worker (varies by document size)
//   - Embedding model: ~500MB (shared across workers via ONNX)
//
// **Latency**:
//   - Small document (< 1MB): 100ms - 1s
//   - Large document (10MB): 1s - 10s
//   - Very large document (100MB): 10s - 60s
//
// # Monitoring
//
// Workers log processing statistics:
//
//	[INFO] Worker worker-1 started
//	[INFO] Claimed document: doc_id=abc123 source=wikipedia
//	[DEBUG] Parsed document: elements=150 duration=1.2s
//	[DEBUG] Generated embeddings: count=150 batch_size=150 duration=500ms
//	[INFO] Stored document: doc_id=abc123 elements=150 relationships=45
//	[INFO] Processed 100 documents, rate=50 docs/sec, success=98 failed=2
//
// Key metrics to monitor:
//   - Documents processed/second
//   - Success vs. failure rate
//   - Average processing time per document
//   - Embedding generation throughput
//   - Queue depth (backlog)
//
// # Concurrent Workers
//
// Run multiple workers in the same process:
//
//	for i := 0; i < 4; i++ {
//		workerID := fmt.Sprintf("worker-%d", i)
//		worker := worker.NewWorker(worker.WorkerConfig{
//			WorkerID: workerID,
//			NumWorkers: 4,
//			// ... other config
//		})
//
//		go func() {
//			err := worker.Run(ctx)
//			if err != nil {
//				log.Printf("Worker %s error: %v", workerID, err)
//			}
//		}()
//	}
//
// Or run workers on separate machines (recommended for scale):
//
//	Machine 1: goworker --config config.toml --workers 10
//	Machine 2: goworker --config config.toml --workers 10
//	Machine 3: goworker --config config.toml --workers 10
//
// # Semantic Analyzer
//
// The SemanticAnalyzer performs cross-document relationship detection:
//
//   - Loads embeddings from analytics storage
//   - Computes similarity matrix
//   - Identifies similar document pairs (above threshold)
//   - Creates cross-document relationships
//   - Stores results back to analytics
//
// This runs as a batch operation when the queue is idle.
//
// # Ontology Extractor
//
// The OntologyExtractor performs entity and relationship extraction:
//
//   - Loads ontology schema (entity types, extraction rules)
//   - Executes SQL-based extraction (no memory loading)
//   - Processes documents in batches
//   - Consolidates raw extractions into canonical entities
//   - LLM-based false positive validation (optional)
//   - Stores results to UDML-O layer
//
// This runs as a batch operation when the queue is idle.
//
// # Worker State
//
// Workers maintain state for batch operations:
//
//   - lastQueueEmptyTime: When queue last became empty
//   - lastNeo4jExportTime: When Neo4j export last ran
//   - lastSemanticAnalysisTime: When semantic analysis last ran
//   - lastOntologyExtractionTime: When ontology extraction last ran
//   - documentsProcessed: Total documents processed (atomic counter)
//   - isLeader: Whether this worker is the leader
//
// # See Also
//
//   - Job Control: go/internal/jobcontrol/ (document queue management)
//   - Content Sources: go/internal/contentsource/ (document ingestion)
//   - Parsers: go/internal/parser/ (document parsing)
//   - Analytics Storage: go/internal/analytics/ (result persistence)
//   - Embeddings: go/internal/embeddings/ (vector generation)
//   - Discovery: docs/features/discovery/ (crawling and link following)
//   - Configuration: docs/configuration/README.md
//   - Examples: examples/distributed-workers/
package worker
