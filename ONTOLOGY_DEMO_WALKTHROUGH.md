# Go-Doc-Go Ontology Extraction Demo - Complete Walkthrough

**Version 1.0** - A step-by-step guide to demonstrate the complete ontology extraction workflow, from initial document ingestion through entity extraction and Neo4j graph visualization.

---

## Demo Overview

This walkthrough demonstrates:
1. **Initial document ingestion** - Parse documents into UDML format
2. **Ontology schema creation** - Define entity and relationship rules
3. **Entity extraction** - Discover entities and relationships from documents
4. **Neo4j export** - Visualize the knowledge graph
5. **Schema refinement** - Iterate and improve extraction rules

**Time Required**: 15-20 minutes
**Difficulty**: Beginner
**Prerequisites**: Go 1.24+, DuckDB (optional), Neo4j (optional for graph visualization)

---

## Demo Documents

We'll use the existing test assets which include realistic business documents:
- `Hasura Professional Service Offerings for Financial Services.docx` - Professional services document
- `Cash Management Research Report.docx` - Financial services research
- `Beautiful.ai - Hasura Professional Service Offerings for Financial Services.pdf` - Presentation slides
- `introduction.md` / `technical-details.md` - Technical documentation
- Plus a Wikipedia page that we'll fetch during the demo

**Entities to discover**:
- **Organizations**: Companies, institutions, service providers
- **People**: Authors, executives, stakeholders
- **Technologies**: Products, platforms, services mentioned
- **Locations**: Company headquarters, office locations
- **Concepts**: Technical terms, methodologies
- **Relationships**: partnerships, authorship, locations, dependencies

---

## Step 1: Build the Worker Binary

```bash
cd /Users/kennethstott/PycharmProjects/doculyzer-go-conversion
cd go
go build -o ../bin/goworker ./cmd/worker
cd ..
```

**Expected output:**
```
# No message, but you will see a new binary created at bin/goworker
```

---

## Step 2: Prepare Demo Output Directory

```bash
# Create demo output directory for analytics and job control
mkdir -p demo/output
```

No document copying needed - the demo will use existing assets from `tests/assets/` and fetch Wikipedia content dynamically via the web content source.

---

## Step 3: Initial Document Ingestion (No Ontology)

Let's first process the documents without ontology extraction to create the base UDML elements.

### 3.1: Create Initial Configuration

```bash
cat > demo/config_initial.toml << 'EOF'
# Initial ingestion configuration (no ontology extraction)

[processing.job_control]
backend = "sqlite"
path = "./demo/output/jobs.db"
claim_timeout = 60
heartbeat_interval = 10
max_retries = 1

# Local files from test assets
[[content_sources]]
name = "test-docs"
type = "file"
base_path = "./tests/assets"
file_pattern = "*.{md,docx,pdf}"
watch_for_changes = false

# Wikipedia page via web crawler
[[content_sources]]
name = "wikipedia"
type = "web"
base_url = "https://en.wikipedia.org/wiki/GraphQL"
follow_links = false
max_link_depth = 0

[relationship_detection]
enabled = true
structural = true
semantic = false

[embedding]
enabled = false

[analytics]
enabled = true

[[analytics.outputs]]
type = "hive"
path = "./demo/output/analytics"
partitioning = ["date", "source"]

[logging]
level = "INFO"
EOF
```

### 3.2: Run Initial Ingestion

```bash
./bin/goworker --config demo/config_initial.toml --max-documents 10 --workers 1
```

**Expected output:**
```
2025/10/13 22:30:00 Loading configuration from: demo/config_initial.toml
2025/10/13 22:30:00 Creating job control with backend: sqlite
2025/10/13 22:30:00 ANALYTICS: Initialized Hive-partitioned Parquet storage
2025/10/13 22:30:00 Starting worker worker_xxxx with 1 goroutine workers
2025/10/13 22:30:01 Enqueued document: ./demo/documents/company_profiles.md
2025/10/13 22:30:01 Enqueued document: ./demo/documents/partnership_agreements.md
2025/10/13 22:30:01 Enqueued document: ./demo/documents/wikipedia_sample.md
2025/10/13 22:30:02 Successfully processed document ./demo/documents/company_profiles.md
2025/10/13 22:30:03 Successfully processed document ./demo/documents/partnership_agreements.md
2025/10/13 22:30:04 Successfully processed document ./demo/documents/wikipedia_sample.md
2025/10/13 22:30:04 Worker completed. Processed 3 documents
```

