# Domain Entity Extraction and Knowledge Graph Testing

This directory contains a comprehensive test suite for the domain entity extraction and relationship detection system, with output to a Neo4j knowledge graph.

## Overview

The test suite demonstrates:
1. **Domain Ontology Application**: Using the financial markets ontology to extract entities from unstructured text
2. **Entity Extraction**: Identifying companies, people, financial metrics, and market concepts
3. **Relationship Detection**: Discovering relationships between entities based on ontology rules
4. **Knowledge Graph Creation**: Building a Neo4j graph database from the extracted information
5. **Graph Analytics**: Analyzing the resulting knowledge graph

## Components

### 1. Financial Markets Ontology
**Location**: `examples/ontologies/financial_markets.yaml`

A comprehensive ontology for financial documents including:
- **30+ Terms**: Companies, executives, financial metrics, market concepts
- **20+ Relationship Rules**: Company relationships, financial reporting, market dynamics
- **Multiple Rule Types**: Semantic, regex, and keyword-based extraction

Key entity types:
- Companies and tickers
- Executives (CEO, CFO)
- Financial metrics (revenue, earnings, growth)
- Market concepts (competition, partnerships, risks)
- Analyst ratings and price targets

### 2. Neo4j Docker Setup
**Location**: `tests/docker/docker-compose.yml`

Pre-configured Neo4j database with:
- Neo4j Community Edition 5.13
- APOC and Graph Data Science plugins
- Optimized memory settings
- Web interface on port 7474
- Bolt protocol on port 7687

### 3. Neo4j Exporter Module
**Location**: `tests/integration/neo4j_exporter.py`

Python module for exporting extraction results to Neo4j:
- Document, Element, and Term nodes
- Relationship creation and management
- Graph statistics and analytics
- Path finding between terms
- GraphML export capability

### 4. Integration Test
**Location**: `tests/integration/test_financial_entity_extraction.py`

Complete end-to-end test that:
- Generates realistic earnings call transcript paragraphs
- Simulates 5 companies across 4 quarters (200+ paragraphs)
- Extracts entities using the financial ontology
- Builds relationships between entities
- Exports everything to Neo4j
- Provides graph analytics

## Installation

### Prerequisites
```bash
# Install required Python packages
pip install neo4j pyyaml

# Install Docker and Docker Compose
# See https://docs.docker.com/get-docker/
```

### Setup Steps

1. **Start Neo4j Database**:
```bash
cd tests/docker
docker-compose up -d

# Wait for Neo4j to start (about 30 seconds)
docker-compose logs -f neo4j
```

2. **Verify Neo4j is Running**:
- Open http://localhost:7474 in your browser
- Login with username: `neo4j`, password: `go-doc-go123`

## Running the Tests

### Basic Test Run
```bash
cd tests/integration
python test_financial_entity_extraction.py
```

This will:
1. Generate simulated earnings call data
2. Extract entities and relationships
3. Export to Neo4j
4. Display statistics

### Expected Output
```
================================================================================
Financial Entity Extraction and Knowledge Graph Test
================================================================================

1. Generating simulated earnings call transcript paragraphs...
   Generated 200 paragraphs from 20 earnings calls

2. Setting up test database...
   Inserted 20 documents and 200 elements

3. Loading financial markets ontology...
   Loaded ontology: financial_markets
   Terms: 35
   Mapping rules: 25
   Relationship rules: 20

4. Extracting entities using domain ontology...
   Processing tech_q1_2024_earnings with 10 elements...
   ...
   Extracted 150 entity mappings
   Discovered 45 relationships

5. Exporting to Neo4j knowledge graph...
   Exported 5 documents
   Exported 50 elements
   Exported 35 terms
   Exported 150 element-term mappings
   Exported 45 domain relationships

6. Graph Statistics:
   Total nodes: 90
   - Documents: 5
   - Elements: 50
   - Terms: 35
   Total relationships: 245
   Average mappings per element: 3.00

   Top extracted terms:
      company: 25 occurrences
      revenue: 20 occurrences
      ceo: 15 occurrences
      guidance: 12 occurrences
      growth_rate: 10 occurrences

✅ Successfully created financial knowledge graph in Neo4j!
```

## Exploring the Knowledge Graph

### Neo4j Browser Queries

After running the test, explore the graph with these Cypher queries:

1. **View all nodes and relationships** (limited):
```cypher
MATCH (n) 
RETURN n 
LIMIT 100
```

2. **Find all companies and their relationships**:
```cypher
MATCH (c:Term {label: 'Company'})<-[:MAPPED_TO]-(e:Element)-[:MAPPED_TO]->(other:Term)
RETURN c, e, other
LIMIT 50
```

