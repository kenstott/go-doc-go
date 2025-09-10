# The Go-Doc-Go Knowledge Engine

## An extensible, declarative, unstructured text data pipeline for building intelligent search and knowledge graphs

The Go-Doc-Go Knowledge Engine is an advanced document processing and knowledge extraction system that transforms unstructured text from various sources into intelligent, searchable knowledge graphs through declarative configuration and domain-specific entity extraction, while maintaining efficient pointers to original content.

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│ Content Sourcer │     │Document Ingester│     │ Storage Manager │     │Domain Ontologist│
└────────┬────────┘     └────────┬────────┘     └────────┬────────┘     └────────┬────────┘
         │                       │                       │                       │
┌────────┼────────┐     ┌────────┼────────┐     ┌────────┼────────┐     ┌────────┼────────┐
│ Confluence API  │     │Parser Adapters  │     │SQLite Backend  │     │Entity Extraction│
│ SharePoint Docs │◄───►│Structure Extract│◄───►│MongoDB Backend │◄───►│Semantic Mapping │
│ S3 Buckets      │     │Embedding Gen    │     │Vector Database │     │Relationship Det │
│ Local Files     │     │Relationship Map │     │Elasticsearch   │     │Business Rules   │
└─────────────────┘     └─────────────────┘     └─────────────────┘     └─────────────────┘

Flow: Sources → Ingestion → Storage ← Domain Analysis (reads from & writes to storage)
```

## Key Features

### 🔧 Extensible & Declarative Pipeline Architecture
- **📝 Configuration-Driven Processing**: Define entire data pipelines through YAML/JSON configuration without coding
- **🧩 Pluggable Components**: Modular content sources, parsers, storage backends, and embedding providers
- **🎛️ Declarative Rules Engine**: Configure entity extraction and relationship detection through business rules
- **🔌 Backend-Agnostic Design**: Automatic capability detection and optimization across different storage systems

### 🧠 Intelligent Knowledge Graph Construction  
- **🎯 Domain Entity Extraction & Relationship Detection**: Apply business ontologies to automatically extract domain-specific entities (customers, products, regulations) and discover semantic relationships between them using configurable rules (semantic similarity, regex, keywords)
- **🌐 Cross-Document Intelligence**: Discover relationships spanning multiple documents and sources for comprehensive knowledge graphs
- **📊 Confidence Scoring & Optimization**: Configurable thresholds for precision vs. recall to match your quality requirements
- **🔗 Semantic Relationship Discovery**: Automatically finds connections between entities based on declarative business rules

### 📄 Advanced Document Processing Pipeline
- **🏗️ Universal Document Model**: Common structured representation across all document types and formats
- **🎯 Element-Level Precision**: Maintains granular accuracy to specific document elements with hierarchical relationships
- **📋 Document Materialization**: Comprehensive reconstruction and format conversion with intelligent element mapping
- **⚡ Batch Processing**: Efficient bulk document retrieval and format conversion for performance optimization

### 🔍 Intelligent Search & Discovery
- **🧭 Contextual Semantic Search**: Advanced embedding techniques incorporating document context (hierarchy, neighbors)
- **🔎 Advanced Structured Search**: Powerful query language with logical operators, similarity thresholds, and backend optimization
- **🏷️ Topic-Aware Organization**: Categorize and filter content by topics for enhanced organization and discovery
- **📈 Pattern Matching & Filtering**: Advanced element type filtering, metadata search, and regex pattern support

### ⚙️ Enterprise-Ready Configuration
- **🎛️ Flexible Full-Text Storage**: Declaratively configure text storage and indexing for optimal performance and efficiency
- **📊 Configurable Vector Representations**: Support different vector dimensions based on content needs
- **🔧 Modular Dependencies**: Only install components you need, with graceful fallbacks when dependencies are missing
- **📊 Document Analytics**: Rich statistics, outlines, and structural analysis capabilities

## Full-Text Storage and Search Configuration

Go-Doc-Go provides flexible full-text storage and indexing options that can be configured independently to optimize for your specific use case:

### Storage Configuration Options

```yaml
storage:
  backend: elasticsearch
  
  # Full-text storage and indexing options
  store_full_text: true        # Whether to store full text for retrieval (default: true)
  index_full_text: true        # Whether to index full text for search (default: true)
  compress_full_text: false    # Whether to enable compression for stored text (default: false)
  full_text_max_length: null   # Maximum length for full text, truncate if longer (default: null)
```

### Common Configuration Patterns

#### 1. Search + Storage (Default - Best Search Quality)
```yaml
storage:
  store_full_text: true
  index_full_text: true
  # Best for: Complete search capabilities with full content retrieval
  # Storage impact: High (stores and indexes full text)
```

#### 2. Search Only (Space Optimized)
```yaml
storage:
  store_full_text: false
  index_full_text: true
  # Best for: Search-focused applications where content retrieval isn't needed
  # Storage impact: Medium (indexes but doesn't store full text)
```

#### 3. Storage Only (Retrieval Focused)
```yaml
storage:
  store_full_text: true
  index_full_text: false
  # Best for: Content archives where retrieval is important but search is basic
  # Storage impact: Medium (stores but doesn't index full text)