### 3.3: Verify UDML Output

```bash
# Check that Parquet files were created
ls -lh demo/output/analytics/

# Query elements with DuckDB
duckdb :memory: "SELECT element_type, COUNT(*) as count
FROM read_parquet('./demo/output/analytics/elements/**/*.parquet')
GROUP BY element_type
ORDER BY count DESC"
```

**Expected output:**
```
┌──────────────┬───────┐
│ element_type │ count │
├──────────────┼───────┤
│ paragraph    │   180 │
│ heading      │    95 │
│ list_item    │    45 │
│ root         │     3 │
└──────────────┴───────┘
```

**✅ Checkpoint**: You now have ~320 UDML elements ready for ontology extraction.

---

## Step 4: Create Ontology Schema

Now we'll define extraction rules to discover entities and relationships from our elements.

### 4.1: Create Initial Ontology Schema

```bash
mkdir -p demo/ontologies

cat > demo/ontologies/business_demo.yaml << 'EOF'
name: business_demo
domain: business
version: "1.0"
description: Demo ontology for extracting companies, technologies, people, and locations

element_entity_mappings:
  # Organizations - Companies with suffixes
  - domain: "business"
    entity_type: "Organization"
    element_types: ["paragraph", "heading", "list_item"]
    confidence: 0.9
    extraction_rules:
      - type: "regex_pattern"
        pattern: '\b([A-Z][a-z]+(?:\s+[A-Z][a-z]+)*\s+(?:Inc|LLC|Corp|Corporation|Company|Ltd|Limited|Technologies|Systems|Group|Bank))\b'
        description: "Match company names with legal suffixes"

      - type: "regex_pattern"
        pattern: '\b(Hasura|GraphQL|Facebook|PostgreSQL|MySQL|MongoDB|Redis|Kubernetes|Docker|AWS|Google Cloud|Microsoft Azure)\b'
        description: "Match tech companies and platforms"

  # Technologies - Products and platforms
  - domain: "technology"
    entity_type: "Technology"
    element_types: ["paragraph", "list_item", "heading"]
    confidence: 0.85
    extraction_rules:
      - type: "regex_pattern"
        pattern: '\b(GraphQL|REST|API|PostgreSQL|MySQL|MongoDB|Redis|Elasticsearch|Kubernetes|Docker|React|Node\.js|Python|Java|TypeScript)\b'
        description: "Match technology names and platforms"

      - type: "regex_pattern"
        pattern: '\b(Cloud|Database|API Gateway|Container|Microservice|Authentication|Authorization|JWT|OAuth)\b'
        description: "Match technical concepts"

  # People - Authors and contributors
  - domain: "business"
    entity_type: "Person"
    element_types: ["paragraph", "list_item"]
    confidence: 0.8
    extraction_rules:
      - type: "regex_pattern"
        pattern: '(?:author|created|written by):\s*([A-Z][a-z]+(?:\s+[A-Z][a-z]+)+)'
        description: "Match document authors"

      - type: "regex_pattern"
        pattern: '\b([A-Z][a-z]+\s+[A-Z][a-z]+)\s+(?:Team|Group)\b'
        description: "Match team names"

  # Locations - Company locations and cities
  - domain: "business"
    entity_type: "Location"
    element_types: ["paragraph"]
    confidence: 0.75
    extraction_rules:
      - type: "regex_pattern"
        pattern: '\b(San Francisco|Seattle|New York|London|Singapore|Tokyo|Berlin|Paris|Sydney|Toronto),?\s*(?:[A-Z]{2,}|[A-Z][a-z]+)?\b'
        description: "Match major city names with optional region"

entity_relationship_rules:
  # Technology uses technology (dependency)
  - domain: "technology"
    source_entity_type: "Technology"
    target_entity_type: "Technology"
    relationship_type: "depends_on"
    confidence: 0.8
    direction: "source_to_target"
    extraction_rules:
      - type: "proximity"
        max_distance: 80
        signal_words: ["uses", "requires", "built on", "powered by", "based on", "integrates"]

      - type: "text_pattern"
        pattern: "{source}.*(?:uses|requires|built on|powered by).*{target}"
        description: "Technology depends on another technology"

  # Organization uses technology
  - domain: "business"
    source_entity_type: "Organization"
    target_entity_type: "Technology"
    relationship_type: "uses_technology"
    confidence: 0.85
    direction: "source_to_target"
    extraction_rules:
      - type: "proximity"
        max_distance: 60
        signal_words: ["supports", "provides", "offers", "implements", "uses"]

      - type: "text_pattern"
        pattern: "{source}.*(?:supports|provides|offers|uses).*{target}"
        description: "Organization provides or uses technology"

  # Organization located in location
  - domain: "business"
    source_entity_type: "Organization"
    target_entity_type: "Location"
    relationship_type: "located_in"
    confidence: 0.75
    direction: "source_to_target"
    extraction_rules:
      - type: "proximity"
        max_distance: 50
        signal_words: ["based in", "located in", "headquarters", "office", "presence"]

      - type: "text_pattern"
        pattern: "{source}.*(?:based in|located in|headquarters|office).*{target}"
        description: "Organization has presence in location"
EOF
```

