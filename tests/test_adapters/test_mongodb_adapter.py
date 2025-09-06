"""
Tests for MongoDB adapter.
"""

import pytest
import json
from unittest.mock import patch, MagicMock, Mock
from typing import Dict, Any
from bson import ObjectId

from go_doc_go.adapter.mongodb import MongoDBAdapter
from conftest import requires_pymongo, requires_docker


class TestMongoDBAdapterUnit:
    """Unit tests for MongoDBAdapter without real MongoDB."""
    
    def test_adapter_initialization_without_pymongo(self):
        """Test that adapter raises error when pymongo is not available."""
        with patch('go_doc_go.adapter.mongodb.PYMONGO_AVAILABLE', False):
            with pytest.raises(ImportError, match="pymongo is required"):
                MongoDBAdapter()
    
    @patch('go_doc_go.adapter.mongodb.pymongo')
    def test_adapter_initialization_with_config(self, mock_pymongo):
        """Test adapter initialization with custom configuration."""
        config = {
            "connection_string": "mongodb://localhost:27017/",
            "database": "test_db",
            "collection": "test_collection",
            "username": "testuser",
            "password": "testpass",
            "auth_source": "admin"
        }
        
        with patch('go_doc_go.adapter.mongodb.PYMONGO_AVAILABLE', True):
            adapter = MongoDBAdapter(config)
            
            assert adapter.default_connection_string == "mongodb://localhost:27017/"
            assert adapter.default_database == "test_db"
            assert adapter.default_collection == "test_collection"
            assert adapter.username == "testuser"
            assert adapter.password == "testpass"
            assert adapter.auth_source == "admin"
    
    def test_supports_location_mongodb_uri(self):
        """Test that adapter correctly identifies MongoDB URIs."""
        with patch('go_doc_go.adapter.mongodb.PYMONGO_AVAILABLE', True):
            adapter = MongoDBAdapter()
            
            # Valid MongoDB URIs
            assert adapter.supports_location({"source": "mongodb://localhost/db/collection"}) is True
            assert adapter.supports_location({"source": "mongodb://host:27017/db/collection/doc_id"}) is True
            
            # Invalid URIs
            assert adapter.supports_location({"source": "http://example.com"}) is False
            assert adapter.supports_location({"source": "/local/path"}) is False
            assert adapter.supports_location({"source": ""}) is False
    
    def test_parse_mongodb_uri(self):
        """Test MongoDB URI parsing."""
        # Standard MongoDB URI
        parsed = MongoDBAdapter._parse_mongodb_uri("mongodb://localhost/test_db/test_collection")
        assert parsed["connection_string"] == "mongodb://localhost"
        assert parsed["database"] == "test_db"
        assert parsed["collection"] == "test_collection"
        
        # URI with document ID
        parsed = MongoDBAdapter._parse_mongodb_uri("mongodb://localhost/test_db/test_collection/507f1f77bcf86cd799439011")
        assert parsed["connection_string"] == "mongodb://localhost"
        assert parsed["database"] == "test_db"
        assert parsed["collection"] == "test_collection"
        assert parsed["document_id"] == "507f1f77bcf86cd799439011"
        
        # URI with document ID and field path
        parsed = MongoDBAdapter._parse_mongodb_uri("mongodb://localhost/test_db/test_collection/doc123/metadata/author")
        assert parsed["connection_string"] == "mongodb://localhost"
        assert parsed["database"] == "test_db"
        assert parsed["collection"] == "test_collection"
        assert parsed["document_id"] == "doc123"
        assert parsed["field_path"] == "metadata/author"
    
    def test_mask_connection_string(self):
        """Test connection string masking for security."""
        # With credentials
        masked = MongoDBAdapter._mask_connection_string("mongodb://user:password@localhost:27017/")
        assert masked == "mongodb://****:****@localhost:27017/"
        assert "password" not in masked
        
        # Without credentials
        masked = MongoDBAdapter._mask_connection_string("mongodb://localhost:27017/")
        assert masked == "mongodb://localhost:27017/"
    
    def test_build_query(self):
        """Test query building from parsed URI data."""
        # With valid ObjectId
        parsed_data = {"document_id": "507f1f77bcf86cd799439011"}
        query = MongoDBAdapter._build_query(parsed_data)
        assert isinstance(query["_id"], ObjectId)
        assert str(query["_id"]) == "507f1f77bcf86cd799439011"
        
        # With string ID
        parsed_data = {"document_id": "custom-string-id"}
        query = MongoDBAdapter._build_query(parsed_data)
        assert query["_id"] == "custom-string-id"
        
        # Without document ID
        parsed_data = {}
        query = MongoDBAdapter._build_query(parsed_data)
        assert query == {}
    
    def test_extract_field(self):
        """Test field extraction with dot notation."""
        document = {
            "_id": "123",
            "name": "Test",
            "metadata": {
                "author": "John Doe",
                "tags": ["tag1", "tag2"],
                "nested": {
                    "value": "deep value"
                }
            },
            "array": [
                {"index": 0, "value": "first"},
                {"index": 1, "value": "second"}
            ]
        }
        
        # No field path - return full document
        value, path = MongoDBAdapter._extract_field(document, {})
        assert value == document
        assert path is None
        
        # Simple field
        value, path = MongoDBAdapter._extract_field(document, {"field_path": "name"})
        assert value == "Test"
        assert path == "name"
        
        # Nested field with dot notation
        value, path = MongoDBAdapter._extract_field(document, {"field_path": "metadata.author"})
        assert value == "John Doe"
        assert path == "metadata.author"
        
        # Deep nested field
        value, path = MongoDBAdapter._extract_field(document, {"field_path": "metadata.nested.value"})
        assert value == "deep value"
        assert path == "metadata.nested.value"
        
        # Array access
        value, path = MongoDBAdapter._extract_field(document, {"field_path": "array[0].value"})
        assert value == "first"
        assert path == "array[0].value"
        
        # Array access with index
        value, path = MongoDBAdapter._extract_field(document, {"field_path": "metadata.tags[1]"})
        assert value == "tag2"
        assert path == "metadata.tags[1]"
    
    @patch('go_doc_go.adapter.mongodb.MongoClient')
    def test_get_content_from_mongodb(self, mock_mongo_client):
        """Test getting content from MongoDB."""
        # Setup mock
        mock_client = MagicMock()
        mock_db = MagicMock()
        mock_collection = MagicMock()
        
        mock_mongo_client.return_value = mock_client
        mock_client.__getitem__.return_value = mock_db
        mock_db.__getitem__.return_value = mock_collection
        
        test_doc = {
            "_id": ObjectId("507f1f77bcf86cd799439011"),
            "title": "Test Document",
            "content": "Test content",
            "metadata": {"author": "Test Author"}
        }
        
        mock_collection.find_one.return_value = test_doc
        mock_client.admin.command.return_value = {"ok": 1}
        
        with patch('go_doc_go.adapter.mongodb.PYMONGO_AVAILABLE', True):
            adapter = MongoDBAdapter()
            adapter._get_client = Mock(return_value=mock_client)
            
            result = adapter.get_content({"source": "mongodb://localhost/test_db/test_collection/507f1f77bcf86cd799439011"})
            
            assert "content" in result
            assert "content_type" in result
            assert result["content_type"] == "json"
            assert "metadata" in result
            assert result["metadata"]["database"] == "test_db"
            assert result["metadata"]["collection"] == "test_collection"


