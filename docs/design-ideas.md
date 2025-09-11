# Domain Ontology Design Ideas

This document contains aspirational features and settings that could enhance the domain ontology system in the future. These are **not currently implemented** but represent potential areas for improvement.

## Domain-Specific Processing Features

### Financial Markets Domain

#### Specialized Data Extraction
- **`extract_monetary_values`**: Automatically parse and normalize monetary amounts (e.g., "$2.3M", "2.3 million dollars")
- **`extract_percentages`**: Extract and normalize percentage values (e.g., "25%", "twenty-five percent")
- **`extract_dates`**: Parse financial dates and quarters (e.g., "Q4 2024", "fourth quarter")
- **`detect_ticker_symbols`**: Identify stock ticker symbols using patterns and validation against exchanges
- **`normalize_company_names`**: Standardize company name variations (e.g., "Apple Inc.", "Apple", "AAPL" → canonical form)

##### Implementation Note: Normalization Issue
**Problem Identified**: The current entity extraction uses `_slugify()` to normalize entity names, but the matching criteria comparison doesn't normalize the target values. This causes mismatches when comparing values like "CFO" vs "cfo".

**Solution Required**:
1. **Extraction Phase**: Apply specialized extractors that normalize values during extraction
2. **Matching Phase**: Apply same normalization to both source and target values before comparison
3. **Consistency**: Ensure all normalization functions (slugify, lowercase, etc.) are applied consistently in both extraction and matching phases

Example of the issue:
- Entity extraction: `speaker_role: "CFO"` → normalized to `"cfo"` 
- Matching criteria: Compares raw `"CFO"` != normalized `"cfo"` → No match found
- Fix: Normalize both sides: `normalize("CFO")` == `normalize("cfo")` → Match!

#### Advanced Relationship Processing
- **`enable_hierarchical_relationships`**: Use document structure to infer relationship hierarchies
- **`infer_transitive_relationships`**: Derive implied relationships (if A relates to B, and B to C, then A relates to C)
- **`min_confidence_for_inference`**: Minimum confidence threshold for inferring transitive relationships

#### Sentiment and Context Analysis
- **`track_sentiment`**: Analyze sentiment of entity mentions (bullish/bearish for financial terms)
- **`identify_forward_looking_statements`**: Detect forward-looking language and mark as predictive content
- **`detect_risk_factors`**: Identify risk-related language and categorize risk types

### Healthcare Domain (Future)

#### Medical Data Extraction
- **`extract_drug_names`**: Parse pharmaceutical names and generic/brand mappings
- **`extract_dosages`**: Normalize medication dosages and frequencies
- **`extract_medical_codes`**: Parse ICD-10, CPT codes, etc.
- **`detect_contraindications`**: Identify drug interaction warnings

#### Clinical Relationships
- **`infer_treatment_outcomes`**: Connect treatments to patient outcomes
- **`track_temporal_progression`**: Follow disease progression over time
- **`detect_comorbidities`**: Identify related conditions mentioned together

### SEC XBRL Domain (Future)

#### XBRL Document Processing
- **`xbrl_namespace_prefixes`**: Support for standard XBRL namespaces (us-gaap, dei, srt, etc.)
- **`normalize_accounting_terms`**: Standardize accounting terminology variations
- **`link_footnotes`**: Connect footnote references to their content
- **`track_amendments`**: Identify amended or restated values
- **`extract_taxonomy_references`**: Parse XBRL taxonomy element references

#### Financial Statement Analysis
- **`validate_calculations`**: Verify mathematical relationships between XBRL elements
- **`detect_disclosure_patterns`**: Identify standard disclosure formats
- **`track_comparative_periods`**: Link current and prior period values

### Legal Domain (Future)

#### Legal Document Processing
- **`extract_case_citations`**: Parse legal citations in standard formats
- **`extract_statutes`**: Identify statute and regulation references
- **`normalize_court_names`**: Standardize court name variations
- **`detect_legal_precedents`**: Identify when cases cite precedents

#### Contract Analysis
- **`extract_contract_terms`**: Parse key contract elements (parties, dates, amounts)
- **`identify_obligations`**: Detect contractual obligations and responsibilities
- **`track_compliance_requirements`**: Flag regulatory compliance mentions

## Implementation Architecture Ideas

### Pluggable Processors
```yaml
domain:
  name: financial_markets
  processors:
    - name: monetary_extractor
      type: regex_plus_validation
      patterns: ["$[0-9.]+[KMB]?", "\\d+\\.\\d+ (million|billion)"]
      validation: numerical_range_check
    - name: ticker_detector  
      type: api_validation
      patterns: ["[A-Z]{1,5}"]
      api_endpoint: stock_exchange_validator
```

### Configuration-Driven Rules
```yaml
smart_extraction:
  financial_metrics:
    revenue_patterns:
      - "revenue (grew|increased) by {percentage}"
      - "revenue of {amount}"
    confidence_boost: 0.1  # Boost confidence when financial context detected
```