```

#### 4. Minimal Storage (Preview Only)
```yaml
storage:
  store_full_text: false
  index_full_text: false
  # Best for: Minimal storage requirements, using only content previews
  # Storage impact: Low (neither stores nor indexes full text)
```

## Document Materialization System

**⚠️ Important:** This is NOT file format conversion (e.g., DOCX → XLSX). Go-Doc-Go parses documents into structured elements, then materializes them as display formats for search results and document previews.

Go-Doc-Go includes a powerful document materialization system designed for **search engines and document preview systems**. After parsing documents into structured elements, it can reconstruct and display them in multiple user-friendly formats with intelligent format-specific optimizations.

### Use Cases

This system is ideal for:
- **🔍 Search Engines**: Display clean previews of document matches without downloading original files
- **📱 Document Portals**: Show readable summaries of complex documents (PDFs, PowerPoints, Excel sheets)
- **🤖 AI/ML Pipelines**: Extract structured content for further processing
- **📊 Content Analysis**: Convert documents to structured data for analysis

### Document Materialization Features

- **📄 Multi-Format Display**: Present parsed documents as text, markdown, HTML, JSON, YAML, and XML with format-specific optimizations
- **🎯 Format-Aware Reconstruction**: Intelligent handling of document-specific elements (slides, headers, footnotes, etc.)
- **⚡ Batch Processing**: Efficient bulk document materialization for performance
- **📊 Rich Metadata**: Include document outlines, statistics, and structural analysis
- **🔄 Content Integration**: Seamlessly combine search results with materialized document content
- **💾 Memory Optimization**: Configurable content length limits and selective materialization options

### Supported Document Types (Input)

Go-Doc-Go can parse these document formats into structured elements:
- **PDF documents** → Text blocks, images, tables, headers/footers
- **Microsoft Word (DOCX)** → Paragraphs, headers, tables, images, footnotes  
- **Microsoft PowerPoint (PPTX)** → Slides, speaker notes, text boxes, images
- **Microsoft Excel (XLSX)** → Worksheets, cells, tables, charts
- **HTML pages** → Elements, links, images, text content
- **Markdown files** → Headers, paragraphs, lists, code blocks, links
- **Plain text files** → Paragraphs, sections
- **CSV files** → Rows, columns, headers
- **JSON files** → Objects, arrays, key-value pairs
- **XML files** → Elements, attributes, hierarchical structure

### Display Format Options (Output)

After parsing, documents can be materialized in these display formats:

| Format | Description | Best For | What You Get |
|--------|-------------|----------|--------------|
| `text` | Plain text with preserved structure | Reading, analysis | Clean text with section separators |
| `markdown` | Structured markdown with tables/headers | Documentation, wikis | Formatted markdown with headers, tables, lists |
| `html` | Styled HTML with CSS classes | Web display, rich rendering | Full HTML with semantic markup and styling |
| `docx_html` | Word-optimized HTML styling | Preserving Word document appearance | HTML with Times New Roman, margins, footnote styling |
| `pptx_html` | Presentation-optimized HTML layout | Slide presentation display | HTML with slide layouts, speaker notes, visual styling |
| `json` | Structured JSON representation | API integration, data processing | Complete element hierarchy and metadata as JSON |
| `yaml` | Human-readable YAML format | Configuration, readable data export | Structured data with comments in YAML format |
| `xml` | Structured XML representation | Legacy systems, data exchange | Semantic XML with proper namespaces |

### How It Works: Parse → Structure → Materialize

**Example: PowerPoint Presentation Processing**

```
Input: presentation.pptx (5 slides with speaker notes)
    ↓ PARSE
Structured Elements:
- slide_1 (element_type: "slide", content: "Introduction to AI")  
- slide_notes_1 (element_type: "slide_notes", content: "Welcome everyone...")
- slide_2 (element_type: "slide", content: "Machine Learning Basics")
- slide_notes_2 (element_type: "slide_notes", content: "ML is a subset...")
- image_1 (element_type: "image", content: "diagram.png")
    ↓ MATERIALIZE
Display Formats:
- HTML: <div class="slide">Introduction to AI</div><div class="slide-notes">Welcome everyone...</div>
- Markdown: # Slide 1: Introduction to AI\n> **Notes:** Welcome everyone...
- Text: --- SLIDE 1 ---\nIntroduction to AI\nSpeaker Notes: Welcome everyone...
```

This is **NOT** converting PowerPoint → Excel. It's extracting content for search and display.

### Document Materialization Examples

```python
from go-doc-go import search_with_documents, get_document_in_format

# Search with document materialization as markdown
results = search_with_documents(
    query_text="machine learning best practices",
    limit=10,
    document_format="markdown",
    include_document_statistics=True,
    include_document_outline=True,
    max_document_length=5000
)

print(f"Found {results.total_results} results")
print(f"Materialized {len(results.materialized_documents)} documents")

for doc_id, doc in results.materialized_documents.items():
    print(f"\nDocument: {doc.title}")
    print(f"Format: {doc.format_type}")
    print(f"Words: {doc.statistics.get('total_words', 0) if doc.statistics else 'N/A'}")
    print(f"Element count: {doc.element_count}")
    print(f"Markdown preview: {doc.formatted_content[:200]}...")
    
    if doc.outline:
        print(f"Document structure: {doc.outline.get('total_elements', 0)} elements")

