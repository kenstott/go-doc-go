# Semantic Search Example

This example demonstrates how to enable semantic search using ONNX embeddings for similarity-based document queries.

## What This Example Does

- Generates vector embeddings for all document elements
- Enables semantic similarity search (not just keyword matching)
- Uses contextual embeddings (includes surrounding text)
- Stores embeddings in PostgreSQL with pgvector or Parquet

## Prerequisites

### 1. ONNX Runtime

```bash
# macOS
pip install onnxruntime-coreml

# Linux
pip install onnxruntime

# Or download from https://onnxruntime.ai
```

### 2. Export Embedding Model to ONNX

```bash
# Install dependencies
pip install onnx sentence-transformers torch optimum[onnxruntime]

# Export model
python -c "
from optimum.onnxruntime import ORTModelForFeatureExtraction
from transformers import AutoTokenizer

model_id = 'sentence-transformers/all-MiniLM-L6-v2'
ort_model = ORTModelForFeatureExtraction.from_pretrained(model_id, export=True)
tokenizer = AutoTokenizer.from_pretrained(model_id)

ort_model.save_pretrained('./models/all-MiniLM-L6-v2')
tokenizer.save_pretrained('./models/all-MiniLM-L6-v2')
print('Model exported to ./models/all-MiniLM-L6-v2')
"
```

### 3. Set ONNX Runtime Library Path

```bash
# Find the library
find ~/.local -name "libonnxruntime*.dylib" 2>/dev/null || \
find /opt/homebrew -name "libonnxruntime*.dylib" 2>/dev/null

# Set environment variable
export ONNXRUNTIME_SHARED_LIBRARY_PATH="/path/to/libonnxruntime.dylib"

# Or add to your shell profile
echo 'export ONNXRUNTIME_SHARED_LIBRARY_PATH="/path/to/libonnxruntime.dylib"' >> ~/.zshrc
```

### 4. PostgreSQL with pgvector (recommended)

```bash
# Run PostgreSQL with pgvector extension
docker run --name godocgo-pgvector \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=godocgo \
  -p 5432:5432 \
  -d pgvector/pgvector:pg15

# Wait for startup
sleep 5

# Enable pgvector extension
docker exec -it godocgo-pgvector psql -U postgres -d godocgo -c "CREATE EXTENSION IF NOT EXISTS vector;"
```

## Quick Start

### 1. Add Documents

```bash
mkdir -p docs
cp /path/to/your/*.pdf docs/
```

### 2. Run Worker with Embeddings

```bash
../../bin/goworker --config config.toml --workers 4
```

This will:
- Parse documents
- Generate embeddings for each element
- Store embeddings in PostgreSQL (with pgvector) or Parquet

### 3. Semantic Search Query

#### Using Python

```python
import psycopg2
import numpy as np
from sentence_transformers import SentenceTransformer

# Load embedding model
model = SentenceTransformer('sentence-transformers/all-MiniLM-L6-v2')

# Connect to database
conn = psycopg2.connect(
    host="localhost",
    database="godocgo",
    user="postgres",
    password="password"
)

# Search function
def semantic_search(query, limit=10):
    # Generate query embedding
    query_embedding = model.encode(query)

    # Find similar elements using cosine similarity
    cursor = conn.cursor()
    cursor.execute("""
        SELECT
            e.element_id,
            e.element_type,
            e.content_preview,
            d.file_path,
            1 - (emb.embedding <=> %s::vector) as similarity
        FROM elements e
        JOIN embeddings emb ON e.element_id = emb.element_id
        JOIN documents d ON e.doc_id = d.doc_id
        ORDER BY emb.embedding <=> %s::vector
        LIMIT %s
    """, (query_embedding.tolist(), query_embedding.tolist(), limit))

    results = cursor.fetchall()
    cursor.close()

    return results

# Example search
results = semantic_search("What are the system requirements?", limit=10)

for element_id, element_type, content, file_path, similarity in results:
    print(f"\n{file_path} ({element_type}) - Similarity: {similarity:.3f}")
    print(f"  {content}")
```