### Dynamic Rule Generation
```yaml
auto_rule_generation:
  enabled: true
  training_documents: 10  # Minimum docs to generate rules
  confidence_threshold: 0.8
  review_required: true  # Human review before activation
```

## Advanced Features

### Cross-Document Intelligence
- **Entity Disambiguation**: Resolve when "Apple" refers to the company vs fruit across documents
- **Temporal Tracking**: Track how entity relationships change over time
- **Comparative Analysis**: Automatically compare similar entities across documents

### Machine Learning Integration
- **Embedding-Based Similarity**: Use document embeddings for better relationship discovery
- **Active Learning**: Suggest new rules based on extraction patterns
- **Confidence Calibration**: ML-based confidence scoring for extractions

### Performance Optimizations
- **Incremental Processing**: Only reprocess changed sections of documents
- **Rule Caching**: Cache compiled regex patterns and validation results
- **Batch Processing**: Process similar documents together for efficiency

## Implementation Priority (Estimated ROI)

### High Priority (Easy wins)
1. **Monetary value extraction** - High utility, relatively simple regex patterns
2. **Percentage extraction** - Similar to monetary, common in financial docs
3. **Ticker symbol detection** - Can validate against known exchange lists

### Medium Priority 
4. **Date extraction** - Complex due to various formats, but very useful
5. **Company name normalization** - Requires entity resolution logic
6. **Sentiment tracking** - Needs sentiment analysis integration

### Lower Priority (Complex features)
7. **Transitive relationship inference** - Requires graph algorithms and careful validation
8. **Forward-looking statement detection** - Complex NLP task
9. **Hierarchical relationships** - Needs document structure analysis

## Migration Strategy

When implementing these features:

1. **Incremental Development**: Implement one feature at a time with full testing
2. **Backward Compatibility**: Ensure existing ontologies continue working
3. **Feature Flags**: Use configuration to enable/disable new features
4. **Validation**: Add comprehensive tests for each new extraction type
5. **Documentation**: Update examples and guides as features are added

## Additional Design Ideas from Development

### Ontology Enhancements

#### Variables and Constants
**Problem**: Massive repetition in ontology definitions
**Solution**: Support for variables/constants in YAML (beyond anchors)

```yaml
variables:
  ${COMPANY_TERMS}: "company, corporation, firm, enterprise"
  ${HIGH_CONFIDENCE}: 0.75
  
rules:
  - semantic_phrase: "${COMPANY_TERMS}, we, our"
    confidence: ${HIGH_CONFIDENCE}
```

#### Template-Based Rule Generation
**Problem**: Similar rules with minor variations
**Solution**: Rule templates with parameter substitution

```yaml
templates:
  executive_rule:
    params: [role, title]
    template:
      term_id: "${role}"
      semantic_phrase: "${title}, chief ${role} officer"
      
generated_rules:
  - from_template: executive_rule
    params: {role: "technology", title: "CTO"}
  - from_template: executive_rule
    params: {role: "marketing", title: "CMO"}
```

#### Aliases and Macros
**Problem**: Common pattern combinations repeated
**Solution**: Named pattern macros

```yaml
macros:
  financial_amount: "\\$?\\d+\\.?\\d*\\s*(million|billion|M|B)"
  percentage: "\\d+\\.?\\d*\\s*%"
  
rules:
  - pattern: "{financial_amount}\\s+in\\s+revenue"
  - pattern: "{percentage}\\s+growth"
```

### Storage Provider Features

#### Entity CRUD Operations
**Status**: Fully implemented across providers (update_entity methods exist)
**Note**: Incremental entity updates are already implemented in domain.py

#### Entity Relationship Management
**Problem**: Relationships stored separately from entities
**Solution**: Graph-native entity storage

```python
class GraphEntity:
    def __init__(self, entity_id, entity_type):
        self.id = entity_id
        self.type = entity_type
        self.attributes = {}
        self.relationships = []  # Embedded relationships
        
    def add_relationship(self, target_id, rel_type):
        self.relationships.append({
            'target': target_id,
            'type': rel_type,
            'created_at': datetime.now()
        })
```

### Critical Bugs & Storage Architecture

#### PostgreSQL AUTOCOMMIT Issue ✅ FIXED
**Status**: ~~Critical bug in postgres.py (line 971)~~ **FIXED on 2025-09-11**
**Problem**: Used `ISOLATION_LEVEL_AUTOCOMMIT` which completely disabled transactions
**Impact**:
- ~~No atomicity - partial document storage can fail halfway through~~ ✅ Fixed
- ~~Fake transaction boundaries - `commit()`/`rollback()` calls do nothing~~ ✅ Fixed  
- ~~Race conditions with multiple instances accessing same data~~ ✅ Fixed
- ~~Data corruption on any failure during multi-table operations~~ ✅ Fixed