# Get single document in specific format
doc_html = get_document_in_format(
    doc_id="doc_123",
    format_type="html",
    include_outline=True,
    include_statistics=True,
    max_length=10000
)

print(f"HTML document: {len(doc_html.formatted_content or '')} characters")
print(f"Document outline: {doc_html.outline}")
print(f"Statistics: {doc_html.statistics}")

# Batch document materialization
from go-doc-go import get_documents_batch_formatted

doc_ids = ["doc_1", "doc_2", "doc_3", "doc_4"]
docs_json = get_documents_batch_formatted(
    doc_ids=doc_ids,
    format_type="json",
    include_statistics=True,
    include_outline=True
)

for doc_id, doc in docs_json.items():
    print(f"Document {doc_id}: {doc.element_count} elements")
    if doc.statistics:
        print(f"  Characters: {doc.statistics.get('total_characters', 0)}")
        print(f"  Element types: {list(doc.statistics.get('element_types', {}).keys())}")
```

### Document Materialization Options

```python
from go-doc-go import DocumentMaterializationOptions

# Configure materialization options
options = DocumentMaterializationOptions(
    include_full_document=True,        # Include complete document structure
    document_format="markdown",        # Output format
    include_document_outline=True,     # Include hierarchical outline
    include_document_statistics=True,  # Include word counts, element stats
    include_full_text=True,           # Include full text content
    max_document_length=50000,        # Truncate if longer
    batch_documents=True,             # Use batch loading for efficiency
    join_elements=True,               # Join elements for full text
    element_separator='\n\n'          # Separator for joined elements
)
```

## Advanced Document Display

The system includes sophisticated document display capabilities that handle format-specific element types with intelligent display conversions:

### Format-Specific Element Display

| Source Element | Text Output | Markdown Output | HTML Output |
|---------------|-------------|-----------------|-------------|
| `slide` | `--- SLIDE N ---` | `# Slide N` | `<div class="slide">` |
| `slide_notes` | `Speaker Notes: ...` | `> **Notes:** ...` | `<div class="slide-notes">` |
| `page_header` | `[HEADER: text]` | `*Header: text*` | `<header>text</header>` |
| `page_footer` | `[FOOTER: text]` | `*Footer: text*` | `<footer>text</footer>` |
| `footnote` | `[FOOTNOTE: text]` | `[^1]: text` | `<span class="footnote">` |
| `text_box` | `[TEXT BOX: text]` | `> **Text Box:** text` | `<div class="text-box">` |
| `image` | `[IMAGE: alt_text]` | `![alt_text](src)` | `<img src="..." alt="...">` |
| `table` | Formatted table | Markdown table | HTML `<table>` |

### Document Format Detection and Display Recommendations

```python
# Analyze document format and get display advice
format_info = db.get_document_format_info("complex_doc_123")

print(f"Source: {format_info['source_format']}")           # 'pptx'
print(f"Detected: {format_info['detected_format']}")       # 'pptx' 
print(f"Elements: {format_info['format_specific_elements']}")  # ['slide', 'slide_notes', 'shape']

for recommendation in format_info['display_recommendations']:
    print(f"• {recommendation}")
# • PowerPoint presentation with 15 slides detected
# • Speaker notes found on 8 slides  
# • Recommend 'pptx_html' format for best presentation layout
# • Use 'markdown' format for readable slide content export

# Check display quality for different formats
validation = db.validate_display_capability("doc_123")
for format_type, assessment in validation['format_assessments'].items():
    print(f"{format_type}: {assessment['quality']} quality")
    print(f"  Supported elements: {assessment['supported_elements']}")
```

## Supported Document Types

Go-Doc-Go can ingest and process a variety of document formats:
- HTML pages
- Markdown files
- Plain text files
- PDF documents
- Microsoft Word documents (DOCX)
- Microsoft PowerPoint presentations (PPTX)
- Microsoft Excel spreadsheets (XLSX)
- CSV files
- XML files
- JSON files

## Content Sources

Go-Doc-Go supports multiple content sources through a modular, pluggable architecture. Each content source has its own optional dependencies, which are only required if you use that specific source:

| Content Source | Description | Required Dependencies | Installation |
|---------------|-------------|----------------------|--------------|
| File System | Local, mounted, and network file systems | None (core) | Default install |
| HTTP/Web | Fetch content from URLs and websites | `requests` | Default install |
| Confluence | Atlassian Confluence wiki content | `atlassian-python-api` | `pip install "go-doc-go[source-confluence]"` |
| JIRA | Atlassian JIRA issue tracking system | `atlassian-python-api` | `pip install "go-doc-go[source-jira]"` |
| Amazon S3 | Cloud storage through S3 | `boto3` | `pip install "go-doc-go[cloud-aws]"` |
| Databases | SQL and NoSQL database content | `sqlalchemy` | `pip install "go-doc-go[source-database]"` |
| ServiceNow | ServiceNow platform content | `pysnow` | `pip install "go-doc-go[source-servicenow]"` |
| MongoDB | MongoDB database content | `pymongo` | `pip install "go-doc-go[source-mongodb]"` |
| SharePoint | Microsoft SharePoint content | `Office365-REST-Python-Client` | `pip install "go-doc-go[source-sharepoint]"` |
| Google Drive | Google Drive content (auto-exports Docs/Sheets/Slides to MS Office) | `google-api-python-client` | `pip install "go-doc-go[source-gdrive]"` |

