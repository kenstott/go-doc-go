# Data Sources Guide

Go-Doc-Go can ingest unstructured data from virtually anywhere through its modular content source architecture.

## Supported Sources Overview

| Source Type | Description | Use Cases                                                            | Required Dependencies |
|-------------|-------------|----------------------------------------------------------------------|----------------------|
| **File System** | Local, network, and mounted filesystems | Document libraries, shared drives                                    | None (core) |
| **Databases** | SQL and NoSQL with TEXT/VARCHAR fields | CMS data, user content, articles                                     | `sqlalchemy` |
| **DuckDB** | Parquet datasets with hive partitioning | SEC filings, data warehouses, analytical datasets                    | `duckdb` |
| **HTTP/Web** | Fetch from URLs and web endpoints | APIs, web scraping, RSS feeds, GraphQL endpoints, JSON:API endpoints | `requests` |
| **Amazon S3** | Cloud object storage | Document archives, data lakes                                        | `boto3` |
| **SharePoint** | Microsoft SharePoint documents and sites | Corporate documents, wikis                                           | `Office365-REST-Python-Client` |
| **Confluence** | Atlassian wiki content | Technical documentation, knowledge bases                             | `atlassian-python-api` |
| **JIRA** | Issue tracking and project data | Support tickets, project descriptions                                | `atlassian-python-api` |
| **Google Drive** | Documents, Sheets, Slides (auto-exports) | Collaborative documents                                              | `google-api-python-client` |
| **MongoDB** | NoSQL document collections | Content management, user data                                        | `pymongo` |
| **ServiceNow** | Enterprise service platform | IT service docs, knowledge articles                                  | `pysnow` |

## Database Sources (Most Common)

### SQL Databases with TEXT/VARCHAR Content

```yaml
content_sources:
  - name: "articles"
    type: "database"
    connection_string: "postgresql://user:pass@host/db"
    query: |
      SELECT 
        id,
        title,
        content,
        created_at,
        author,
        category
      FROM articles 
      WHERE status = 'published'
    
    # Map query results to document fields
    field_mapping:
      doc_id: "id"
      title: "title" 
      content: "content"
      metadata:
        author: "author"
        category: "category"
        created: "created_at"
```

### DuckDB Parquet Datasets (Hive-Partitioned)

```yaml
content_sources:
  - name: "sec-filings"
    type: "duckdb"
    database_path: "/data/sec-parquet"  # Path to parquet dataset
    enable_hive_partitioning: true
    
    # Connection configuration
    connection_config:
      threads: 4
      memory_limit: "2GB"
    
    # Each query generates documents from the dataset
    queries:
      - name: "10k-mda-sections"
        description: "Management Discussion & Analysis from 10-K filings"
        sql: |
          SELECT 
            cik,
            filing_type,
            year,
            section,
            paragraph_number,
            paragraph_text,
            filing_date
          FROM read_parquet('cik=*/filing_type=10K/*/mda.parquet', hive_partitioning=true)
          WHERE year >= 2022 
            AND paragraph_text IS NOT NULL
            AND LENGTH(paragraph_text) > 100
        
        # Columns that uniquely identify each document
        id_columns: ["cik", "filing_type", "year", "paragraph_number"]
        
        # Column containing the document text
        content_column: "paragraph_text"
        
        # Columns to include in metadata (optional - defaults to all non-content columns)
        metadata_columns: ["filing_date", "section"]
        
        # Document type for parsing
        doc_type: "sec_mda"
      
      - name: "earnings-transcripts"
        description: "CEO and CFO comments from earnings calls"
        sql: |
          SELECT *
          FROM read_parquet('*/*/earnings.parquet', hive_partitioning=true)
          WHERE speaker_role IN ('CEO', 'CFO')
            AND LENGTH(paragraph_text) > 50
        id_columns: ["cik", "year", "quarter", "speaker_name", "paragraph_number"]
        content_column: "paragraph_text"
        metadata_columns: ["speaker_role", "call_date", "company_name"]
        doc_type: "earnings_transcript"
```

### Complex Database Queries

