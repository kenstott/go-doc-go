# Neo4j Test Container

This directory contains the Neo4j graph database configuration for running tests.

## Quick Start

```bash
# Start the container
docker-compose -f compose.yaml up -d

# Wait for health check (Neo4j takes ~20-30 seconds to start)
docker-compose -f compose.yaml ps

# Test the connection
curl http://localhost:7474

# Access Neo4j Browser
open http://localhost:7474
# Login: neo4j / testpass123

# Stop the container
docker-compose -f compose.yaml down
```

## Configuration

- **HTTP Port**: 7474 (Neo4j Browser)
- **Bolt Port**: 7687 (Database connections)
- **Username**: neo4j
- **Password**: testpass123
- **Memory**: 512MB heap, 256MB pagecache

## Initialize with Test Data

After the container starts, you can load the test data:

```bash
# Using cypher-shell
docker exec -it go-doc-go-test-neo4j cypher-shell -u neo4j -p testpass123 -f /var/lib/neo4j/import/init.cypher

# Or via Neo4j Browser
# 1. Open http://localhost:7474
# 2. Login with neo4j/testpass123
# 3. Copy and paste the contents of init.cypher
```

## Connection Examples

### Python with py2neo
```python
from py2neo import Graph

# Connect to Neo4j
graph = Graph("bolt://localhost:7687", auth=("neo4j", "testpass123"))

# Run a query
result = graph.run("MATCH (n) RETURN count(n) as count").data()
print(f"Total nodes: {result[0]['count']}")

# Create a node
graph.run("CREATE (d:Document {doc_id: $id, doc_type: $type})", 
          id="test-doc-3", type="pdf")
```

### Python with neo4j driver
```python
from neo4j import GraphDatabase

# Create driver
driver = GraphDatabase.driver(
    "bolt://localhost:7687", 
    auth=("neo4j", "testpass123")
)

# Execute query
with driver.session() as session:
    result = session.run("MATCH (d:Document) RETURN d.doc_id as id")
    for record in result:
        print(record["id"])

driver.close()
```

### Environment Variables
```bash
export TEST_NEO4J_URI=bolt://localhost:7687
export TEST_NEO4J_USER=neo4j
export TEST_NEO4J_PASSWORD=testpass123
```

## Cypher Queries

### Basic Operations
```cypher
// Count all nodes
MATCH (n) RETURN count(n);

// Find all documents
MATCH (d:Document) RETURN d;

// Find elements in a document
MATCH (d:Document {doc_id: 'test-doc-1'})-[:CONTAINS]->(e:Element)
RETURN e.element_id, e.content_preview;

// Find entities mentioned in elements
MATCH (e:Element)-[:MENTIONS]->(entity:Entity)
RETURN e.element_id, entity.name, entity.entity_type;

// Find similar elements
MATCH (e1:Element)-[r:SIMILAR_TO]->(e2:Element)
WHERE r.score > 0.8
RETURN e1.element_id, e2.element_id, r.score;
```

### Full-Text Search
```cypher
// Search documents
CALL db.index.fulltext.queryNodes('document_search', 'test')
YIELD node, score
RETURN node.doc_id, score;

// Search elements
CALL db.index.fulltext.queryNodes('element_search', 'paragraph')
YIELD node, score
RETURN node.element_id, node.content_preview, score;
```

## Performance Optimizations

The instance is configured for testing with:
- Limited memory (512MB heap + 256MB pagecache)
- Community edition (no clustering overhead)
- APOC plugin for advanced operations
- Indexes and constraints pre-created

⚠️ **WARNING**: These settings are ONLY for testing. For production, use appropriate memory settings and Neo4j Enterprise.

## Troubleshooting

If Neo4j fails to start:
```bash
# Check logs
docker-compose -f compose.yaml logs neo4j-test

# Common issues:
# 1. Port 7474 or 7687 already in use
# 2. Not enough memory - increase Docker memory
# 3. Permission issues with volumes

# Reset everything
docker-compose -f compose.yaml down -v
docker-compose -f compose.yaml up -d
```

## Integration with Tests

```python
import os
from neo4j import GraphDatabase
from neo4j.exceptions import ServiceUnavailable

def neo4j_available():
    try:
        driver = GraphDatabase.driver(
            os.getenv('TEST_NEO4J_URI', 'bolt://localhost:7687'),
            auth=(
                os.getenv('TEST_NEO4J_USER', 'neo4j'),
                os.getenv('TEST_NEO4J_PASSWORD', 'testpass123')
            )
        )
        with driver.session() as session:
            session.run("RETURN 1")
        driver.close()
        return True
    except ServiceUnavailable:
        return False

# pytest fixture
@pytest.fixture
def neo4j_graph():
    if not neo4j_available():
        pytest.skip("Neo4j not available")
    
    from py2neo import Graph
    return Graph(
        "bolt://localhost:7687",
        auth=("neo4j", "testpass123")
    )
```

## Graph Model

The test data creates this graph structure:
```
Document (test-doc-1)
    |-- CONTAINS --> Element (elem-1) 
    |                    |-- MENTIONS --> Entity (entity-1: Test Corporation)
    |                    |-- SIMILAR_TO --> Element (elem-2)
    |
    |-- CONTAINS --> Element (elem-2)
                         |-- MENTIONS --> Entity (entity-2: John Doe)

Document (test-doc-2)
```

This provides a basic graph for testing document structures, entity extraction, and relationship queries.