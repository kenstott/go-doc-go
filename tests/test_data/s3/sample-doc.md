# Sample Documentation

## Introduction

This is a sample document used for testing S3 integration with the document management system.

## Features

The system supports:
- Markdown parsing with [internal links](./nested/details.md)
- Code highlighting
- Table support

## Code Example

```python
def process_document(doc_id: str) -> Dict[str, Any]:
    """Process a document by ID."""
    return {"status": "processed", "id": doc_id}
```

## References

- [Configuration Guide](./config.yaml)
- [Data Schema](./data.json)
- [Employee Records](./employees.csv)