#### Using DuckDB (Parquet embeddings)

```python
import duckdb
import numpy as np
from sentence_transformers import SentenceTransformer

# Load model
model = SentenceTransformer('sentence-transformers/all-MiniLM-L6-v2')

# Connect to DuckDB
conn = duckdb.connect()

# Search function
def semantic_search_parquet(query, limit=10):
    # Generate query embedding
    query_emb = model.encode(query)

    # Load embeddings and elements
    results = conn.execute("""
        SELECT
            e.element_id,
            e.element_type,
            e.content_preview,
            e.doc_id,
            emb.embedding
        FROM read_parquet('./output/analytics.parquet/elements/*.parquet') e
        JOIN read_parquet('./output/analytics.parquet/embeddings/*.parquet') emb
          ON e.element_id = emb.element_id
    """).fetchall()

    # Calculate similarity in Python (DuckDB doesn't have vector ops)
    scored = []
    for element_id, elem_type, content, doc_id, embedding in results:
        emb_array = np.array(embedding)
        similarity = np.dot(query_emb, emb_array) / (
            np.linalg.norm(query_emb) * np.linalg.norm(emb_array)
        )
        scored.append((element_id, elem_type, content, doc_id, similarity))

    # Sort by similarity
    scored.sort(key=lambda x: x[4], reverse=True)
    return scored[:limit]

# Example search
results = semantic_search_parquet("installation instructions", limit=5)

for element_id, elem_type, content, doc_id, similarity in results:
    print(f"\n{doc_id} ({elem_type}) - Similarity: {similarity:.3f}")
    print(f"  {content}")
```

## Configuration Options

### Basic Embeddings

```toml
[embedding]
enabled = true
provider = "onnx"
model_path = "./models/all-MiniLM-L6-v2"
```

### Contextual Embeddings (GraphRAG-lite)

Includes surrounding text for better semantic understanding:

```toml
[embedding]
enabled = true
provider = "onnx"
model_path = "./models/all-MiniLM-L6-v2"

# Contextual settings
contextual = true
predecessor_count = 2    # Include 2 previous elements
successor_count = 2      # Include 2 following elements
```

### Advanced Settings

```toml
[embedding]
enabled = true
provider = "onnx"
model_path = "./models/all-MiniLM-L6-v2"

# Performance
batch_size = 32          # Process 32 elements at once
max_sequence_length = 512 # Truncate long text
num_threads = 4          # ONNX threads

# Quality
normalize_embeddings = true
pooling_strategy = "mean"  # or "cls", "max"

# Contextual
contextual = true
predecessor_count = 2
successor_count = 2
context_separator = " [SEP] "
```

## Embedding Models

### Recommended Models

| Model | Size | Dimensions | Speed | Quality |
|-------|------|------------|-------|---------|
| `all-MiniLM-L6-v2` | 80MB | 384 | Fast | Good |
| `all-mpnet-base-v2` | 420MB | 768 | Medium | Better |
| `BAAI/bge-small-en-v1.5` | 133MB | 384 | Fast | Very Good |
| `BAAI/bge-base-en-v1.5` | 438MB | 768 | Medium | Excellent |

### Changing Models

```bash
# Export a different model
python -c "
from optimum.onnxruntime import ORTModelForFeatureExtraction
from transformers import AutoTokenizer

model_id = 'BAAI/bge-base-en-v1.5'
ort_model = ORTModelForFeatureExtraction.from_pretrained(model_id, export=True)
tokenizer = AutoTokenizer.from_pretrained(model_id)

ort_model.save_pretrained('./models/bge-base-en-v1.5')
tokenizer.save_pretrained('./models/bge-base-en-v1.5')
"
```

Update config:

```toml
[embedding]
model_path = "./models/bge-base-en-v1.5"
```

## Performance Optimization

### Batch Size

