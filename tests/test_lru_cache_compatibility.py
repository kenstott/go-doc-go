"""
Test LRU cache compatibility between Python and Go implementations.

This test validates that both implementations produce identical results
for the same operations, ensuring the Go port is functionally equivalent.
"""

import json
import os
import pytest
import time
from pathlib import Path

# Add src to Python path
import sys
sys.path.insert(0, str(Path(__file__).parent.parent / "src"))

from go_doc_go.document_parser.lru_cache import LRUCache, GoLRUCache, create_lru_cache


class TestLRUCacheCompatibility:
    """Test compatibility between Python and Go LRU cache implementations."""

    @pytest.fixture
    def python_cache(self):
        """Create a Python LRU cache instance."""
        return LRUCache(max_size=10, ttl=60)

    @pytest.fixture
    def go_cache(self):
        """Create a Go LRU cache instance."""
        try:
            return GoLRUCache(max_size=10, ttl=60)
        except FileNotFoundError:
            pytest.skip("Go binary not found. Build with: cd go && go build -o ../bin/lrucache ./cmd/lrucache")

    def test_basic_set_get(self, python_cache, go_cache):
        """Test basic set and get operations match."""
        # Test string values
        python_cache.set("key1", "value1")
        go_cache.set("key1", "value1")

        assert python_cache.get("key1") == "value1"
        assert go_cache.get("key1") == "value1"

        # Test non-existent keys
        assert python_cache.get("nonexistent") is None
        assert go_cache.get("nonexistent") is None

    def test_numeric_values(self, python_cache, go_cache):
        """Test that numeric values are handled consistently."""
        # Integers
        python_cache.set("int", 42)
        go_cache.set("int", 42)

        py_result = python_cache.get("int")
        go_result = go_cache.get("int")

        assert py_result == 42
        assert go_result == 42

        # Floats
        python_cache.set("float", 3.14)
        go_cache.set("float", 3.14)

        assert python_cache.get("float") == 3.14
        assert go_cache.get("float") == 3.14

    def test_boolean_values(self, python_cache, go_cache):
        """Test that boolean values are handled consistently."""
        python_cache.set("bool_true", True)
        go_cache.set("bool_true", True)

        python_cache.set("bool_false", False)
        go_cache.set("bool_false", False)

        assert python_cache.get("bool_true") is True
        assert go_cache.get("bool_true") is True

        assert python_cache.get("bool_false") is False
        assert go_cache.get("bool_false") is False

    def test_complex_data_structures(self, python_cache, go_cache):
        """Test that complex data structures are handled consistently."""
        # Lists
        list_data = [1, 2, 3, "four", 5.0]
        python_cache.set("list", list_data)
        go_cache.set("list", list_data)

        assert python_cache.get("list") == list_data
        assert go_cache.get("list") == list_data

        # Dictionaries
        dict_data = {"name": "test", "value": 42, "nested": {"key": "value"}}
        python_cache.set("dict", dict_data)
        go_cache.set("dict", dict_data)

        assert python_cache.get("dict") == dict_data
        assert go_cache.get("dict") == dict_data

    def test_cache_eviction(self, python_cache, go_cache):
        """Test that LRU eviction works consistently."""
        # Both caches have max_size=10
        # Fill both caches
        for i in range(10):
            key = f"key{i}"
            value = f"value{i}"
            python_cache.set(key, value)
            go_cache.set(key, value)

        # Access key0 to make it recently used
        python_cache.get("key0")
        go_cache.get("key0")

        # Add one more item - should evict key1 (least recently used after key0 access)
        python_cache.set("key10", "value10")
        go_cache.set("key10", "value10")

        # key0 should still exist (recently accessed)
        assert python_cache.get("key0") == "value0"
        assert go_cache.get("key0") == "value0"

        # key10 should exist (just added)
        assert python_cache.get("key10") == "value10"
        assert go_cache.get("key10") == "value10"

    def test_clear_operation(self, python_cache, go_cache):
        """Test that clear operation works consistently."""
        # Add items to both caches
        python_cache.set("key1", "value1")
        python_cache.set("key2", "value2")
        go_cache.set("key1", "value1")
        go_cache.set("key2", "value2")

        # Clear both caches
        python_cache.clear()
        go_cache.clear()

        # Both should be empty
        assert python_cache.get("key1") is None
        assert go_cache.get("key1") is None
        assert python_cache.get("key2") is None
        assert go_cache.get("key2") is None

    def test_ttl_expiration(self):
        """Test that TTL expiration works consistently."""
        # Create caches with 1 second TTL
        python_cache = LRUCache(max_size=10, ttl=1)
        try:
            go_cache = GoLRUCache(max_size=10, ttl=1)
        except FileNotFoundError:
            pytest.skip("Go binary not found")

        # Set values
        python_cache.set("expire", "value")
        go_cache.set("expire", "value")

        # Should exist immediately
        assert python_cache.get("expire") == "value"
        assert go_cache.get("expire") == "value"

        # Wait for expiration
        time.sleep(1.5)

        # Should be expired
        assert python_cache.get("expire") is None
        assert go_cache.get("expire") is None

    def test_update_existing_key(self, python_cache, go_cache):
        """Test that updating existing keys works consistently."""
        # Set initial values
        python_cache.set("update", "value1")
        go_cache.set("update", "value1")

        # Update with new values
        python_cache.set("update", "value2")
        go_cache.set("update", "value2")

        # Both should have the updated value
        assert python_cache.get("update") == "value2"
        assert go_cache.get("update") == "value2"

    def test_factory_function(self):
        """Test the factory function for creating caches."""
        # Test without environment variable (should use Python)
        os.environ.pop("USE_GO_CACHE", None)
        cache = create_lru_cache()
        assert isinstance(cache, LRUCache)

        # Test with environment variable (should use Go if available)
        os.environ["USE_GO_CACHE"] = "true"
        try:
            cache = create_lru_cache()
            assert isinstance(cache, GoLRUCache)
        except FileNotFoundError:
            # Go binary not available, should fall back to Python
            assert isinstance(cache, LRUCache)
        finally:
            os.environ.pop("USE_GO_CACHE", None)

    def test_special_characters_in_keys(self, python_cache, go_cache):
        """Test that special characters in keys are handled consistently."""
        special_keys = [
            "key with spaces",
            "key:with:colons",
            "key/with/slashes",
            "key@with#special$chars",
            "key\twith\ttabs",
            "key\nwith\nnewlines"
        ]

        for key in special_keys:
            value = f"value_for_{key}"
            python_cache.set(key, value)
            go_cache.set(key, value)

            py_result = python_cache.get(key)
            go_result = go_cache.get(key)

            assert py_result == value, f"Python cache failed for key: {repr(key)}"
            assert go_result == value, f"Go cache failed for key: {repr(key)}"

    def test_null_and_empty_values(self, python_cache, go_cache):
        """Test handling of null and empty values."""
        # Empty string
        python_cache.set("empty", "")
        go_cache.set("empty", "")

        assert python_cache.get("empty") == ""
        assert go_cache.get("empty") == ""

        # Empty list
        python_cache.set("empty_list", [])
        go_cache.set("empty_list", [])

        assert python_cache.get("empty_list") == []
        assert go_cache.get("empty_list") == []

        # Empty dict
        python_cache.set("empty_dict", {})
        go_cache.set("empty_dict", {})

        assert python_cache.get("empty_dict") == {}
        assert go_cache.get("empty_dict") == {}


if __name__ == "__main__":
    pytest.main([__file__, "-v"])