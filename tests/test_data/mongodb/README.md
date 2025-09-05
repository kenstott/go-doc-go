# MongoDB Test Data

This directory contains sample MongoDB documents and initialization scripts for testing.

## Files

### sample_documents.json
Sample MongoDB documents in extended JSON format including:
- Article documents with text content
- Product documents with nested specifications
- User documents with profile and settings
- Order documents with references to users and products  
- Log entries with metadata

### init/
Directory for MongoDB initialization scripts that run when the Docker container starts.

## Document Types

### Articles
- Text content documents
- Author references
- Tags and categories
- Timestamps

### Products  
- Nested specifications
- Inventory tracking
- Price and categories

### Users
- Profile information
- Nested address data
- Account settings
- Document references

### Orders
- Customer references
- Item arrays
- Shipping information
- Status tracking

### Logs
- Service logs
- Request metadata
- Performance metrics

## Usage

These documents are automatically loaded when running MongoDB tests:

```bash
# Start MongoDB with test data
docker-compose -f docker-compose.test.yml up mongodb

# Import sample documents manually
mongoimport --host localhost --db test_db --collection test_collection \
  --file tests/test_data/mongodb/sample_documents.json --jsonArray
```

## BSON Types

Documents demonstrate various BSON types:
- ObjectId (`$oid`)
- Date (`$date`)
- Arrays
- Nested documents
- Numbers (int, float)
- Strings
- Booleans