```yaml
content_sources:
  - name: "support_tickets"
    type: "database"
    connection_string: "mysql://user:pass@host/helpdesk"
    query: |
      SELECT 
        t.id,
        CONCAT(t.subject, '\n\n', t.description) as content,
        t.created_at,
        c.name as customer,
        u.name as assigned_to,
        GROUP_CONCAT(tag.name) as tags
      FROM tickets t
      LEFT JOIN customers c ON t.customer_id = c.id
      LEFT JOIN users u ON t.assigned_to = u.id
      LEFT JOIN ticket_tags tt ON t.id = tt.ticket_id
      LEFT JOIN tags tag ON tt.tag_id = tag.id
      WHERE t.status IN ('open', 'in_progress')
      GROUP BY t.id
    
    batch_size: 1000
    field_mapping:
      doc_id: "id"
      content: "content"
      metadata:
        customer: "customer"
        assignee: "assigned_to"
        tags: "tags"
        created: "created_at"
```

### JSON Document Mode

```yaml
content_sources:
  - name: "product_catalog"
    type: "database"
    connection_string: "postgresql://user:pass@host/ecommerce"
    query: "products"
    json_mode: true  # Export entire rows as JSON documents
    
    # Optional: specify which columns to include (default: all except ID)
    json_columns: ["name", "description", "price", "category", "specs"]
    
    # Include metadata in the JSON content (default: true)
    json_include_metadata: true
    
    # Standard options still apply
    id_column: "product_id"
    metadata_columns: ["category", "brand", "created_at"]
    timestamp_column: "updated_at"
    batch_size: 500
```

### Binary Content Handling

```yaml
content_sources:
  - name: "document_blobs"
    type: "database"
    connection_string: "postgresql://user:pass@host/docstore"
    query: |
      SELECT 
        doc_id,
        file_name,
        binary_content,
        mime_type,
        created_at
      FROM documents
      WHERE binary_content IS NOT NULL
    
    field_mapping:
      doc_id: "doc_id"
      title: "file_name"
      content: "binary_content"  # Will be handled as binary data
      metadata:
        mime_type: "mime_type"
        created: "created_at"
    
    # Truncate very large binary content for performance
    max_content_length: 1048576  # 1MB limit
```

## File System Sources

### Basic File Ingestion

```yaml
content_sources:
  - name: "documentation" 
    type: "file"
    base_path: "/shared/docs"
    file_pattern: "**/*.{md,txt,pdf,docx}"
    max_depth: 5
    
    # Extract metadata from file paths
    path_metadata:
      department: "/{department}/**"
      project: "/{department}/{project}/**"
      version: "**/v{version}/*"
```

### Advanced File Configuration

```yaml
content_sources:
  - name: "corporate_docs"
    type: "file" 
    base_path: "/mnt/corporate"
    
    # Multiple patterns
    include_patterns:
      - "**/*.pdf"
      - "**/*.docx" 
      - "**/*.pptx"
      - "**/README.md"
    
    # Exclusions
    exclude_patterns:
      - "**/archive/**"
      - "**/temp/**"
      - "**/.git/**"
    
    # Follow document links
    follow_links: true
    max_link_depth: 2
    
    # Metadata extraction
    extract_file_metadata: true
    custom_metadata:
      source_system: "corporate_files"
      ingestion_date: "{{ now() }}"
```

## Cloud Storage Sources

### Amazon S3

```yaml
content_sources:
  - name: "s3_documents"
    type: "s3"
    bucket: "company-documents"
    prefix: "docs/"
    
    # AWS credentials (or use IAM roles)
    aws_access_key_id: "${AWS_ACCESS_KEY}" 
    aws_secret_access_key: "${AWS_SECRET_KEY}"
    region: "us-west-2"
    
    # Filter by file types
    include_extensions: [".pdf", ".docx", ".md"]
    
    # Parallel processing
    max_workers: 10
    batch_size: 100
```

### SharePoint

```yaml
content_sources:
  - name: "sharepoint_docs"
    type: "sharepoint"
    site_url: "https://company.sharepoint.com/sites/documents"
    
    # Authentication
    client_id: "${SHAREPOINT_CLIENT_ID}"
    client_secret: "${SHAREPOINT_CLIENT_SECRET}"
    tenant_id: "${SHAREPOINT_TENANT_ID}"
    
    # Libraries to scan
    libraries:
      - "Shared Documents"
      - "Project Files"
      - "Policies"
    
    # API preference (graph or rest)
    api_type: "graph"  # or "rest"
    max_items: 5000
```

## Enterprise Integration Sources

### Confluence