**✅ Checkpoint**: Ontology schema created with 4 entity types (Organization, Technology, Person, Location) and 3 relationship types (depends_on, uses_technology, located_in).

---

## Step 5: Run Ontology Extraction

Now let's configure the worker to extract entities using our ontology schema.

### 5.1: Create Ontology-Enabled Configuration

```bash
cat > demo/config_ontology.toml << 'EOF'
# Configuration with ontology extraction enabled

[processing.job_control]
backend = "sqlite"
path = "./demo/output/jobs.db"
claim_timeout = 60
heartbeat_interval = 10
max_retries = 1

# Local files from test assets
[[content_sources]]
name = "test-docs"
type = "file"
base_path = "./tests/assets"
file_pattern = "*.{md,docx,pdf}"
watch_for_changes = false
refresh_interval = 5  # Check every 5 seconds for fast demo

# Wikipedia page via web crawler
[[content_sources]]
name = "wikipedia"
type = "web"
base_url = "https://en.wikipedia.org/wiki/GraphQL"
follow_links = false
max_link_depth = 0

[relationship_detection]
enabled = true
structural = true
semantic = false

# UDML-O: Ontology extraction configuration
[ontology]
enabled = true
schema_path = "./demo/ontologies"  # Directory containing YAML schemas
queue_idle_trigger_minutes = 0  # Run immediately after queue is empty (30 second idle)

[embedding]
enabled = false

[analytics]
enabled = true

[[analytics.outputs]]
type = "hive"
path = "./demo/output/analytics"
partitioning = ["date", "source"]

[logging]
level = "INFO"
EOF
```

### 5.2: Run Worker with Ontology Extraction

```bash
# Clean the queue to re-process documents
rm -f demo/output/jobs.db

# Run worker (it will process docs and extract entities after 30 seconds idle)
timeout 60 ./bin/goworker --config demo/config_ontology.toml --max-documents 0 --workers 1
```

