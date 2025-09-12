# Storage Backends Guide

Go-Doc-Go supports multiple storage backends, each optimized for different use cases. Choose the backend that best fits your requirements for scale, performance, and features.

## Backend Comparison

| Backend | Best For | Pros | Cons | Vector Search | Full-Text Search | Graph Relations |
|---------|----------|------|------|---------------|------------------|-----------------|
| **SQLite** | Development, small datasets | Simple, no setup, portable | Single-threaded, limited scale | With extensions | ✅ | Via SQL joins |
| **PostgreSQL** | Production, ACID compliance | Mature, reliable, SQL standard | Setup required | With pgvector | ✅ | Via SQL joins |
| **Elasticsearch** | Search-heavy workloads | Excellent search, analytics | Complex setup, memory-heavy | ✅ | ✅ | Limited |
| **MongoDB** | Flexible schema needs | Document-native, scalable | NoSQL learning curve | ✅ | ✅ | Via references |
| **Neo4j** | Graph analysis & visualization | Native graph operations, Cypher | Specialized queries | ✅ | ✅ | ✅ Native |

## Neo4j: Special Status as Graph Adjunct

⚠️ **Important**: Neo4j has a **unique dual role** in Go-Doc-Go - it can serve as both a primary storage backend AND as a specialized graph store alongside any other backend.

### The Graph Adjunct Pattern (Recommended)

**Why it's special**: Unlike other backends that are mutually exclusive, Neo4j can be added to ANY primary storage configuration to provide native graph capabilities:

```yaml
# Primary storage: reliable document and element storage
storage:
  backend: "postgresql"  # or sqlite, elasticsearch, mongodb
  host: "postgres-server"
  database: "go_doc_go_primary"

# Graph adjunct: specialized relationship analysis  
neo4j:
  enabled: true
  uri: "bolt://neo4j-server:7687"
  username: "neo4j"
  password: "${NEO4J_PASSWORD}"
  
  # Sync configuration
  export_on_ingestion: true      # Auto-export new data
  export_batch_size: 1000
  
  # What to include in graph
  include_documents: true
  include_elements: true
  include_relationships: true
  include_entity_relationships: true  # Domain entities
```

**Result**: You get the **best of both worlds**:
- **Primary backend** handles document storage, search, and reliability
- **Neo4j** provides powerful graph queries and visualization
- **Automatic synchronization** keeps both in sync

## SQLite (Default)

Perfect for development, testing, and smaller datasets.

```yaml
storage:
  backend: "sqlite"
  path: "./data/documents.db"
  
  # Optional: Enable extensions
  enable_fts: true          # Full-text search
  enable_vector: false      # Vector search (requires sqlean)
  
  # Storage optimization
  page_size: 4096
  cache_size: 10000
  journal_mode: "WAL"
```

**Pros:**
- Zero setup - just specify a file path
- Perfect for development and testing
- Excellent for datasets under 1TB
- ACID compliant transactions

**Cons:**
- Single writer (though multiple readers)
- Limited concurrent access
- No built-in replication

**Use Cases:**
- Development and testing
- Single-user applications
- Document archives up to millions of documents
- Embedded applications

## PostgreSQL (Recommended for Production)

Robust, scalable, production-ready with excellent ecosystem support.

```yaml
storage:
  backend: "postgresql"
  host: "localhost"
  port: 5432
  database: "go_doc_go"
  username: "postgres"
  password: "${DB_PASSWORD}"
  
  # Connection pooling
  pool_size: 20
  max_overflow: 30
  pool_timeout: 30
  
  # Performance tuning
  statement_timeout: 300000  # 5 minutes
  batch_size: 1000
```

**With Vector Search (pgvector):**

```yaml
storage:
  backend: "postgresql"
  # ... connection details ...
  
  # Vector search configuration
  enable_vector_search: true
  vector_dimension: 384
  vector_index_type: "ivfflat"  # or "hnsw"
  
embedding:
  enabled: true
  provider: "fastembed"
  model: "BAAI/bge-small-en-v1.5"
  dimensions: 384
```

**Pros:**
- Battle-tested reliability
- Excellent concurrency support
- Rich ecosystem and tooling
- MVCC for time-travel queries
- Mature backup and replication

**Cons:**
- Requires database server setup
- More complex than SQLite
- Resource overhead for small datasets

**Use Cases:**
- Production applications
- Multi-user environments
- Large datasets (10TB+)
- Applications requiring high availability

## Elasticsearch

Optimized for search-heavy workloads with advanced analytics.

```yaml
storage:
  backend: "elasticsearch"
  hosts:
    - "http://localhost:9200"
  
  # Authentication
  username: "elastic"
  password: "${ELASTIC_PASSWORD}"
  
  # Index configuration
  index_prefix: "go-doc-go"
  number_of_shards: 3
  number_of_replicas: 1
  
  # Search optimization
  enable_vector_search: true
  vector_dimension: 384
  similarity_metric: "cosine"
  
  # Full-text configuration
  store_full_text: true
  index_full_text: true
  analyzer: "standard"
```