```yaml
content_sources:
  - name: "confluence_wiki"
    type: "confluence"
    url: "https://company.atlassian.net"
    username: "${CONFLUENCE_USER}"
    api_token: "${CONFLUENCE_TOKEN}"
    
    # Spaces to include
    spaces:
      - "TECH"  # Technical documentation
      - "PROD"  # Product documentation
      - "HR"    # HR policies
    
    # Content filtering
    include_page_types: ["page", "blogpost"]
    exclude_labels: ["draft", "obsolete"]
    
    # Include page hierarchy
    include_page_tree: true
```

### JIRA

```yaml
content_sources:
  - name: "jira_issues"
    type: "jira" 
    url: "https://company.atlassian.net"
    username: "${JIRA_USER}"
    api_token: "${JIRA_TOKEN}"
    
    # JQL query for content
    jql: |
      project in (SUPPORT, TECH, PROD) 
      AND created >= -90d 
      AND description is not EMPTY
    
    # Fields to include
    fields:
      - "summary"
      - "description" 
      - "comments"
      - "labels"
      - "priority"
    
    # Include comments as separate documents
    include_comments: true
    max_results: 10000
```

### Google Drive

```yaml
content_sources:
  - name: "google_drive"
    type: "google_drive"
    
    # Service account credentials
    credentials_file: "/path/to/service-account.json"
    
    # Folders to scan
    folder_names:
      - "Company Documentation"
      - "Project Files"
      - "Meeting Notes"
    
    # Auto-export Google formats to Office formats
    export_google_formats: true
    export_mappings:
      "application/vnd.google-apps.document": "docx"
      "application/vnd.google-apps.spreadsheet": "xlsx" 
      "application/vnd.google-apps.presentation": "pptx"
    
    max_files: 5000
```

## API and Web Sources

### Generic HTTP/REST APIs

```yaml
content_sources:
  - name: "api_content"
    type: "http"
    
    # API configuration
    base_url: "https://api.company.com"
    endpoints:
      - path: "/articles"
        method: "GET"
        params:
          status: "published"
          limit: 1000
      - path: "/knowledge-base"
        method: "GET"
        
    # Authentication
    auth_type: "bearer"
    auth_token: "${API_TOKEN}"
    
    # Response parsing
    content_field: "body"
    title_field: "title"
    id_field: "id"
    
    # Pagination
    pagination:
      type: "offset"
      limit_param: "limit"
      offset_param: "offset"
      page_size: 100
```

### JSON:API Standard Support

```yaml
content_sources:
  - name: "jsonapi_content"
    type: "web"
    
    # JSON:API endpoints
    start_urls:
      - "https://api.company.com/v1/articles"
      - "https://api.company.com/v1/posts"
    
    # JSON:API specific configuration
    api_format: "jsonapi"
    
    # Authentication for JSON:API
    headers:
      Authorization: "Bearer ${API_TOKEN}"
      Content-Type: "application/vnd.api+json"
      Accept: "application/vnd.api+json"
    
    # JSON:API response parsing
    content_selectors:
      title: "data[*].attributes.title"
      body: "data[*].attributes.content"
      id: "data[*].id"
      type: "data[*].type"
    
    # Handle JSON:API relationships and included resources
    include_relationships: true
    follow_links: true
    link_selector: "links.next"  # Follow pagination links
    
    max_pages: 100
    delay: 0.5
```

### GraphQL API Support

```yaml
content_sources:
  - name: "graphql_content" 
    type: "web"
    
    # GraphQL endpoint
    start_urls:
      - "https://api.company.com/graphql"
    
    # GraphQL query configuration
    method: "POST"
    headers:
      Authorization: "Bearer ${GRAPHQL_TOKEN}"
      Content-Type: "application/json"
    
    # GraphQL query body
    request_body: |
      {
        "query": "query GetArticles($first: Int!) { 
          articles(first: $first) { 
            nodes { 
              id title content createdAt author { name email } 
            } 
            pageInfo { hasNextPage endCursor }
          } 
        }",
        "variables": { "first": 100 }
      }
    
    # Parse GraphQL response
    content_selectors:
      title: "data.articles.nodes[*].title"
      body: "data.articles.nodes[*].content"
      id: "data.articles.nodes[*].id"
      author: "data.articles.nodes[*].author.name"
      created: "data.articles.nodes[*].createdAt"
```