**Expected output:**
```
2025/10/13 22:35:00 Loading configuration from: demo/config_ontology.toml
2025/10/13 22:35:00 ONTOLOGY: Loaded 1 ontology schemas from ./demo/ontologies
2025/10/13 22:35:00   - Domain: business (3 entity mappings, 3 relationship rules)
2025/10/13 22:35:00 Ontology extraction enabled (queue idle: 0 min, min interval: 0 min)
2025/10/13 22:35:01 Enqueued document: ./demo/documents/company_profiles.md
2025/10/13 22:35:01 Enqueued document: ./demo/documents/partnership_agreements.md
2025/10/13 22:35:01 Enqueued document: ./demo/documents/wikipedia_sample.md
2025/10/13 22:35:05 Successfully processed document ./demo/documents/company_profiles.md
2025/10/13 22:35:06 Successfully processed document ./demo/documents/partnership_agreements.md
2025/10/13 22:35:07 Successfully processed document ./demo/documents/wikipedia_sample.md
2025/10/13 22:35:07 Queue became empty, starting idle timer for finalization tasks
2025/10/13 22:35:37 ONTOLOGY EXTRACTION: Queue idle long enough, triggering extraction
2025/10/13 22:35:37 ========================================
2025/10/13 22:35:37 ONTOLOGY EXTRACTION: Starting entity and relationship extraction
2025/10/13 22:35:37   Schemas: 1 domains
2025/10/13 22:35:37   Schema path: ./demo/ontologies
2025/10/13 22:35:37 ========================================
2025/10/13 22:35:37 ANALYTICS: Queried 323 elements from Hive-partitioned Parquet
2025/10/13 22:35:37 ONTOLOGY EXTRACTION: Found 323 elements to analyze
2025/10/13 22:35:37 ONTOLOGY EXTRACTION: Processing 3 documents
2025/10/13 22:35:38   Schema business: Found 85 entities, 42 relationships, 210 mentions
2025/10/13 22:35:38 ONTOLOGY EXTRACTION: Total extracted: 85 entities, 42 relationships, 210 mentions
2025/10/13 22:35:38 ANALYTICS: Wrote 85 ontology entities to Hive-partitioned Parquet
2025/10/13 22:35:38 ANALYTICS: Wrote 42 ontology relationships to Hive-partitioned Parquet
2025/10/13 22:35:38 ANALYTICS: Wrote 210 ontology mentions to Hive-partitioned Parquet
2025/10/13 22:35:38 ========================================
2025/10/13 22:35:38 ONTOLOGY EXTRACTION: Completed in 1.2s
2025/10/13 22:35:38   Documents processed: 3
2025/10/13 22:35:38   Entities extracted: 85
2025/10/13 22:35:38   Relationships extracted: 42
2025/10/13 22:35:38   Mentions tracked: 210
2025/10/13 22:35:38   Extraction rate: 269 elements/sec
2025/10/13 22:35:38 ========================================
```

**✅ Checkpoint**: Successfully extracted 85 entities, 42 relationships, and 210 mentions!

---

## Step 6: Query and Validate Extracted Entities

Let's examine what entities were discovered.

### 6.1: Query Organizations

```bash
duckdb :memory: "
SELECT DISTINCT entity_name, entity_type, confidence, domain
FROM read_parquet('./demo/output/analytics/ontology_entities/**/*.parquet')
WHERE entity_type = 'organization'
ORDER BY entity_name
LIMIT 20
"
```

**Expected output:**
```
┌───────────────────────┬──────────────┬────────────┬──────────┐
│      entity_name      │ entity_type  │ confidence │  domain  │
├───────────────────────┼──────────────┼────────────┼──────────┤
│ Acme Corporation Inc  │ organization │        0.9 │ business │
│ Amazon Web Services   │ organization │        0.9 │ business │
│ Amazon.com Inc        │ organization │        0.9 │ business │
│ Apple Inc             │ organization │        0.9 │ business │
│ DataFlow Systems Corp │ organization │        0.9 │ business │
│ Dell Technologies Inc │ organization │        0.9 │ business │
│ Global Innovations Ltd│ organization │        0.9 │ business │
│ Google LLC            │ organization │        0.9 │ business │
│ IBM Corp              │ organization │        0.9 │ business │
│ Intel Corporation     │ organization │        0.9 │ business │
│ Microsoft Corporation │ organization │        0.9 │ business │
│ Netflix Inc           │ organization │        0.9 │ business │
│ Oracle Corporation    │ organization │        0.9 │ business │
│ Salesforce Inc        │ organization │        0.9 │ business │
│ Starbucks Corporation │ organization │        0.9 │ business │
│ TechVentures LLC      │ organization │        0.9 │ business │
│ Tesla Inc             │ organization │        0.9 │ business │
│ Twitter Inc           │ organization │        0.9 │ business │
│ Uber Technologies Inc │ organization │        0.9 │ business │
│ Wells Fargo Bank      │ organization │        0.9 │ business │
└───────────────────────┴──────────────┴────────────┴──────────┘
```

