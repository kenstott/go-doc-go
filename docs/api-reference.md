# Go-Doc-Go API Reference

## Overview

Go-Doc-Go provides a comprehensive REST API for document processing, search, pipeline management, and system configuration. The API is built with Flask and supports both JSON requests/responses and WebSocket connections for real-time updates.

## Base URL

```
http://localhost:5002/api
```

## Authentication

Currently, the API does not require authentication for local deployments. For production deployments, authentication should be configured based on your security requirements.

## Content Types

- **Request**: `application/json`
- **Response**: `application/json`
- **File Upload**: `multipart/form-data`

---

## Core API Endpoints

### Health & Status

#### Health Check
```http
GET /health
```

Returns the health status of the API server.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

#### API Information
```http
GET /api/info
```

Returns API version and available endpoints information.

**Response:**
```json
{
  "version": "1.0.0",
  "endpoints": {
    "search": "/api/search",
    "pipelines": "/api/pipelines",
    "config": "/api/config",
    "ontologies": "/api/ontologies"
  }
}
```

---

## Search API

### Structured Search
```http
POST /api/search/structured
```

Performs a structured search with advanced query capabilities.

**Request Body:**
```json
{
  "queries": [
    {
      "query": "revenue growth",
      "columns": ["content", "metadata"],
      "doc_id_columns": ["doc_id"],
      "element_id_columns": ["element_id"],
      "top_k": 10,
      "min_score": 0.7
    }
  ],
  "filters": {
    "doc_type": "earnings_call",
    "date_range": {
      "start": "2024-01-01",
      "end": "2024-12-31"
    }
  },
  "aggregations": {
    "group_by": "doc_type",
    "metrics": ["count", "avg_score"]
  }
}
```

**Response:**
```json
{
  "results": [
    {
      "doc_id": "doc_123",
      "element_id": "elem_456",
      "content": "Revenue grew by 25% year-over-year...",
      "score": 0.85,
      "metadata": {
        "doc_type": "earnings_call",
        "company": "TechCorp"
      }
    }
  ],
  "aggregations": {
    "doc_type_counts": {
      "earnings_call": 15,
      "annual_report": 8
    }
  },
  "total_results": 23
}
```

### Simple Structured Search
```http
POST /api/search/structured/simple
```

A simplified version of structured search for basic queries.

**Request Body:**
```json
{
  "query": "machine learning applications",
  "limit": 20,
  "min_score": 0.5
}
```

### Standard Search
```http
POST /api/search
```

Basic semantic search across all documents.

**Request Body:**
```json
{
  "query": "artificial intelligence trends",
  "top_k": 10,
  "filters": {
    "doc_type": ["research_paper", "article"]
  }
}
```

### Advanced Search
```http
POST /api/search/advanced
```

Advanced search with multiple query types and complex filters.

**Request Body:**
```json
{
  "semantic_query": "AI ethics and governance",
  "keyword_query": "regulation compliance",
  "filters": {
    "date_range": {
      "field": "published_date",
      "start": "2023-01-01",
      "end": "2024-12-31"
    },
    "categories": ["technology", "law"]
  },
  "boost_recent": true,
  "include_related": true
}
```

### Document Sources
```http
POST /api/search/sources
```

Retrieves unique document sources based on search criteria.

**Request Body:**
```json
{
  "filters": {
    "doc_type": "financial_report"
  },
  "group_by": "source",
  "include_metadata": true
}
```

---

## Pipeline Management API

### List Pipelines
```http
GET /api/pipelines
```

Lists all configured pipelines.

**Query Parameters:**
- `active_only` (boolean): Filter active pipelines only (default: true)
- `tags` (string): Comma-separated list of tags to filter by
- `limit` (integer): Maximum number of results (default: 50)

**Response:**
```json
{
  "pipelines": [
    {
      "id": 1,
      "name": "SEC Filings Pipeline",
      "description": "Processes SEC filing documents",
      "tags": ["financial", "sec"],
      "is_active": true,
      "version": 3,
      "created_at": "2024-01-10T08:00:00Z"
    }
  ],
  "total": 5
}
```

### Create Pipeline
```http
POST /api/pipelines
```

Creates a new processing pipeline.