### Web Scraping

```yaml
content_sources:
  - name: "company_blog"
    type: "web"
    
    # URLs to scrape
    start_urls:
      - "https://company.com/blog"
      - "https://company.com/news"
    
    # Content extraction
    content_selectors:
      title: "h1.post-title"
      body: ".post-content"
      date: ".post-date"
      author: ".post-author"
    
    # Link following
    follow_links: true
    link_selector: "a.read-more"
    max_depth: 2
    
    # Rate limiting
    delay: 1.0  # seconds between requests
    max_pages: 1000
```

## NoSQL and Specialized Sources

### MongoDB

```yaml
content_sources:
  - name: "mongodb_content"
    type: "mongodb"
    
    connection_string: "mongodb://localhost:27017/content_db"
    collection: "articles"
    
    # Query filter
    query:
      status: "published"
      content_type: {"$in": ["article", "blog_post"]}
      created_at: {"$gte": "2024-01-01"}
    
    # Field mapping
    field_mapping:
      doc_id: "_id"
      title: "headline"
      content: "body"
      metadata:
        author: "author.name"
        tags: "tags"
        created: "published_date"
        
    batch_size: 500
```

### ServiceNow

```yaml
content_sources:
  - name: "servicenow_kb"
    type: "servicenow"
    
    instance_url: "https://company.service-now.com"
    username: "${SNOW_USER}"
    password: "${SNOW_PASS}"
    
    # Tables to extract
    tables:
      - name: "kb_knowledge"
        fields: ["short_description", "text", "topic", "author"]
        filter: "workflow_state=published"
      - name: "incident"
        fields: ["short_description", "description", "resolution_notes"]
        filter: "state=6^opened_at>=javascript:gs.daysAgo(90)"
    
    max_records: 10000
```

### Microsoft Exchange

```yaml
content_sources:
  - name: "exchange_emails"
    type: "exchange"
    
    # EWS (Exchange Web Services) Configuration
    server_url: "https://mail.company.com/EWS/Exchange.asmx"
    username: "${EXCHANGE_USER}"
    password: "${EXCHANGE_PASS}"
    
    # Authentication method: 'basic', 'ntlm', or 'oauth2'
    auth_method: "ntlm"
    
    # Folders to scan
    folders:
      - "Inbox"
      - "Sent Items"
      - "Public Folders/Knowledge Base"
    
    # Email filtering
    filters:
      - from_contains: ["@company.com"]
      - subject_contains: ["Policy", "Documentation", "Manual"]
      - date_range:
          start: "2024-01-01"
          end: "2024-12-31"
    
    # Content extraction
    include_attachments: true
    attachment_extensions: [".pdf", ".docx", ".xlsx", ".pptx"]
    max_attachment_size: 10485760  # 10MB
    
    # Rate limiting
    max_emails_per_folder: 1000
    delay_between_requests: 0.1
```

### Microsoft Graph API (Exchange Online)

```yaml
content_sources:
  - name: "graph_emails"
    type: "exchange"
    
    # Microsoft Graph API Configuration
    api_type: "graph"  # Use Graph API instead of EWS
    tenant_id: "${AZURE_TENANT_ID}"
    client_id: "${AZURE_CLIENT_ID}"
    client_secret: "${AZURE_CLIENT_SECRET}"
    
    # User mailboxes to scan
    mailboxes:
      - "user1@company.com"
      - "shared-docs@company.com"
      - "support@company.com"
    
    # Graph API specific options
    scopes: ["https://graph.microsoft.com/.default"]
    
    # Email filtering via Graph query
    odata_filter: |
      (from/emailAddress/address eq 'docs@company.com') and
      (receivedDateTime ge 2024-01-01T00:00:00Z) and
      (hasAttachments eq true)
    
    # Content options
    include_body: true
    body_type: "text"  # 'text' or 'html'
    include_attachments: true
    max_items: 1000
```

## Configuration Tips

### Environment Variables

Use environment variables for sensitive information:

```yaml
# .env file
DB_CONNECTION=postgresql://user:password@localhost/db
API_TOKEN=your-secret-token
AWS_ACCESS_KEY=your-access-key

# config.yaml
content_sources:
  - name: "database"
    type: "database" 
    connection_string: "${DB_CONNECTION}"
  - name: "api"
    type: "http"
    auth_token: "${API_TOKEN}"
```

