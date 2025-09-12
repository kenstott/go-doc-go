"""
Tests for MongoDB content source.
"""

import pytest
import json
import time
from datetime import datetime
from unittest.mock import patch, MagicMock, Mock
from typing import Dict, Any, List
from bson import ObjectId

# Import from parent directory's conftest
import os
import sys
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'test_adapters'))
from conftest import requires_pymongo, requires_docker

from go_doc_go.content_source.mongodb import MongoDBContentSource


class TestMongoDBContentSourceUnit:
    """Unit tests for MongoDBContentSource without real MongoDB."""
    
    def test_content_source_initialization_without_pymongo(self):
        """Test that content source raises error when pymongo is not available."""
        config = {
            "name": "test-mongodb-source",
            "database_name": "test_db",
            "collection_name": "test_collection"
        }
        
        with patch('go_doc_go.content_source.mongodb.PYMONGO_AVAILABLE', False):
            with pytest.raises(ImportError, match="pymongo is required"):
                MongoDBContentSource(config)
    
    @patch('go_doc_go.content_source.mongodb.MongoClient')
    def test_content_source_initialization_with_config(self, mock_mongo_client):
        """Test content source initialization with configuration."""
        mock_client = MagicMock()
        mock_mongo_client.return_value = mock_client
        mock_client.admin.command.return_value = {"ok": 1}
        
        config = {
            "name": "test-mongodb-source",
            "connection_string": "mongodb://localhost:27017/",
            "database_name": "test_db",
            "collection_name": "test_collection",
            "query": {"type": "article"},
            "projection": {"content": 1, "name": 1},
            "id_field": "_id",
            "content_field": "content",
            "timestamp_field": "updated_at",
            "limit": 500,
            "sort_by": [("created_at", -1)]
        }
        
        with patch('go_doc_go.content_source.mongodb.PYMONGO_AVAILABLE', True):
            source = MongoDBContentSource(config)
            
            assert source.connection_string == "mongodb://localhost:27017/"
            assert source.database_name == "test_db"
            assert source.collection_name == "test_collection"
            assert source.query == {"type": "article"}
            assert source.projection == {"content": 1, "name": 1}
            assert source.id_field == "_id"
            assert source.content_field == "content"
            assert source.timestamp_field == "updated_at"
            assert source.limit == 500
            assert source.sort_by == [("created_at", -1)]
    
    @patch('go_doc_go.content_source.mongodb.MongoClient')
    def test_get_safe_connection_string(self, mock_mongo_client):
        """Test safe connection string generation."""
        mock_client = MagicMock()
        mock_mongo_client.return_value = mock_client
        mock_client.admin.command.return_value = {"ok": 1}
        
        config = {
            "name": "test-source",
            "connection_string": "mongodb://user:password@localhost:27017/",
            "database_name": "test_db",
            "collection_name": "test_collection"
        }
        
        with patch('go_doc_go.content_source.mongodb.PYMONGO_AVAILABLE', True):
            source = MongoDBContentSource(config)
            safe_conn = source.get_safe_connection_string()
            
            assert "password" not in safe_conn
            assert "user:****" in safe_conn
            assert "localhost:27017" in safe_conn
    
    def test_extract_mongo_id(self):
        """Test MongoDB ID extraction from source identifiers."""
        config = {
            "name": "test-source",
            "database_name": "test_db",
            "collection_name": "test_collection"
        }
        
        with patch('go_doc_go.content_source.mongodb.PYMONGO_AVAILABLE', True):
            with patch('go_doc_go.content_source.mongodb.MongoClient'):
                source = MongoDBContentSource(config)
                
                # Simple ObjectId string
                mongo_id = source._extract_mongo_id("507f1f77bcf86cd799439011")
                assert mongo_id == "507f1f77bcf86cd799439011"
                
                # MongoDB URI format
                mongo_id = source._extract_mongo_id("mongodb://test_db/test_collection/507f1f77bcf86cd799439011")
                assert mongo_id == "507f1f77bcf86cd799439011"
                
                # Custom string ID
                mongo_id = source._extract_mongo_id("custom-id-123")
                assert mongo_id == "custom-id-123"
    
    def test_get_doc_id_str(self):
        """Test document ID string extraction."""
        config = {
            "name": "test-source",
            "database_name": "test_db",
            "collection_name": "test_collection"
        }
        
        with patch('go_doc_go.content_source.mongodb.PYMONGO_AVAILABLE', True):
            with patch('go_doc_go.content_source.mongodb.MongoClient'):
                source = MongoDBContentSource(config)
                
                # Document with ObjectId
                doc = {"_id": ObjectId("507f1f77bcf86cd799439011"), "name": "test"}
                doc_id = source._get_doc_id_str(doc)
                assert doc_id == "507f1f77bcf86cd799439011"
                
                # Document with string ID
                doc = {"_id": "custom-id", "name": "test"}
                doc_id = source._get_doc_id_str(doc)
                assert doc_id == "custom-id"
                
                # Document with custom ID field
                source.id_field = "custom_id"
                doc = {"custom_id": "my-id", "_id": "ignored"}
                doc_id = source._get_doc_id_str(doc)
                assert doc_id == "my-id"