**Solution Implemented**:
```python
# OLD (BROKEN):
self.conn.set_isolation_level(ISOLATION_LEVEL_AUTOCOMMIT)

# NEW (FIXED):
# Normal operations use READ_COMMITTED for proper transactions
self.conn.set_isolation_level(psycopg2.extensions.ISOLATION_LEVEL_READ_COMMITTED)

# Only temporarily switch to AUTOCOMMIT for CREATE EXTENSION
if need_create_extension:
    self.conn.set_isolation_level(ISOLATION_LEVEL_AUTOCOMMIT)
    self.cursor.execute("CREATE EXTENSION vector")
    # Immediately restore transaction mode
    self.conn.set_isolation_level(psycopg2.extensions.ISOLATION_LEVEL_READ_COMMITTED)
```

**Test Results**: All transaction tests pass - rollback and commit work correctly.

#### Storage Concurrency Analysis
**Current State**: Minimal concurrency control across storage providers
- **SQLite**: No WAL mode enabled, database-level locking blocks readers during writes
- **PostgreSQL**: Broken due to AUTOCOMMIT mode (see above)
- **File storage**: Pure in-memory dictionaries, no locking mechanisms

**Recommendations for Multi-Instance Access**:
1. **Production**: "Just use PostgreSQL" (after fixing transaction bug)
   - Native MVCC handles concurrent access correctly
   - Row-level locking prevents conflicts
   - Connection pooling (PgBouncer) handles multiple instances
2. **Development/Single Instance**: SQLite is sufficient
   - Consider enabling WAL mode for better concurrency

### Append-Only Storage Pattern

#### The Universal Pattern
**Concept**: Instead of updating records, only append new versions with timestamps. Deletes become tombstone records.

**Benefits**:
- **Time Travel**: Query any historical state for free
- **Audit Trail**: Complete history of all changes
- **No Conflicts**: Append-only means no update conflicts
- **Works Anywhere**: S3, flat files, any database
- **Recovery**: Can replay from any point in time

#### Implementation Pattern
```python
# Write operations
def write(key, value, op='UPDATE'):
    append_to_log({
        'key': key,
        'value': value,
        'timestamp': now(),
        'op': op  # UPDATE or DELETE
    })

# Current state query
def get_current_state(records):
    # Group by key, take latest timestamp
    latest_by_key = {}
    for record in records:
        key = record['key']
        if key not in latest_by_key or record['timestamp'] > latest_by_key[key]['timestamp']:
            latest_by_key[key] = record
    
    # Remove deletes (tombstones)
    return {k: v for k, v in latest_by_key.items() if v['op'] != 'DELETE'}

# Time travel query
def get_state_at_time(records, timestamp):
    # Filter records before timestamp, then get current state
    historical = [r for r in records if r['timestamp'] <= timestamp]
    return get_current_state(historical)
```

#### Storage-Specific Implementations

**PostgreSQL/SQLite**:
```sql
-- Append-only table
CREATE TABLE changes (
    key TEXT,
    value JSONB,
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    op TEXT CHECK (op IN ('UPDATE', 'DELETE')),
    PRIMARY KEY (key, timestamp)
);

-- Current state view
CREATE VIEW current_state AS
WITH latest AS (
    SELECT DISTINCT ON (key) *
    FROM changes
    ORDER BY key, timestamp DESC
)
SELECT * FROM latest WHERE op != 'DELETE';
```

**S3/Parquet**:
```python
# Append changes to dated files
/data/2024/01/01/changes_0001.parquet
/data/2024/01/01/changes_0002.parquet

# Read current state with Spark/Pandas
df = read_parquet("/data/*/*/*.parquet")
current = df.groupby("key").last()
current = current[current.op != 'DELETE']
```

#### The Neo4j Exception
**Neo4j should use traditional mutable CRUD** because:
- Graph databases optimize for relationship traversal, not time series
- Version chains create relationship explosion
- Temporal queries fight the graph model

**Hybrid Approach**:
```python
class StorageRouter:
    def __init__(self):
        self.postgres = PostgreSQLAppendOnly()  # Source of truth with history
        self.neo4j = Neo4jMutable()            # Derived current state
        
    def write(self, doc):
        # Append to PostgreSQL (source of truth)
        self.postgres.append(doc)
        
        # Update Neo4j (current state only)
        if doc['op'] == 'DELETE':
            self.neo4j.delete(doc['key'])
        else:
            self.neo4j.upsert(doc['key'], doc['value'])
```

#### When to Use Append-Only

**Use Append-Only For**:
- Audit requirements (compliance, legal)
- Time-travel queries needed
- High-concurrency writes
- Immutable storage (S3, HDFS)
- Event sourcing architectures