## Storage Backends

Go-Doc-Go supports multiple storage backends through a modular, pluggable architecture. Each backend has its own optional dependencies, which are only required if you use that specific storage method:

| Storage Backend | Description | Topic Support | Vector Search | Full-Text Search | Required Dependencies | Installation |
|-----------------|-------------|---------------|---------------|------------------|----------------------|--------------|
| File-based | Simple storage using the file system | ✅ | ❌ | ❌ | None (core) | Default install |
| SQLite | Lightweight, embedded database | ✅ | ❌ | ✅ | None (core) | Default install |
| SQLite Enhanced | SQLite with vector extension support | ✅ | ✅ | ✅ | `sqlean.py` | `pip install "go-doc-go[db-core]"` |
| Neo4J | Graph database with native relationship support | ✅ | ✅ | ✅ | `neo4j` | `pip install "go-doc-go[db-neo4j]"` |
| PostgreSQL | Robust relational database for production | ✅ | ❌ | ✅ | `psycopg2` | `pip install "go-doc-go[db-postgresql]"` |
| PostgreSQL + pgvector | PostgreSQL with vector search | ✅ | ✅ | ✅ | `psycopg2`, `pgvector` | `pip install "go-doc-go[db-postgresql,db-vector]"` |
| MongoDB | Document-oriented database | ✅ | ✅ | ✅ | `pymongo` | `pip install "go-doc-go[db-mongodb]"` |
| MySQL/MariaDB | Popular open-source SQL database | ✅ | ❌ | ✅ | `sqlalchemy`, `pymysql` | `pip install "go-doc-go[db-mysql]"` |
| Oracle | Enterprise SQL database | ✅ | ❌ | ✅ | `sqlalchemy`, `cx_Oracle` | `pip install "go-doc-go[db-oracle]"` |
| Microsoft SQL Server | Enterprise SQL database | ✅ | ❌ | ✅ | `sqlalchemy`, `pymssql` | `pip install "go-doc-go[db-mssql]"` |
| **Elasticsearch** | **Distributed search and analytics** | ✅ | ✅ | ✅ | `elasticsearch` | `pip install "go-doc-go[db-elasticsearch]"` |

## Enhanced Search Capabilities

Go-Doc-Go provides powerful, flexible search capabilities across all database backends with support for pattern matching, element type filtering, metadata search, configurable full-text indexing, and seamless document materialization.

### Search with Document Materialization

```python
from go-doc-go import search_with_documents, search_simple_structured

# Enhanced search with materialized documents
results = search_with_documents(
    query_text="quarterly financial reports",
    limit=15,
    include_topics=["finance%", "quarterly%"],
    exclude_topics=["draft%", "deprecated%"],
    # Document materialization options
    document_format="markdown",
    include_document_outline=True,
    include_document_statistics=True,
    max_document_length=10000,
    batch_documents=True
)

print(f"Search completed in {results.execution_time_ms:.1f}ms")
print(f"Materialization took {results.materialization_time_ms:.1f}ms")
print(f"Found {results.total_results} results across {len(results.documents)} documents")

# Access search results
for item in results.results:
    print(f"Element: {item.element_type} - Score: {item.similarity:.3f}")
    print(f"Preview: {item.content_preview}")

# Access materialized documents
for doc_id, doc in results.materialized_documents.items():
    print(f"\nDocument: {doc.title}")
    print(f"Format: {doc.format_type}")
    print(f"Length: {len(doc.formatted_content or '')} characters")
    
    if doc.statistics:
        stats = doc.statistics
        print(f"Words: {stats.get('total_words', 0)}")
        print(f"Elements: {stats.get('total_elements', 0)}")
        print(f"Element types: {list(stats.get('element_types', {}).keys())}")
    
    if doc.outline:
        print(f"Outline sections: {doc.outline.get('total_sections', 0)}")

# Simple structured search with document materialization
results = search_simple_structured(
    query_text="machine learning algorithms",
    limit=10,
    similarity_threshold=0.8,
    include_topics=["ai%", "ml%"],
    days_back=30,
    element_types=["header", "paragraph"],
    # Document options
    document_format="html",
    include_document_statistics=True
)
```

### Batch Document Retrieval

```python
from go-doc-go import get_documents_batch_formatted

# Efficiently retrieve multiple documents in formatted output
doc_ids = ["report_q1", "report_q2", "report_q3", "report_q4"]

# Get all quarterly reports as markdown with statistics
quarterly_reports = get_documents_batch_formatted(
    doc_ids=doc_ids,
    format_type="markdown",
    include_statistics=True,
    include_outline=True,
    max_length=20000
)

for doc_id, doc in quarterly_reports.items():
    print(f"\n{doc.title or doc_id}")
    print(f"Elements: {doc.element_count}")
    
    if doc.statistics:
        print(f"Words: {doc.statistics.get('total_words', 0)}")
        print(f"Tables: {doc.statistics.get('element_types', {}).get('table', 0)}")
        print(f"Headers: {doc.statistics.get('element_types', {}).get('header', 0)}")
    
    # Save to file
    if doc.formatted_content:
        with open(f"{doc_id}.md", "w", encoding="utf-8") as f:
            f.write(doc.formatted_content)
```