@requires_pymongo
@requires_docker
class TestMongoDBContentSourceIntegration:
    """Integration tests for MongoDBContentSource with real MongoDB."""
    
    @pytest.fixture
    def mongodb_source(self, mongodb_client, mongodb_config):
        """Create MongoDBContentSource configured for test MongoDB."""
        config = {
            "name": "test-mongodb-source",
            "connection_string": mongodb_config["connection_string"],
            "database_name": mongodb_config["database_name"],
            "collection_name": mongodb_config["collection_name"]
        }
        
        source = MongoDBContentSource(config)
        source.client = mongodb_client
        source.db = mongodb_client[mongodb_config["database_name"]]
        source.collection = source.db[mongodb_config["collection_name"]]
        
        return source
    
    def test_fetch_document_from_mongodb(self, mongodb_source, mongodb_config, insert_test_documents, sample_mongodb_documents):
        """Test fetching document from MongoDB."""
        # Insert test document
        doc = sample_mongodb_documents[0]
        doc_ids = insert_test_documents(doc)
        doc_id = doc_ids[0]
        
        # Fetch document
        result = mongodb_source.fetch_document(str(doc_id))
        
        assert "id" in result
        assert result["id"].startswith("mongodb://")
        assert "content" in result
        
        # Parse content
        content = json.loads(result["content"])
        assert content["name"] == "Document 1"
        assert content["type"] == "article"
        
        assert result["metadata"]["database"] == mongodb_config["database_name"]
        assert result["metadata"]["collection"] == mongodb_config["collection_name"]
        assert result["metadata"]["id_value"] == str(doc_id)
    
    def test_fetch_document_with_content_field(self, mongodb_source, insert_test_documents):
        """Test fetching document with specific content field."""
        # Configure content field
        mongodb_source.content_field = "body"
        
        # Insert document with body field
        doc = {
            "title": "Test Article",
            "body": "This is the main content of the article",
            "metadata": {"author": "Test Author"}
        }
        doc_ids = insert_test_documents(doc)
        doc_id = doc_ids[0]
        
        # Fetch document
        result = mongodb_source.fetch_document(str(doc_id))
        
        # Should only get the body field as content
        assert result["content"] == "This is the main content of the article"
        assert result["metadata"]["content_field"] == "body"
    
    def test_list_documents(self, mongodb_source, insert_test_documents, sample_mongodb_documents):
        """Test listing documents in MongoDB collection."""
        # Insert multiple documents
        insert_test_documents(sample_mongodb_documents)
        
        # List documents
        documents = mongodb_source.list_documents()
        
        assert len(documents) >= 3
        
        for doc in documents:
            assert "id" in doc
            assert doc["id"].startswith("mongodb://")
            assert "metadata" in doc
            assert "database" in doc["metadata"]
            assert "collection" in doc["metadata"]
            assert "id_value" in doc["metadata"]
    
    def test_list_documents_with_query(self, mongodb_source, insert_test_documents, sample_mongodb_documents):
        """Test listing documents with query filter."""
        # Insert test documents
        insert_test_documents(sample_mongodb_documents)
        
        # Configure query to filter by type
        mongodb_source.query = {"type": "article"}
        
        # List documents
        documents = mongodb_source.list_documents()
        
        # Should only get documents with type="article"
        assert len(documents) >= 1
        
        # Verify filtered results
        for doc in documents:
            doc_id = doc["metadata"]["id_value"]
            # Fetch full document to verify type
            full_doc = mongodb_source.fetch_document(doc_id)
            content = json.loads(full_doc["content"])
            assert content.get("type") == "article"
    
    def test_list_documents_with_limit_and_sort(self, mongodb_source, insert_test_documents):
        """Test listing documents with limit and sorting."""
        # Insert documents with timestamps
        docs = [
            {"name": f"Doc {i}", "timestamp": i, "content": f"Content {i}"}
            for i in range(10)
        ]
        insert_test_documents(docs)
        
        # Configure limit and sort
        mongodb_source.limit = 5
        mongodb_source.sort_by = [("timestamp", -1)]  # Sort by timestamp descending
        
        # List documents
        documents = mongodb_source.list_documents()
        
        assert len(documents) <= 5
    
    def test_has_changed_with_timestamp(self, mongodb_source, insert_test_documents):
        """Test document change detection using timestamp field."""
        # Insert document with timestamp
        now = datetime.now()
        doc = {
            "name": "Timestamped Document",
            "content": "Initial content",
            "updated_at": now
        }
        doc_ids = insert_test_documents(doc)
        doc_id = doc_ids[0]
        
        # Configure timestamp field
        mongodb_source.timestamp_field = "updated_at"
        
        # Check has_changed with current timestamp - should be False
        assert mongodb_source.has_changed(str(doc_id), now.timestamp()) is False
        
        # Update document with new timestamp
        time.sleep(0.1)
        new_time = datetime.now()
        mongodb_source.collection.update_one(
            {"_id": doc_id},
            {"$set": {"updated_at": new_time, "content": "Updated content"}}
        )
        
        # Clear cache to force check
        mongodb_source.content_cache.clear()
        
        # Should detect change
        assert mongodb_source.has_changed(str(doc_id), now.timestamp()) is True
    
    def test_has_changed_without_timestamp(self, mongodb_source, insert_test_documents):
        """Test change detection when no timestamp field is available."""
        # Insert document without timestamp
        doc = {"name": "No Timestamp", "content": "Test content"}
        doc_ids = insert_test_documents(doc)
        doc_id = doc_ids[0]
        
        # Without timestamp field, should always return True
        mongodb_source.timestamp_field = None
        assert mongodb_source.has_changed(str(doc_id), 1000.0) is True
    
    def test_follow_links_with_references(self, mongodb_source, insert_test_documents):
        """Test following references between MongoDB documents."""
        # Insert documents with references
        doc1_id = ObjectId()
        doc2_id = ObjectId()
        doc3_id = ObjectId()
        
        docs = [
            {
                "_id": doc1_id,
                "name": "Document 1",
                "content": "Main document",
                "references": [str(doc2_id), str(doc3_id)]
            },
            {
                "_id": doc2_id,
                "name": "Document 2",
                "content": "Referenced document 2"
            },
            {
                "_id": doc3_id,
                "name": "Document 3",
                "content": "Referenced document 3"
            }
        ]
        insert_test_documents(docs)
        
        # Configure reference following
        mongodb_source.follow_references = True
        mongodb_source.reference_field = "references"
        
        # Get main document content
        main_doc = mongodb_source.fetch_document(str(doc1_id))
        
        # Follow references
        linked_docs = mongodb_source.follow_links(
            main_doc["content"],
            str(doc1_id)
        )
        
        # Should find referenced documents
        assert len(linked_docs) == 2
        
        linked_ids = [doc["metadata"]["id_value"] for doc in linked_docs]
        assert str(doc2_id) in linked_ids
        assert str(doc3_id) in linked_ids
    
    def test_content_caching(self, mongodb_source, insert_test_documents):
        """Test content caching mechanism."""
        # Insert test document
        doc = {"name": "Cache Test", "content": "Original content"}
        doc_ids = insert_test_documents(doc)
        doc_id = doc_ids[0]
        
        # Clear cache
        mongodb_source.content_cache.clear()
        
        # First fetch - should query MongoDB
        result1 = mongodb_source.fetch_document(str(doc_id))
        assert result1["content"]
        
        # Check cache was populated
        assert len(mongodb_source.content_cache) == 1
        
        # Update document in MongoDB
        mongodb_source.collection.update_one(
            {"_id": doc_id},
            {"$set": {"content": "Modified content"}}
        )
        
        # Second fetch - should use cache
        result2 = mongodb_source.fetch_document(str(doc_id))
        
        # Content should be the same (cached)
        content1 = json.loads(result1["content"])
        content2 = json.loads(result2["content"])
        assert content1["content"] == content2["content"]
        assert content2["content"] == "Original content"  # Not modified
    
    def test_json_serialization(self, mongodb_source, insert_test_documents):
        """Test JSON serialization of MongoDB-specific types."""
        # Insert document with ObjectId and datetime
        doc = {
            "_id": ObjectId(),
            "name": "BSON Types Test",
            "created_at": datetime.now(),
            "reference_id": ObjectId(),
            "data": {"nested_date": datetime.now()}
        }
        doc_ids = insert_test_documents(doc)
        doc_id = doc_ids[0]
        
        # Fetch document
        result = mongodb_source.fetch_document(str(doc_id))
        
        # Content should be valid JSON
        content = json.loads(result["content"])
        
        # Check that BSON types were serialized
        assert "created_at" in content
        assert "reference_id" in content
        assert "data" in content
        assert "nested_date" in content["data"]
    
    def test_projection(self, mongodb_source, insert_test_documents):
        """Test document projection to limit returned fields."""
        # Insert document with many fields
        doc = {
            "name": "Full Document",
            "content": "Main content",
            "large_field": "x" * 10000,  # Large field we don't want
            "metadata": {"author": "Test", "tags": ["tag1", "tag2"]}
        }
        doc_ids = insert_test_documents(doc)
        doc_id = doc_ids[0]
        
        # Configure projection to exclude large field
        mongodb_source.projection = {"name": 1, "content": 1, "metadata": 1}
        
        # Fetch document
        result = mongodb_source.fetch_document(str(doc_id))
        content = json.loads(result["content"])
        
        # Should have projected fields
        assert "name" in content
        assert "content" in content
        assert "metadata" in content
        
        # Should not have excluded field
        assert "large_field" not in content