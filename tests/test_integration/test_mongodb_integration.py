"""
Integration tests for MongoDB adapter and content source with document processing.
"""

import pytest
import json
import time
from datetime import datetime
from typing import Dict, Any, List
from unittest.mock import patch
from bson import ObjectId

# Add test_adapters to path
import os
import sys
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'test_adapters'))
from conftest import requires_pymongo, requires_docker

from go_doc_go.adapter.mongodb import MongoDBAdapter
from go_doc_go.content_source.mongodb import MongoDBContentSource
from go_doc_go.document_parser.factory import get_parser_for_content


@requires_pymongo
@requires_docker
class TestMongoDBIntegration:
    """End-to-end integration tests for MongoDB with document processing."""
    
    @pytest.fixture
    def mongodb_components(self, mongodb_client, mongodb_config):
        """Create MongoDB adapter and content source for testing."""
        config = {
            "connection_string": mongodb_config["connection_string"],
            "database_name": mongodb_config["database_name"],
            "collection_name": mongodb_config["collection_name"]
        }
        
        # Create adapter
        adapter = MongoDBAdapter(config)
        
        # Create content source
        source = MongoDBContentSource(config)
        source.client = mongodb_client
        source.db = mongodb_client[mongodb_config["database_name"]]
        source.collection = source.db[mongodb_config["collection_name"]]
        
        return adapter, source
    
    def test_json_document_pipeline(self, mongodb_components, insert_test_documents):
        """Test complete pipeline for JSON documents in MongoDB."""
        adapter, source = mongodb_components
        
        # Create complex JSON document
        doc = {
            "type": "configuration",
            "name": "Application Config",
            "settings": {
                "database": {
                    "host": "localhost",
                    "port": 27017,
                    "options": {
                        "ssl": True,
                        "retry": 3
                    }
                },
                "features": ["auth", "logging", "caching"],
                "limits": {
                    "max_connections": 100,
                    "timeout": 30
                }
            },
            "metadata": {
                "version": "1.0.0",
                "updated_at": datetime.now(),
                "updated_by": "admin"
            }
        }
        
        # Insert document
        doc_ids = insert_test_documents(doc)
        doc_id = doc_ids[0]
        
        # Fetch document using content source
        doc_data = source.fetch_document(str(doc_id))
        
        assert doc_data["content_type"] == "application/json"
        content = json.loads(doc_data["content"])
        assert content["name"] == "Application Config"
        assert content["settings"]["database"]["port"] == 27017
        
        # Parse document
        parser = get_parser_for_content(doc_data)
        parsed = parser.parse(doc_data)
        
        # Verify parsing results
        assert "document" in parsed
        assert "elements" in parsed
        assert len(parsed["elements"]) > 0
        
        # Check that nested structure is preserved
        element_contents = [e.get("content_preview", "") for e in parsed["elements"]]
        joined_content = " ".join(element_contents)
        
        assert "localhost" in joined_content
        assert "27017" in str(joined_content)
        assert "ssl" in joined_content
    
    def test_document_with_arrays(self, mongodb_components, insert_test_documents):
        """Test handling of documents with arrays."""
        adapter, source = mongodb_components
        
        # Document with various array types
        doc = {
            "name": "Array Test Document",
            "simple_array": [1, 2, 3, 4, 5],
            "string_array": ["apple", "banana", "cherry"],
            "object_array": [
                {"id": 1, "name": "Item 1", "active": True},
                {"id": 2, "name": "Item 2", "active": False},
                {"id": 3, "name": "Item 3", "active": True}
            ],
            "nested_arrays": [
                [1, 2, 3],
                [4, 5, 6],
                [7, 8, 9]
            ]
        }
        
        doc_ids = insert_test_documents(doc)
        doc_id = doc_ids[0]
        
        # Test accessing array elements via adapter
        source_uri = f"mongodb://localhost/test_db/test_collection/{str(doc_id)}/object_array/1/name"
        result = adapter.get_content({"source": source_uri})
        assert result["content"] == "Item 2"
        
        # Test accessing nested array
        source_uri = f"mongodb://localhost/test_db/test_collection/{str(doc_id)}/nested_arrays/1/2"
        result = adapter.get_content({"source": source_uri})
        assert result["content"] == "6"
    
    def test_reference_following_pipeline(self, mongodb_components, insert_test_documents):
        """Test following references between MongoDB documents."""
        adapter, source = mongodb_components
        
        # Create a network of related documents
        user_id = ObjectId()
        post1_id = ObjectId()
        post2_id = ObjectId()
        comment1_id = ObjectId()
        comment2_id = ObjectId()
        
        documents = [
            {
                "_id": user_id,
                "type": "user",
                "name": "John Doe",
                "email": "john@example.com",
                "posts": [str(post1_id), str(post2_id)]
            },
            {
                "_id": post1_id,
                "type": "post",
                "title": "First Post",
                "content": "This is my first post",
                "author": str(user_id),
                "comments": [str(comment1_id)]
            },
            {
                "_id": post2_id,
                "type": "post",
                "title": "Second Post",
                "content": "Another interesting post",
                "author": str(user_id),
                "comments": [str(comment2_id)]
            },
            {
                "_id": comment1_id,
                "type": "comment",
                "text": "Great post!",
                "author": "Anonymous",
                "post": str(post1_id)
            },
            {
                "_id": comment2_id,
                "type": "comment",
                "text": "Thanks for sharing",
                "author": "Reader",
                "post": str(post2_id)
            }
        ]
        
        insert_test_documents(documents)
        
        # Configure source for reference following
        source.follow_references = True
        source.reference_field = "posts"
        source.max_link_depth = 2
        
        # Fetch user document
        user_doc = source.fetch_document(str(user_id))
        
        # Follow references to posts
        linked_docs = source.follow_links(user_doc["content"], str(user_id))
        
        # Should find the two posts
        assert len(linked_docs) >= 2
        
        # Verify post content
        post_titles = []
        for doc in linked_docs:
            content = json.loads(doc["content"])
            if content.get("type") == "post":
                post_titles.append(content["title"])
        
        assert "First Post" in post_titles
        assert "Second Post" in post_titles
    
    def test_change_detection_pipeline(self, mongodb_components, insert_test_documents):
        """Test change detection and incremental updates."""
        adapter, source = mongodb_components
        
        # Insert document with timestamp
        initial_time = datetime.now()
        doc = {
            "name": "Changing Document",
            "content": "Initial content",
            "version": 1,
            "updated_at": initial_time
        }
        
        doc_ids = insert_test_documents(doc)
        doc_id = doc_ids[0]
        
        # Configure timestamp field
        source.timestamp_field = "updated_at"
        
        # Initial fetch
        initial_doc = source.fetch_document(str(doc_id))
        initial_timestamp = initial_doc["metadata"]["timestamp_value"]
        
        # Check for changes - should be False
        assert source.has_changed(str(doc_id), initial_timestamp) is False
        
        # Update document
        time.sleep(0.1)
        new_time = datetime.now()
        source.collection.update_one(
            {"_id": doc_id},
            {
                "$set": {
                    "content": "Updated content",
                    "version": 2,
                    "updated_at": new_time
                }
            }
        )
        
        # Clear cache
        source.content_cache.clear()
        
        # Should detect change
        assert source.has_changed(str(doc_id), initial_timestamp) is True
        
        # Fetch updated document
        updated_doc = source.fetch_document(str(doc_id))
        updated_content = json.loads(updated_doc["content"])
        
        assert updated_content["version"] == 2
        assert updated_content["content"] == "Updated content"
    
    def test_large_collection_handling(self, mongodb_components, mongodb_collection):
        """Test handling of large collections with pagination."""
        adapter, source = mongodb_components
        
        # Insert many documents
        docs = []
        for i in range(100):
            docs.append({
                "index": i,
                "name": f"Document {i}",
                "content": f"Content for document {i}",
                "category": "test" if i % 2 == 0 else "sample",
                "created_at": datetime.now()
            })
        
        mongodb_collection.insert_many(docs)
        
        # Configure source with limit
        source.limit = 20
        source.sort_by = [("index", 1)]
        
        # List documents
        documents = source.list_documents()
        
        # Should respect limit
        assert len(documents) <= 20
        
        # Verify documents are sorted
        indices = []
        for doc in documents:
            doc_id = doc["metadata"]["id_value"]
            full_doc = mongodb_collection.find_one({"_id": ObjectId(doc_id)})
            if full_doc:
                indices.append(full_doc["index"])
        
        # Should be in ascending order
        assert indices == sorted(indices)
    
    def test_nested_field_extraction(self, mongodb_components, insert_test_documents):
        """Test extraction of deeply nested fields."""
        adapter, source = mongodb_components
        
        # Document with deep nesting
        doc = {
            "root": {
                "level1": {
                    "level2": {
                        "level3": {
                            "level4": {
                                "value": "deeply nested value",
                                "array": [
                                    {"item": "first"},
                                    {"item": "second"},
                                    {"item": "third"}
                                ]
                            }
                        }
                    }
                }
            }
        }
        
        doc_ids = insert_test_documents(doc)
        doc_id = doc_ids[0]
        
        # Extract deeply nested value
        source_uri = f"mongodb://localhost/test_db/test_collection/{str(doc_id)}/root/level1/level2/level3/level4/value"
        result = adapter.get_content({"source": source_uri})
        assert result["content"] == "deeply nested value"
        
        # Extract from nested array
        source_uri = f"mongodb://localhost/test_db/test_collection/{str(doc_id)}/root/level1/level2/level3/level4/array/1/item"
        result = adapter.get_content({"source": source_uri})
        assert result["content"] == "second"
    
    def test_bson_type_handling(self, mongodb_components, insert_test_documents):
        """Test handling of various BSON types."""
        adapter, source = mongodb_components
        
        # Document with various BSON types
        doc = {
            "object_id": ObjectId(),
            "date": datetime.now(),
            "binary": b"binary data here",
            "null_value": None,
            "boolean": True,
            "integer": 42,
            "float": 3.14159,
            "regex": {"$regex": "pattern.*"},
            "timestamp": datetime.now().timestamp()
        }
        
        doc_ids = insert_test_documents(doc)
        doc_id = doc_ids[0]
        
        # Fetch document
        doc_data = source.fetch_document(str(doc_id))
        
        # Content should be valid JSON
        content = json.loads(doc_data["content"])
        
        # Verify types are handled
        assert "object_id" in content
        assert "date" in content
        assert content["boolean"] is True
        assert content["integer"] == 42
        assert content["float"] == 3.14159
        assert content["null_value"] is None
    
    def test_query_and_projection(self, mongodb_components, insert_test_documents):
        """Test document querying and projection."""
        adapter, source = mongodb_components
        
        # Insert various documents
        docs = [
            {"type": "article", "title": "Article 1", "content": "Long content...", "tags": ["tech", "news"]},
            {"type": "blog", "title": "Blog 1", "content": "Blog content...", "tags": ["personal"]},
            {"type": "article", "title": "Article 2", "content": "More content...", "tags": ["science"]},
            {"type": "blog", "title": "Blog 2", "content": "Another blog...", "tags": ["travel"]},
            {"type": "article", "title": "Article 3", "content": "Final content...", "tags": ["tech", "tutorial"]}
        ]
        
        insert_test_documents(docs)
        
        # Configure query to get only articles with tech tag
        source.query = {"type": "article", "tags": "tech"}
        source.projection = {"title": 1, "tags": 1}  # Only get title and tags
        
        # List documents
        documents = source.list_documents()
        
        # Should only get tech articles
        for doc in documents:
            doc_id = doc["metadata"]["id_value"]
            full_doc = source.fetch_document(doc_id)
            content = json.loads(full_doc["content"])
            
            # Should be an article with tech tag
            if "type" in content:  # type might be excluded by projection
                assert content["type"] == "article"
            if "tags" in content:
                assert "tech" in content["tags"]
            
            # Should not have content field due to projection
            assert "content" not in content or content["content"] is None
    
    def test_concurrent_access(self, mongodb_components, insert_test_documents):
        """Test concurrent access to MongoDB documents."""
        adapter, source = mongodb_components
        
        # Insert multiple documents
        doc_ids = []
        for i in range(10):
            doc = {
                "index": i,
                "name": f"Concurrent Doc {i}",
                "content": f"Content {i}"
            }
            ids = insert_test_documents(doc)
            doc_ids.extend(ids)
        
        # Fetch all documents
        documents = []
        for doc_id in doc_ids:
            doc = source.fetch_document(str(doc_id))
            documents.append(doc)
        
        # Verify all documents were fetched
        assert len(documents) == 10
        
        # Verify content integrity
        for i, doc in enumerate(documents):
            content = json.loads(doc["content"])
            assert "Concurrent Doc" in content["name"]
    
    def test_error_recovery(self, mongodb_components):
        """Test error handling and recovery."""
        adapter, source = mongodb_components
        
        # Test non-existent document
        fake_id = str(ObjectId())
        with pytest.raises(ValueError, match="Document not found"):
            source.fetch_document(fake_id)
        
        # Test invalid MongoDB URI
        with pytest.raises(ValueError, match="Invalid MongoDB URI"):
            adapter.get_content({"source": "http://not-mongodb.com/db/collection"})
        
        # Test invalid field path
        doc = {"simple": "value"}
        doc_ids = source.collection.insert_one(doc).inserted_id
        
        source_uri = f"mongodb://localhost/test_db/test_collection/{str(doc_ids)}/nonexistent/field"
        with pytest.raises(ValueError, match="Field not found"):
            adapter.get_content({"source": source_uri})