```toml
[embedding]
batch_size = 64  # Larger batches = faster (more memory)
```

### GPU Acceleration

```toml
[embedding]
provider = "onnx"
execution_provider = "CUDAExecutionProvider"  # Use GPU
# or "CoreMLExecutionProvider" for macOS
```

### Memory Management

```toml
[embedding]
batch_size = 16            # Smaller batches
max_sequence_length = 256  # Shorter sequences
embedding_cache_size = 5000 # Cache embeddings
```

## Use Cases

### Document Similarity

Find documents similar to a reference document:

```python
# Get reference document embedding
ref_embedding = get_document_embedding("reference-doc-id")

# Find similar documents
results = conn.execute("""
    SELECT
        d.doc_id,
        d.file_path,
        1 - (AVG(emb.embedding) <=> %s::vector) as similarity
    FROM documents d
    JOIN elements e ON d.doc_id = e.doc_id
    JOIN embeddings emb ON e.element_id = emb.element_id
    GROUP BY d.doc_id, d.file_path
    ORDER BY similarity DESC
    LIMIT 10
""", (ref_embedding.tolist(),))
```

### Question Answering

Find relevant context for questions:

```python
def answer_question(question):
    # Find relevant passages
    results = semantic_search(question, limit=5)

    # Concatenate context
    context = "\n\n".join([content for _, _, content, _, _ in results])

    # Use context with LLM (Claude, GPT, etc.)
    answer = llm.generate(f"Context: {context}\n\nQuestion: {question}")

    return answer, results
```

### Semantic Clustering

Group similar elements:

```python
from sklearn.cluster import KMeans

# Load all embeddings
embeddings_df = pd.read_parquet('./output/analytics.parquet/embeddings')

# Cluster
kmeans = KMeans(n_clusters=10)
embeddings_df['cluster'] = kmeans.fit_predict(embeddings_df['embedding'].tolist())

# Analyze clusters
for cluster_id in range(10):
    cluster_elements = embeddings_df[embeddings_df['cluster'] == cluster_id]
    print(f"\nCluster {cluster_id} ({len(cluster_elements)} elements)")
    # ... analyze cluster content
```

## Monitoring

### Embedding Generation Progress

```bash
# Watch log output
../../bin/goworker --config config.toml 2>&1 | grep -i embed

# Example output:
# [INFO] Embedding: Processing batch 1/100 (32 elements)
# [INFO] Embedding: Generated 1000 embeddings (2.3s)
# [INFO] Embedding: Average time per element: 2.3ms
```

### Verify Embeddings

```sql
-- PostgreSQL
SELECT COUNT(*) FROM embeddings;

-- Check dimensions
SELECT element_id, array_length(embedding, 1) as dimensions
FROM embeddings
LIMIT 5;
```

## Troubleshooting

### ONNX Runtime not found

```bash
# Verify library exists
ls $ONNXRUNTIME_SHARED_LIBRARY_PATH

# If not found, reinstall
pip uninstall onnxruntime onnxruntime-coreml
pip install onnxruntime-coreml  # or onnxruntime
```

### Slow embedding generation

```toml
# Increase batch size
[embedding]
batch_size = 64

# Use more threads
num_threads = 8

# Reduce sequence length
max_sequence_length = 256
```

### High memory usage

```toml
# Reduce batch size
[embedding]
batch_size = 16

# Limit sequence length
max_sequence_length = 256

# Disable contextual embeddings
contextual = false
```

## Next Steps

1. **Combine with Neo4j**: Export embeddings to Neo4j for graph + semantic queries
2. **Add ontology extraction**: Semantic search + entity extraction
3. **Scale horizontally**: Use [../distributed-workers/](../distributed-workers/) with embeddings

## Related Documentation

- [Embeddings Guide](../../docs/features/embeddings/README.md)
- [Configuration Reference](../../docs/configuration/README.md)
- [Performance Tuning](../../docs/operations/scaling.md)