**Advanced Search Configuration:**

```yaml
storage:
  backend: "elasticsearch"
  # ... connection details ...
  
  # Custom mappings for better search
  custom_mappings:
    elements:
      properties:
        content_preview:
          type: "text"
          analyzer: "english"
          fields:
            keyword:
              type: "keyword"
        embedding:
          type: "dense_vector"
          dims: 384
          index: true
          similarity: "cosine"
```

**Pros:**
- Excellent full-text search capabilities
- Built-in analytics and aggregations
- Horizontal scaling
- Rich query DSL
- Real-time search

**Cons:**
- High memory usage
- Complex cluster management
- Can be overkill for simple use cases

**Use Cases:**
- Search-centric applications
- Real-time analytics
- Large-scale text processing
- Multi-tenant applications

## MongoDB

Document-native storage with flexible schema support.

```yaml
storage:
  backend: "mongodb"
  host: "localhost"
  port: 27017
  database: "go_doc_go"
  username: "${MONGO_USER}"
  password: "${MONGO_PASSWORD}"
  
  # Connection options
  max_pool_size: 100
  min_pool_size: 10
  max_idle_time: 30000
  
  # Collections
  documents_collection: "documents"
  elements_collection: "elements"
  relationships_collection: "relationships"
  
  # Indexing
  create_indexes: true
  text_index_language: "english"
```

**With Vector Search (MongoDB Atlas):**

```yaml
storage:
  backend: "mongodb"
  # ... connection details ...
  
  # Atlas Vector Search
  enable_vector_search: true
  vector_dimension: 384
  vector_index_name: "vector_index"
  similarity_metric: "cosine"
```

**Pros:**
- Schema flexibility
- Document-native operations
- Good horizontal scaling
- Rich aggregation framework

**Cons:**
- NoSQL learning curve
- Less mature ecosystem than SQL
- Memory usage can be high

**Use Cases:**
- Applications with evolving schemas
- Document-centric workflows
- Rapid prototyping
- Mixed structured/unstructured data

## Neo4j Deployment Patterns

Neo4j's unique dual-role architecture offers unprecedented flexibility in Go-Doc-Go deployments.

### Pattern 1: Neo4j as Primary Storage

For graph-native applications where relationships are the primary concern:

```yaml
storage:
  backend: "neo4j"
  uri: "bolt://localhost:7687"
  username: "neo4j"
  password: "${NEO4J_PASSWORD}"
  
  # Performance settings
  max_connection_lifetime: 1800
  max_connection_pool_size: 100
  connection_acquisition_timeout: 60
  
  # Graph-specific options
  create_constraints: true
  create_indexes: true
```

**When to use**: Pure knowledge graph applications, research platforms, relationship-heavy analytics.

### Pattern 2: Neo4j as Graph Adjunct (Recommended)

The **game-changer**: Add powerful graph capabilities to ANY existing storage backend without migration:

```yaml
# Your existing reliable primary storage (unchanged)
storage:
  backend: "postgresql"  # or sqlite, mongodb, elasticsearch
  # ... your existing configuration ...

# Add Neo4j graph capabilities (new addition)
neo4j:
  enabled: true
  uri: "bolt://neo4j-cluster:7687" 
  username: "neo4j"
  password: "${NEO4J_PASSWORD}"
  
  # Automatic synchronization
  export_on_ingestion: true      # Real-time sync
  export_batch_size: 1000
  export_schedule: "hourly"      # Optional: batch sync
  
  # Selective graph construction  
  include_documents: true
  include_elements: true
  include_relationships: true
  include_entity_relationships: true    # Domain ontology data
  include_cross_document_links: true    # Multi-document connections
  
  # Graph optimization
  merge_duplicate_entities: true        # Deduplicate entities across docs
  create_document_clusters: true        # Group related documents
  compute_centrality_metrics: true      # Graph analytics
```

**Why this is revolutionary**:
- **Zero migration risk** - Your primary storage remains unchanged
- **Incremental adoption** - Start with simple queries, grow into complex graph analytics
- **Best tool for each job** - PostgreSQL for reliability, Neo4j for graph insights
- **Automatic synchronization** - No manual data management

### Example: Graph Queries You Can't Do Elsewhere

With Neo4j as a graph adjunct, you unlock powerful relationship queries:

```cypher
// Find all documents that reference both "brake system" and "safety standard"
MATCH (d:Document)-[:CONTAINS]->(e1:Element)-[:MENTIONS]->(entity1:Entity {type: "brake_system"})
MATCH (d)-[:CONTAINS]->(e2:Element)-[:MENTIONS]->(entity2:Entity {type: "safety_standard"})
RETURN d.title, entity1.name, entity2.name

// Discover knowledge paths between concepts
MATCH path = (start:Entity {name: "ABS"})-[*1..4]-(end:Entity {name: "crash test"})
RETURN path, length(path) as degrees_of_separation

// Find most influential documents (highest centrality)
CALL gds.pageRank.stream('knowledge-graph')
YIELD nodeId, score
MATCH (d:Document) WHERE id(d) = nodeId
RETURN d.title, score ORDER BY score DESC LIMIT 10

// Cluster similar documents by shared entities
CALL gds.louvain.stream('document-graph')
YIELD nodeId, communityId  
MATCH (d:Document) WHERE id(d) = nodeId
RETURN communityId, collect(d.title) as related_docs
```

**The Power**: These queries run on your **complete knowledge graph** while your primary storage handles day-to-day document operations efficiently.

### Why This Works: Structured Foundation vs. Text Blobs

**The Key Differentiator**: Go-Doc-Go starts with a **structured document graph** as the foundation for knowledge extraction, not raw text blobs. This makes knowledge graph construction **1000x more effective**.

**Traditional Approach (Text Blobs)**:
```
PDF → Extract text → Split into chunks → Try to find relationships in flat text
```
Result: **Lost context, unclear relationships, poor entity extraction**

**Go-Doc-Go Approach (Structured Foundation)**:
```
PDF → Parse into structured elements → Preserve hierarchy → Extract entities with context → Build knowledge graph
```
Result: **Rich context, clear relationships, precise entity extraction**

**Example: Why Structure Matters**

From a safety manual PDF:

**Text Blob Approach**:
```
"ABS system must comply with FMVSS-135 brake fluid temperature sensors..."
```
→ Entities: [ABS, FMVSS-135, brake, fluid, temperature, sensors]
→ Relationships: ??? (unclear from flat text)

**Structured Document Approach**:
```
Document: "Vehicle Safety Manual v2.1"
  Section: "Brake System Requirements" 
    Subsection: "ABS Compliance"
      Paragraph: "ABS system must comply with FMVSS-135"
      Table Row: "brake fluid temperature sensors | -40°C to +150°C | mandatory"
```

→ **Rich Knowledge Graph**:
- Entity: "ABS system" (type: brake_component, context: safety_requirements)
- Entity: "FMVSS-135" (type: safety_standard, applies_to: brake_systems)  
- Relationship: ABS_system → COMPLIES_WITH → FMVSS-135
- Context: From "Vehicle Safety Manual v2.1", "Brake System Requirements" section
- Metadata: Document structure, table data, hierarchical position

**The Neo4j Advantage**: When you export this structured data to Neo4j, you get:
- **Precise entity relationships** based on document structure
- **Contextual queries** that understand document hierarchy  
- **Cross-document connections** between related structured content
- **Graph analytics** on meaningful, structured relationships

**Pros:**
- Native graph operations
- Excellent for relationship queries
- Cypher query language
- Graph algorithms built-in

**Cons:**
- Specialized use case
- Different query paradigm
- Can be complex for simple document storage

**Use Cases:**
- Knowledge graph analysis
- Recommendation systems
- Fraud detection
- Social network analysis

## Hybrid Architectures

**Neo4j's Unique Status**: Unlike other backends, Neo4j can be combined with **any** primary storage backend as a graph adjunct. This enables powerful hybrid architectures impossible with other databases.

### PostgreSQL + Neo4j (Most Popular)

The ultimate combination: PostgreSQL's reliability + Neo4j's graph power.

```yaml
# Primary storage
storage:
  backend: "postgresql"
  host: "postgres-server"
  database: "go_doc_go_primary"
  
# Search index
elasticsearch:
  enabled: true
  hosts: ["http://elasticsearch-cluster:9200"]
  sync_on_write: true
  index_prefix: "search-"
```

### PostgreSQL + Neo4j

Reliable storage with graph analysis capabilities.

```yaml
# Primary storage  
storage:
  backend: "postgresql"
  host: "postgres-server"
  database: "go_doc_go_primary"
  
# Graph export
neo4j:
  enabled: true
  uri: "bolt://neo4j-server:7687"
  export_schedule: "daily"  # or "on_ingestion"
```

## Performance Tuning

### General Optimization

```yaml
storage:
  # Batch processing
  batch_size: 1000
  max_workers: 8
  
  # Memory management
  max_memory_usage: "4GB"
  cache_size: "1GB"
  
  # Connection management
  pool_size: 20
  max_connections: 100
  connection_timeout: 30
```

### Backend-Specific Tuning

