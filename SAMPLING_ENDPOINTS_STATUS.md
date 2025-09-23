# Sampling Endpoints Status

## Summary
The sampling endpoints have been successfully added to the primary server at `/api/sampling/*` with an MCP passthrough server for integration. However, they require a PostgreSQL analytics database with specific views to function properly.

## Completed Work

### 1. API Routes Added (`/api/sampling/`)
- **POST /api/sampling/elements** - Sample elements with flexible filtering
- **POST /api/sampling/corpus-stats** - Get corpus statistics
- **POST /api/sampling/documents** - Sample documents
- **POST /api/sampling/custom-query** - Execute custom SQL queries
- **GET /api/sampling/schema** - Get schema information
- **POST /api/sampling/ontology-sample** - Comprehensive sampling for ontology generation

### 2. MCP Passthrough Server
Created `src/go_doc_go/mcp/mcp_passthrough_server.py` that:
- Acts as a passthrough to the primary server's sampling endpoints
- Provides MCP tool functions for database sampling
- Can be configured with any primary server URL

### 3. Files Created/Modified
- `src/go_doc_go/api/sampling_routes.py` - Flask Blueprint with sampling endpoints
- `src/go_doc_go/mcp/mcp_passthrough_server.py` - MCP passthrough server
- `src/go_doc_go/mcp/database_sampler.py` - Generic database sampler class
- `src/go_doc_go/server.py` - Registered sampling Blueprint

## Requirements for Operation

### Database Prerequisites
The sampling endpoints require a PostgreSQL analytics database with:

1. **Database View**: `element_document_enriched`
   - Joins elements with document attributes
   - Includes columns like:
     - `element_id`, `doc_id`, `element_type`
     - `structural_name`, `structural_path`
     - `content_preview`, `has_temporal_value`
     - `format_type`, `document_category`

2. **Environment Variables**:
   ```bash
   POSTGRES_HOST=localhost
   POSTGRES_PORT=5432
   POSTGRES_DB=doculyzer
   POSTGRES_USER=postgres
   POSTGRES_PASSWORD=postgres
   ```

### Pipeline Context
The sampling functionality is designed to work with pipeline-specific analytics databases. Each pipeline execution creates its own analytics storage, and the sampling should target a specific pipeline's data.

## Current Limitations

1. **No Default Database**: The system no longer uses `get_document_database()` - database connections are pipeline-specific
2. **View Dependency**: Requires the `element_document_enriched` view which joins elements and documents
3. **PostgreSQL Required**: Current implementation uses PostgreSQL-specific SQL queries

## Usage Examples

### Via MCP Passthrough
```python
from go_doc_go.mcp.mcp_passthrough_server import create_mcp_tools

# Create MCP tools
tools = create_mcp_tools("http://localhost:5002")

# Get corpus statistics
stats = tools["get_corpus_stats"]()

# Sample elements
elements = tools["sample_elements"](
    filters='{"element_type": "xml_element"}',
    limit=100
)
```

### Direct API Calls
```python
import requests

# Sample elements with filters
response = requests.post(
    "http://localhost:5002/api/sampling/elements",
    json={
        "filters": {"element_type": "xml_element"},
        "limit": 100,
        "stratify_by": "structural_name"
    }
)
```

## Next Steps

To make the sampling endpoints fully functional:

1. **Option A: Use Existing Pipeline Database**
   - Modify sampling routes to accept a pipeline_id parameter
   - Use the pipeline's configured analytics backend
   - Query data directly from the pipeline's storage

2. **Option B: Create Dedicated Analytics Database**
   - Set up a PostgreSQL database with required views
   - Configure connection in environment variables
   - Ensure data is populated from pipeline executions

3. **Option C: Use Parquet Data Lake Directly**
   - Modify sampler to use DuckDB for querying parquet files
   - Remove dependency on PostgreSQL views
   - Work directly with the data lake structure

## Testing

A test script is available at `test_sampling_endpoints.py`. Once the database prerequisites are met, run:

```bash
python test_sampling_endpoints.py
```

This will test all sampling endpoints and verify functionality.