3. **Trace revenue discussions**:
```cypher
MATCH (r:Term {label: 'Revenue'})<-[:MAPPED_TO]-(e:Element)
MATCH (e)-[:BELONGS_TO]->(d:Document)
RETURN r, e, d
```

4. **Find competitive relationships**:
```cypher
MATCH (c1:Term {label: 'Company'})<-[:MAPPED_TO]-(e1:Element)
WHERE e1.content_preview CONTAINS 'compet'
MATCH (e1)-[:MAPPED_TO]->(c2:Term {label: 'Competitor'})
RETURN c1, e1, c2
```

5. **Analyze CEO statements**:
```cypher
MATCH (ceo:Term {label: 'Chief Executive Officer'})<-[:MAPPED_TO]-(e:Element)
MATCH (e)-[:MAPPED_TO]->(topic:Term)
WHERE topic.label <> 'Chief Executive Officer'
RETURN ceo, e, topic
LIMIT 30
```

6. **Financial metrics network**:
```cypher
MATCH path = (t1:Term)-[:MAPPED_TO*2]-(t2:Term)
WHERE t1.label IN ['Revenue', 'Earnings', 'Growth Rate']
AND t2.label IN ['Revenue', 'Earnings', 'Growth Rate']
AND t1 <> t2
RETURN path
LIMIT 20
```

### Graph Patterns to Explore

1. **Company Networks**: See how companies are mentioned together
2. **Executive Discussions**: Track what CEOs and CFOs talk about
3. **Financial Metrics**: Understand relationships between revenue, earnings, and growth
4. **Risk and Opportunity**: Find discussions of risks and opportunities
5. **Temporal Analysis**: Compare entities across quarters

## Customizing the Test

### Using Your Own Data

To use actual earnings call transcripts from your database:

1. Modify `test_financial_entity_extraction.py`:
```python
# Replace the simulator with database queries
from go_doc_go.content_source.database import DatabaseContentSource

source = DatabaseContentSource(config)
documents = source.get_documents(
    filters={'doc_type': 'earnings_call'},
    limit=10
)
```

2. Process actual paragraphs:
```python
for doc in documents:
    elements = db.get_elements_by_document(doc['doc_id'])
    relationships = detector.detect_relationships(doc, elements)
```

### Adding More Ontologies

Create additional ontologies for other domains:
- Healthcare: Clinical trials, drug development, medical devices
- Legal: Contracts, litigation, regulatory compliance
- Technology: Patents, research papers, technical documentation

### Performance Testing

For larger datasets:
```python
# Process in batches
batch_size = 100
for i in range(0, len(documents), batch_size):
    batch = documents[i:i+batch_size]
    process_batch(batch)
```

## Troubleshooting

### Neo4j Connection Issues
```bash
# Check if Neo4j is running
docker ps | grep neo4j

# View Neo4j logs
docker logs go-doc-go-neo4j

# Restart Neo4j
cd tests/docker
docker-compose restart neo4j
```

### Memory Issues with Large Datasets
```yaml
# Increase Neo4j memory in docker-compose.yml
NEO4J_dbms_memory_heap_max__size: 4G
NEO4J_dbms_memory_pagecache_size: 2G
```

### Slow Entity Extraction
- Adjust confidence thresholds in the ontology
- Use embeddings for better semantic matching
- Process documents in parallel

## Advanced Features

### 1. Cross-Document Relationships
The system can find relationships across different documents:
```python
# Find relationships between companies mentioned in different calls
cross_doc_rels = detector.detect_cross_document_relationships(doc_ids)
```

### 2. Temporal Analysis
Track how entities and relationships change over time:
```cypher
MATCH (d:Document)-[:CONTAINS]->(e:Element)-[:MAPPED_TO]->(t:Term)
WHERE t.label = 'Revenue'
RETURN d.metadata.quarter, avg(toFloat(e.metadata.value)) as avg_revenue
ORDER BY d.metadata.quarter
```

### 3. Graph Embeddings
Export the graph for machine learning:
```python
# Export to GraphML for embedding generation
exporter.export_to_graphml('/path/to/export.graphml')
```

## Next Steps

1. **Expand Ontologies**: Add more domain-specific ontologies
2. **Improve Extraction**: Use embeddings for better semantic matching
3. **Add Validation**: Create ground truth annotations for accuracy measurement
4. **Scale Testing**: Test with larger document collections
5. **Graph Analytics**: Implement community detection and centrality measures
6. **Machine Learning**: Use the knowledge graph for downstream ML tasks

## Contributing

To add new test cases or ontologies:
1. Create ontology YAML files in `examples/ontologies/`
2. Add test documents in `tests/test_documents/`
3. Write integration tests in `tests/integration/`
4. Document your additions in this README

## License

This test suite is part of the go-doc-go project and follows the same license terms.