// Package analytics provides storage backends and corpus exploration capabilities.
//
// The analytics package implements the Storage interface for persisting parsed documents,
// elements, relationships, embeddings, and ontology extractions. It supports multiple
// storage backends optimized for different query patterns and scale requirements.
//
// # Storage Backends
//
// **Parquet (Hive-Partitioned)**:
//   - File-based storage in Apache Parquet format
//   - Hive-style partitioning by source_name for efficient filtering
//   - Optimized for analytical queries and billion-scale datasets
//   - Supports temporal versioning (partition by processing date)
//   - DuckDB-compatible for SQL queries
//   - Ideal for: Data science, machine learning, batch analytics
//
// **Neo4j**:
//   - Graph database storage
//   - Native support for relationship traversal
//   - Cypher query language
//   - Ideal for: Knowledge graphs, relationship exploration, semantic networks
//
// # Usage
//
// Create a storage backend:
//
//	// Parquet storage
//	parquetStorage, err := analytics.NewParquetStorage(analytics.ParquetConfig{
//		BasePath:              "./output",
//		HivePartitionBySource: true,
//		TemporalPartitioning:  true,
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer parquetStorage.Close()
//
//	// Store parsed documents
//	err = parquetStorage.AppendDocuments([]analytics.Document{...})
//	err = parquetStorage.AppendElements([]analytics.Element{...})
//	err = parquetStorage.AppendRelationships([]analytics.Relationship{...})
//
// # Data Model (UDML Layers)
//
// The Storage interface persists five UDML (Universal Document Markup Language) layers:
//
// **UDML-D (Documents)**:
//   - Document metadata (ID, title, URL, content type)
//   - Processing timestamps
//   - Element and relationship counts
//
// **UDML-E (Elements)**:
//   - Universal document elements (headings, paragraphs, tables, etc.)
//   - Query-optimized promoted fields (PageNumber, SectionLevel, RowIndex, etc.)
//   - Code-specific fields (FunctionName, ClassName, Namespace, LineNumber)
//   - Hierarchical structure via ParentID
//
// **UDML-R (Relationships)**:
//   - Structural relationships (parent-child, section-content)
//   - Semantic relationships (similar-to, references, depends-on)
//   - Cross-document relationships
//
// **UDML-V (Vector Embeddings)**:
//   - Element embeddings for semantic search
//   - Contextual text used for embedding generation
//   - Supports similarity search and clustering
//
// **UDML-L (Links)**:
//   - Hyperlinks and cross-references
//   - Code dependencies and imports
//   - Used for discovery/crawling
//
// **UDML-O (Ontology Instances)**:
//   - Extracted entities (people, organizations, locations, etc.)
//   - Entity relationships (works-at, located-in, mentions)
//   - Entity mentions (where entities appear in documents)
//   - Canonical entities (deduplicated from raw extractions)
//   - LLM validation results (false positive detection)
//
// # Query Operations
//
// The Storage interface provides two categories of operations:
//
// **Standard Queries** (all backends):
//
//	// Query elements with filters
//	elements, err := storage.QueryElements(map[string]interface{}{
//		"source_name":      "wikipedia",
//		"element_type":     "heading",
//		"page_number":      5,
//		"latest_only":      true,  // Temporal filtering
//	})
//
//	// Query ontology entities
//	entities, err := storage.QueryOntologyEntities(map[string]interface{}{
//		"entity_type": "person",
//		"confidence":  0.8,  // Minimum confidence threshold
//	})
//
// **Corpus Exploration** (interactive analysis):
//
//	// Semantic similarity search
//	results, err := storage.SearchSemanticSimilarity(
//		queryVector,
//		filters,
//		threshold,  // 0.7 = 70% similarity
//		limit,      // Top K results
//	)
//
//	// Regex pattern matching
//	results, err := storage.SearchByRegex(
//		`\b[A-Z][a-z]+ [A-Z][a-z]+\b`,  // Person name pattern
//		filters,
//		1000,
//	)
//
//	// Term frequency analysis
//	frequencies, err := storage.ComputeTermFrequencies(
//		[]string{"machine learning", "AI", "neural network"},
//		true,  // case-sensitive
//		filters,
//	)
//
//	// Co-occurrence analysis
//	cooccur, err := storage.FindCooccurrences(
//		"machine learning",
//		"neural network",
//		"element",  // context window: "element", "paragraph", "document"
//		filters,
//		100,  // max examples
//	)
//
// # Temporal Filtering
//
// All query methods support temporal filtering for versioned corpora:
//
//	filters := map[string]interface{}{
//		"source_name": "wikipedia",
//		"latest_only": true,  // Only return latest version of each doc_id
//	}
//
//	// Or query corpus state at specific date
//	filters := map[string]interface{}{
//		"as_of_date": "2024-01-15",  // Corpus state on Jan 15, 2024
//	}
//
// This enables time-travel queries and corpus evolution analysis.
//
// # Ontology Extraction (SQL-Based)
//
// For billion-scale datasets, the package provides SQL-based entity extraction
// that runs entirely within the database layer (no memory loading):
//
//	// Get all document IDs (for distributed batching)
//	docIDs, err := storage.GetAllDocIDs(filters)
//
//	// Extract entities for a batch of documents
//	entityCount, err := storage.ExtractAndStoreEntities(
//		runID,              // Extraction run ID (versioning)
//		"person",           // Entity type
//		docIDs[0:1000],     // Document batch
//		mappingJSON,        // Extraction rules (from ontology schema)
//		filters,
//		conceptEmbeddings,  // Pre-computed semantic vectors
//	)
//
//	// Consolidate raw extractions into canonical entities
//	err = storage.ConsolidateEntities(
//		runID,
//		"max_confidence",  // Resolution strategy
//		llmClient,         // For false positive validation
//		schema,            // For cross-domain merging
//	)
//
// This streaming approach processes billions of elements with O(1) memory.
//
// # Parquet Schema Structure
//
// Parquet storage uses Hive-style partitioning for efficient filtering:
//
//	base_path/
//	├── documents/
//	│   └── source_name=wikipedia/
//	│       └── processing_date=2024-01-15/
//	│           └── data.parquet
//	├── elements/
//	│   └── source_name=wikipedia/
//	│       └── processing_date=2024-01-15/
//	│           └── data.parquet
//	├── relationships/
//	│   └── source_name=wikipedia/
//	│       └── data.parquet
//	├── embeddings/
//	│   └── source_name=wikipedia/
//	│       └── data.parquet
//	└── ontology_entities/
//	    ├── run_id=20240115_120000/
//	    │   ├── entity_type=person/
//	    │   │   └── data.parquet
//	    │   └── entity_type=organization/
//	    │       └── data.parquet
//	    └── canonical_entities/
//	        ├── run_id=20240115_120000/
//	        │   └── data.parquet
//	        └── entity_mentions/
//	            └── data.parquet
//
// # Neo4j Graph Model
//
// Neo4j storage creates the following node and relationship types:
//
// **Nodes**:
//   - Document: Represents processed documents
//   - Element: Universal document elements
//   - OntologyEntity: Extracted entities
//
// **Relationships**:
//   - CONTAINS: Document → Element (containment)
//   - PARENT_OF: Element → Element (hierarchy)
//   - RELATES_TO: Element → Element (semantic relationships)
//   - ENTITY_MENTION: OntologyEntity → Element (where entity appears)
//   - ENTITY_RELATIONSHIP: OntologyEntity → OntologyEntity (entity relationships)
//
// # Performance Characteristics
//
// **Parquet (Hive-Partitioned)**:
//   - Write throughput: 100K+ elements/second
//   - Query latency: 100ms - 5s (depends on filter selectivity)
//   - Storage efficiency: 5-10x compression ratio
//   - Scalability: Tested up to 1B elements
//   - Memory usage: O(batch_size) for streaming operations
//
// **Neo4j**:
//   - Write throughput: 10K+ nodes/second (batched)
//   - Query latency: <100ms for graph traversals
//   - Storage efficiency: Higher overhead than Parquet
//   - Scalability: Tested up to 100M nodes
//   - Memory usage: Depends on query complexity
//
// # Corpus Statistics
//
// Compute aggregate statistics about the corpus:
//
//	stats, err := storage.AggregateStatistics(
//		[]string{
//			"element_type_distribution",
//			"document_count",
//			"total_elements",
//			"avg_content_length",
//		},
//		filters,
//	)
//
//	fmt.Printf("Documents: %d\n", stats.DocumentCount)
//	fmt.Printf("Elements: %d\n", stats.TotalElements)
//	for elemType, count := range stats.ElementTypeDistribution {
//		fmt.Printf("  %s: %d\n", elemType, count)
//	}
//
// # Factory Pattern
//
// Create storage backends using the factory:
//
//	import "github.com/kennethstott/doculyzer-go-conversion/internal/analytics"
//
//	// From configuration
//	config := map[string]interface{}{
//		"type":      "parquet",
//		"base_path": "./output",
//		"hive_partition_by_source": true,
//	}
//	storage, err := analytics.NewStorage(config)
//
// # Error Handling
//
// Storage operations return errors for:
//
//   - Connection failures (Neo4j)
//   - File I/O errors (Parquet)
//   - Invalid data (schema violations)
//   - Query syntax errors
//
// Errors are wrapped with context using fmt.Errorf with %w.
//
// # Concurrency
//
// Storage backends are designed for concurrent writes:
//
//   - Parquet: Thread-safe, uses file-level locking
//   - Neo4j: Connection pooling, batched writes
//
// Multiple goroutines can write to the same storage backend concurrently.
//
// # See Also
//
//   - UDML Specification: docs/features/udml/
//   - Ontology Extraction: docs/features/ontology/
//   - Configuration: docs/configuration/storage.md
//   - MCP Server: docs/features/mcp/ (corpus exploration via MCP protocol)
//   - Examples: examples/distributed-workers/, examples/semantic-search/
package analytics