### Document Display in Multiple Formats

```python
from go-doc-go import get_document_in_format

# Display a parsed document in multiple formats
doc_id = "technical_specification_v2"

# Get as markdown for documentation
markdown_doc = get_document_in_format(
    doc_id=doc_id,
    format_type="markdown",
    include_outline=True,
    max_length=50000
)

# Get as HTML for web display
html_doc = get_document_in_format(
    doc_id=doc_id,
    format_type="html",
    include_statistics=True
)

# Get as JSON for API integration
json_doc = get_document_in_format(
    doc_id=doc_id,
    format_type="json",
    include_full_text=True
)

print(f"Markdown: {len(markdown_doc.formatted_content or '')} chars")
print(f"HTML: {len(html_doc.formatted_content or '')} chars")
print(f"JSON: {len(json_doc.formatted_content or '')} chars")

# Check for errors
if markdown_doc.materialization_error:
    print(f"Error: {markdown_doc.materialization_error}")
```

## Intelligent Knowledge Graph Construction

The Go-Doc-Go Knowledge Engine's declarative ontology system transforms unstructured documents into intelligent, searchable knowledge graphs through automated business entity extraction and semantic relationship discovery. This extensible pipeline enables organizations to apply their domain expertise at scale through configuration-driven processing.

### Why Domain Entity Extraction Matters

- **🎯 Business Context**: Automatically identify customers, products, regulations, components, or any domain-specific entities
- **🔍 Knowledge Discovery**: Uncover hidden relationships between entities across thousands of documents
- **📊 Compliance & Audit**: Track regulatory references, safety requirements, and compliance relationships
- **🚀 Accelerated Analysis**: Convert months of manual document review into minutes of automated extraction

### Core Capabilities

- **🧩 Flexible Entity Definition**: Define any entity type relevant to your domain (people, products, regulations, etc.)
- **🎯 Two-Phase Processing**: First extracts entities from document elements, then discovers relationships
- **📝 Multiple Extraction Methods**: 
  - **Semantic Similarity**: Match concepts using AI embeddings (e.g., "brake system" matches "ABS", "anti-lock", "braking mechanism")
  - **Pattern Matching**: Extract codes, IDs, references using regex (e.g., "PROD-12345", "Section 4.2.1")
  - **Keyword Detection**: Simple keyword and phrase matching
- **🔗 Relationship Discovery**: Automatically finds connections between entities based on business rules
- **📊 Confidence Scoring**: Configurable thresholds for precision vs. recall optimization
- **🌐 Cross-Document Intelligence**: Discover relationships spanning multiple documents and sources

### Real-World Applications

| Industry | Entity Examples | Relationship Examples |
|----------|----------------|----------------------|
| **Automotive** | Components, Safety Standards, Diagnostic Codes | Component COMPLIES_WITH Standard, Code DIAGNOSES Component |
| **Pharmaceutical** | Drugs, Clinical Trials, Adverse Events | Drug TESTED_IN Trial, Drug CAUSES Adverse_Event |
| **Financial** | Companies, Executives, Metrics | Executive SPEAKS_ABOUT Metric, Company REPORTS Revenue |
| **Legal** | Parties, Clauses, Obligations | Party BOUND_BY Clause, Clause REFERENCES Regulation |
| **Engineering** | Systems, Requirements, Tests | System SATISFIES Requirement, Test VALIDATES System |

### Quick Example

```yaml
# Define your domain ontology
name: automotive
version: "1.0"

terms:
  - id: brake_system
    name: "Brake System"
    aliases: ["ABS", "anti-lock", "braking"]
    
  - id: safety_requirement
    name: "Safety Requirement"
    
element_mappings:
  # Semantic matching for brake systems
  - term: brake_system
    rule_type: semantic
    semantic_phrase: "brake ABS anti-lock rotor pad caliper"
    threshold: 0.7
    
  # Pattern matching for safety standards
  - term: safety_requirement
    rule_type: regex
    patterns: ["FMVSS-[0-9]+", "ISO[\\s-]?26262"]

relationship_rules:
  - id: compliance_relationship
    source_terms: ["brake_system"]
    target_terms: ["safety_requirement"]
    relationship_type: COMPLIES_WITH
    constraints:
      hierarchy_level: -1  # Same document
```

### Configuration

```yaml
relationship_detection:
  domain:
    enabled: true
    ontologies:
      - path: "./ontologies/automotive.yaml"
        active: true
      - path: "./ontologies/regulatory.yaml"
        active: true
    entity_extraction:
      min_confidence: 0.5
      batch_size: 100
    relationship_detection:
      min_confidence: 0.6
```

### Usage

```python
from go_doc_go import Config
from go_doc_go.domain import OntologyManager

# Process documents with domain extraction
config = Config("config.yaml")
ontology_manager = config.get_ontology_manager()

# Query extracted entities
brake_systems = db.find_elements_by_term("brake_system", domain="automotive")
print(f"Found {len(brake_systems)} brake system references")

# Analyze entity relationships
relationships = db.get_entity_relationships(entity_type="brake_system")
for rel in relationships:
    print(f"{rel.source} -> {rel.relationship_type} -> {rel.target}")

# Get domain statistics
stats = db.get_term_statistics("automotive")
print(f"Domain coverage: {stats['total_terms']} unique concepts found")
```