**Request Body:**
```json
{
  "name": "New Pipeline",
  "description": "Pipeline for processing documents",
  "config_yaml": "content_sources:\n  - type: file\n    path: /data",
  "tags": ["production", "daily"],
  "template_name": "standard_processing"
}
```

### Get Pipeline
```http
GET /api/pipelines/{pipeline_id}
```

Retrieves a specific pipeline configuration.

### Update Pipeline
```http
PUT /api/pipelines/{pipeline_id}
```

Updates a pipeline configuration with optimistic locking.

**Request Body:**
```json
{
  "name": "Updated Pipeline Name",
  "config_yaml": "...",
  "is_active": true,
  "expected_version": 3
}
```

### Delete Pipeline
```http
DELETE /api/pipelines/{pipeline_id}
```

Deletes a pipeline and all its execution history.

### Clone Pipeline
```http
POST /api/pipelines/{pipeline_id}/clone
```

Creates a copy of an existing pipeline.

**Request Body:**
```json
{
  "name": "Cloned Pipeline",
  "created_by": "username"
}
```

### Execute Pipeline
```http
POST /api/pipelines/{pipeline_id}/execute
```

Starts pipeline execution.

**Request Body:**
```json
{
  "worker_count": 4,
  "documents_total": 1000,
  "execution_metadata": {
    "triggered_by": "scheduler",
    "priority": "high"
  }
}
```

**Response:**
```json
{
  "message": "Pipeline execution started",
  "execution": {
    "run_id": "run_20240115_103000_abc123",
    "pipeline_id": 1,
    "status": "running",
    "started_at": "2024-01-15T10:30:00Z"
  }
}
```

### Get Pipeline Executions
```http
GET /api/pipelines/{pipeline_id}/executions
```

Retrieves execution history for a pipeline.

**Query Parameters:**
- `limit` (integer): Maximum number of results (default: 20)

### Export Pipeline
```http
GET /api/pipelines/{pipeline_id}/export
```

Exports pipeline configuration as a YAML file.

### Import Pipeline
```http
POST /api/pipelines/import
```

Imports a pipeline from a YAML file.

**Form Data:**
- `file`: YAML file to import
- `name`: Optional override for pipeline name
- `created_by`: Creator username

---

## Execution Monitoring API

### Get Execution Status
```http
GET /api/pipelines/executions/{run_id}/status
```

Gets real-time execution status including progress data.

**Response:**
```json
{
  "execution": {
    "run_id": "run_20240115_103000_abc123",
    "status": "running",
    "documents_processed": 450,
    "documents_total": 1000,
    "progress_percentage": 45.0
  },
  "progress": {
    "current_document": "doc_789",
    "processing_rate": 15.5,
    "estimated_completion": "2024-01-15T11:00:00Z"
  },
  "recent_events": [
    {
      "timestamp": "2024-01-15T10:35:00Z",
      "event_type": "document_processed",
      "details": "Successfully processed doc_456"
    }
  ]
}
```

### Update Execution Progress
```http
PUT /api/pipelines/executions/{run_id}/progress
```

Updates execution progress (typically called by workers).

**Request Body:**
```json
{
  "documents_processed": 500,
  "documents_total": 1000,
  "status": "running",
  "errors_count": 2,
  "warnings_count": 5
}
```

### Cancel Execution
```http
POST /api/pipelines/executions/{run_id}/cancel
```

Cancels an active pipeline execution.

### Get Execution Logs
```http
GET /api/pipelines/executions/{run_id}/logs
```

Retrieves execution logs with pagination.

**Query Parameters:**
- `start_index` (integer): Starting index for logs (default: 0)

**Response:**
```json
{
  "run_id": "run_20240115_103000_abc123",
  "logs": [
    {
      "timestamp": "2024-01-15T10:30:00Z",
      "level": "INFO",
      "message": "Starting pipeline execution",
      "module": "go_doc_go.pipeline"
    }
  ],
  "total_count": 250,
  "start_index": 0,
  "has_more": true
}
```

### List Active Executions
```http
GET /api/pipelines/executions/active
```

Lists all currently active executions across all pipelines.

### Get Pipeline Dashboard
```http
GET /api/pipelines/dashboard
```

Retrieves comprehensive dashboard data for all pipelines.