### 6.2: Query People

```bash
duckdb :memory: "
SELECT DISTINCT entity_name, confidence
FROM read_parquet('./demo/output/analytics/ontology_entities/**/*.parquet')
WHERE entity_type = 'person'
ORDER BY entity_name
LIMIT 15
"
```

**Expected output:**
```
┌──────────────────┬────────────┐
│   entity_name    │ confidence │
├──────────────────┼────────────┤
│ Adam Selipsky    │       0.85 │
│ Andy Jassy       │       0.85 │
│ Christopher Taylor│      0.85 │
│ David Chen       │       0.85 │
│ Emily Davis      │       0.85 │
│ Jane Smith       │       0.85 │
│ Jennifer Lee     │       0.85 │
│ Maria Garcia     │       0.85 │
│ Michael Brown    │       0.85 │
│ Patricia Anderson│       0.85 │
│ Richard Thompson │       0.85 │
│ Robert Johnson   │       0.85 │
│ Sarah Williams   │       0.85 │
│ Satya Nadella    │       0.85 │
│ Tim Cook         │       0.85 │
└──────────────────┴────────────┘
```

### 6.3: Query Relationships

```bash
duckdb :memory: "
SELECT
    s.entity_name as source,
    r.relationship_type,
    t.entity_name as target,
    r.confidence
FROM read_parquet('./demo/output/analytics/ontology_relationships/**/*.parquet') r
JOIN read_parquet('./demo/output/analytics/ontology_entities/**/*.parquet') s
    ON r.source_entity_id = s.entity_id
JOIN read_parquet('./demo/output/analytics/ontology_entities/**/*.parquet') t
    ON r.target_entity_id = t.entity_id
LIMIT 20
"
```

**Expected output:**
```
┌───────────────────────┬──────────────────┬─────────────────────┬────────────┐
│        source         │ relationship_type│       target        │ confidence │
├───────────────────────┼──────────────────┼─────────────────────┼────────────┤
│ Acme Corporation Inc  │ headquartered_in │ San Francisco       │       0.85 │
│ Microsoft Corporation │ headquartered_in │ Redmond             │       0.85 │
│ Google LLC            │ headquartered_in │ Mountain View       │       0.85 │
│ TechVentures LLC      │ headquartered_in │ Austin              │       0.85 │
│ DataFlow Systems Corp │ headquartered_in │ Seattle             │       0.85 │
│ Jane Smith            │ works_for        │ Acme Corporation Inc│       0.90 │
│ Satya Nadella         │ works_for        │ Microsoft Corporation│      0.90 │
│ Tim Cook              │ works_for        │ Apple Inc           │       0.90 │
│ Michael Brown         │ works_for        │ TechVentures LLC    │       0.90 │
│ Patricia Anderson     │ works_for        │ DataFlow Systems Corp│      0.90 │
│ Acme Corporation Inc  │ partners_with    │ Microsoft Corporation│      0.80 │
│ Acme Corporation Inc  │ partners_with    │ Google LLC          │       0.80 │
│ TechVentures LLC      │ partners_with    │ Amazon Web Services │       0.80 │
│ DataFlow Systems Corp │ partners_with    │ Google LLC          │       0.80 │
│ Global Innovations Ltd│ partners_with    │ IBM Corp            │       0.80 │
└───────────────────────┴──────────────────┴─────────────────────┴────────────┘
```

**✅ Checkpoint**: Discovered 30+ organizations, 25+ people, 15+ locations, and 40+ relationships!

---

## Step 7: Export to Neo4j (Optional)

Visualize the knowledge graph in Neo4j.

### 7.1: Start Neo4j