### Performance Optimization

```yaml
content_sources:
  - name: "large_database"
    type: "database"
    connection_string: "postgresql://host/db"
    query: "SELECT * FROM articles"
    
    # Batch processing
    batch_size: 1000        # Process in chunks
    max_workers: 4          # Parallel workers (reserved for future use)
    connection_pool_size: 8 # DB connection pool
    
    # Memory management
    stream_results: true    # Stream results for large datasets
    max_content_length: 50000  # Truncate very long content
    
    # Advanced performance options
    field_mapping:
      doc_id: "id"
      content: ["title", "content", "summary"]  # Concatenate multiple fields
      metadata:
        author.name: "author_name"              # Nested metadata paths
        author.email: "author_email"
        department.name: "dept_name"
```

### DuckDB Connection Testing

```yaml
content_sources:
  - name: "sec-filings"
    type: "duckdb"
    database_path: "/data/sec-parquet"
    enable_hive_partitioning: true
    
    connection_config:
      threads: 8
      memory_limit: "4GB"
      max_expression_depth: 1000
    
    queries:
      - name: "10k-filings"
        sql: "SELECT * FROM read_parquet('cik=*/filing_type=10K/*/mda.parquet', hive_partitioning=true)"
        id_columns: ["cik", "filing_type", "year"]
        content_column: "paragraph_text"
```

**Testing DuckDB Connection:**

```python
from go_doc_go.content_source.duckdb import DuckDBContentSource

config = {
    "name": "test-duckdb",
    "database_path": "/data/parquet",
    "queries": [{"name": "test", "sql": "SELECT 1 as test", "id_columns": ["test"], "content_column": "test"}]
}

source = DuckDBContentSource(config)
if source.test_connection():
    print("✅ DuckDB connection successful")
    
    # Get query information
    query_info = source.get_query_info()
    for query in query_info:
        print(f"Query '{query['name']}': {len(query['id_columns'])} ID columns")
else:
    print("❌ DuckDB connection failed")
```

### Content Filtering

```yaml
content_sources:
  - name: "filtered_content"
    type: "file"
    base_path: "/docs"
    
    # Size filters
    min_file_size: 100      # bytes
    max_file_size: 10485760 # 10MB
    
    # Date filters
    modified_after: "2024-01-01"
    modified_before: "2024-12-31"
    
    # Content filters
    content_filters:
      - type: "regex"
        pattern: "\\b(confidential|secret)\\b"
        action: "exclude"
      - type: "length"
        min_length: 50
        max_length: 100000
```

## Monitoring and Debugging

### Enable Detailed Logging

```yaml
logging:
  level: "DEBUG"
  handlers:
    - type: "file"
      filename: "ingestion.log"
    - type: "console"
      
content_sources:
  - name: "debug_source"
    type: "database"
    # ... configuration
    
    # Debug options
    log_queries: true
    log_results_sample: 5  # Log first 5 results
    validate_content: true  # Validate before processing
```

### Health Checks

```python
from go_doc_go import Config

config = Config("config.yaml") 

# Test all content sources
for source_config in config.content_sources:
    try:
        source = config.create_content_source(source_config)
        test_docs = list(source.get_documents(limit=1))
        print(f"✅ {source_config['name']}: {len(test_docs)} documents available")
    except Exception as e:
        print(f"❌ {source_config['name']}: {str(e)}")
```

## Troubleshooting and Common Issues

### Database Connection Issues

**Issue: "SQLAlchemy is required but not available"**
```bash
# Install SQLAlchemy
pip install sqlalchemy

# For specific database drivers:
pip install psycopg2-binary  # PostgreSQL
pip install mysql-connector-python  # MySQL
pip install pymysql  # MySQL alternative
```

**Issue: "MySQL connector not available, trying PyMySQL driver"**
- This is normal fallback behavior
- Install MySQL connector for better performance: `pip install mysql-connector-python`

**Issue: Database connection timeouts**
```yaml
content_sources:
  - name: "database_with_timeouts"
    connection_string: "postgresql://user:pass@host/db?connect_timeout=30"
    connection_pool_size: 5  # Reduce pool size
    stream_results: true     # Enable streaming for large results
```

**Issue: Binary content not displaying properly**
- Binary data is automatically detected and handled
- Content appears as `<binary data: N bytes>` when not UTF-8 decodable
- Use `max_content_length` to limit binary data processing