**PostgreSQL:**
```yaml
storage:
  backend: "postgresql"
  # ... connection details ...
  
  # PostgreSQL-specific tuning
  work_mem: "256MB"
  shared_buffers: "2GB"  
  effective_cache_size: "8GB"
  random_page_cost: 1.1  # For SSDs
  
  # Bulk loading optimization
  maintenance_work_mem: "1GB"
  checkpoint_completion_target: 0.9
```

**Elasticsearch:**
```yaml
storage:
  backend: "elasticsearch"
  # ... connection details ...
  
  # ES-specific tuning
  refresh_interval: "30s"     # Less frequent refreshes
  number_of_replicas: 0       # For bulk loading
  index_buffer_size: "512mb"  # More memory for indexing
  
  # Bulk API settings
  bulk_size: 1000
  bulk_timeout: "60s"
  max_retries: 3
```

## Storage Patterns

### Development Pattern

```yaml
# Simple, fast setup
storage:
  backend: "sqlite"
  path: "./dev_data.db"
  enable_fts: true

embedding:
  provider: "fastembed"  # Fast local embeddings
  model: "BAAI/bge-small-en-v1.5"
```

### Production Pattern

```yaml
# Scalable, reliable
storage:
  backend: "postgresql"
  host: "${DB_HOST}"
  database: "go_doc_go_prod"
  pool_size: 50
  
embedding:
  provider: "fastembed"
  model: "BAAI/bge-base-en-v1.5"  # Better quality
  
# Optional: Search optimization
elasticsearch:
  enabled: true
  hosts: ["${SEARCH_CLUSTER}"]
  sync_on_write: false  # Async for performance
  sync_schedule: "*/5 * * * *"  # Every 5 minutes
```

### Analytics Pattern

```yaml
# Optimized for analysis and reporting
storage:
  backend: "postgresql"  # Primary storage
  # ... connection details ...

# Search and analytics
elasticsearch:
  enabled: true
  # ... ES configuration ...
  
# Graph analysis
neo4j:
  enabled: true
  export_schedule: "daily"
  include_entity_relationships: true
```

## Migration Between Backends

Go-Doc-Go supports migrating data between storage backends:

```bash
# Export from current backend
go-doc-go export --config current_config.yaml --output backup.jsonl

# Import to new backend  
go-doc-go import --config new_config.yaml --input backup.jsonl
```

Or programmatically:

```python
from go_doc_go import Config
from go_doc_go.storage import migrate_storage

# Migrate from SQLite to PostgreSQL
source_config = Config("sqlite_config.yaml")
target_config = Config("postgres_config.yaml")

migrate_storage(
    source_config=source_config,
    target_config=target_config,
    batch_size=1000,
    include_embeddings=True
)
```

## Monitoring and Maintenance

### Health Checks

```python
from go_doc_go import Config

config = Config("config.yaml")
db = config.get_storage_backend()

# Check backend health
health = db.health_check()
print(f"Status: {health['status']}")
print(f"Documents: {health['document_count']}")
print(f"Elements: {health['element_count']}")
print(f"Storage size: {health['storage_size']}")
```

### Performance Monitoring

```yaml
# Enable metrics collection
monitoring:
  enabled: true
  metrics_backend: "prometheus"  # or "statsd"
  
storage:
  backend: "postgresql"
  # ... connection details ...
  
  # Performance monitoring
  log_slow_queries: true
  slow_query_threshold: 1000  # milliseconds
  track_query_stats: true
```

### Backup and Recovery

**PostgreSQL:**
```bash
# Backup
pg_dump go_doc_go > backup.sql

# Restore
psql go_doc_go < backup.sql
```

**Elasticsearch:**
```bash
# Backup
curl -X PUT "localhost:9200/_snapshot/backup/snapshot_1?wait_for_completion=true"

# Restore  
curl -X POST "localhost:9200/_snapshot/backup/snapshot_1/_restore"
```

## Troubleshooting

### Common Issues

**Connection timeouts:**
```yaml
storage:
  connection_timeout: 60
  pool_timeout: 30
  retry_attempts: 3
  retry_delay: 5
```

**Memory issues:**
```yaml  
storage:
  batch_size: 500        # Reduce batch size
  max_workers: 4         # Reduce parallelism
  stream_results: true   # Don't load all into memory
```

**Slow queries:**
```yaml
storage:
  # Add indexes
  create_indexes: true
  
  # Query optimization
  use_prepared_statements: true
  enable_query_cache: true
  
  # For PostgreSQL
  enable_jit: false  # Disable for small queries
  work_mem: "256MB"  # More memory for sorting
```

## Next Steps

- [Scaling Guide](scaling.md) - Horizontal scaling and performance optimization
- [Configuration Reference](configuration.md) - Detailed configuration options
- [API Reference](api.md) - Programmatic storage backend management
- [Embeddings Guide](embeddings.md) - Vector search configuration