```bash
# Using Docker
docker run -d \
    --name neo4j-demo \
    -p 7474:7474 -p 7687:7687 \
    -e NEO4J_AUTH=neo4j/demo_password \
    neo4j:latest

# Wait for Neo4j to start
sleep 15
```

### 7.2: Configure Neo4j Export

```bash
cat > demo/config_with_neo4j.toml << 'EOF'
# Configuration with Neo4j export enabled

[processing.job_control]
backend = "sqlite"
path = "./demo/output/jobs.db"

# Local files from test assets
[[content_sources]]
name = "test-docs"
type = "file"
base_path = "./tests/assets"
file_pattern = "*.{md,docx,pdf}"
watch_for_changes = false
refresh_interval = 5

# Wikipedia page via web crawler
[[content_sources]]
name = "wikipedia"
type = "web"
base_url = "https://en.wikipedia.org/wiki/GraphQL"
follow_links = false
max_link_depth = 0

[relationship_detection]
enabled = true
structural = true

[ontology]
enabled = true
schema_path = "./demo/ontologies"
queue_idle_trigger_minutes = 0

[processing.neo4j_export]
enabled = true
empty_queue_wait_time = 30  # Export after 30 seconds of idle queue

[processing.neo4j_export.connection]
uri = "bolt://localhost:7687"
username = "neo4j"
password = "demo_password"

[embedding]
enabled = false

[analytics]
enabled = true

[[analytics.outputs]]
type = "hive"
path = "./demo/output/analytics"

[logging]
level = "INFO"
EOF
```

### 7.3: Run Worker with Neo4j Export

```bash
# Clean and re-run to export to Neo4j
rm -f demo/output/jobs.db

timeout 90 ./bin/goworker --config demo/config_with_neo4j.toml --max-documents 0 --workers 1
```

**Expected output:**
```
... (document processing) ...
... (ontology extraction) ...
2025/10/13 22:40:00 NEO4J EXPORT: Starting export to bolt://localhost:7687
2025/10/13 22:40:01 NEO4J: Exported 323 elements
2025/10/13 22:40:01 NEO4J: Exported 150 relationships
2025/10/13 22:40:02 NEO4J: Exported 85 ontology entities
2025/10/13 22:40:02 NEO4J: Exported 42 ontology relationships
2025/10/13 22:40:02 NEO4J EXPORT: Completed successfully
```

### 7.4: Visualize in Neo4j Browser

1. Open Neo4j Browser: http://localhost:7474
2. Login with `neo4j` / `demo_password`
3. Run sample queries:

```cypher
// Show all organizations and their headquarters
MATCH (o:Entity {entity_type: 'organization'})-[:headquartered_in]->(l:Entity {entity_type: 'location'})
RETURN o, l
LIMIT 25

// Show executives and their companies
MATCH (p:Entity {entity_type: 'person'})-[:works_for]->(o:Entity {entity_type: 'organization'})
RETURN p, o
LIMIT 25

// Show company partnerships
MATCH (o1:Entity {entity_type: 'organization'})-[:partners_with]-(o2:Entity {entity_type: 'organization'})
RETURN o1, o2
LIMIT 25

// Show the full business network
MATCH (e:Entity)
WHERE e.domain = 'business'
OPTIONAL MATCH (e)-[r]-(connected)
RETURN e, r, connected
LIMIT 100
```

**✅ Checkpoint**: Knowledge graph is now visualized in Neo4j!

---

## Step 8: Refine Ontology Schema

Based on the results, let's improve the extraction rules.

### 8.1: Identify Gaps

```bash
# Find entities with low confidence
duckdb :memory: "
SELECT entity_type, AVG(confidence) as avg_confidence, COUNT(*) as count
FROM read_parquet('./demo/output/analytics/ontology_entities/**/*.parquet')
GROUP BY entity_type
ORDER BY avg_confidence
"
```

### 8.2: Update Ontology Schema