### DuckDB Issues

**Issue: "DuckDB is required but not available"**
```bash
pip install duckdb
```

**Issue: Thread configuration warnings**
```yaml
# Explicitly configure threads to avoid warnings
content_sources:
  - name: "duckdb_optimized"
    type: "duckdb"
    connection_config:
      threads: 4  # Match your CPU cores
      memory_limit: "2GB"
```

**Issue: Parquet file not found errors**
```yaml
# Ensure hive partitioning is enabled for partitioned datasets
content_sources:
  - name: "hive_parquet"
    type: "duckdb"
    enable_hive_partitioning: true
    queries:
      - sql: "SELECT * FROM read_parquet('data/*/*/*.parquet', hive_partitioning=true)"
```

### Performance Issues

**Issue: Memory usage too high**
```yaml
content_sources:
  - name: "memory_optimized"
    type: "database"
    batch_size: 100          # Reduce batch size
    stream_results: true     # Enable streaming
    max_content_length: 10000  # Limit content size
```

**Issue: Slow database queries**
```yaml
content_sources:
  - name: "optimized_queries"
    type: "database"
    query: |
      SELECT id, content, created_at
      FROM articles 
      WHERE created_at >= '2024-01-01'  -- Add WHERE clauses
      AND status = 'published'          -- Limit result set
      ORDER BY created_at DESC
      LIMIT 10000                       -- Limit total results
    
    # Use indexes on filtered columns
    # CREATE INDEX idx_articles_created_status ON articles(created_at, status);
```

### Authentication Issues

**Issue: SharePoint/Graph API authentication failures**
```yaml
# Ensure proper Azure app registration permissions
content_sources:
  - name: "sharepoint_auth"
    type: "sharepoint"
    # Use service principal authentication
    client_id: "${AZURE_CLIENT_ID}"
    client_secret: "${AZURE_CLIENT_SECRET}"
    tenant_id: "${AZURE_TENANT_ID}"
    
    # Required permissions in Azure:
    # - Sites.Read.All
    # - Files.Read.All
```

**Issue: API rate limiting**
```yaml
content_sources:
  - name: "rate_limited_api"
    type: "web"
    delay: 2.0  # Increase delay between requests
    max_pages: 100  # Limit total requests
    
    # Use authentication to get higher rate limits
    headers:
      Authorization: "Bearer ${API_TOKEN}"
```

### Content Source Discovery

**Test content source connections:**
```python
from go_doc_go.content_source.factory import get_content_source

def test_content_sources(configs):
    """Test all configured content sources."""
    for config in configs:
        try:
            source = get_content_source(config)
            
            # Test connection if available
            if hasattr(source, 'test_connection'):
                if source.test_connection():
                    print(f"✅ {config['name']}: Connection successful")
                else:
                    print(f"❌ {config['name']}: Connection failed")
            else:
                # Try to list a few documents
                docs = source.list_documents()
                print(f"✅ {config['name']}: Found {len(docs)} documents")
                
        except Exception as e:
            print(f"❌ {config['name']}: Error - {str(e)}")
```

### Debugging Tips

**Enable detailed logging:**
```python
import logging

# Enable debug logging for content sources
logging.getLogger('go_doc_go.content_source').setLevel(logging.DEBUG)

# Enable SQL query logging
logging.getLogger('sqlalchemy.engine').setLevel(logging.INFO)
```

**Check configuration validity:**
```python
def validate_config(config):
    """Validate content source configuration."""
    required_fields = ['name', 'type']
    for field in required_fields:
        if field not in config:
            raise ValueError(f"Missing required field: {field}")
    
    source_type = config['type']
    type_specific_requirements = {
        'database': ['connection_string'],
        'duckdb': ['database_path', 'queries'],
        'file': ['base_path'],
        's3': ['bucket'],
        'web': ['start_urls']
    }
    
    if source_type in type_specific_requirements:
        for field in type_specific_requirements[source_type]:
            if field not in config:
                raise ValueError(f"{source_type} source missing required field: {field}")
```

## Next Steps

- [Storage Backends](storage.md) - Choose where to store your processed data
- [Configuration Guide](configuration.md) - Advanced configuration options  
- [Scaling Guide](scaling.md) - Handle large-scale data ingestion
- [API Reference](api.md) - Programmatic access to content sources