**Use Mutable CRUD For**:
- Graph databases (Neo4j)
- Search indices (Elasticsearch/Solr - they're derived anyway)
- High-frequency updates with no history need
- Storage-constrained environments

#### Compaction Strategy
```python
def compact_log(records, keep_history_days=30):
    cutoff = datetime.now() - timedelta(days=keep_history_days)
    current = get_current_state(records)
    
    compacted = []
    for record in records:
        if record['timestamp'] > cutoff:  # Keep recent history
            compacted.append(record)
        elif record['key'] in current and current[record['key']] == record:
            compacted.append(record)  # Keep current state
        # Drop old non-current records
    
    return compacted
```

### AI Agent Temporal Query Abstraction

#### The Pragmatic Realization
**Why build complex temporal storage when AI agents can handle the complexity?**

The data is already there in our append-only PostgreSQL or other storage backends. Instead of building elaborate temporal Neo4j models with version chains and state nodes, an AI agent can:

1. **Abstract away temporal SQL complexity** - Users ask natural language questions, AI handles the nasty SQL
2. **Reconstruct point-in-time graphs on demand** - No need to store all historical graph states
3. **Provide intuitive interfaces** - "Show me the knowledge graph from last quarter" becomes simple

#### The Problem with Temporal SQL
Even with PostgreSQL's temporal extensions, time-travel queries are painful:

```sql
-- "Show me entities as of January 1st" becomes:
WITH point_in_time AS (
  SELECT '2024-01-01'::timestamp AS target_time
),
valid_entities AS (
  SELECT DISTINCT ON (entity_id) *
  FROM entities e, point_in_time p
  WHERE e.created_at <= p.target_time
    AND (e.deleted_at IS NULL OR e.deleted_at > p.target_time)
  ORDER BY entity_id, updated_at DESC
)
SELECT * FROM valid_entities;

-- "Find relationships that existed between these dates" is even worse:
SELECT * FROM relationships r
WHERE r.created_at <= '2024-01-01'
  AND (r.deleted_at IS NULL OR r.deleted_at > '2024-01-01')
  AND r.source_id NOT IN (
    SELECT entity_id FROM entity_deletions 
    WHERE deleted_at <= '2024-01-01'
  )
  AND r.target_id NOT IN (
    SELECT entity_id FROM entity_deletions 
    WHERE deleted_at <= '2024-01-01'
  );
```

#### The AI Agent Solution

```python
class TemporalQueryAgent:
    """AI agent that handles temporal complexity transparently."""
    
    def query(self, natural_language_query: str) -> Any:
        # User asks: "What did TechCorp's relationships look like in Q3 2023?"
        
        # Agent:
        # 1. Parses temporal intent (Q3 2023 = July 1 - Sept 30, 2023)
        # 2. Generates appropriate SQL for each storage backend
        # 3. Assembles results into coherent response
        # 4. Returns clean graph visualization or summary
        
        time_range = self.extract_temporal_context(natural_language_query)
        entities = self.get_entities_at_time(time_range)
        relationships = self.get_relationships_at_time(time_range)
        
        if user_wants_visualization:
            return self.create_graph_visualization(entities, relationships)
        else:
            return self.generate_summary(entities, relationships)
    
    def extract_temporal_context(self, query: str) -> TimeRange:
        # LLM parses: "last quarter", "6 months ago", "between Jan and March"
        # Returns: Precise datetime ranges
        pass
    
    def handle_complex_temporal_sql(self, time_range: TimeRange) -> str:
        # Agent generates the nasty SQL so users don't have to
        # Handles all the edge cases, deletions, tombstones, etc.
        pass
```

#### Benefits of Agent-Based Approach

1. **No new infrastructure** - Use existing storage as-is
2. **Flexible query patterns** - Agent adapts to any temporal question
3. **Progressive enhancement** - Start simple, add capabilities over time
4. **Storage agnostic** - Works with PostgreSQL, SQLite, even flat files
5. **Natural language interface** - No need to learn temporal SQL

#### Example User Interactions

```
User: "Show me all entities that were related to OpenAI in 2019"
Agent: [Queries PostgreSQL, reconstructs 2019 graph, returns results]

User: "How did the executive team change over the last year?"
Agent: [Identifies executive entities, tracks changes, creates timeline]

User: "Find any relationships that existed temporarily but were later removed"
Agent: [Complex temporal join query, identifies transient relationships]

User: "Compare the knowledge graph between product launch and now"
Agent: [Two point-in-time queries, diff computation, change summary]
```

#### Implementation Simplicity

Instead of the complex Neo4j temporal patterns, just:

```python
class TemporalStorage:
    def __init__(self, postgres_conn):
        self.db = postgres_conn
        self.agent = TemporalQueryAgent()
    
    def query(self, natural_language: str):
        # Agent handles all complexity
        return self.agent.query(natural_language)
    
    def get_state_at(self, timestamp):
        # Simple SQL that agent generates
        sql = self.agent.generate_point_in_time_sql(timestamp)
        return self.db.execute(sql)
```

#### When to Build vs When to Use AI

**Build temporal infrastructure when:**
- Microsecond query latency required
- Regulatory requirement for immutable audit logs
- Need to support thousands of concurrent temporal queries

**Use AI agent abstraction when:**
- Temporal queries are occasional
- Users prefer natural language
- Flexibility more important than performance
- Want to avoid maintenance burden

#### The Bottom Line

We already have the data. We already have AI. Why build complex temporal graph databases when an AI agent can reconstruct any historical state on demand from simple append-only storage?

This transforms the time-traveling knowledge graph from a **storage architecture challenge** into a **query interface opportunity** - much simpler, more maintainable, and users get natural language temporal queries instead of learning complex graph query languages.

### Time-Traveling Knowledge Graph (Original Complex Approach)

*Note: Preserved for reference, but see "AI Agent Temporal Query Abstraction" above for the pragmatic approach*

#### The Vision
**A Neo4j-based knowledge graph with full temporal capabilities** - query any state at any point in time, track relationship evolution, and perform temporal graph analytics.

#### Implementation Patterns

**1. Versioned Nodes and Relationships**
```cypher
// Every entity has version nodes
(:Entity {id: 'OpenAI'})-[:HAS_VERSION]->(:Version {
    valid_from: datetime('2015-12-11'),
    valid_to: datetime('9999-12-31'),
    properties: {name: 'OpenAI', type: 'company', status: 'non-profit'}
})

// Relationships also versioned
(:Entity {id: 'OpenAI'})-[:EMPLOYED {
    valid_from: datetime('2019-03-11'),
    valid_to: datetime('2022-11-17'),
    role: 'CEO'
}]->(:Entity {id: 'Sam Altman'})
```

**2. State Node Pattern**
```cypher
// Current state on main node, history in state nodes
(:Element {id: 'elem_123', current_content: 'latest'})-[:HAS_STATE]->
    (:State {content: 'v1', valid_from: datetime('2024-01-01'), valid_to: datetime('2024-01-15')})
(:Element {id: 'elem_123'})-[:HAS_STATE]->
    (:State {content: 'v2', valid_from: datetime('2024-01-15'), valid_to: datetime('9999-12-31')})
```

**3. Temporal Views**
```cypher
// Hide complexity behind views/procedures
CALL apoc.custom.asProcedure(
  'graph_at_time',
  'WITH $timestamp AS t
   MATCH (n:Entity)-[:HAS_VERSION]->(v:Version)
   WHERE v.valid_from <= t < v.valid_to
   MATCH (n)-[r]-(m:Entity)
   WHERE r.valid_from <= t < r.valid_to
   RETURN n, r, m',
  'READ'
)

// Simple time-travel queries
CALL graph_at_time('2023-01-01') YIELD n, r, m
RETURN n, r, m
```

#### Killer Use Cases

**1. Compliance & Forensics**
```cypher
// "Show all relationships at time of incident"
CALL time.travel($incident_date)
MATCH (person:Person)-[r*..3]-(entity:Entity)
WHERE ALL(rel IN r WHERE 
  rel.valid_from <= $incident_date < rel.valid_to)
RETURN person, r, entity
```

**2. Knowledge Evolution**
```cypher
// "How did our understanding evolve?"
MATCH (concept:Concept {name: 'COVID-19'})
WITH concept
MATCH (concept)<-[:MENTIONS]-(doc:Document)
RETURN doc.date, COUNT(doc) AS mentions, 
       AVG(doc.confidence) AS understanding_level
ORDER BY doc.date
```

**3. Relationship Archaeology**
```cypher
// "Find hidden connections that existed in the past"
MATCH (e1:Entity {id: $entity1}), (e2:Entity {id: $entity2})
MATCH path = (e1)-[rels*]-(e2)
WHERE ALL(r IN rels WHERE r.valid_from <= $date < r.valid_to)
RETURN path, 
       [r IN rels | r.valid_from] AS connection_dates
```

**4. Anomaly Detection**
```cypher
// "Detect unusual relationship patterns over time"
MATCH (e:Entity)-[r:TRANSACTED_WITH]-(other:Entity)
WITH e, COUNT(DISTINCT other) AS connections_at_time, 
     date(r.valid_from) AS date
RETURN e, date, connections_at_time,
       stdev(connections_at_time) OVER (PARTITION BY e ORDER BY date 
                                        ROWS BETWEEN 30 PRECEDING AND CURRENT ROW) AS volatility
WHERE volatility > 2  // Anomaly threshold
```

#### Hybrid Architecture

```python
class TemporalKnowledgeGraph:
    def __init__(self):
        self.neo4j = Neo4j()  # Graph with versioned nodes
        self.postgres = PostgreSQL()  # Complete change log
        
    def write(self, subject, predicate, object, op='CREATE'):
        # Log to PostgreSQL (source of truth)
        self.postgres.append({
            'subject': subject,
            'predicate': predicate,
            'object': object,
            'timestamp': now(),
            'op': op
        })
        
        # Update Neo4j versioned graph
        if op == 'CREATE':
            self.create_versioned_triple(subject, predicate, object)
        elif op == 'DELETE':
            self.end_version(subject, predicate, object)
            
    def time_travel_query(self, cypher, timestamp):
        # Rewrite query with temporal filters
        return self.neo4j.query(
            self.add_temporal_filters(cypher, timestamp)
        )
        
    def get_evolution(self, entity_id):
        # Get complete history from PostgreSQL
        return self.postgres.query(
            "SELECT * FROM changes WHERE subject = ? OR object = ? ORDER BY timestamp",
            [entity_id, entity_id]
        )
```

#### Technical Challenges & Solutions

**Challenge**: Query performance with temporal filters
**Solution**: Composite indexes on (id, valid_from, valid_to), materialized current-state views

**Challenge**: Relationship versioning creates edge explosion
**Solution**: Use relationship properties for versioning, create virtual edges in queries

**Challenge**: Complex temporal queries
**Solution**: Build Cypher query generators, provide high-level temporal operators

#### Why It's Worth The Complexity

1. **Regulatory Compliance** - Prove exact state at any point
2. **Forensic Analysis** - Reconstruct knowledge at time of event
3. **Causal Reasoning** - Track how knowledge led to decisions
4. **Trust Evolution** - See how trust networks changed
5. **Knowledge Debugging** - Find when incorrect information entered the system
6. **Temporal Analytics** - Trend analysis on graph structure evolution

#### Implementation Phases

**Phase 1**: Add version nodes to existing Neo4j schema
**Phase 2**: Build temporal view layer with APOC procedures
**Phase 3**: Create time-travel query interface
**Phase 4**: Add temporal analytics capabilities
**Phase 5**: Build visualization for knowledge evolution

### Testing Improvements

#### Known Output Testing
**Problem**: Tests validating implementation instead of design
**Solution**: Test against known files with expected outputs

```python
class TestKnownOutputs:
    def test_financial_document(self):
        # Use real document with known entities
        doc = load_test_document('earnings_call_q4_2024.txt')
        entities = extract_entities(doc)
        
        # Validate against expected entities
        expected = load_expected_output('earnings_call_q4_2024_entities.json')
        assert entities == expected
```

#### Performance Benchmarking
**Problem**: No performance regression detection
**Solution**: Automated performance tests

```python
@pytest.mark.benchmark
def test_entity_extraction_performance(benchmark):
    doc = load_large_document()
    result = benchmark(extract_entities, doc)
    assert benchmark.stats['mean'] < 1.0  # Must complete in < 1 second
```

### LLM Integration

#### Ontology Generation
**Concept**: Use LLMs to generate domain ontologies from sample documents

```python
class LLMOntologyGenerator:
    def generate_from_samples(self, documents, domain):
        prompt = f"Generate a {domain} ontology from these documents..."
        
        # Extract patterns and entities
        entities = self.llm.extract_entities(documents)
        relationships = self.llm.discover_relationships(entities)
        
        # Generate YAML ontology
        return self.build_ontology(entities, relationships)
```

#### Rule Refinement
**Concept**: Use LLM to refine extraction rules based on failures

```python
class RuleRefinementAgent:
    def refine_rule(self, rule, failed_examples, successful_examples):
        prompt = f"""
        Current rule: {rule}
        Failed on: {failed_examples}
        Succeeded on: {successful_examples}
        
        Suggest improved rule:
        """
        return self.llm.complete(prompt)
```

### MCP (Model Context Protocol) Integration

#### Expose Go-Doc-Go as MCP Server
**Transform the document processing system into an MCP-compatible tool that AI assistants can use**

The Model Context Protocol enables AI assistants to interact with external systems through a standardized interface. By exposing Go-Doc-Go as an MCP server, any MCP-compatible AI (Claude, etc.) can:

1. **Query the knowledge graph directly**
2. **Trigger document ingestion pipelines**
3. **Access entity and relationship data**
4. **Perform temporal queries on historical data**

#### MCP Server Implementation

```python
from mcp.server import MCPServer, Tool, Resource

class GoDocGoMCPServer(MCPServer):
    """MCP server exposing Go-Doc-Go capabilities."""
    
    def __init__(self, config_path: str):
        super().__init__("go-doc-go")
        self.go_doc_go = GoDocGoSystem(config_path)
        self.register_tools()
        self.register_resources()
    
    def register_tools(self):
        """Register available tools for AI assistants."""
        
        @self.tool("ingest_document")
        async def ingest_document(uri: str, doc_type: str = "auto") -> dict:
            """Ingest a document from URI into the knowledge graph."""
            return await self.go_doc_go.ingest(uri, doc_type)
        
        @self.tool("query_entities")
        async def query_entities(
            entity_type: str = None,
            domain: str = None,
            time_range: str = None
        ) -> list:
            """Query entities from the knowledge graph."""
            return await self.go_doc_go.query_entities(
                entity_type, domain, time_range
            )
        
        @self.tool("find_relationships")
        async def find_relationships(
            source_entity: str = None,
            target_entity: str = None,
            relationship_type: str = None
        ) -> list:
            """Find relationships between entities."""
            return await self.go_doc_go.find_relationships(
                source_entity, target_entity, relationship_type
            )
        
        @self.tool("temporal_query")
        async def temporal_query(
            natural_language: str
        ) -> dict:
            """Execute temporal query in natural language."""
            return await self.go_doc_go.temporal_agent.query(natural_language)
    
    def register_resources(self):
        """Register data resources accessible to AI."""
        
        @self.resource("knowledge_graph")
        async def get_graph_schema():
            """Returns the current knowledge graph schema."""
            return {
                "entity_types": self.go_doc_go.get_entity_types(),
                "relationship_types": self.go_doc_go.get_relationship_types(),
                "domains": self.go_doc_go.get_active_domains()
            }
        
        @self.resource("document_stats")
        async def get_stats():
            """Returns processing statistics."""
            return self.go_doc_go.get_statistics()
```

#### MCP Client Configuration

```json
{
  "mcpServers": {
    "go-doc-go": {
      "command": "python",
      "args": ["-m", "go_doc_go.mcp_server", "--config", "config.yaml"],
      "env": {
        "GO_DOC_GO_CONFIG": "/path/to/config.yaml"
      }
    }
  }
}
```

### Headless Claude Code for Agentic AI Planning

#### The Vision
**A headless version of Claude Code that can be orchestrated by AI agents for complex document processing workflows**

Instead of interactive CLI usage, create a programmatic API that AI agents can use to plan and execute sophisticated document analysis pipelines.

#### Headless Architecture

```python
class HeadlessClaudeCode:
    """Headless Claude Code for AI agent orchestration."""
    
    def __init__(self, workspace_dir: str):
        self.workspace = workspace_dir
        self.planner = DocumentProcessingPlanner()
        self.executor = PipelineExecutor()
        self.monitor = ProgressMonitor()
    
    async def plan_and_execute(self, objective: str) -> dict:
        """
        AI agent provides high-level objective, system plans and executes.
        
        Example objective: 
        "Analyze all Q4 earnings calls, extract financial metrics, 
         build knowledge graph, and identify trend anomalies"
        """
        
        # 1. Planning Phase - AI breaks down objective
        plan = await self.planner.create_plan(objective)
        # Returns structured plan:
        # {
        #   "stages": [
        #     {"id": "ingest", "type": "document_ingestion", 
        #      "sources": ["s3://earnings/q4/*.pdf"]},
        #     {"id": "extract", "type": "entity_extraction",
        #      "domain": "financial_markets"},
        #     {"id": "graph", "type": "build_knowledge_graph"},
        #     {"id": "analyze", "type": "anomaly_detection",
        #      "metrics": ["revenue", "guidance"]}
        #   ],
        #   "dependencies": {"extract": ["ingest"], "graph": ["extract"]}
        # }
        
        # 2. Validation - Check feasibility
        validation = await self.validate_plan(plan)
        if not validation.is_valid:
            return {"error": validation.issues}
        
        # 3. Execution - Run pipeline with monitoring
        execution_id = await self.executor.start(plan)
        
        # 4. Progress tracking
        async for progress in self.monitor.track(execution_id):
            yield {
                "stage": progress.current_stage,
                "progress": progress.percentage,
                "status": progress.status
            }
        
        # 5. Results compilation
        results = await self.executor.get_results(execution_id)
        return {
            "execution_id": execution_id,
            "plan": plan,
            "results": results,
            "artifacts": self.get_artifacts(execution_id)
        }
    
    async def interactive_planning(self, agent_interface):
        """
        Allow AI agent to iteratively refine plans.
        """
        while True:
            feedback = await agent_interface.get_feedback()
            if feedback.type == "modify_plan":
                self.planner.modify(feedback.modifications)
            elif feedback.type == "add_stage":
                self.planner.add_stage(feedback.stage)
            elif feedback.type == "execute":
                return await self.executor.start(self.planner.current_plan)
```

#### AI Agent Integration Pattern

```python
class DocumentAnalysisAgent:
    """AI agent that orchestrates headless Claude Code."""
    
    def __init__(self):
        self.headless_cc = HeadlessClaudeCode("/workspace")
        self.knowledge_base = KnowledgeBase()
    
    async def handle_request(self, user_request: str):
        """
        User: "Compare our competitor earnings with ours and find gaps"
        """
        
        # 1. Understand intent
        intent = self.analyze_intent(user_request)
        
        # 2. Create execution plan
        plan = f"""
        Objective: {intent.objective}
        Steps:
        1. Ingest competitor earnings from {intent.competitor_sources}
        2. Ingest our earnings from {intent.our_sources}
        3. Extract financial entities using financial_markets domain
        4. Build comparative knowledge graph
        5. Identify gaps and opportunities
        6. Generate executive summary
        """
        
        # 3. Execute via headless Claude Code
        async for progress in self.headless_cc.plan_and_execute(plan):
            # Update user on progress
            await self.update_user(progress)
        
        # 4. Interpret results
        results = await self.headless_cc.get_results()
        summary = self.generate_summary(results)
        
        return summary
```

#### Use Cases for Headless Mode

1. **Batch Processing Pipelines**
   - Nightly document ingestion
   - Scheduled knowledge graph updates
   - Automated report generation

2. **Multi-Agent Workflows**
   - Research agent finds documents
   - Analysis agent processes them via headless CC
   - Synthesis agent creates reports

3. **CI/CD Integration**
   - Validate document schemas in PR checks
   - Test entity extraction on new ontologies
   - Performance regression testing

4. **API-First Architecture**
   ```python
   # REST API wrapping headless Claude Code
   @app.post("/api/process")
   async def process_documents(request: ProcessRequest):
       result = await headless_cc.plan_and_execute(request.objective)
       return {"job_id": result.execution_id}
   
   @app.get("/api/status/{job_id}")
   async def get_status(job_id: str):
       return await headless_cc.monitor.get_status(job_id)
   ```

#### Benefits of Headless Operation

1. **Scalability** - Run multiple instances in parallel
2. **Automation** - No human interaction required
3. **Integration** - Fits into existing MLOps pipelines
4. **Flexibility** - AI agents can adapt plans dynamically
5. **Observability** - Full tracking of all operations

#### Implementation Roadmap

**Phase 1: Core API**
- Extract CLI logic into programmatic API
- Add async/await support throughout
- Create job management system

**Phase 2: Planning Engine**
- Build plan DSL (Domain Specific Language)
- Implement plan validation
- Add dependency resolution

**Phase 3: MCP Integration**
- Implement MCP server protocol
- Add tool and resource definitions
- Create example MCP clients

**Phase 4: Agent Framework**
- Build agent orchestration layer
- Add feedback loops
- Implement learning from execution history

**Phase 5: Production Features**
- Add authentication/authorization
- Implement rate limiting
- Build monitoring dashboard

## Updated Implementation Priority (ROI Analysis)

### Immediate ROI (Already Implemented)
- **YAML Anchors** - ∞ ROI (zero cost, 49% reduction achieved)
- **Incremental Entity Updates** - Already implemented in domain.py

### Critical Bugs (Must fix immediately)
1. ~~**PostgreSQL Transaction Fix** - CRITICAL BUG (1 day, enables safe concurrency, prevents data corruption)~~ ✅ **FIXED 2025-09-11**

### Very High ROI (Days to implement, major impact)
2. **Normalization Fix** - Critical bug fix, enables entity matching (2 days, fixes core functionality)
3. **Known Output Testing** - Improves quality, reduces bugs (3 days, prevents regressions)
4. **Monetary Value Extraction** - High demand feature (3 days, immediate value)
5. **Percentage Extraction** - Common in financial docs (2 days, pairs with monetary)

### High ROI (Week to implement, significant value)
6. **Append-Only Storage Pattern** - Time travel + audit trail (3-5 days, solves multiple problems)
7. **SQLite WAL Mode** - Simple concurrency improvement (1 day, better dev experience)
8. **Template-Based Rules** - Reduces ontology maintenance (4 days, developer productivity)
9. **Ticker Symbol Detection** - Financial markets focus (3 days, high precision possible)
10. **Date Extraction** - Universal need (5 days, complex but valuable)

### Medium ROI (Weeks to implement, moderate value)
11. **AI Agent Temporal Query Abstraction** - Natural language time travel (5-7 days, high user value, low complexity)
12. **MCP Server Integration** - AI assistant integration (7-10 days, enables Claude/other AI tools)
13. **Company Name Normalization** - Entity resolution (7 days, improves accuracy)
14. **Variables/Constants in YAML** - Beyond anchors (5 days, nice to have)
15. **Performance Benchmarking** - Regression detection (4 days, quality improvement)
16. **Sentiment Tracking** - Market analysis (7 days, requires ML integration)

### Lower ROI (Complex implementation, specialized value)
17. **Headless Claude Code** - Full agentic orchestration (20-30 days, complex architecture, niche use cases)
18. **LLM Ontology Generation** - Experimental (10+ days, requires LLM infrastructure)
19. **Transitive Relationships** - Graph algorithms (7 days, limited use cases)
20. **Rule Refinement Agent** - Advanced AI (14+ days, requires feedback loop)
21. **Forward-Looking Statements** - Complex NLP (10+ days, regulatory focus)

### ROI Calculation Factors
- **Implementation Time**: Days of developer effort
- **Impact**: Number of users/use cases affected
- **Maintenance**: Ongoing support requirements
- **Dependencies**: External systems or libraries needed
- **Risk**: Probability of successful implementation

## Notes

- All features in this document are **aspirational** and not currently implemented
- Remove any non-functional settings from ontology files to avoid confusion
- Use this document to guide future development priorities
- Consider user feedback and real-world usage patterns when prioritizing features
- **YAML anchors** proved to be the highest ROI improvement (zero cost, immediate benefit)