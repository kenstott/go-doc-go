# Optional Dependencies

This document describes the optional dependencies for various adapters and content sources in the Go-Doc-Go system. These dependencies are not required for the core functionality but enable specific integrations.

## SharePoint Integration

SharePoint integration supports two different API approaches. You can install one or both depending on your needs:

### REST API (Office365-REST-Python-Client)

```bash
pip install Office365-REST-Python-Client
```

**Use cases:**
- Traditional SharePoint authentication
- On-premises SharePoint servers
- SharePoint 2013/2016/2019
- When you need full SharePoint API access

### Microsoft Graph API

```bash
pip install msgraph-sdk azure-identity
```

**Use cases:**
- Modern cloud-based authentication
- SharePoint Online (Microsoft 365)
- When integrating with other Microsoft 365 services
- Better performance for cloud scenarios

### Configuration

Create a `.env.sharepoint` file with your credentials:

```env
# Required
SHAREPOINT_TENANT_ID=your-tenant-id
SHAREPOINT_CLIENT_ID=your-client-id
SHAREPOINT_CLIENT_SECRET=your-client-secret
SHAREPOINT_SITE_URL=https://yoursite.sharepoint.com

# Optional
SHAREPOINT_API_TYPE=auto  # Options: 'rest', 'graph', 'auto'
SHAREPOINT_TEST_PATH=/Shared Documents
SHAREPOINT_MAX_FILE_SIZE=104857600  # 100MB in bytes
SHAREPOINT_TIMEOUT=30
```

### Usage Example

```python
from go_doc_go.adapter.sharepoint import SharePointAdapter

# Adapter will auto-detect which API to use based on installed libraries
adapter = SharePointAdapter({
    'tenant_id': 'your-tenant-id',
    'client_id': 'your-client-id', 
    'client_secret': 'your-client-secret',
    'site_url': 'https://yoursite.sharepoint.com',
    'api_type': 'auto'  # or 'rest' or 'graph'
})

# Retrieve a document
content = adapter.get_content({
    'source': 'sharepoint://yoursite.sharepoint.com/Shared Documents/report.docx'
})
```

## S3 Integration

### AWS SDK (boto3)

```bash
pip install boto3
```

**Required for:**
- Amazon S3 storage
- S3-compatible storage (MinIO, Wasabi, etc.)
- AWS credentials via IAM roles or access keys

### Configuration

```env
AWS_ACCESS_KEY_ID=your-access-key
AWS_SECRET_ACCESS_KEY=your-secret-key
AWS_REGION=us-west-2
S3_ENDPOINT_URL=https://s3.amazonaws.com  # Optional, for S3-compatible services
```

## MongoDB Integration

### PyMongo

```bash
pip install pymongo
```

**Required for:**
- MongoDB document storage
- GridFS for large files
- MongoDB Atlas cloud database

### Configuration

```env
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=documents
MONGODB_COLLECTION=content
```

## Confluence Integration

### Atlassian Python API

```bash
pip install atlassian-python-api
```

**Required for:**
- Atlassian Confluence wiki integration
- Confluence Cloud or Server
- Page and attachment retrieval

### Configuration

```env
CONFLUENCE_URL=https://your-domain.atlassian.net
CONFLUENCE_USERNAME=your-email@example.com
CONFLUENCE_API_TOKEN=your-api-token
CONFLUENCE_SPACE_KEY=DOCS
```

## ServiceNow Integration

### PySnow

```bash
pip install pysnow
```

**Required for:**
- ServiceNow ITSM integration
- Knowledge base articles
- Incident and change management documents

### Configuration

```env
SERVICENOW_INSTANCE=your-instance.service-now.com
SERVICENOW_USERNAME=your-username
SERVICENOW_PASSWORD=your-password
```

## Document Parsing Dependencies

### PDF Processing

```bash
pip install PyMuPDF  # or pip install pymupdf
```

**Required for:**
- PDF text extraction
- PDF metadata extraction
- PDF image extraction

### Office Documents

```bash
pip install python-docx  # Word documents
pip install openpyxl     # Excel spreadsheets
pip install python-pptx  # PowerPoint presentations
```

**Required for:**
- Microsoft Office document parsing
- OpenXML format support
- Document structure extraction

### Web Content

```bash
pip install beautifulsoup4  # HTML parsing
pip install lxml            # XML parsing (faster than built-in)
pip install python-dateutil # Date parsing
```

**Required for:**
- HTML content parsing
- XML document processing
- Enhanced date/time handling

## Installation Profiles

### Minimal Installation

```bash
# Core only - file system support
pip install go-doc-go
```

### Cloud Storage

```bash
# S3 and cloud storage
pip install go-doc-go[s3]
# or manually:
pip install boto3
```

### Enterprise Integration

```bash
# SharePoint, Confluence, ServiceNow
pip install go-doc-go[enterprise]
# or manually:
pip install Office365-REST-Python-Client atlassian-python-api pysnow
```

### Full Installation

```bash
# All optional dependencies
pip install go-doc-go[all]
```

## Checking Available Features

You can check which features are available in your installation:

```python
from go_doc_go.adapter.sharepoint import OFFICE365_AVAILABLE, MSGRAPH_AVAILABLE
from go_doc_go.adapter.s3 import BOTO3_AVAILABLE

print(f"SharePoint REST API: {OFFICE365_AVAILABLE}")
print(f"SharePoint Graph API: {MSGRAPH_AVAILABLE}")
print(f"S3 Support: {BOTO3_AVAILABLE}")
```

## Handling Missing Dependencies

The system gracefully handles missing optional dependencies:

1. **At Import Time**: Adapters check for required libraries and set availability flags
2. **At Runtime**: Clear error messages indicate which library needs to be installed
3. **In Tests**: Tests are automatically skipped if required dependencies are missing

Example error handling:

```python
try:
    adapter = SharePointAdapter(config)
except ImportError as e:
    print(f"SharePoint not available: {e}")
    print("Install with: pip install Office365-REST-Python-Client")
```

## Development Setup

For development and testing, install all optional dependencies:

```bash
# Clone the repository
git clone https://github.com/your-org/go-doc-go.git
cd go-doc-go

# Create virtual environment
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install in development mode with all dependencies
pip install -e ".[all,dev]"

# Run tests (will skip tests for missing dependencies)
pytest -xvs
```

## Troubleshooting

### SharePoint Issues

1. **Authentication Failures**
   - Verify tenant ID, client ID, and client secret
   - Ensure app registration has correct permissions
   - Check if MFA is blocking authentication

2. **API Selection**
   - Set `SHAREPOINT_API_TYPE=rest` to force REST API
   - Set `SHAREPOINT_API_TYPE=graph` to force Graph API
   - Use `auto` to let the system choose based on available libraries

### S3 Issues

1. **Access Denied**
   - Check IAM permissions for the S3 bucket
   - Verify credentials are correct
   - Ensure bucket policy allows access

2. **Region Issues**
   - Specify region explicitly in configuration
   - Check if bucket is in a different region

### MongoDB Issues

1. **Connection Timeouts**
   - Check network connectivity
   - Verify MongoDB is running
   - Check firewall rules

2. **Authentication**
   - Ensure user has correct database permissions
   - Check authentication mechanism compatibility