### Documentation

- 📖 **[Complete Domain Entity Extraction Guide](docs/domain-ontology.md)** - Architecture, API reference, and advanced features
- 🚀 **[Quick Start Guide](docs/domain-quickstart.md)** - Get started in 5 minutes
- 🔧 **[Ontology Management Guide](docs/ontology-management.md)** - Best practices for creating and maintaining ontologies
- 📚 **[Example Ontologies](examples/ontologies/)** - Pre-built ontologies for common domains

## Advanced Structured Search System

Go-Doc-Go includes a powerful, backend-agnostic structured search system that provides sophisticated querying capabilities with automatic optimization based on backend capabilities.

### Structured Search with Document Materialization

```python
from go-doc-go import search_structured, SearchQueryRequest, SearchCriteriaGroupRequest
from go-doc-go.storage.search import (
    LogicalOperatorEnum, SemanticSearchRequest, TopicSearchRequest, 
    DateSearchRequest, DateRangeOperatorEnum
)

# Build complex structured query
query = SearchQueryRequest(
    criteria_group=SearchCriteriaGroupRequest(
        operator=LogicalOperatorEnum.AND,
        semantic_search=SemanticSearchRequest(
            query_text="security policies and procedures",
            similarity_threshold=0.8
        ),
        topic_search=TopicSearchRequest(
            include_topics=["security%", "policy%"],
            exclude_topics=["deprecated%", "draft%"],
            min_confidence=0.7
        ),
        date_search=DateSearchRequest(
            operator=DateRangeOperatorEnum.RELATIVE_DAYS,
            relative_value=90  # Last 90 days
        )
    ),
    limit=20,
    include_similarity_scores=True,
    include_element_dates=True
)

# Execute with document materialization
results = search_structured(
    query=query,
    text=True,
    content=True,
    # Document materialization options
    include_full_document=True,
    document_format="markdown",
    include_document_outline=True,
    include_document_statistics=True,
    max_document_length=15000,
    batch_documents=True
)

print(f"Query ID: {results.query_id}")
print(f"Execution time: {results.execution_time_ms:.1f}ms")
print(f"Materialization time: {results.materialization_time_ms:.1f}ms")
print(f"Total results: {results.total_results}")
print(f"Documents materialized: {len(results.materialized_documents)}")

# Process results with materialized content
for item in results.results:
    print(f"\nElement: {item.element_id}")
    print(f"Score: {item.similarity:.3f}")
    print(f"Topics: {item.topics}")
    print(f"Text preview: {item.text[:200] if item.text else 'N/A'}...")
    
    # Access materialized document
    if item.doc_id in results.materialized_documents:
        doc = results.materialized_documents[item.doc_id]
        print(f"Document: {doc.title}")
        print(f"Markdown length: {len(doc.formatted_content or '')} chars")
        
        if doc.statistics:
            print(f"Document stats: {doc.statistics.get('total_words', 0)} words")
```

## Architecture

The system is built with a modular architecture:

1. **Content Sources**: Adapters for different content origins (with conditional dependencies)
2. **Document Parsers**: Transform content into structured elements (with format-specific dependencies)
3. **Document Database**: Stores metadata, elements, and relationships (with backend-specific dependencies)
4. **Content Resolver**: Retrieves original content when needed
5. **Embedding Generator**: Creates vector representations for semantic search (with model-specific dependencies)
6. **Relationship Detector**: Identifies connections between document elements
7. **Topic Manager**: Organizes content by topics for enhanced categorization and filtering
8. **Structured Search Engine**: Advanced query processing with backend capability detection
9. **Full-Text Engine**: Configurable text storage and indexing for optimal search performance
10. **📄 Document Materializer**: Advanced document reconstruction and format conversion system
11. **⚡ Batch Processor**: Efficient bulk operations for document retrieval and processing

## Getting Started

### Flexible Installation

Go-Doc-Go supports a modular installation system where you can choose which components to install based on your specific needs:

```bash
# Minimal installation (core functionality only)
pip install go-doc-go

# Install with specific database backend
pip install "go-doc-go[db-postgresql]"    # PostgreSQL support
pip install "go-doc-go[db-mongodb]"       # MongoDB support
pip install "go-doc-go[db-neo4j]"         # Neo4j support
pip install "go-doc-go[db-mysql]"         # MySQL support
pip install "go-doc-go[db-elasticsearch]" # Elasticsearch support
pip install "go-doc-go[db-core]"          # SQLite extensions + SQLAlchemy

# Install with specific content sources
pip install "go-doc-go[source-database]"     # Database content sources
pip install "go-doc-go[source-confluence]"   # Confluence content sources
pip install "go-doc-go[source-jira]"         # JIRA content sources
pip install "go-doc-go[source-gdrive]"       # Google Drive content sources
pip install "go-doc-go[source-sharepoint]"   # SharePoint content sources
pip install "go-doc-go[source-servicenow]"   # ServiceNow content sources
pip install "go-doc-go[source-mongodb]"      # MongoDB content sources

# Install with specific embedding provider
pip install "go-doc-go[huggingface]"    # HuggingFace/PyTorch support
pip install "go-doc-go[openai]"         # OpenAI API support
pip install "go-doc-go[fastembed]"      # FastEmbed support (15x faster)

# Install with AWS S3 support
pip install "go-doc-go[cloud-aws]"

# Install additional components
pip install "go-doc-go[scientific]"     # NumPy and scientific libraries
pip install "go-doc-go[document_parsing]"  # Additional document parsing utilities

# Install all database backends
pip install "go-doc-go[db-all]"

# Install all content sources
pip install "go-doc-go[source-all]"

# Install all embedding providers
pip install "go-doc-go[embedding-all]"

# Install everything
pip install "go-doc-go[all]"
```