**Response:**
```json
{
  "pipeline_stats": {
    "total_pipelines": 10,
    "active_pipelines": 7,
    "inactive_pipelines": 3,
    "templates_count": 5
  },
  "execution_stats": {
    "total_executions": 150,
    "active_executions": 2,
    "completed_executions": 140,
    "failed_executions": 8,
    "cancelled_executions": 0
  },
  "recent_activity": [...],
  "system_info": {
    "total_active_monitors": 2,
    "database_path": "pipeline_config.db"
  }
}
```

---

## Configuration API

### Get Configuration
```http
GET /api/config
```

Retrieves the current system configuration.

**Response:**
```json
{
  "storage": {
    "backend": "postgresql",
    "connection": {
      "host": "localhost",
      "port": 5432,
      "database": "go_doc_go"
    }
  },
  "embedding": {
    "model": "sentence-transformers/all-MiniLM-L6-v2",
    "dimensions": 384
  },
  "content_sources": [
    {
      "name": "s3_documents",
      "type": "s3",
      "config": {
        "bucket": "document-bucket",
        "region": "us-east-1"
      }
    }
  ]
}
```

### Update Configuration
```http
PUT /api/config
```

Updates system configuration.

**Request Body:**
```json
{
  "embedding": {
    "model": "all-mpnet-base-v2",
    "dimensions": 768
  }
}
```

### Validate Configuration
```http
POST /api/config/validate
```

Validates configuration without applying changes.

**Request Body:**
```json
{
  "config": {
    "storage": {
      "backend": "sqlite",
      "path": "/data/documents.db"
    }
  }
}
```

**Response:**
```json
{
  "valid": true,
  "warnings": [],
  "errors": []
}
```

---

## Ontology Management API

### List Ontologies
```http
GET /api/ontologies
```

Lists all available ontologies.

**Response:**
```json
{
  "ontologies": [
    {
      "name": "financial_markets",
      "description": "Financial markets domain ontology",
      "version": "1.0.0",
      "entity_types": ["company", "executive", "metric"],
      "relationship_types": ["employs", "reports", "owns"]
    }
  ]
}
```

### Get Ontology
```http
GET /api/ontologies/{name}
```

Retrieves a specific ontology definition.

### Update Ontology
```http
PUT /api/ontologies/{name}
```

Updates an ontology definition.

**Request Body:**
```json
{
  "description": "Updated description",
  "entity_types": [...],
  "relationship_types": [...],
  "rules": [...]
}
```

### Activate Domain
```http
POST /api/domain/{name}/activate
```

Activates a domain ontology for entity extraction.

### Deactivate Domain
```http
POST /api/domain/{name}/deactivate
```

Deactivates a domain ontology.

### Get Active Domains
```http
GET /api/domain/active
```

Lists currently active domain ontologies.

---

## Analytics Registry API

### Get Analytics Registry
```http
GET /api/analytics/registry
```

Retrieves available analytics backends and their capabilities.

**Response:**
```json
{
  "backends": {
    "duckdb": {
      "name": "DuckDB",
      "type": "sql",
      "capabilities": ["sql_query", "parquet", "csv_export"],
      "performance": {
        "query_speed": "fast",
        "scalability": "medium"
      }
    },
    "pandas": {
      "name": "Pandas",
      "type": "dataframe",
      "capabilities": ["statistical_analysis", "pivot_tables"],
      "performance": {
        "memory_usage": "high",
        "processing_speed": "medium"
      }
    }
  }
}
```

### Get Analytics Backend Details
```http
GET /api/analytics/registry/{backend_name}
```

Retrieves detailed information about a specific analytics backend.

### Recommend Analytics Backend
```http
POST /api/analytics/recommend
```

Recommends the best analytics backend based on requirements.

**Request Body:**
```json
{
  "data_size": "large",
  "query_type": "complex_sql",
  "export_formats": ["parquet", "csv"],
  "performance_priority": "speed"
}
```

**Response:**
```json
{
  "recommended": "duckdb",
  "score": 0.95,
  "reasons": [
    "Best performance for large datasets",
    "Native SQL support",
    "Efficient parquet handling"
  ],
  "alternatives": [
    {
      "backend": "clickhouse",
      "score": 0.88
    }
  ]
}
```

---

## WebSocket Endpoints

### Progress Monitoring
```
ws://localhost:5002/ws/progress
```