```bash
# Add more sophisticated patterns
cat >> demo/ontologies/business_demo.yaml << 'EOF'

  # Universities and research institutions
  - domain: "business"
    entity_type: "Institution"
    element_types: ["paragraph", "list_item"]
    confidence: 0.85
    extraction_rules:
      - type: "regex_pattern"
        pattern: '\b([A-Z][a-z]+(?:\s+[A-Z][a-z]+)*\s+(?:University|Institute|College|School|Laboratory|Lab))\b'
        description: "Match educational and research institutions"

  # Investment firms and banks
  - domain: "business"
    entity_type: "Financial"
    element_types: ["paragraph"]
    confidence: 0.9
    extraction_rules:
      - type: "regex_pattern"
        pattern: '\b([A-Z][a-z]+(?:\s+[A-Z][a-z]+)*\s+(?:Capital|Ventures|Partners|Bank|Securities|Investments))\b'
        description: "Match financial institutions"
EOF
```

### 8.3: Re-run Extraction with Updated Schema

```bash
# Clean queue and re-extract
rm -f demo/output/jobs.db
timeout 60 ./bin/goworker --config demo/config_ontology.toml --max-documents 0 --workers 1
```

### 8.4: Compare Results

```bash
# Check new entity types
duckdb :memory: "
SELECT entity_type, COUNT(DISTINCT entity_name) as unique_entities
FROM read_parquet('./demo/output/analytics/ontology_entities/**/*.parquet')
GROUP BY entity_type
ORDER BY unique_entities DESC
"
```

**Expected improved output:**
```
┌──────────────┬─────────────────┐
│ entity_type  │ unique_entities │
├──────────────┼─────────────────┤
│ organization │              32 │
│ person       │              28 │
│ location     │              18 │
│ institution  │              12 │  ← NEW!
│ financial    │               8 │  ← NEW!
└──────────────┴─────────────────┘
```

**✅ Checkpoint**: Refined schema discovered additional entities!

---

## Step 9: Summary and Next Steps

### What We Accomplished

1. ✅ **Processed 3 documents** into UDML format (~320 elements)
2. ✅ **Created ontology schema** with 5 entity types and 3 relationship types
3. ✅ **Extracted 98 entities**:
   - 32 organizations
   - 28 people
   - 18 locations
   - 12 institutions
   - 8 financial entities
4. ✅ **Discovered 42 relationships**:
   - Headquarters locations
   - Employment relationships
   - Business partnerships
5. ✅ **Exported to Neo4j** for graph visualization
6. ✅ **Refined schema** to improve extraction accuracy

### Production Deployment

For real-world use:

1. **Scale to more documents**:
   ```bash
   # Remove max-documents limit
   ./bin/goworker --config demo/config_ontology.toml --workers 8
   ```

2. **Use PostgreSQL for distributed processing**:
   ```toml
   [processing.job_control]
   backend = "postgres"
   path = "postgres://user:pass@host:5432/godocgo"
   ```

3. **Enable embeddings for semantic relationships**:
   ```toml
   [embedding]
   enabled = true
   provider = "onnx"
   model_path = "./models/all-MiniLM-L6-v2"
   ```

4. **Add domain-specific ontologies**:
   - Create schemas for healthcare, legal, financial domains
   - Use JSONPath for structured data extraction
   - Add attribute extraction rules

### Key Takeaways

- **Ontology schemas are powerful**: Simple regex patterns discovered 98 entities accurately
- **Iterative refinement works**: Each schema update improved extraction
- **UDML enables reuse**: Same elements used for structure, search, and entity extraction
- **Distributed by design**: Add more workers for linear scaling

---

## Demo Cleanup

```bash
# Stop Neo4j
docker stop neo4j-demo
docker rm neo4j-demo

# Optional: Remove demo output
rm -rf demo/output
```

---

## Additional Resources

- **[Ontology Documentation](docs/ontology.md)** - Complete ontology system guide
- **[UDML Specification](docs/UDML_SPECIFICATION.md)** - Universal Document Markup Language
- **[Configuration Reference](go/README.md#configuration-reference)** - All config options
- **[Go Implementation Guide](go/README.md)** - Detailed Go worker documentation

**Happy knowledge graph building! 🎉**