### Configuration

Create a configuration file `config.yaml`:

```yaml
storage:
  backend: elasticsearch  # Options: file, sqlite, mongodb, postgresql, elasticsearch, sqlalchemy
  topic_support: true  # Enable topic features
  
  # Full-text storage and indexing configuration
  store_full_text: true      # Store full text for retrieval
  index_full_text: true      # Index full text for search
  compress_full_text: true   # Enable compression for large documents
  full_text_max_length: 100000  # Limit very large documents (100KB max)
  
  # Elasticsearch-specific configuration
  elasticsearch:
    hosts: ["localhost:9200"]
    username: "elastic"  # optional
    password: "changeme"  # optional
    index_prefix: "go-doc-go"
    vector_dimension: 384

embedding:
  enabled: true
  # Embedding provider: choose between "huggingface", "openai", or "fastembed"
  provider: "huggingface"
  model: "sentence-transformers/all-MiniLM-L6-v2"
  dimensions: 384  # Configurable based on content needs
  contextual: true  # Enable contextual embeddings

content_sources:
  # Local file content source (core, no extra dependencies)
  - name: "documentation"
    type: "file"
    base_path: "./docs"
    file_pattern: "**/*.md"
    max_link_depth: 2
    topics: ["documentation", "user-guides"]  # Assign topics to this source

relationship_detection:
  enabled: true
  link_pattern: r"\[\[(.*?)\]\]|href=[\"\'](.*?)[\"\']"

logging:
  level: "INFO"
  file: "./logs/docpointer.log"
```

### Basic Usage with Document Materialization

```python
from go-doc-go import Config, ingest_documents
from go-doc-go import search_with_documents, get_document_in_format

# Load configuration
config = Config("config.yaml")

# Initialize storage
db = config.initialize_database()

# Ingest documents
stats = ingest_documents(config)
print(f"Processed {stats['documents']} documents with {stats['elements']} elements")

# Search with document materialization
results = search_with_documents(
    query_text="machine learning algorithms",
    limit=10,
    document_format="markdown",
    include_document_statistics=True,
    include_document_outline=True,
    max_document_length=5000
)

print(f"Found {results.total_results} results")
print(f"Materialized {len(results.materialized_documents)} documents")

# Process results
for item in results.results:
    print(f"Element: {item.element_type} - Score: {item.similarity:.3f}")
    print(f"Content: {item.content_preview}")

# Process materialized documents
for doc_id, doc in results.materialized_documents.items():
    print(f"\nDocument: {doc.title}")
    print(f"Format: {doc.format_type}")
    print(f"Length: {len(doc.formatted_content or '')} characters")
    
    if doc.statistics:
        print(f"Words: {doc.statistics.get('total_words', 0)}")
        print(f"Elements: {doc.element_count}")
    
    # Save markdown to file
    if doc.formatted_content:
        filename = f"{doc_id.replace('/', '_')}.md"
        with open(filename, "w", encoding="utf-8") as f:
            f.write(doc.formatted_content)
        print(f"Saved to {filename}")

# Get specific document in different formats
doc_markdown = get_document_in_format("doc_123", "markdown", include_outline=True)
doc_html = get_document_in_format("doc_123", "html", include_statistics=True)
doc_json = get_document_in_format("doc_123", "json", max_length=10000)

print(f"Markdown: {len(doc_markdown.formatted_content or '')} chars")
print(f"HTML: {len(doc_html.formatted_content or '')} chars")
print(f"JSON: {len(doc_json.formatted_content or '')} chars")
```

## Document Materialization Examples

### Example 1: Search and Export Documents

```python
from go-doc-go import search_with_documents
import os

# Search for quarterly reports and export as markdown
results = search_with_documents(
    query_text="quarterly financial report",
    include_topics=["finance%", "quarterly%"],
    exclude_topics=["draft%"],
    limit=20,
    document_format="markdown",
    include_document_statistics=True,
    max_document_length=50000
)

# Create export directory
os.makedirs("exported_reports", exist_ok=True)

# Export each document
for doc_id, doc in results.materialized_documents.items():
    if doc.formatted_content and not doc.materialization_error:
        filename = f"exported_reports/{doc_id.replace('/', '_')}.md"
        
        with open(filename, "w", encoding="utf-8") as f:
            # Add metadata header
            f.write(f"# {doc.title or doc_id}\n\n")
            if doc.statistics:
                f.write(f"- **Words:** {doc.statistics.get('total_words', 0)}\n")
                f.write(f"- **Elements:** {doc.element_count}\n")
                f.write(f"- **Source:** {doc.source}\n\n")
            f.write("---\n\n")
            f.write(doc.formatted_content)
        
        print(f"Exported: {filename}")
    else:
        print(f"Skipped {doc_id}: {doc.materialization_error}")
```