@requires_pymongo
@requires_docker
class TestMongoDBAdapterIntegration:
    """Integration tests for MongoDBAdapter with real MongoDB."""
    
    def test_get_content_from_mongodb(self, mongodb_collection, insert_test_documents, sample_mongodb_documents):
        """Test getting content from actual MongoDB instance."""
        # Insert test document
        doc = sample_mongodb_documents[0]
        doc_ids = insert_test_documents(doc)
        doc_id = doc_ids[0]
        
        # Create adapter with correct connection string
        config = {
            "connection_string": "mongodb://admin:admin123@localhost:27017/",
            "database": "test_db",
            "collection": "test_collection"
        }
        adapter = MongoDBAdapter(config)
        
        # Get content
        source = f"mongodb://localhost/test_db/test_collection/{str(doc_id)}"
        result = adapter.get_content({"source": source})
        
        assert "content" in result
        content = json.loads(result["content"])
        assert content["name"] == "Document 1"
        assert content["type"] == "article"
        assert result["content_type"] == "json"
        assert result["metadata"]["document_id"] == str(doc_id)
    
    def test_get_field_from_document(self, mongodb_collection, insert_test_documents, sample_mongodb_documents):
        """Test getting specific field from MongoDB document."""
        # Insert test document
        doc = sample_mongodb_documents[1]  # Document with sections
        doc_ids = insert_test_documents(doc)
        doc_id = doc_ids[0]
        
        adapter = MongoDBAdapter({
            "connection_string": "mongodb://admin:admin123@localhost:27017/",
            "database": "test_db",
            "collection": "test_collection"
        })
        
        # Get specific field
        source = f"mongodb://localhost/test_db/test_collection/{str(doc_id)}/sections/0/title"
        result = adapter.get_content({"source": source})
        
        content = result["content"]
        assert content == "Introduction"
        assert result["metadata"]["field_path"] == "sections/0/title"
    
    def test_get_nested_field(self, mongodb_collection, insert_test_documents, sample_mongodb_documents):
        """Test getting deeply nested field."""
        # Insert document with nested structure
        doc = sample_mongodb_documents[2]
        doc_ids = insert_test_documents(doc)
        doc_id = doc_ids[0]
        
        adapter = MongoDBAdapter({
            "connection_string": "mongodb://admin:admin123@localhost:27017/",
            "database": "test_db",
            "collection": "test_collection"
        })
        
        # Get deeply nested field
        source = f"mongodb://localhost/test_db/test_collection/{str(doc_id)}/nested/level1/level2/level3"
        result = adapter.get_content({"source": source})
        
        assert result["content"] == "deeply nested value"
        assert result["metadata"]["field_type"] == "str"
    
    def test_get_metadata_without_content(self, mongodb_collection, insert_test_documents, sample_mongodb_documents):
        """Test getting metadata without retrieving full document content."""
        # Insert test document
        doc = sample_mongodb_documents[0]
        doc_ids = insert_test_documents(doc)
        doc_id = doc_ids[0]
        
        adapter = MongoDBAdapter({
            "connection_string": "mongodb://admin:admin123@localhost:27017/",
            "database": "test_db",
            "collection": "test_collection"
        })
        
        source = f"mongodb://localhost/test_db/test_collection/{str(doc_id)}"
        metadata = adapter.get_metadata({"source": source})
        
        assert metadata["database"] == "test_db"
        assert metadata["collection"] == "test_collection"
        assert metadata["document_id"] == str(doc_id)
        assert "collection_size" in metadata
        assert "document_count" in metadata
    
    def test_get_binary_content(self, mongodb_collection, insert_test_documents):
        """Test getting document as binary content."""
        # Insert document with binary data
        doc = {
            "name": "Binary Test",
            "data": b"Binary content here",
            "type": "binary"
        }
        doc_ids = insert_test_documents(doc)
        doc_id = doc_ids[0]
        
        adapter = MongoDBAdapter({
            "connection_string": "mongodb://admin:admin123@localhost:27017/",
            "database": "test_db",
            "collection": "test_collection"
        })
        
        source = f"mongodb://localhost/test_db/test_collection/{str(doc_id)}"
        binary_content = adapter.get_binary_content({"source": source})
        
        assert isinstance(binary_content, bytes)
        # Content should be JSON representation of document
        decoded = json.loads(binary_content.decode('utf-8'))
        assert decoded["name"] == "Binary Test"
    
    def test_error_handling_nonexistent_document(self, mongodb_collection):
        """Test error handling for non-existent documents."""
        adapter = MongoDBAdapter({
            "connection_string": "mongodb://admin:admin123@localhost:27017/",
            "database": "test_db",
            "collection": "test_collection"
        })
        
        # Try to get non-existent document
        fake_id = str(ObjectId())
        source = f"mongodb://localhost/test_db/test_collection/{fake_id}"
        
        with pytest.raises(ValueError, match="Document not found"):
            adapter.get_content({"source": source})
    
    def test_resolve_uri(self, mongodb_collection):
        """Test URI resolution."""
        adapter = MongoDBAdapter({
            "connection_string": "mongodb://admin:admin123@localhost:27017/",
            "database": "test_db",
            "collection": "test_collection"
        })
        
        # Valid MongoDB URI
        uri = "mongodb://localhost/test_db/test_collection/507f1f77bcf86cd799439011"
        result = adapter.resolve_uri(uri)
        
        assert result["source"] == uri
        assert result["connection_string"] == "mongodb://localhost"
        assert result["database"] == "test_db"
        assert result["collection"] == "test_collection"
        assert result["document_id"] == "507f1f77bcf86cd799439011"
        
        # Invalid URI
        with pytest.raises(ValueError, match="Not a MongoDB URI"):
            adapter.resolve_uri("http://example.com")
    
    def test_caching(self, mongodb_collection, insert_test_documents, sample_mongodb_documents):
        """Test content caching mechanism."""
        doc = sample_mongodb_documents[0]
        doc_ids = insert_test_documents(doc)
        doc_id = doc_ids[0]
        
        adapter = MongoDBAdapter({
            "connection_string": "mongodb://admin:admin123@localhost:27017/",
            "database": "test_db",
            "collection": "test_collection"
        })
        
        source = f"mongodb://localhost/test_db/test_collection/{str(doc_id)}"
        
        # First call - should fetch from MongoDB
        result1 = adapter.get_content({"source": source})
        
        # Modify the document in database
        mongodb_collection.update_one(
            {"_id": doc_id},
            {"$set": {"name": "Modified Document"}}
        )
        
        # Second call - should use cache
        result2 = adapter.get_content({"source": source})
        
        # Content should be the same (cached)
        assert result1["content"] == result2["content"]
        content = json.loads(result2["content"])
        assert content["name"] == "Document 1"  # Original name, not modified
    
    def test_cleanup(self, mongodb_collection):
        """Test adapter cleanup."""
        adapter = MongoDBAdapter({
            "connection_string": "mongodb://admin:admin123@localhost:27017/",
            "database": "test_db",
            "collection": "test_collection"
        })
        
        # Create some cached data
        adapter.content_cache["test"] = {"data": "test"}
        adapter.metadata_cache["test"] = {"meta": "test"}
        
        # Cleanup
        adapter.cleanup()
        
        assert len(adapter.clients) == 0
        assert len(adapter.content_cache) == 0
        assert len(adapter.metadata_cache) == 0