Connect to receive real-time progress updates for all executions.

**Message Format:**
```json
{
  "type": "progress_event",
  "run_id": "run_20240115_103000_abc123",
  "data": {
    "documents_processed": 100,
    "current_document": "doc_456",
    "status": "processing"
  },
  "timestamp": "2024-01-15T10:35:00Z"
}
```

### Execution-Specific Progress
```
ws://localhost:5002/ws/progress/{run_id}
```

Connect to receive updates for a specific execution.

### Pipeline Progress
```
ws://localhost:5002/ws/pipelines/{pipeline_id}/progress
```

Connect to receive updates for all executions of a specific pipeline.

---

## Error Responses

All endpoints follow a consistent error response format:

```json
{
  "error": "Error Type",
  "message": "Detailed error message",
  "details": {
    "field": "Additional context"
  }
}
```

### Common HTTP Status Codes

- `200 OK`: Successful request
- `201 Created`: Resource successfully created
- `400 Bad Request`: Invalid request parameters
- `404 Not Found`: Resource not found
- `409 Conflict`: Concurrency conflict (e.g., version mismatch)
- `422 Unprocessable Entity`: Valid request but semantic errors
- `500 Internal Server Error`: Server-side error

---

## Rate Limiting

Currently, no rate limiting is implemented. For production deployments, consider implementing rate limiting based on your requirements.

---

## CORS Configuration

The API supports CORS for cross-origin requests. Default configuration allows all origins in development mode. For production, configure specific allowed origins.

---

## Examples

### Example: Full Document Processing Pipeline

```python
import requests
import json

# 1. Create a pipeline
pipeline_data = {
    "name": "Document Processing Pipeline",
    "description": "Processes documents from S3",
    "config_yaml": """
content_sources:
  - type: s3
    config:
      bucket: my-documents
      prefix: /inbox/
processing:
  parsers:
    pdf: enabled
    docx: enabled
  embeddings:
    enabled: true
    model: all-MiniLM-L6-v2
""",
    "tags": ["production", "s3"]
}

response = requests.post(
    "http://localhost:5002/api/pipelines",
    json=pipeline_data
)
pipeline = response.json()["pipeline"]

# 2. Execute the pipeline
execution_data = {
    "worker_count": 4,
    "documents_total": 100
}

response = requests.post(
    f"http://localhost:5002/api/pipelines/{pipeline['id']}/execute",
    json=execution_data
)
execution = response.json()["execution"]

# 3. Monitor execution progress
import time

while True:
    response = requests.get(
        f"http://localhost:5002/api/pipelines/executions/{execution['run_id']}/status"
    )
    status = response.json()
    
    print(f"Progress: {status['execution']['documents_processed']}/{status['execution']['documents_total']}")
    
    if status['execution']['status'] in ['completed', 'failed', 'cancelled']:
        break
    
    time.sleep(5)

# 4. Search processed documents
search_data = {
    "query": "revenue growth",
    "top_k": 10,
    "filters": {
        "pipeline_run_id": execution['run_id']
    }
}

response = requests.post(
    "http://localhost:5002/api/search",
    json=search_data
)
results = response.json()["results"]

for result in results:
    print(f"Score: {result['score']}, Content: {result['content'][:100]}...")
```

### Example: WebSocket Progress Monitoring

```javascript
// Connect to WebSocket for real-time updates
const ws = new WebSocket('ws://localhost:5002/ws/progress');

ws.onopen = () => {
    console.log('Connected to progress monitoring');
};

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    
    if (data.type === 'progress_event') {
        console.log(`Pipeline ${data.run_id}: ${data.data.documents_processed} documents processed`);
        updateProgressBar(data.data);
    }
};

ws.onerror = (error) => {
    console.error('WebSocket error:', error);
};

ws.onclose = () => {
    console.log('Disconnected from progress monitoring');
};
```

---

## API Versioning

The current API version is v1. Future versions will be accessible via versioned endpoints (e.g., `/api/v2/`).

---

## Support

For API support and questions:
- Documentation: [https://github.com/your-org/go-doc-go/docs](https://github.com/your-org/go-doc-go/docs)
- Issues: [https://github.com/your-org/go-doc-go/issues](https://github.com/your-org/go-doc-go/issues)