### Example 2: Batch Document Analysis

```python
from go-doc-go import get_documents_batch_formatted
import json

# Get all technical documents and analyze their structure
doc_ids = ["tech_spec_v1", "tech_spec_v2", "api_guide", "user_manual"]

docs = get_documents_batch_formatted(
    doc_ids=doc_ids,
    format_type="json",
    include_statistics=True,
    include_outline=True
)

# Analyze document structure
analysis = {
    "total_documents": len(docs),
    "total_words": 0,
    "total_elements": 0,
    "element_type_distribution": {},
    "documents": []
}

for doc_id, doc in docs.items():
    if doc.statistics and not doc.materialization_error:
        doc_stats = {
            "doc_id": doc_id,
            "title": doc.title,
            "words": doc.statistics.get('total_words', 0),
            "elements": doc.element_count,
            "element_types": doc.statistics.get('element_types', {})
        }
        
        analysis["documents"].append(doc_stats)
        analysis["total_words"] += doc_stats["words"]
        analysis["total_elements"] += doc_stats["elements"]
        
        # Aggregate element types
        for elem_type, count in doc_stats["element_types"].items():
            analysis["element_type_distribution"][elem_type] = (
                analysis["element_type_distribution"].get(elem_type, 0) + count
            )

# Save analysis
with open("document_analysis.json", "w") as f:
    json.dump(analysis, f, indent=2)

print(f"Analyzed {analysis['total_documents']} documents")
print(f"Total words: {analysis['total_words']:,}")
print(f"Total elements: {analysis['total_elements']:,}")
print("Element type distribution:")
for elem_type, count in sorted(analysis["element_type_distribution"].items()):
    print(f"  {elem_type}: {count}")
```

### Example 3: Document Format Comparison

```python
from go-doc-go import get_document_in_format
import time

doc_id = "complex_presentation_2024"

# Get document in multiple formats and compare
formats = ["text", "markdown", "html", "docx_html", "pptx_html"]
format_results = {}

for format_type in formats:
    start_time = time.time()
    
    doc = get_document_in_format(
        doc_id=doc_id,
        format_type=format_type,
        include_statistics=True,
        max_length=100000
    )
    
    processing_time = (time.time() - start_time) * 1000
    
    if not doc.materialization_error:
        format_results[format_type] = {
            "length": len(doc.formatted_content or ''),
            "processing_time_ms": processing_time,
            "quality": "high" if doc.formatted_content else "low",
            "stats": doc.statistics
        }
        
        # Save sample of each format
        if doc.formatted_content:
            with open(f"sample_{doc_id}_{format_type}.txt", "w", encoding="utf-8") as f:
                f.write(doc.formatted_content[:1000])  # First 1000 chars
    else:
        format_results[format_type] = {
            "error": doc.materialization_error,
            "processing_time_ms": processing_time
        }

# Print comparison
print(f"Format Comparison for {doc_id}:")
print("-" * 60)
for format_type, result in format_results.items():
    if "error" not in result:
        print(f"{format_type:12}: {result['length']:6,} chars, "
              f"{result['processing_time_ms']:6.1f}ms, {result['quality']}")
    else:
        print(f"{format_type:12}: ERROR - {result['error']}")
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Recommended Configurations

### Minimal Setup with Document Materialization
```bash
pip install "go-doc-go[db-core]"
```

### High-Performance Search with Document Export
```bash
pip install "go-doc-go[db-elasticsearch,fastembed]"
```
Configuration:
```yaml
storage:
  backend: elasticsearch
  store_full_text: true
  index_full_text: true
  compress_full_text: true
  full_text_max_length: 100000
```

### Production Setup with Full Document Capabilities
```bash
pip install "go-doc-go[db-postgresql,source-database,fastembed]"
```
Configuration:
```yaml
storage:
  backend: postgresql
  store_full_text: true   # Enable document materialization
  index_full_text: true   # Enable search
  compress_full_text: true
  topic_support: true
```

### Enterprise Configuration with Complete Document Management
```bash
pip install "go-doc-go[db-all,embedding-all,source-all,cloud-aws]"
```

# Verified Compatibility

Tested and working with:
- ✅ All storage backends with full-text configuration and document materialization
- ✅ Complete document retrieval and format conversion (text, markdown, HTML, JSON, YAML, XML)
- ✅ Advanced document reconstruction with format-specific optimizations (DOCX, PPTX, PDF)
- ✅ Document format detection and reconstruction quality validation
- ✅ Batch document processing and bulk format conversion
- ✅ Enhanced search integration with materialized document content
- ✅ Document statistics, outlines, and structural analysis
- ✅ Performance-optimized document materialization with configurable options
- ✅ Advanced structured search system with document materialization integration
- ✅ Storage optimization recommendations and configuration monitoring
- ✅ Format-specific document reconstruction with intelligent element mapping
