"""
MongoDB Implementation with Enhanced Full-Text Support

This module provides a complete MongoDB implementation of the DocumentDatabase
with full structured search capabilities and comprehensive full-text configuration support.
"""

import logging
import os
import re
import gzip
from datetime import datetime, timedelta
from typing import Dict, Any, List, Optional, Tuple, Union, TYPE_CHECKING

import time

from .element_element import ElementHierarchical

# Import types for type checking only - these won't be imported at runtime
if TYPE_CHECKING:
    import numpy as np
    from numpy.typing import NDArray
    from pymongo import MongoClient
    from pymongo.database import Database
    from pymongo.collection import Collection

    # Define type aliases for type checking
    VectorType = NDArray[np.float32]  # NumPy array type for vectors
    MongoDBType = Database  # MongoDB database type
    MongoCollectionType = Collection  # MongoDB collection type
else:
    # Runtime type aliases - use generic Python types
    VectorType = List[float]  # Generic list of floats for vectors
    MongoDBType = Any  # Generic type for MongoDB database
    MongoCollectionType = Any  # Generic type for MongoDB collection

from .element_relationship import ElementRelationship
from .base import DocumentDatabase
from .element_element import ElementType, ElementBase  # Import existing enum and ElementBase

# Import structured search components
from .structured_search import (
    StructuredSearchQuery, SearchCriteriaGroup, BackendCapabilities, SearchCapability,
    UnsupportedSearchError, TextSearchCriteria, EmbeddingSearchCriteria, DateSearchCriteria,
    TopicSearchCriteria, MetadataSearchCriteria, ElementSearchCriteria,
    LogicalOperator, DateRangeOperator, SimilarityOperator
)

logger = logging.getLogger(__name__)

# Define global flags for availability - these will be set at runtime
PYMONGO_AVAILABLE = False
NUMPY_AVAILABLE = False

# Try to import MongoDB library at runtime
try:
    import pymongo
    from pymongo import MongoClient
    from pymongo.errors import ConnectionFailure, DuplicateKeyError

    PYMONGO_AVAILABLE = True
except ImportError:
    logger.warning("pymongo not available. Install with 'pip install pymongo'.")
    MongoClient = None
    ConnectionFailure = Exception  # Fallback type for exception handling
    DuplicateKeyError = Exception  # Fallback type for exception handling

# Try to import NumPy conditionally at runtime
try:
    import numpy as np

    NUMPY_AVAILABLE = True
except ImportError:
    logger.warning("numpy not available. Install with 'pip install numpy'.")

# Try to import the config
try:
    from ..config import Config

    config = Config(os.environ.get("GO_DOC_GO_CONFIG_PATH", "./config.yaml"))
except Exception as e:
    logger.warning(f"Error configuring MongoDB provider: {str(e)}")
    config = None


class MongoDBDocumentDatabase(DocumentDatabase):
    """MongoDB implementation of document database with comprehensive full-text and structured search support."""

    def __init__(self, conn_params: Dict[str, Any]):
        """
        Initialize MongoDB document database with full-text configuration support.

        Args:
            conn_params: Connection parameters for MongoDB including full-text options
        """
        # Call parent constructor to set up full-text configuration
        super().__init__(conn_params)

        # MongoDB specific connection parameters
        self.client = None
        self.db: MongoDBType = None  # Type hint using our conditional alias
        self.vector_search = False
        self.embedding_generator = None
        self.vector_dimension = None

        # Full-text search capabilities
        self._full_text_search_enabled = False
        self._text_index_created = False

        if config:
            self.vector_dimension = config.config.get('embedding', {}).get('dimensions', 384)
        else:
            self.vector_dimension = 384  # Default if config not available

        # Log full-text configuration
        logger.info(f"MongoDB initialized with full-text config: "
                   f"store={self.store_full_text}, index={self.index_full_text}, "
                   f"compress={self.compress_full_text}, max_length={self.full_text_max_length}")

    # ========================================
    # FULL-TEXT CONFIGURATION SUPPORT
    # ========================================

    def supports_full_text_search(self) -> bool:
        """
        MongoDB supports full-text search through text indexes and $text operator.

        Returns:
            True if full-text search is enabled and text indexes are available
        """
        return self._full_text_search_enabled and self.index_full_text

    def _get_available_search_fields(self) -> List[str]:
        """
        Get list of fields available for text search based on configuration.

        Returns:
            List of field names that can be searched
        """
        fields = ['content_preview']  # Always available

        if self.index_full_text and self.supports_full_text_search():
            fields.append('full_content')

        return fields

    def get_full_text_usage_recommendations(self) -> Dict[str, Any]:
        """
        Get recommendations for optimizing full-text usage based on current configuration.

        Returns:
            Dictionary with optimization recommendations
        """
        config = self.get_text_storage_config()
        storage = self.get_storage_size_estimate()

        recommendations = {
            'current_config': 'optimal',
            'suggestions': [],
            'warnings': []
        }

        # MongoDB-specific recommendations
        if not self._text_index_created and self.index_full_text:
            recommendations['warnings'].append(
                "Text indexing is enabled but MongoDB text indexes haven't been created. "
                "Call create_text_indexes() to enable full-text search."
            )

        if self.index_full_text and not self.store_full_text:
            recommendations['suggestions'].append(
                "Consider enabling store_full_text=True to allow retrieval of original content "
                "in addition to search capabilities."
            )

        if storage['overall_storage'] == 'High' and not self.compress_full_text:
            recommendations['suggestions'].append(
                "High storage usage detected. Consider enabling compress_full_text=True "
                "to reduce storage requirements in MongoDB."
            )

        if self.full_text_max_length is None:
            recommendations['suggestions'].append(
                "No length limit set for full text. Consider setting full_text_max_length "
                "to prevent very large documents from consuming excessive MongoDB storage."
            )

        # MongoDB Atlas specific recommendations
        if self.vector_search:
            recommendations['suggestions'].append(
                "MongoDB Atlas vector search is available. Consider using unified_search() "
                "to combine text similarity with vector search for better results."
            )

        return recommendations

    def _process_full_text_content(self, content: str) -> Tuple[Optional[str], Optional[bytes]]:
        """
        Process full text content according to configuration settings.

        Args:
            content: Original full text content

        Returns:
            Tuple of (processed_content, compressed_content) where one may be None based on config
        """
        if not content:
            return None, None

        # Apply length limit if configured
        if self.full_text_max_length and len(content) > self.full_text_max_length:
            content = content[:self.full_text_max_length]
            logger.debug(f"Truncated content to {self.full_text_max_length} characters")

        processed_content = None
        compressed_content = None

        # Store original content if configured
        if self.store_full_text:
            if self.compress_full_text:
                # Store compressed version
                compressed_content = gzip.compress(content.encode('utf-8'))
                logger.debug(f"Compressed content from {len(content)} to {len(compressed_content)} bytes")
            else:
                # Store uncompressed
                processed_content = content

        return processed_content, compressed_content

    def _retrieve_full_text_content(self, element: Dict[str, Any]) -> Optional[str]:
        """
        Retrieve full text content from element, handling compression if enabled.

        Args:
            element: Element document from MongoDB

        Returns:
            Decompressed full text content or None
        """
        if not self.store_full_text:
            return None

        # Check for compressed content first
        if 'full_content_compressed' in element and element['full_content_compressed']:
            try:
                compressed_data = element['full_content_compressed']
                # Handle both binary data and base64 encoded strings
                if isinstance(compressed_data, str):
                    import base64
                    compressed_data = base64.b64decode(compressed_data)

                decompressed = gzip.decompress(compressed_data)
                return decompressed.decode('utf-8')
            except Exception as e:
                logger.warning(f"Failed to decompress content: {str(e)}")
                return None

        # Check for uncompressed content
        if 'full_content' in element:
            return element['full_content']

        return None

    # ========================================
    # CORE DATABASE OPERATIONS
    # ========================================

    def initialize(self) -> None:
        """Initialize the database by connecting and creating collections with full-text support."""
        if not PYMONGO_AVAILABLE:
            raise ImportError("pymongo is required for MongoDB support")

        # Extract connection parameters
        host = self.conn_params.get('host', 'localhost')
        port = self.conn_params.get('port', 27017)
        username = self.conn_params.get('username')
        password = self.conn_params.get('password')
        db_name = self.conn_params.get('db_name', 'go-doc-go')

        # Build connection string
        connection_string = "mongodb://"
        if username and password:
            connection_string += f"{username}:{password}@"
        connection_string += f"{host}:{port}/{db_name}"

        # Add additional connection options
        options = self.conn_params.get('options', {})
        if options:
            option_str = "&".join(f"{k}={v}" for k, v in options.items())
            connection_string += f"?{option_str}"

        # Connect to MongoDB
        try:
            self.client = MongoClient(connection_string)
            # Ping the server to verify connection
            self.client.admin.command('ping')
            logger.info(f"Connected to MongoDB at {host}:{port}")

            # Select database
            self.db = self.client[db_name]

            # Create collections and indexes with full-text support
            self._create_collections()

            # Check if vector search is available
            self._check_vector_search()

            logger.info(f"Initialized MongoDB database with vector search: {self.vector_search}, "
                       f"full-text search: {self.supports_full_text_search()}")

        except ConnectionFailure as e:
            logger.error(f"Error connecting to MongoDB: {str(e)}")
            raise

    def _check_vector_search(self) -> None:
        """Check if vector search capabilities are available."""
        try:
            # Check MongoDB version (5.0+ required for some vector features)
            server_info = self.client.server_info()
            version = server_info.get('version', '0.0.0')
            major_version = int(version.split('.')[0])

            if major_version >= 5:
                # Check if Atlas Vector Search is available
                try:
                    # Try getting list of search indexes to see if feature is available
                    # indexes = list(self.db.command({"listSearchIndexes": "elements"}))
                    self.vector_search = True
                    logger.info("MongoDB vector search is available")
                    return
                except Exception as e:
                    logger.debug(f"Vector search not available: {str(e)}")

            logger.info(f"MongoDB version: {version}, vector search unavailable")
            self.vector_search = False

        except Exception as e:
            logger.warning(f"Error checking vector search availability: {str(e)}")
            self.vector_search = False

    def _create_collections(self) -> None:
        """Create collections and indexes with full-text search support."""
        # Documents collection
        if "documents" not in self.db.list_collection_names():
            self.db.create_collection("documents")

        # Create indexes for documents collection
        self.db.documents.create_index("doc_id", unique=True)
        self.db.documents.create_index("source")

        # Elements collection - Enhanced for full-text support
        if "elements" not in self.db.list_collection_names():
            self.db.create_collection("elements")

        # Create indexes for elements collection
        self.db.elements.create_index("element_id", unique=True)
        self.db.elements.create_index("element_pk", unique=True)
        self.db.elements.create_index("doc_id")
        self.db.elements.create_index("parent_id")
        self.db.elements.create_index("element_type")

        # Create full-text search indexes if enabled
        if self.index_full_text:
            self._create_full_text_indexes()

        # Other collections
        if "relationships" not in self.db.list_collection_names():
            self.db.create_collection("relationships")

        self.db.relationships.create_index("relationship_id", unique=True)
        self.db.relationships.create_index("source_id")
        self.db.relationships.create_index("relationship_type")

        if "embeddings" not in self.db.list_collection_names():
            self.db.create_collection("embeddings")

        self.db.embeddings.create_index("element_pk", unique=True)
        self.db.embeddings.create_index("topics")
        self.db.embeddings.create_index("confidence")

        if "processing_history" not in self.db.list_collection_names():
            self.db.create_collection("processing_history")

        self.db.processing_history.create_index("source_id", unique=True)

        if "counters" not in self.db.list_collection_names():
            self.db.create_collection("counters")
            if not self.db.counters.find_one({"_id": "element_pk"}):
                self.db.counters.insert_one({"_id": "element_pk", "seq": 0})

        logger.info("Created MongoDB collections and indexes with full-text support")

    def _create_full_text_indexes(self) -> None:
        """Create MongoDB text indexes for full-text search."""
        try:
            # Define text index fields based on configuration
            text_index_fields = [("content_preview", "text")]

            if self.store_full_text and not self.compress_full_text:
                # Only index uncompressed full_content
                text_index_fields.append(("full_content", "text"))

            # Create compound text index
            if len(text_index_fields) > 0:
                self.db.elements.create_index(text_index_fields, name="fulltext_search_index")
                self._text_index_created = True
                self._full_text_search_enabled = True
                logger.info(f"Created MongoDB text index on fields: {[field[0] for field in text_index_fields]}")

        except Exception as e:
            logger.warning(f"Could not create full-text indexes: {str(e)}")
            self._text_index_created = False
            self._full_text_search_enabled = False

    def create_text_indexes(self):
        """
        Create or recreate text indexes for better search performance.
        Call this after database initialization for better performance.
        """
        if not self.db:
            raise ValueError("Database not initialized")

        # Drop existing text index if it exists
        try:
            self.db.elements.drop_index("fulltext_search_index")
            logger.info("Dropped existing text index")
        except:
            pass  # Index may not exist

        # Recreate text indexes
        self._create_full_text_indexes()

    def close(self) -> None:
        """Close the database connection."""
        if self.client:
            self.client.close()
            self.client = None
            self.db = None

    # ========================================
    # DOCUMENT STORAGE OPERATIONS
    # ========================================

    def store_document(self, document: Dict[str, Any], elements: List[Dict[str, Any]],
                       relationships: List[Dict[str, Any]]) -> None:
        """
        Store a document with its elements and relationships, respecting full-text configuration.
        """
        if not self.db:
            raise ValueError("Database not initialized")

        source = document.get("source", "")
        content_hash = document.get("content_hash", "")

        # Check if document already exists with this source
        existing_doc = self.db.documents.find_one({"source": source}) if source else None

        if existing_doc:
            # Document exists, update it
            doc_id = existing_doc["doc_id"]
            document["doc_id"] = doc_id  # Use existing doc_id

            # Update all elements to use the existing doc_id
            for element in elements:
                element["doc_id"] = doc_id

            self.update_document(doc_id, document, elements, relationships)
            return

        # New document, proceed with creation
        doc_id = document["doc_id"]

        try:
            # Store document
            document_with_timestamps = {
                **document,
                "created_at": document.get("created_at", time.time()),
                "updated_at": document.get("updated_at", time.time())
            }

            self.db.documents.insert_one(document_with_timestamps)

            # Process elements with full-text configuration support
            elements_to_insert = []
            for i, element in enumerate(elements):
                # Generate MongoDB compatible representation with element_pk
                mongo_element = {**element}

                # Generate a unique element_pk if not present
                if "element_pk" not in mongo_element:
                    counter = self.db.counters.find_one_and_update(
                        {"_id": "element_pk"},
                        {"$inc": {"seq": 1}},
                        upsert=True,
                        return_document=True
                    )
                    mongo_element["element_pk"] = counter["seq"]
                    element["element_pk"] = mongo_element["element_pk"]

                # Process full text content according to configuration
                if 'full_content' in mongo_element:
                    full_content = mongo_element['full_content']
                    processed_content, compressed_content = self._process_full_text_content(full_content)

                    # Remove original full_content
                    del mongo_element['full_content']

                    # Store processed content based on configuration
                    if processed_content is not None:
                        mongo_element['full_content'] = processed_content
                    elif compressed_content is not None:
                        # Store compressed content as binary data
                        import base64
                        mongo_element['full_content_compressed'] = base64.b64encode(compressed_content).decode('ascii')

                elements_to_insert.append(mongo_element)

            # Store elements in bulk if there are any
            if elements_to_insert:
                self.db.elements.insert_many(elements_to_insert)

            # Store relationships in bulk if there are any
            if relationships:
                self.db.relationships.insert_many(relationships)

            # Update processing history
            if source:
                self.update_processing_history(source, content_hash)

            logger.info(
                f"Stored document {doc_id} with {len(elements)} elements and {len(relationships)} relationships "
                f"(full-text config: store={self.store_full_text}, compress={self.compress_full_text})")

        except Exception as e:
            logger.error(f"Error storing document {doc_id}: {str(e)}")
            raise

    def update_document(self, doc_id: str, document: Dict[str, Any],
                        elements: List[Dict[str, Any]],
                        relationships: List[Dict[str, Any]]) -> None:
        """
        Update an existing document with full-text configuration support.
        """
        if not self.db:
            raise ValueError("Database not initialized")

        # Check if document exists
        existing_doc = self.db.documents.find_one({"doc_id": doc_id})
        if not existing_doc:
            raise ValueError(f"Document not found: {doc_id}")

        try:
            # Update document timestamps
            document["updated_at"] = time.time()
            if "created_at" not in document and "created_at" in existing_doc:
                document["created_at"] = existing_doc["created_at"]

            # Get all element IDs and PKs for this document
            element_ids = [element["element_id"] for element in
                           self.db.elements.find({"doc_id": doc_id}, {"element_id": 1})]
            element_pks = [element["element_pk"] for element in
                           self.db.elements.find({"doc_id": doc_id}, {"element_pk": 1})]

            # Delete all existing relationships and embeddings
            if element_ids:
                self.db.relationships.delete_many({"source_id": {"$in": element_ids}})
            if element_pks:
                self.db.embeddings.delete_many({"element_pk": {"$in": element_pks}})

            # Delete all existing elements for this document
            self.db.elements.delete_many({"doc_id": doc_id})

            # Replace the document
            self.db.documents.replace_one({"doc_id": doc_id}, document)

            # Process elements with full-text support
            elements_to_insert = []
            for element in elements:
                mongo_element = {**element}

                # Generate a unique element_pk if not present
                if "element_pk" not in mongo_element:
                    counter = self.db.counters.find_one_and_update(
                        {"_id": "element_pk"},
                        {"$inc": {"seq": 1}},
                        upsert=True,
                        return_document=True
                    )
                    mongo_element["element_pk"] = counter["seq"]
                    element["element_pk"] = mongo_element["element_pk"]

                # Process full text content according to configuration
                if 'full_content' in mongo_element:
                    full_content = mongo_element['full_content']
                    processed_content, compressed_content = self._process_full_text_content(full_content)

                    # Remove original full_content
                    del mongo_element['full_content']

                    # Store processed content based on configuration
                    if processed_content is not None:
                        mongo_element['full_content'] = processed_content
                    elif compressed_content is not None:
                        import base64
                        mongo_element['full_content_compressed'] = base64.b64encode(compressed_content).decode('ascii')

                elements_to_insert.append(mongo_element)

            # Insert new elements
            if elements_to_insert:
                self.db.elements.insert_many(elements_to_insert)

            # Insert new relationships
            if relationships:
                self.db.relationships.insert_many(relationships)

            # Update processing history
            source = document.get("source", "")
            content_hash = document.get("content_hash", "")
            if source:
                self.update_processing_history(source, content_hash)

            logger.info(f"Updated document {doc_id} with {len(elements)} elements and {len(relationships)} relationships")

        except Exception as e:
            logger.error(f"Error updating document {doc_id}: {str(e)}")
            raise

    def get_document(self, doc_id: str) -> Optional[Dict[str, Any]]:
        """Get document metadata by ID."""
        if not self.db:
            raise ValueError("Database not initialized")

        document = self.db.documents.find_one({"doc_id": doc_id})

        if not document:
            return None

        # Remove MongoDB's _id field
        if "_id" in document:
            del document["_id"]

        return document

    def get_last_processed_info(self, source_id: str) -> Optional[Dict[str, Any]]:
        """Get information about when a document was last processed."""
        if not self.db:
            raise ValueError("Database not initialized")

        try:
            history = self.db.processing_history.find_one({"source_id": source_id})
            if not history:
                return None

            # Remove MongoDB's _id field
            if "_id" in history:
                del history["_id"]

            return history
        except Exception as e:
            logger.error(f"Error getting processing history for {source_id}: {str(e)}")
            return None

    def update_processing_history(self, source_id: str, content_hash: str) -> None:
        """Update the processing history for a document."""
        if not self.db:
            raise ValueError("Database not initialized")

        try:
            # Check if record exists
            existing = self.db.processing_history.find_one({"source_id": source_id})
            processing_count = 1  # Default for new records

            if existing:
                processing_count = existing.get("processing_count", 0) + 1

            # Update or insert record
            self.db.processing_history.update_one(
                {"source_id": source_id},
                {
                    "$set": {
                        "content_hash": content_hash,
                        "last_modified": time.time(),
                        "processing_count": processing_count
                    }
                },
                upsert=True
            )

            logger.debug(f"Updated processing history for {source_id}")

        except Exception as e:
            logger.error(f"Error updating processing history for {source_id}: {str(e)}")

    def get_document_elements(self, doc_id: str) -> List[Dict[str, Any]]:
        """
        Get elements for a document with full-text content restoration.
        """
        if not self.db:
            raise ValueError("Database not initialized")

        # First try to get document by doc_id
        document = self.db.documents.find_one({"doc_id": doc_id})

        if not document:
            # If not found, try by source
            document = self.db.documents.find_one({"source": doc_id})
            if not document:
                return []
            doc_id = document["doc_id"]

        elements = list(self.db.elements.find({"doc_id": doc_id}))

        # Process each element to restore full_content and remove MongoDB fields
        for element in elements:
            if "_id" in element:
                del element["_id"]

            # Restore full_content from compressed storage if needed and requested
            if self.store_full_text and 'full_content_compressed' in element:
                full_content = self._retrieve_full_text_content(element)
                if full_content:
                    element['full_content'] = full_content
                # Remove compressed version from result
                del element['full_content_compressed']

        return elements

    def get_document_relationships(self, doc_id: str) -> List[Dict[str, Any]]:
        """Get relationships for a document."""
        if not self.db:
            raise ValueError("Database not initialized")

        # First get all element IDs for the document
        element_ids = [element["element_id"] for element in
                       self.db.elements.find({"doc_id": doc_id}, {"element_id": 1})]

        if not element_ids:
            return []

        # Find relationships involving these elements
        relationships = list(self.db.relationships.find({"source_id": {"$in": element_ids}}))

        # Remove MongoDB's _id field from each relationship
        for relationship in relationships:
            if "_id" in relationship:
                del relationship["_id"]

        return relationships

    def get_element(self, element_id_or_pk: Union[str, int]) -> Optional[Dict[str, Any]]:
        """
        Get element by ID or PK with full-text content restoration.
        """
        if not self.db:
            raise ValueError("Database not initialized")

        element = None

        # Try to interpret as element_pk (integer) first
        try:
            element_pk = int(element_id_or_pk)
            element = self.db.elements.find_one({"element_pk": element_pk})
        except (ValueError, TypeError):
            # If not an integer, treat as element_id (string)
            element = self.db.elements.find_one({"element_id": element_id_or_pk})

        if not element:
            return None

        # Remove MongoDB's _id field
        if "_id" in element:
            del element["_id"]

        # Restore full_content from compressed storage if needed
        if self.store_full_text and 'full_content_compressed' in element:
            full_content = self._retrieve_full_text_content(element)
            if full_content:
                element['full_content'] = full_content
            # Remove compressed version from result
            del element['full_content_compressed']

        return element

    def delete_document(self, doc_id: str) -> bool:
        """Delete a document and all associated elements and relationships."""
        if not self.db:
            raise ValueError("Database not initialized")

        # Check if document exists
        if not self.db.documents.find_one({"doc_id": doc_id}):
            return False

        try:
            # Get all element IDs for this document
            element_ids = [element["element_id"] for element in
                           self.db.elements.find({"doc_id": doc_id}, {"element_id": 1})]

            # Get all element PKs for this document
            element_pks = [element["element_pk"] for element in
                           self.db.elements.find({"doc_id": doc_id}, {"element_pk": 1})]

            # Delete embeddings for these elements
            if element_pks:
                self.db.embeddings.delete_many({"element_pk": {"$in": element_pks}})

            # Delete relationships involving these elements
            if element_ids:
                self.db.relationships.delete_many({"source_id": {"$in": element_ids}})

            # Delete elements
            self.db.elements.delete_many({"doc_id": doc_id})

            # Delete document
            self.db.documents.delete_one({"doc_id": doc_id})

            logger.info(f"Deleted document {doc_id} with {len(element_ids)} elements")
            return True

        except Exception as e:
            logger.error(f"Error deleting document {doc_id}: {str(e)}")
            return False

    # ========================================
    # LEGACY SEARCH METHODS (Enhanced with full-text support)
    # ========================================

    def find_documents(self, query: Dict[str, Any] = None, limit: int = 100) -> List[Dict[str, Any]]:
        """
        Find documents matching query with support for LIKE patterns.
        """
        if not self.db:
            raise ValueError("Database not initialized")

        # Build MongoDB query
        mongo_query = {}

        if query:
            for key, value in query.items():
                if key == "metadata":
                    # Handle metadata exact matches
                    for meta_key, meta_value in value.items():
                        mongo_query[f"metadata.{meta_key}"] = meta_value
                elif key == "metadata_like":
                    # Handle metadata LIKE patterns
                    for meta_key, meta_value in value.items():
                        regex_pattern = self._convert_like_to_regex(meta_value)
                        mongo_query[f"metadata.{meta_key}"] = {"$regex": regex_pattern, "$options": "i"}
                elif key == "metadata_ilike":
                    # Case-insensitive metadata LIKE patterns (same as metadata_like in MongoDB)
                    for meta_key, meta_value in value.items():
                        regex_pattern = self._convert_like_to_regex(meta_value)
                        mongo_query[f"metadata.{meta_key}"] = {"$regex": regex_pattern, "$options": "i"}
                elif key.endswith("_ilike"):
                    # Case-insensitive LIKE pattern
                    field_name = key[:-6]  # Remove '_ilike' suffix
                    regex_pattern = self._convert_like_to_regex(value)
                    mongo_query[field_name] = {"$regex": regex_pattern, "$options": "i"}
                elif key.endswith("_like"):
                    # LIKE pattern for regular fields
                    field_name = key[:-5]  # Remove '_like' suffix
                    regex_pattern = self._convert_like_to_regex(value)
                    mongo_query[field_name] = {"$regex": regex_pattern}
                elif isinstance(value, list):
                    # Handle list fields with IN clause
                    mongo_query[key] = {"$in": value}
                else:
                    # Exact match for regular fields
                    mongo_query[key] = value

        # Execute query
        documents = list(self.db.documents.find(mongo_query).limit(limit))

        # Remove MongoDB's _id field from each document
        for document in documents:
            if "_id" in document:
                del document["_id"]

        return documents

    def find_elements(self, query: Dict[str, Any] = None, limit: int = 100) -> List[Dict[str, Any]]:
        """
        Find elements matching query with support for LIKE patterns and ElementType enums.
        """
        if not self.db:
            raise ValueError("Database not initialized")

        # Build MongoDB query
        mongo_query = {}

        if query:
            for key, value in query.items():
                if key == "metadata":
                    # Handle metadata exact matches
                    for meta_key, meta_value in value.items():
                        mongo_query[f"metadata.{meta_key}"] = meta_value
                elif key == "metadata_like":
                    # Handle metadata LIKE patterns
                    for meta_key, meta_value in value.items():
                        regex_pattern = self._convert_like_to_regex(meta_value)
                        mongo_query[f"metadata.{meta_key}"] = {"$regex": regex_pattern, "$options": "i"}
                elif key == "metadata_ilike":
                    # Case-insensitive metadata LIKE patterns (same as metadata_like in MongoDB)
                    for meta_key, meta_value in value.items():
                        regex_pattern = self._convert_like_to_regex(meta_value)
                        mongo_query[f"metadata.{meta_key}"] = {"$regex": regex_pattern, "$options": "i"}
                elif key.endswith("_ilike"):
                    # Case-insensitive LIKE pattern
                    field_name = key[:-6]  # Remove '_ilike' suffix
                    regex_pattern = self._convert_like_to_regex(value)
                    mongo_query[field_name] = {"$regex": regex_pattern, "$options": "i"}
                elif key.endswith("_like"):
                    # LIKE pattern for regular fields
                    field_name = key[:-5]  # Remove '_like' suffix
                    regex_pattern = self._convert_like_to_regex(value)
                    mongo_query[field_name] = {"$regex": regex_pattern}
                elif key == "element_type":
                    # Handle ElementType enums, strings, and lists
                    type_values = self._prepare_element_type_query(value)
                    if type_values:
                        if len(type_values) == 1:
                            mongo_query["element_type"] = type_values[0]
                        else:
                            mongo_query["element_type"] = {"$in": type_values}
                elif isinstance(value, list):
                    # Handle other list fields with IN clause
                    mongo_query[key] = {"$in": value}
                else:
                    # Exact match for regular fields
                    mongo_query[key] = value

        # Execute query
        elements = list(self.db.elements.find(mongo_query).limit(limit))

        # Process each element to restore full_content and remove MongoDB fields
        for element in elements:
            if "_id" in element:
                del element["_id"]

            # Restore full_content from compressed storage if needed
            if self.store_full_text and 'full_content_compressed' in element:
                full_content = self._retrieve_full_text_content(element)
                if full_content:
                    element['full_content'] = full_content
                # Remove compressed version from result
                del element['full_content_compressed']

        return elements

    def search_elements_by_content(self, search_text: str, limit: int = 10) -> List[Dict[str, Any]]:
        """
        Search elements by content using MongoDB full-text search when available.

        Implementation respects full-text configuration:
        - Uses MongoDB $text operator if text indexes are available
        - Falls back to regex search on content_preview
        - Searches full_content field if indexed and available
        """
        if not self.db:
            raise ValueError("Database not initialized")

        elements = []

        # Try MongoDB full-text search first if available
        if self.supports_full_text_search() and self._text_index_created:
            try:
                # Use MongoDB's $text operator for full-text search
                text_search_results = list(self.db.elements.find(
                    {"$text": {"$search": search_text}},
                    {"score": {"$meta": "textScore"}}
                ).sort([("score", {"$meta": "textScore"})]).limit(limit))

                if text_search_results:
                    logger.debug(f"MongoDB full-text search found {len(text_search_results)} results")
                    elements = text_search_results

            except Exception as e:
                logger.warning(f"MongoDB full-text search failed, falling back to regex: {str(e)}")

        # Fall back to regex search if full-text search didn't work or found no results
        if not elements:
            # Search content_preview (always available)
            regex_results = list(self.db.elements.find(
                {"content_preview": {"$regex": search_text, "$options": "i"}}
            ).limit(limit))

            # Also search uncompressed full_content if it's stored and indexed
            if self.store_full_text and not self.compress_full_text:
                full_content_results = list(self.db.elements.find(
                    {"full_content": {"$regex": search_text, "$options": "i"}}
                ).limit(limit))

                # Combine results and remove duplicates
                combined_results = regex_results + full_content_results
                seen_ids = set()
                elements = []
                for elem in combined_results:
                    elem_id = elem.get('element_id')
                    if elem_id not in seen_ids:
                        elements.append(elem)
                        seen_ids.add(elem_id)
                        if len(elements) >= limit:
                            break

            else:
                elements = regex_results

        # Remove MongoDB's _id field and restore full_content if requested
        for element in elements:
            if "_id" in element:
                del element["_id"]

            # Restore full_content from compressed storage if needed
            if self.store_full_text and 'full_content_compressed' in element:
                full_content = self._retrieve_full_text_content(element)
                if full_content:
                    element['full_content'] = full_content
                # Remove compressed version from result
                if 'full_content_compressed' in element:
                    del element['full_content_compressed']

        logger.debug(f"Content search for '{search_text}' returned {len(elements)} results")
        return elements

    # ========================================
    # EMBEDDING SEARCH METHODS
    # ========================================

    def store_embedding(self, element_pk: int, embedding: VectorType) -> None:
        """
        Store embedding for an element.
        """
        if not self.db:
            raise ValueError("Database not initialized")

        # Verify element exists
        element = self.db.elements.find_one({"element_pk": element_pk})
        if not element:
            raise ValueError(f"Element not found: {element_pk}")

        # Update vector dimension based on actual data
        self.vector_dimension = max(self.vector_dimension, len(embedding))

        try:
            # Store or update embedding with default topics and confidence
            self.db.embeddings.update_one(
                {"element_pk": element_pk},
                {
                    "$set": {
                        "embedding": embedding,
                        "dimensions": len(embedding),
                        "topics": [],  # Default to empty topics
                        "confidence": 1.0,  # Default confidence
                        "created_at": time.time()
                    }
                },
                upsert=True
            )

            logger.debug(f"Stored embedding for element {element_pk}")

        except Exception as e:
            logger.error(f"Error storing embedding for {element_pk}: {str(e)}")
            raise

    def get_embedding(self, element_pk: int) -> Optional[VectorType]:
        """
        Get embedding for an element.
        """
        if not self.db:
            raise ValueError("Database not initialized")

        embedding_doc = self.db.embeddings.find_one({"element_pk": element_pk})
        if not embedding_doc:
            return None

        return embedding_doc.get("embedding")

    def search_by_embedding(self, query_embedding: VectorType, limit: int = 10,
                            filter_criteria: Dict[str, Any] = None) -> List[Tuple[int, float]]:
        """
        Search elements by embedding similarity with optional filtering.
        Returns (element_pk, similarity) tuples for consistency with other implementations.
        """
        if not self.db:
            raise ValueError("Database not initialized")

        try:
            if self.vector_search:
                return self._search_by_vector_index(query_embedding, limit, filter_criteria)
            else:
                return self._search_by_cosine_similarity(query_embedding, limit, filter_criteria)
        except Exception as e:
            logger.error(f"Error searching by embedding: {str(e)}")
            # Fall back to cosine similarity
            return self._search_by_cosine_similarity(query_embedding, limit, filter_criteria)

    def _search_by_vector_index(self, query_embedding: VectorType, limit: int = 10,
                                filter_criteria: Dict[str, Any] = None) -> List[Tuple[int, float]]:
        """Search embeddings using MongoDB Atlas Vector Search with filtering."""
        try:
            # Define the vector search pipeline
            pipeline = [{
                "$vectorSearch": {
                    "index": "embeddings_vector_index",
                    "path": "embedding",
                    "queryVector": query_embedding,
                    "numCandidates": limit * 5,  # Get more candidates for better results
                    "limit": limit * 10  # Get more results than needed to allow for filtering
                }
            }, {
                "$lookup": {
                    "from": "elements",
                    "localField": "element_pk",
                    "foreignField": "element_pk",
                    "as": "element"
                }
            }, {
                "$unwind": "$element"
            }, {
                "$lookup": {
                    "from": "documents",
                    "localField": "element.doc_id",
                    "foreignField": "doc_id",
                    "as": "document"
                }
            }, {
                "$unwind": {
                    "path": "$document",
                    "preserveNullAndEmptyArrays": True
                }
            }]

            # Add filter stages if criteria provided
            if filter_criteria:
                match_conditions = {}

                for key, value in filter_criteria.items():
                    if key == "element_type" and isinstance(value, list):
                        # Handle list of allowed element types
                        match_conditions["element.element_type"] = {"$in": value}
                    elif key == "doc_id" and isinstance(value, list):
                        # Handle list of document IDs to include
                        match_conditions["element.doc_id"] = {"$in": value}
                    elif key == "exclude_doc_id" and isinstance(value, list):
                        # Handle list of document IDs to exclude
                        match_conditions["element.doc_id"] = {"$nin": value}
                    elif key == "exclude_doc_source" and isinstance(value, list):
                        # Handle list of document sources to exclude
                        match_conditions["document.source"] = {"$nin": value}
                    else:
                        # Simple equality filter
                        match_conditions[f"element.{key}"] = value

                if match_conditions:
                    pipeline.append({"$match": match_conditions})

            # Add projection and limit
            pipeline.extend([
                {
                    "$project": {
                        "_id": 0,
                        "element_pk": 1,
                        "score": {"$meta": "vectorSearchScore"}
                    }
                },
                {
                    "$limit": limit
                }
            ])

            # Execute the search
            results = list(self.db.embeddings.aggregate(pipeline))

            # Format results as (element_pk, similarity_score)
            return [(doc["element_pk"], doc["score"]) for doc in results]

        except Exception as e:
            logger.error(f"Error using vector search index: {str(e)}")
            raise

    def _search_by_cosine_similarity(self, query_embedding: VectorType, limit: int = 10,
                                     filter_criteria: Dict[str, Any] = None) -> List[Tuple[int, float]]:
        """Fall back to calculating cosine similarity with filtering."""
        # Check if NumPy is available and use appropriate implementation
        if not NUMPY_AVAILABLE:
            return self._search_by_cosine_similarity_pure_python(query_embedding, limit, filter_criteria)
        else:
            return self._search_by_cosine_similarity_numpy(query_embedding, limit, filter_criteria)

    def _search_by_cosine_similarity_numpy(self, query_embedding: VectorType, limit: int = 10,
                                           filter_criteria: Dict[str, Any] = None) -> List[Tuple[int, float]]:
        """NumPy implementation of cosine similarity search."""
        # Begin building a pipeline for embeddings with element and document data
        pipeline = [
            {
                "$lookup": {
                    "from": "elements",
                    "localField": "element_pk",
                    "foreignField": "element_pk",
                    "as": "element"
                }
            },
            {
                "$unwind": "$element"
            },
            {
                "$lookup": {
                    "from": "documents",
                    "localField": "element.doc_id",
                    "foreignField": "doc_id",
                    "as": "document"
                }
            },
            {
                "$unwind": {
                    "path": "$document",
                    "preserveNullAndEmptyArrays": True
                }
            }
        ]

        # Add filter criteria if provided
        if filter_criteria:
            match_conditions = {}

            for key, value in filter_criteria.items():
                if key == "element_type" and isinstance(value, list):
                    # Handle list of allowed element types
                    match_conditions["element.element_type"] = {"$in": value}
                elif key == "doc_id" and isinstance(value, list):
                    # Handle list of document IDs to include
                    match_conditions["element.doc_id"] = {"$in": value}
                elif key == "exclude_doc_id" and isinstance(value, list):
                    # Handle list of document IDs to exclude
                    match_conditions["element.doc_id"] = {"$nin": value}
                elif key == "exclude_doc_source" and isinstance(value, list):
                    # Handle list of document sources to exclude
                    match_conditions["document.source"] = {"$nin": value}
                else:
                    # Simple equality filter
                    match_conditions[f"element.{key}"] = value

            if match_conditions:
                pipeline.append({"$match": match_conditions})

        # Add projection to get just what we need
        pipeline.append({
            "$project": {
                "_id": 0,
                "element_pk": 1,
                "embedding": 1
            }
        })

        # Execute aggregation to get filtered embeddings
        filtered_embeddings = list(self.db.embeddings.aggregate(pipeline))

        # Calculate cosine similarity for each embedding
        similarities = []
        query_array = np.array(query_embedding)

        for doc in filtered_embeddings:
            element_pk = doc["element_pk"]
            embedding = doc["embedding"]

            if embedding and len(embedding) == len(query_embedding):
                # Use NumPy for efficient calculation
                embedding_array = np.array(embedding)
                dot_product = np.dot(query_array, embedding_array)
                norm1 = np.linalg.norm(query_array)
                norm2 = np.linalg.norm(embedding_array)

                if norm1 == 0 or norm2 == 0:
                    similarity = 0.0
                else:
                    similarity = float(dot_product / (norm1 * norm2))

                similarities.append((element_pk, similarity))

        # Sort by similarity (highest first) and limit results
        similarities.sort(key=lambda x: x[1], reverse=True)
        return similarities[:limit]

    def _search_by_cosine_similarity_pure_python(self, query_embedding: VectorType, limit: int = 10,
                                                 filter_criteria: Dict[str, Any] = None) -> List[Tuple[int, float]]:
        """Pure Python implementation of cosine similarity search when NumPy is not available."""
        # Begin building a pipeline for embeddings with element and document data
        pipeline = [
            {
                "$lookup": {
                    "from": "elements",
                    "localField": "element_pk",
                    "foreignField": "element_pk",
                    "as": "element"
                }
            },
            {
                "$unwind": "$element"
            },
            {
                "$lookup": {
                    "from": "documents",
                    "localField": "element.doc_id",
                    "foreignField": "doc_id",
                    "as": "document"
                }
            },
            {
                "$unwind": {
                    "path": "$document",
                    "preserveNullAndEmptyArrays": True
                }
            }
        ]

        # Add filter criteria if provided
        if filter_criteria:
            match_conditions = {}

            for key, value in filter_criteria.items():
                if key == "element_type" and isinstance(value, list):
                    # Handle list of allowed element types
                    match_conditions["element.element_type"] = {"$in": value}
                elif key == "doc_id" and isinstance(value, list):
                    # Handle list of document IDs to include
                    match_conditions["element.doc_id"] = {"$in": value}
                elif key == "exclude_doc_id" and isinstance(value, list):
                    # Handle list of document IDs to exclude
                    match_conditions["element.doc_id"] = {"$nin": value}
                elif key == "exclude_doc_source" and isinstance(value, list):
                    # Handle list of document sources to exclude
                    match_conditions["document.source"] = {"$nin": value}
                else:
                    # Simple equality filter
                    match_conditions[f"element.{key}"] = value

            if match_conditions:
                pipeline.append({"$match": match_conditions})

        # Add projection to get just what we need
        pipeline.append({
            "$project": {
                "_id": 0,
                "element_pk": 1,
                "embedding": 1
            }
        })

        # Execute aggregation to get filtered embeddings
        filtered_embeddings = list(self.db.embeddings.aggregate(pipeline))

        # Calculate cosine similarity for each embedding using pure Python
        similarities = []

        for doc in filtered_embeddings:
            element_pk = doc["element_pk"]
            embedding = doc["embedding"]

            if embedding and len(embedding) == len(query_embedding):
                # Calculate using pure Python
                dot_product = sum(a * b for a, b in zip(query_embedding, embedding))
                mag1 = sum(a * a for a in query_embedding) ** 0.5
                mag2 = sum(b * b for b in embedding) ** 0.5

                if mag1 == 0 or mag2 == 0:
                    similarity = 0.0
                else:
                    similarity = float(dot_product / (mag1 * mag2))

                similarities.append((element_pk, similarity))

        # Sort by similarity (highest first) and limit results
        similarities.sort(key=lambda x: x[1], reverse=True)
        return similarities[:limit]

    def search_by_text(self, search_text: str, limit: int = 10,
                       filter_criteria: Dict[str, Any] = None) -> List[Tuple[int, float]]:
        """
        Search elements by semantic similarity to the provided text.
        Returns (element_pk, similarity) tuples for consistency with other implementations.
        """
        if not self.db:
            raise ValueError("Database not initialized")

        try:
            # Import embedding generator on-demand
            try:
                from ..embeddings import get_embedding_generator
                if self.embedding_generator is None:
                    self.embedding_generator = get_embedding_generator(config)
            except ImportError as e:
                logger.error(f"Embedding generator not available: {str(e)}")
                raise ValueError("Embedding libraries are not installed.")

            # Generate embedding for the search text
            query_embedding = self.embedding_generator.generate(search_text)

            # Use the embedding to search, passing the filter criteria
            return self.search_by_embedding(query_embedding, limit, filter_criteria)

        except Exception as e:
            logger.error(f"Error in semantic search by text: {str(e)}")
            # Return empty list on error
            return []

    def get_outgoing_relationships(self, element_pk: int) -> List[ElementRelationship]:
        """
        Find all relationships where the specified element_pk is the source.
        """
        if not self.db:
            raise ValueError("Database not initialized")

        relationships = []

        # Get the element to find its element_id and type
        element = self.get_element(element_pk)
        if not element:
            logger.warning(f"Element with PK {element_pk} not found")
            return []

        element_id = element.get("element_id")
        if not element_id:
            logger.warning(f"Element with PK {element_pk} has no element_id")
            return []

        element_type = element.get("element_type", "")

        try:
            # Use aggregation pipeline to join relationships with elements
            pipeline = [
                # Match relationships where this element is the source
                {"$match": {"source_id": element_id}},

                # Lookup target elements
                {"$lookup": {
                    "from": "elements",
                    "localField": "target_reference",
                    "foreignField": "element_id",
                    "as": "target_element"
                }},

                # Unwind target_element array (or preserve null with preserveNullAndEmptyArrays)
                {"$unwind": {
                    "path": "$target_element",
                    "preserveNullAndEmptyArrays": True
                }}
            ]

            # Execute the aggregation pipeline
            results = list(self.db.relationships.aggregate(pipeline))

            # Process results
            for result in results:
                # Remove MongoDB's _id field
                if "_id" in result:
                    del result["_id"]

                # Extract target element information if available
                target_element_pk = None
                target_element_type = None
                target_content_preview = None

                if "target_element" in result and result["target_element"]:
                    target_element = result["target_element"]
                    target_element_pk = target_element.get("element_pk")
                    target_element_type = target_element.get("element_type")
                    target_content_preview = target_element.get("content_preview", "")

                    # Remove the target_element object from the result
                    del result["target_element"]

                # Create enriched relationship
                relationship = ElementRelationship(
                    relationship_id=result.get("relationship_id", ""),
                    source_id=element_id,
                    source_element_pk=element_pk,
                    source_element_type=element_type,
                    relationship_type=result.get("relationship_type", ""),
                    target_reference=result.get("target_reference", ""),
                    target_element_pk=target_element_pk,
                    target_element_type=target_element_type,
                    target_content_preview=target_content_preview,
                    doc_id=result.get("doc_id"),
                    metadata=result.get("metadata", {}),
                    is_source=True
                )

                relationships.append(relationship)

            return relationships

        except Exception as e:
            logger.error(f"Error getting outgoing relationships for element {element_pk}: {str(e)}")
            return []

    # ========================================
    # STRUCTURED SEARCH IMPLEMENTATION
    # ========================================

    def get_backend_capabilities(self) -> BackendCapabilities:
        """
        MongoDB supports comprehensive search capabilities including full-text search.
        """
        supported = {
            # Core search types
            SearchCapability.TEXT_SIMILARITY,
            SearchCapability.EMBEDDING_SIMILARITY,

            # Date capabilities
            SearchCapability.DATE_FILTERING,
            SearchCapability.DATE_RANGE_QUERIES,
            SearchCapability.FISCAL_YEAR_DATES,
            SearchCapability.RELATIVE_DATES,
            SearchCapability.DATE_AGGREGATIONS,

            # Topic capabilities
            SearchCapability.TOPIC_FILTERING,
            SearchCapability.TOPIC_LIKE_PATTERNS,
            SearchCapability.TOPIC_CONFIDENCE_FILTERING,

            # Metadata capabilities
            SearchCapability.METADATA_EXACT,
            SearchCapability.METADATA_LIKE,
            SearchCapability.METADATA_RANGE,
            SearchCapability.METADATA_EXISTS,
            SearchCapability.NESTED_METADATA,

            # Element capabilities
            SearchCapability.ELEMENT_TYPE_FILTERING,
            SearchCapability.ELEMENT_HIERARCHY,
            SearchCapability.ELEMENT_RELATIONSHIPS,

            # Logical operations
            SearchCapability.LOGICAL_AND,
            SearchCapability.LOGICAL_OR,
            SearchCapability.LOGICAL_NOT,
            SearchCapability.NESTED_QUERIES,

            # Scoring and ranking
            SearchCapability.CUSTOM_SCORING,
            SearchCapability.SIMILARITY_THRESHOLDS,
            SearchCapability.BOOST_FACTORS,
            SearchCapability.SCORE_COMBINATION,

            # Advanced features
            SearchCapability.FACETED_SEARCH,
        }

        # Add full-text search if available
        if self.supports_full_text_search():
            supported.add(SearchCapability.FULL_TEXT_SEARCH)

        # Add vector search if Atlas Vector Search is available
        if self.vector_search:
            supported.add(SearchCapability.VECTOR_SEARCH)

        return BackendCapabilities(supported)

    def execute_structured_search(self, query: StructuredSearchQuery) -> List[Dict[str, Any]]:
        """Execute a structured search query using MongoDB's aggregation framework."""
        if not self.db:
            raise ValueError("Database not initialized")

        # Validate query support
        missing = self.validate_query_support(query)
        if missing:
            raise UnsupportedSearchError(missing)

        try:
            # Execute the root criteria group
            raw_results = self._execute_criteria_group(query.criteria_group)

            # Process and enrich results
            final_results = self._process_search_results(raw_results, query)

            # Apply pagination
            start_idx = query.offset
            end_idx = start_idx + query.limit

            return final_results[start_idx:end_idx]

        except Exception as e:
            logger.error(f"Error executing structured search: {str(e)}")
            return []

    def _execute_criteria_group(self, group: SearchCriteriaGroup) -> List[Dict[str, Any]]:
        """Execute a single criteria group and return scored results."""

        # Collect results from all criteria in this group
        all_results = []

        # Execute individual criteria
        if group.text_criteria:
            text_results = self._execute_text_criteria(group.text_criteria)
            all_results.append(("text", text_results))

        if group.embedding_criteria:
            embedding_results = self._execute_embedding_criteria(group.embedding_criteria)
            all_results.append(("embedding", embedding_results))

        if group.date_criteria:
            date_results = self._execute_date_criteria(group.date_criteria)
            all_results.append(("date", date_results))

        if group.topic_criteria:
            topic_results = self._execute_topic_criteria(group.topic_criteria)
            all_results.append(("topic", topic_results))

        if group.metadata_criteria:
            metadata_results = self._execute_metadata_criteria(group.metadata_criteria)
            all_results.append(("metadata", metadata_results))

        if group.element_criteria:
            element_results = self._execute_element_criteria(group.element_criteria)
            all_results.append(("element", element_results))

        # Execute sub-groups recursively
        for sub_group in group.sub_groups:
            sub_results = self._execute_criteria_group(sub_group)
            all_results.append(("subgroup", sub_results))

        # Combine results based on the group's logical operator
        return self._combine_results(all_results, group.operator)

    def _execute_text_criteria(self, criteria: TextSearchCriteria) -> List[Dict[str, Any]]:
        """Execute text search using either full-text search or embedding similarity."""
        try:
            # First try MongoDB full-text search if available and preferred
            if (self.supports_full_text_search() and self._text_index_created and
                getattr(criteria, 'prefer_full_text_search', False)):

                return self._execute_mongodb_text_search(criteria)

            # Fall back to embedding-based semantic search
            else:
                return self._execute_embedding_based_text_search(criteria)

        except Exception as e:
            logger.error(f"Error executing text criteria: {str(e)}")
            return []

    def _execute_mongodb_text_search(self, criteria: TextSearchCriteria) -> List[Dict[str, Any]]:
        """Execute MongoDB native full-text search."""
        try:
            # Use MongoDB $text operator
            text_query = {"$text": {"$search": criteria.query_text}}

            # Execute text search with scoring
            results = list(self.db.elements.find(
                text_query,
                {"score": {"$meta": "textScore"}, "element_pk": 1}
            ).sort([("score", {"$meta": "textScore"})]).limit(1000))

            # Filter by similarity threshold if specified
            filtered_results = []
            for result in results:
                # MongoDB textScore is not directly comparable to cosine similarity,
                # but we can use it relatively
                text_score = result.get("score", 0.0)

                # Normalize MongoDB text score to approximate similarity (0-1 range)
                # MongoDB text scores are typically between 0.5 and 1.5 for relevant results
                normalized_score = min(1.0, max(0.0, text_score / 2.0))

                if self._compare_similarity(normalized_score, criteria.similarity_threshold,
                                          criteria.similarity_operator):
                    filtered_results.append({
                        'element_pk': result['element_pk'],
                        'scores': {
                            'text_similarity': normalized_score * criteria.boost_factor,
                            'mongodb_text_score': text_score
                        }
                    })

            logger.debug(f"MongoDB text search for '{criteria.query_text}' found {len(filtered_results)} results")
            return filtered_results

        except Exception as e:
            logger.error(f"MongoDB text search failed: {str(e)}")
            # Fall back to embedding search
            return self._execute_embedding_based_text_search(criteria)

    def _execute_embedding_based_text_search(self, criteria: TextSearchCriteria) -> List[Dict[str, Any]]:
        """Execute embedding-based semantic text search."""
        # Generate embedding for the query text
        query_embedding = self._generate_embedding(criteria.query_text)

        # Perform similarity search
        similarity_results = self.search_by_embedding(
            query_embedding,
            limit=1000,  # Get many results for filtering
            filter_criteria=None
        )

        # Filter by similarity threshold and operator
        filtered_results = []
        for element_pk, similarity in similarity_results:
            if self._compare_similarity(similarity, criteria.similarity_threshold, criteria.similarity_operator):
                filtered_results.append({
                    'element_pk': element_pk,
                    'scores': {
                        'text_similarity': similarity * criteria.boost_factor
                    }
                })

        return filtered_results

    def _execute_embedding_criteria(self, criteria: EmbeddingSearchCriteria) -> List[Dict[str, Any]]:
        """Execute direct embedding vector search."""
        try:
            similarity_results = self.search_by_embedding(
                criteria.embedding_vector,
                limit=1000,
                filter_criteria=None
            )

            filtered_results = []
            for element_pk, similarity in similarity_results:
                if self._compare_similarity(similarity, criteria.similarity_threshold, criteria.similarity_operator):
                    filtered_results.append({
                        'element_pk': element_pk,
                        'scores': {
                            'embedding_similarity': similarity * criteria.boost_factor
                        }
                    })

            return filtered_results

        except Exception as e:
            logger.error(f"Error executing embedding criteria: {str(e)}")
            return []

    def _execute_date_criteria(self, criteria: DateSearchCriteria) -> List[Dict[str, Any]]:
        """Execute date-based filtering using MongoDB aggregation."""
        try:
            # Build date filter based on operator
            if criteria.operator == DateRangeOperator.WITHIN:
                element_pks = self._get_element_pks_in_date_range(criteria.start_date, criteria.end_date)

            elif criteria.operator == DateRangeOperator.AFTER:
                element_pks = self._get_element_pks_in_date_range(criteria.exact_date, None)

            elif criteria.operator == DateRangeOperator.BEFORE:
                element_pks = self._get_element_pks_in_date_range(None, criteria.exact_date)

            elif criteria.operator == DateRangeOperator.EXACTLY:
                # For exactly, we need a tight range around the date
                start_of_day = criteria.exact_date.replace(hour=0, minute=0, second=0, microsecond=0)
                end_of_day = criteria.exact_date.replace(hour=23, minute=59, second=59, microsecond=999999)
                element_pks = self._get_element_pks_in_date_range(start_of_day, end_of_day)

            elif criteria.operator == DateRangeOperator.RELATIVE_DAYS:
                end_date = datetime.now()
                start_date = end_date - timedelta(days=criteria.relative_value)
                element_pks = self._get_element_pks_in_date_range(start_date, end_date)

            elif criteria.operator == DateRangeOperator.RELATIVE_MONTHS:
                end_date = datetime.now()
                start_date = end_date - timedelta(days=criteria.relative_value * 30)  # Approximate
                element_pks = self._get_element_pks_in_date_range(start_date, end_date)

            elif criteria.operator == DateRangeOperator.FISCAL_YEAR:
                # Assume fiscal year starts in July (customize as needed)
                start_date = datetime(criteria.year - 1, 7, 1)
                end_date = datetime(criteria.year, 6, 30, 23, 59, 59)
                element_pks = self._get_element_pks_in_date_range(start_date, end_date)

            elif criteria.operator == DateRangeOperator.CALENDAR_YEAR:
                start_date = datetime(criteria.year, 1, 1)
                end_date = datetime(criteria.year, 12, 31, 23, 59, 59)
                element_pks = self._get_element_pks_in_date_range(start_date, end_date)

            elif criteria.operator == DateRangeOperator.QUARTER:
                quarter_starts = {1: (1, 1), 2: (4, 1), 3: (7, 1), 4: (10, 1)}
                quarter_ends = {1: (3, 31), 2: (6, 30), 3: (9, 30), 4: (12, 31)}

                start_month, start_day = quarter_starts[criteria.quarter]
                end_month, end_day = quarter_ends[criteria.quarter]

                start_date = datetime(criteria.year, start_month, start_day)
                end_date = datetime(criteria.year, end_month, end_day, 23, 59, 59)
                element_pks = self._get_element_pks_in_date_range(start_date, end_date)

            # Also filter by specificity levels if needed
            if hasattr(criteria, 'specificity_levels') and criteria.specificity_levels:
                element_pks = self._filter_by_specificity(element_pks, criteria.specificity_levels)

            # Convert to result format
            results = []
            for element_pk in element_pks:
                results.append({
                    'element_pk': element_pk,
                    'scores': {
                        'date_relevance': 1.0  # Could calculate date relevance score
                    }
                })

            return results

        except Exception as e:
            logger.error(f"Error executing date criteria: {str(e)}")
            return []

    def _execute_topic_criteria(self, criteria: TopicSearchCriteria) -> List[Dict[str, Any]]:
        """Execute topic-based filtering using MongoDB array operators."""
        try:
            topic_results = self.search_by_text_and_topics(
                search_text=None,
                include_topics=criteria.include_topics,
                exclude_topics=criteria.exclude_topics,
                min_confidence=criteria.min_confidence,
                limit=1000
            )

            results = []
            for result in topic_results:
                results.append({
                    'element_pk': result['element_pk'],
                    'scores': {
                        'topic_confidence': result['confidence'] * criteria.boost_factor
                    }
                })

            return results

        except Exception as e:
            logger.error(f"Error executing topic criteria: {str(e)}")
            return []

    def _execute_metadata_criteria(self, criteria: MetadataSearchCriteria) -> List[Dict[str, Any]]:
        """Execute metadata-based filtering using MongoDB's flexible document structure."""
        try:
            # Build MongoDB query for metadata filtering
            match_conditions = {}

            # Add exact matches
            for key, value in criteria.exact_matches.items():
                match_conditions[f"metadata.{key}"] = value

            # Add LIKE patterns using regex
            for key, pattern in criteria.like_patterns.items():
                regex_pattern = self._convert_like_to_regex(pattern)
                match_conditions[f"metadata.{key}"] = {"$regex": regex_pattern, "$options": "i"}

            # Add range filters
            for key, range_filter in criteria.range_filters.items():
                field_conditions = {}
                if 'gte' in range_filter:
                    field_conditions["$gte"] = range_filter['gte']
                if 'lte' in range_filter:
                    field_conditions["$lte"] = range_filter['lte']
                if 'gt' in range_filter:
                    field_conditions["$gt"] = range_filter['gt']
                if 'lt' in range_filter:
                    field_conditions["$lt"] = range_filter['lt']

                if field_conditions:
                    match_conditions[f"metadata.{key}"] = field_conditions

            # Add exists filters
            for key in criteria.exists_filters:
                match_conditions[f"metadata.{key}"] = {"$exists": True}

            # Execute query
            elements = list(self.db.elements.find(match_conditions, {"element_pk": 1}).limit(1000))
            element_pks = [elem["element_pk"] for elem in elements]

            results = []
            for element_pk in element_pks:
                results.append({
                    'element_pk': element_pk,
                    'scores': {
                        'metadata_relevance': 1.0
                    }
                })

            return results

        except Exception as e:
            logger.error(f"Error executing metadata criteria: {str(e)}")
            return []

    def _execute_element_criteria(self, criteria: ElementSearchCriteria) -> List[Dict[str, Any]]:
        """Execute element-based filtering using MongoDB queries."""
        try:
            # Build MongoDB query for element filtering
            match_conditions = {}

            # Add element type filter
            if criteria.element_types:
                type_values = self._prepare_element_type_query(criteria.element_types)
                if type_values:
                    if len(type_values) == 1:
                        match_conditions["element_type"] = type_values[0]
                    else:
                        match_conditions["element_type"] = {"$in": type_values}

            # Add document ID filters
            if criteria.doc_ids:
                match_conditions["doc_id"] = {"$in": criteria.doc_ids}

            if criteria.exclude_doc_ids:
                match_conditions["doc_id"] = {"$nin": criteria.exclude_doc_ids}

            # Add content length filters using MongoDB's string length operator
            content_length_conditions = {}
            if hasattr(criteria, 'content_length_min') and criteria.content_length_min is not None:
                content_length_conditions["$gte"] = criteria.content_length_min
            if hasattr(criteria, 'content_length_max') and criteria.content_length_max is not None:
                content_length_conditions["$lte"] = criteria.content_length_max

            if content_length_conditions:
                # Use $expr to evaluate string length
                match_conditions["$expr"] = {
                    "$and": [
                        {f"${op}": [{"$strLenCP": "$content_preview"}, value]}
                        for op, value in content_length_conditions.items()
                    ]
                }

            # Add parent element filters
            if hasattr(criteria, 'parent_element_ids') and criteria.parent_element_ids:
                match_conditions["parent_id"] = {"$in": criteria.parent_element_ids}

            # Execute query
            elements = list(self.db.elements.find(match_conditions, {"element_pk": 1}).limit(1000))
            element_pks = [elem["element_pk"] for elem in elements]

            results = []
            for element_pk in element_pks:
                results.append({
                    'element_pk': element_pk,
                    'scores': {
                        'element_match': 1.0
                    }
                })

            return results

        except Exception as e:
            logger.error(f"Error executing element criteria: {str(e)}")
            return []

    def _combine_results(self, all_results: List[Tuple[str, List[Dict[str, Any]]]],
                         operator: LogicalOperator) -> List[Dict[str, Any]]:
        """Combine results from multiple criteria using logical operators."""

        if not all_results:
            return []

        if len(all_results) == 1:
            return all_results[0][1]  # Return the single result set

        # Extract just the result lists
        result_sets = [results for _, results in all_results]

        if operator == LogicalOperator.AND:
            return self._intersect_results(result_sets)
        elif operator == LogicalOperator.OR:
            return self._union_results(result_sets)
        elif operator == LogicalOperator.NOT:
            # NOT operation: first set minus all other sets
            if len(result_sets) >= 2:
                return self._subtract_results(result_sets[0], result_sets[1:])
            else:
                return result_sets[0]

        return []

    @staticmethod
    def _intersect_results(result_sets: List[List[Dict[str, Any]]]) -> List[Dict[str, Any]]:
        """Find intersection of multiple result sets."""
        if not result_sets:
            return []

        # Get element_pks from all sets and combine scores
        element_pk_sets = []
        element_scores = {}  # element_pk -> combined scores

        for result_set in result_sets:
            pk_set = set()
            for result in result_set:
                element_pk = result['element_pk']
                pk_set.add(element_pk)

                # Accumulate scores
                if element_pk not in element_scores:
                    element_scores[element_pk] = {}

                for score_type, score_value in result.get('scores', {}).items():
                    if score_type not in element_scores[element_pk]:
                        element_scores[element_pk][score_type] = []
                    element_scores[element_pk][score_type].append(score_value)

            element_pk_sets.append(pk_set)

        # Find intersection
        common_pks = element_pk_sets[0]
        for pk_set in element_pk_sets[1:]:
            common_pks = common_pks.intersection(pk_set)

        # Build result list
        results = []
        for element_pk in common_pks:
            results.append({
                'element_pk': element_pk,
                'scores': element_scores[element_pk]
            })

        return results

    @staticmethod
    def _union_results(result_sets: List[List[Dict[str, Any]]]) -> List[Dict[str, Any]]:
        """Find union of multiple result sets."""
        element_scores = {}  # element_pk -> combined scores

        for result_set in result_sets:
            for result in result_set:
                element_pk = result['element_pk']

                if element_pk not in element_scores:
                    element_scores[element_pk] = {}

                for score_type, score_value in result.get('scores', {}).items():
                    if score_type not in element_scores[element_pk]:
                        element_scores[element_pk][score_type] = []
                    element_scores[element_pk][score_type].append(score_value)

        # Build result list
        results = []
        for element_pk, scores in element_scores.items():
            results.append({
                'element_pk': element_pk,
                'scores': scores
            })

        return results

    @staticmethod
    def _subtract_results(base_set: List[Dict[str, Any]],
                          subtract_sets: List[List[Dict[str, Any]]]) -> List[Dict[str, Any]]:
        """Subtract multiple sets from base set."""
        base_pks = {result['element_pk'] for result in base_set}

        # Collect all PKs to subtract
        subtract_pks = set()
        for subtract_set in subtract_sets:
            for result in subtract_set:
                subtract_pks.add(result['element_pk'])

        # Return base results that are not in subtract sets
        final_pks = base_pks - subtract_pks

        return [result for result in base_set if result['element_pk'] in final_pks]

    def _process_search_results(self, raw_results: List[Dict[str, Any]],
                                query: StructuredSearchQuery) -> List[Dict[str, Any]]:
        """Process and enrich search results with full-text content restoration."""

        # Calculate combined scores
        for result in raw_results:
            result['final_score'] = self._calculate_combined_score(
                result.get('scores', {}),
                query.score_combination,
                query.custom_weights
            )

        # Sort by final score
        raw_results.sort(key=lambda x: x['final_score'], reverse=True)

        # Enrich with element details
        enriched_results = []
        for result in raw_results:
            element_pk = result['element_pk']
            element = self.get_element(element_pk)  # This will handle full-text restoration

            if not element:
                continue

            enriched_result = {
                'element_pk': element_pk,
                'element_id': element.get('element_id'),
                'doc_id': element.get('doc_id'),
                'element_type': element.get('element_type'),
                'content_preview': element.get('content_preview'),
                'final_score': result['final_score']
            }

            # Include full content if requested and available
            if hasattr(query, 'include_full_content') and query.include_full_content and 'full_content' in element:
                enriched_result['full_content'] = element['full_content']

            if query.include_similarity_scores:
                enriched_result['scores'] = result.get('scores', {})

            if query.include_metadata:
                enriched_result['metadata'] = element.get('metadata', {})

            if query.include_topics:
                enriched_result['topics'] = self.get_embedding_topics(element_pk)

            if query.include_element_dates:
                element_id = element.get('element_id')
                if element_id:
                    enriched_result['extracted_dates'] = self.get_element_dates(element_id)
                    enriched_result['date_count'] = len(enriched_result['extracted_dates'])

            enriched_results.append(enriched_result)

        return enriched_results

    @staticmethod
    def _calculate_combined_score(scores: Dict[str, List[float] | float],
                                  combination_method: str,
                                  weights: Dict[str, float]) -> float:
        """Calculate final combined score from multiple score types."""

        if not scores:
            return 0.0

        # Average scores of the same type
        avg_scores = {}
        for score_type, score_list in scores.items():
            if isinstance(score_list, list) and len(score_list) > 0:
                avg_scores[score_type] = sum(score_list) / len(score_list)
            elif isinstance(score_list, float):
                avg_scores[score_type] = score_list

        if not avg_scores:
            return 0.0

        if combination_method == "multiply":
            final_score = 1.0
            for score_type, score in avg_scores.items():
                weight = weights.get(score_type, 1.0)
                final_score *= (score * weight)
            return final_score

        elif combination_method == "add":
            final_score = 0.0
            for score_type, score in avg_scores.items():
                weight = weights.get(score_type, 1.0)
                final_score += (score * weight)
            return final_score

        elif combination_method == "max":
            weighted_scores = []
            for score_type, score in avg_scores.items():
                weight = weights.get(score_type, 1.0)
                weighted_scores.append(score * weight)
            return max(weighted_scores)

        elif combination_method == "weighted_avg":
            total_weighted_score = 0.0
            total_weight = 0.0
            for score_type, score in avg_scores.items():
                weight = weights.get(score_type, 1.0)
                total_weighted_score += (score * weight)
                total_weight += weight
            return total_weighted_score / total_weight if total_weight > 0 else 0.0

        return 0.0

    @staticmethod
    def _compare_similarity(similarity: float, threshold: float,
                            operator: SimilarityOperator) -> bool:
        """Compare similarity score against threshold using specified operator."""
        if operator == SimilarityOperator.GREATER_THAN:
            return similarity > threshold
        elif operator == SimilarityOperator.GREATER_EQUAL:
            return similarity >= threshold
        elif operator == SimilarityOperator.LESS_THAN:
            return similarity < threshold
        elif operator == SimilarityOperator.LESS_EQUAL:
            return similarity <= threshold
        elif operator == SimilarityOperator.EQUALS:
            return abs(similarity - threshold) < 0.001  # Small epsilon for float comparison
        return False

    def _generate_embedding(self, search_text: str) -> List[float]:
        """Generate embedding for search text."""
        try:
            from ..embeddings import get_embedding_generator

            if self.embedding_generator is None:
                if not config:
                    logger.error("Config not available for embedding generator")
                    raise ValueError("Config not available")
                self.embedding_generator = get_embedding_generator(config)

            return self.embedding_generator.generate(search_text)
        except Exception as e:
            logger.error(f"Error generating embedding: {str(e)}")
            raise

    def _get_element_pks_in_date_range(self, start_date: Optional[datetime],
                                       end_date: Optional[datetime]) -> List[int]:
        """Get element_pks that have dates within the specified range using MongoDB aggregation."""
        if not (start_date or end_date):
            return []

        # Build aggregation pipeline to find elements with dates in range
        pipeline = [
            {"$match": {"extracted_dates": {"$exists": True, "$ne": []}}},
            {"$unwind": "$extracted_dates"},
            {"$match": {}}  # We'll add date conditions here
        ]

        # Add date range conditions
        date_conditions = {}
        if start_date:
            date_conditions["extracted_dates.timestamp"] = {"$gte": start_date.timestamp()}
        if end_date:
            if "extracted_dates.timestamp" in date_conditions:
                date_conditions["extracted_dates.timestamp"]["$lte"] = end_date.timestamp()
            else:
                date_conditions["extracted_dates.timestamp"] = {"$lte": end_date.timestamp()}

        pipeline[2]["$match"] = date_conditions

        # Group back to get distinct element_pks
        pipeline.extend([
            {"$group": {"_id": "$element_pk"}},
            {"$project": {"element_pk": "$_id", "_id": 0}}
        ])

        # Execute aggregation
        results = list(self.db.elements.aggregate(pipeline))
        return [result["element_pk"] for result in results]

    def _filter_by_specificity(self, element_pks: List[int],
                               allowed_levels: List[str]) -> List[int]:
        """Filter element PKs by date specificity levels."""
        if not element_pks or not allowed_levels:
            return element_pks

        # Use aggregation to filter by specificity levels
        pipeline = [
            {"$match": {"element_pk": {"$in": element_pks}, "extracted_dates": {"$exists": True, "$ne": []}}},
            {"$unwind": "$extracted_dates"},
            {"$match": {"extracted_dates.specificity_level": {"$in": allowed_levels}}},
            {"$group": {"_id": "$element_pk"}},
            {"$project": {"element_pk": "$_id", "_id": 0}}
        ]

        results = list(self.db.elements.aggregate(pipeline))
        return [result["element_pk"] for result in results]

    # ========================================
    # ENHANCED SEARCH HELPER METHODS
    # ========================================

    @staticmethod
    def supports_like_patterns() -> bool:
        """MongoDB supports regex which can emulate LIKE patterns."""
        return True

    @staticmethod
    def supports_case_insensitive_like() -> bool:
        """MongoDB regex supports case-insensitive matching."""
        return True

    @staticmethod
    def supports_element_type_enums() -> bool:
        """MongoDB implementation supports ElementType enums."""
        return True

    @staticmethod
    def _prepare_element_type_query(element_types: Union[
        ElementType,
        List[ElementType],
        str,
        List[str],
        None
    ]) -> Optional[List[str]]:
        """Prepare element type values for database queries using existing ElementType enum."""
        if element_types is None:
            return None

        if isinstance(element_types, ElementType):
            return [element_types.value]
        elif isinstance(element_types, str):
            return [element_types]
        elif isinstance(element_types, list):
            result = []
            for et in element_types:
                if isinstance(et, ElementType):
                    result.append(et.value)
                elif isinstance(et, str):
                    result.append(et)
            return result if result else None

        return None

    @staticmethod
    def get_element_types_by_category() -> Dict[str, List[ElementType]]:
        """Get categorized lists of ElementType enums."""
        return {
            "text_elements": [
                ElementType.HEADER,
                ElementType.PARAGRAPH,
                ElementType.BLOCKQUOTE,
                ElementType.TEXT_BOX
            ],

            "structural_elements": [
                ElementType.ROOT,
                ElementType.PAGE,
                ElementType.BODY,
                ElementType.PAGE_HEADER,
                ElementType.PAGE_FOOTER
            ],

            "list_elements": [
                ElementType.LIST,
                ElementType.LIST_ITEM
            ],

            "table_elements": [
                ElementType.TABLE,
                ElementType.TABLE_ROW,
                ElementType.TABLE_HEADER_ROW,
                ElementType.TABLE_CELL,
                ElementType.TABLE_HEADER
            ],

            "media_elements": [
                ElementType.IMAGE,
                ElementType.CHART,
                ElementType.SHAPE,
                ElementType.SHAPE_GROUP
            ],

            "code_elements": [
                ElementType.CODE_BLOCK
            ],

            "presentation_elements": [
                ElementType.SLIDE,
                ElementType.SLIDE_NOTES,
                ElementType.PRESENTATION_BODY,
                ElementType.SLIDE_MASTERS,
                ElementType.SLIDE_TEMPLATES,
                ElementType.SLIDE_LAYOUT,
                ElementType.SLIDE_MASTER
            ],

            "data_elements": [
                ElementType.JSON_OBJECT,
                ElementType.JSON_ARRAY,
                ElementType.JSON_FIELD,
                ElementType.JSON_ITEM
            ],

            "xml_elements": [
                ElementType.XML_ELEMENT,
                ElementType.XML_TEXT,
                ElementType.XML_LIST,
                ElementType.XML_OBJECT
            ]
        }

    def find_elements_by_category(self, category: str, **other_filters) -> List[Dict[str, Any]]:
        """Find elements by predefined category using ElementType enums."""
        categories = self.get_element_types_by_category()

        if category not in categories:
            available = list(categories.keys())
            raise ValueError(f"Unknown category: {category}. Available: {available}")

        element_types = categories[category]
        query = {"element_type": element_types}
        query.update(other_filters)

        return self.find_elements(query)

    @staticmethod
    def _convert_like_to_regex(like_pattern: str) -> str:
        """Convert SQL LIKE pattern to MongoDB regex pattern."""
        # Escape special regex characters except % and _
        escaped = re.escape(like_pattern)

        # Convert % to .* (match any characters)
        escaped = escaped.replace(r'\%', '.*')

        # Convert _ to . (match single character)
        escaped = escaped.replace(r'\_', '.')

        return escaped

    # ========================================
    # TOPIC SUPPORT METHODS (Enhanced)
    # ========================================

    def supports_topics(self) -> bool:
        """MongoDB implementation supports topics."""
        return True

    def store_embedding_with_topics(self, element_pk: int, embedding: VectorType,
                                    topics: List[str], confidence: float = 1.0) -> None:
        """Store embedding for an element with topic assignments."""
        if not self.db:
            raise ValueError("Database not initialized")

        # Verify element exists
        element = self.db.elements.find_one({"element_pk": element_pk})
        if not element:
            raise ValueError(f"Element not found: {element_pk}")

        # Update vector dimension based on actual data
        self.vector_dimension = max(self.vector_dimension, len(embedding))

        try:
            # Store or update embedding with topics
            self.db.embeddings.update_one(
                {"element_pk": element_pk},
                {
                    "$set": {
                        "embedding": embedding,
                        "dimensions": len(embedding),
                        "topics": topics,
                        "confidence": confidence,
                        "created_at": time.time()
                    }
                },
                upsert=True
            )

            logger.debug(f"Stored embedding with topics for element {element_pk}: {topics}")

        except Exception as e:
            logger.error(f"Error storing embedding with topics for {element_pk}: {str(e)}")
            raise

    def search_by_text_and_topics(self, search_text: str = None,
                                  include_topics: Optional[List[str]] = None,
                                  exclude_topics: Optional[List[str]] = None,
                                  min_confidence: float = 0.7,
                                  limit: int = 10) -> List[Dict[str, Any]]:
        """Search elements by text with topic filtering using regex patterns."""
        if not self.db:
            raise ValueError("Database not initialized")

        try:
            # Generate embedding for search text if provided
            query_embedding = None
            if search_text:
                try:
                    from ..embeddings import get_embedding_generator
                    if self.embedding_generator is None:
                        self.embedding_generator = get_embedding_generator(config)
                except ImportError as e:
                    logger.error(f"Embedding generator not available: {str(e)}")
                    raise ValueError("Embedding libraries are not installed.")

                query_embedding = self.embedding_generator.generate(search_text)

            # Build aggregation pipeline for topic-aware search
            if search_text and self.vector_search:
                return self._search_by_text_and_topics_vector(
                    query_embedding, include_topics, exclude_topics, min_confidence, limit
                )
            else:
                return self._search_by_text_and_topics_fallback(
                    query_embedding, include_topics, exclude_topics, min_confidence, limit
                )

        except Exception as e:
            logger.error(f"Error in topic-aware search: {str(e)}")
            return []

    def _search_by_text_and_topics_vector(self, query_embedding: VectorType,
                                          include_topics: Optional[List[str]] = None,
                                          exclude_topics: Optional[List[str]] = None,
                                          min_confidence: float = 0.7,
                                          limit: int = 10) -> List[Dict[str, Any]]:
        """Search using MongoDB Atlas Vector Search with topic filtering."""
        try:
            # Start with vector search pipeline
            pipeline = [{
                "$vectorSearch": {
                    "index": "embeddings_vector_index",
                    "path": "embedding",
                    "queryVector": query_embedding,
                    "numCandidates": limit * 5,
                    "limit": limit * 10
                }
            }]

            # Add confidence and topic filtering
            match_conditions = {"confidence": {"$gte": min_confidence}}

            # Add topic filtering conditions
            match_conditions = self._add_topic_filters_mongodb(match_conditions, include_topics, exclude_topics)

            if match_conditions:
                pipeline.append({"$match": match_conditions})

            # Add projection and limit
            pipeline.extend([
                {
                    "$project": {
                        "_id": 0,
                        "element_pk": 1,
                        "confidence": 1,
                        "topics": 1,
                        "score": {"$meta": "vectorSearchScore"}
                    }
                },
                {
                    "$limit": limit
                }
            ])

            # Execute the search
            results = list(self.db.embeddings.aggregate(pipeline))

            # Format results
            formatted_results = []
            for doc in results:
                formatted_results.append({
                    'element_pk': doc["element_pk"],
                    'similarity': float(doc["score"]),
                    'confidence': float(doc["confidence"]),
                    'topics': doc.get("topics", [])
                })

            return formatted_results

        except Exception as e:
            logger.error(f"Error using vector search with topics: {str(e)}")
            raise

    def _search_by_text_and_topics_fallback(self, query_embedding: Optional[VectorType] = None,
                                            include_topics: Optional[List[str]] = None,
                                            exclude_topics: Optional[List[str]] = None,
                                            min_confidence: float = 0.7,
                                            limit: int = 10) -> List[Dict[str, Any]]:
        """Fallback search using aggregation pipeline with topic filtering."""

        # Base query with confidence filtering
        match_conditions = {"confidence": {"$gte": min_confidence}}

        # Add topic filtering conditions
        match_conditions = self._add_topic_filters_mongodb(match_conditions, include_topics, exclude_topics)

        # Build aggregation pipeline
        pipeline = [
            {"$match": match_conditions},
            {
                "$project": {
                    "_id": 0,
                    "element_pk": 1,
                    "embedding": 1,
                    "confidence": 1,
                    "topics": 1
                }
            }
        ]

        # Execute aggregation to get filtered embeddings
        filtered_embeddings = list(self.db.embeddings.aggregate(pipeline))

        # Calculate similarities if we have a query embedding
        results = []
        for doc in filtered_embeddings:
            result = {
                'element_pk': doc["element_pk"],
                'confidence': float(doc["confidence"]),
                'topics': doc.get("topics", [])
            }

            # Calculate similarity if we have a query embedding
            if query_embedding:
                try:
                    embedding = doc["embedding"]
                    if embedding and len(embedding) == len(query_embedding):
                        if NUMPY_AVAILABLE:
                            # Use NumPy for efficient calculation
                            query_array = np.array(query_embedding)
                            embedding_array = np.array(embedding)
                            dot_product = np.dot(query_array, embedding_array)
                            norm1 = np.linalg.norm(query_array)
                            norm2 = np.linalg.norm(embedding_array)
                            similarity = float(dot_product / (norm1 * norm2)) if norm1 != 0 and norm2 != 0 else 0.0
                        else:
                            # Pure Python calculation
                            dot_product = sum(a * b for a, b in zip(query_embedding, embedding))
                            mag1 = sum(a * a for a in query_embedding) ** 0.5
                            mag2 = sum(b * b for b in embedding) ** 0.5
                            similarity = float(dot_product / (mag1 * mag2)) if mag1 != 0 and mag2 != 0 else 0.0
                        result['similarity'] = similarity
                    else:
                        result['similarity'] = 0.0
                except Exception as e:
                    logger.warning(f"Error calculating similarity for element {doc['element_pk']}: {str(e)}")
                    result['similarity'] = 0.0
            else:
                result['similarity'] = 1.0  # No text search, all results have equal similarity

            results.append(result)

        # Sort by similarity if we calculated it
        if query_embedding:
            results.sort(key=lambda x: x['similarity'], reverse=True)

        return results[:limit]

    def _add_topic_filters_mongodb(self, match_conditions: Dict[str, Any],
                                   include_topics: Optional[List[str]] = None,
                                   exclude_topics: Optional[List[str]] = None) -> Dict[str, Any]:
        """Add topic filtering conditions to MongoDB match query."""

        # Add include topic filters
        if include_topics:
            # Convert LIKE patterns to regex patterns and create OR conditions
            include_conditions = []
            for topic_pattern in include_topics:
                # Convert SQL LIKE pattern to regex
                regex_pattern = self._convert_like_to_regex(topic_pattern)
                include_conditions.append({"topics": {"$regex": regex_pattern, "$options": "i"}})

            if include_conditions:
                match_conditions["$or"] = include_conditions

        # Add exclude topic filters
        if exclude_topics:
            exclude_conditions = []
            for topic_pattern in exclude_topics:
                # Convert SQL LIKE pattern to regex
                regex_pattern = self._convert_like_to_regex(topic_pattern)
                exclude_conditions.append({"topics": {"$not": {"$regex": regex_pattern, "$options": "i"}}})

            if exclude_conditions:
                match_conditions["$and"] = exclude_conditions

        return match_conditions

    def get_topic_statistics(self) -> Dict[str, Dict[str, Any]]:
        """Get statistics about topic distribution across embeddings."""
        if not self.db:
            raise ValueError("Database not initialized")

        try:
            # Use MongoDB aggregation pipeline to get topic statistics
            pipeline = [
                # Match embeddings that have topics
                {"$match": {"topics": {"$exists": True, "$ne": []}}},

                # Unwind topics array to get individual topics
                {"$unwind": "$topics"},

                # Lookup elements to get doc_id
                {
                    "$lookup": {
                        "from": "elements",
                        "localField": "element_pk",
                        "foreignField": "element_pk",
                        "as": "element"
                    }
                },

                # Unwind element array
                {"$unwind": "$element"},

                # Group by topic to get statistics
                {
                    "$group": {
                        "_id": "$topics",
                        "embedding_count": {"$sum": 1},
                        "document_count": {"$addToSet": "$element.doc_id"},
                        "avg_confidence": {"$avg": "$confidence"}
                    }
                },

                # Project final results
                {
                    "$project": {
                        "_id": 0,
                        "topic": "$_id",
                        "embedding_count": 1,
                        "document_count": {"$size": "$document_count"},
                        "avg_embedding_confidence": "$avg_confidence"
                    }
                },

                # Sort by embedding count descending
                {"$sort": {"embedding_count": -1}}
            ]

            results = list(self.db.embeddings.aggregate(pipeline))

            # Format results into the expected dictionary structure
            statistics = {}
            for result in results:
                statistics[result["topic"]] = {
                    'embedding_count': int(result["embedding_count"]),
                    'document_count': int(result["document_count"]),
                    'avg_embedding_confidence': float(result["avg_embedding_confidence"])
                }

            return statistics

        except Exception as e:
            logger.error(f"Error getting topic statistics: {str(e)}")
            return {}

    def get_embedding_topics(self, element_pk: int) -> List[str]:
        """Get topics assigned to a specific embedding."""
        if not self.db:
            raise ValueError("Database not initialized")

        try:
            embedding_doc = self.db.embeddings.find_one(
                {"element_pk": element_pk},
                {"topics": 1}
            )

            if not embedding_doc:
                return []

            return embedding_doc.get("topics", [])

        except Exception as e:
            logger.error(f"Error getting topics for element {element_pk}: {str(e)}")
            return []

    # ========================================
    # HIERARCHY METHODS
    # ========================================

    def get_results_outline(self, elements: List[Tuple[int, float]]) -> List["ElementHierarchical"]:
        """Create a hierarchical outline showing element ancestry."""
        from .element_element import ElementHierarchical

        # Dictionary to store element_pk -> score mapping for quick lookup
        element_scores = {element_pk: score for element_pk, score in elements}

        # Set to track processed element_pks to avoid duplicates
        processed_elements = set()

        # Final result structure
        result_tree: List[ElementHierarchical] = []

        # Process each element from the search results
        for element_pk, score in elements:
            if element_pk in processed_elements:
                continue

            # Find the complete ancestry path for this element
            ancestry_path = self._get_element_ancestry_path(element_pk)

            if not ancestry_path:
                continue

            # Mark this element as processed
            processed_elements.add(element_pk)

            # Start with the root level
            current_level = result_tree

            # Process each ancestor from root to the target element
            for i, ancestor in enumerate(ancestry_path):
                ancestor_pk = ancestor.element_pk

                # Check if this ancestor is already in the current level
                existing_idx = None
                for idx, existing_element in enumerate(current_level):
                    if existing_element.element_pk == ancestor_pk:
                        existing_idx = idx
                        break

                if existing_idx is not None:
                    # Ancestor exists, get its children
                    current_level = current_level[existing_idx].child_elements  # Get children list
                else:
                    # Ancestor doesn't exist, add it with its score (or None if not in search results)
                    ancestor_score = element_scores.get(ancestor_pk)
                    children = []
                    ancestor.score = ancestor_score
                    h_ancestor = ancestor.to_hierarchical()
                    h_ancestor.child_elements = children
                    current_level.append(h_ancestor)
                    current_level = children

        return result_tree

    def _get_element_ancestry_path(self, element_pk: int) -> List[ElementBase]:
        """Get the complete ancestry path for an element, from root to the element itself."""
        # Get the element
        element_dict = self.get_element(element_pk)
        if not element_dict:
            return []

        # Convert to ElementBase instance
        element = ElementBase(**element_dict)

        # Start building the ancestry path with the element itself
        ancestry = [element]

        # Track to avoid circular references
        visited = {element_pk}

        # Current element to process
        current_pk = element_pk

        # Traverse up the hierarchy using parent_id
        while True:
            # Get the current element
            current_element = self.get_element(current_pk)
            if not current_element:
                break

            # Get parent ID
            parent_id = current_element.get('parent_id')
            if not parent_id:
                break

            # Get the parent element
            parent_dict = self.get_element(parent_id)
            if not parent_dict:
                break

            # Check for circular references
            parent_pk = parent_dict.get('id') or parent_dict.get('pk') or parent_dict.get('element_id')
            if parent_pk in visited:
                break

            # Convert to ElementBase
            parent = ElementBase(**parent_dict)

            # Add to visited set
            visited.add(parent_pk)

            # Add parent to the beginning of the ancestry list (root first)
            ancestry.insert(0, parent)

            # Move up to the parent
            current_pk = parent_id

        return ancestry

    # ========================================
    # DATE STORAGE AND SEARCH METHODS
    # ========================================

    def store_element_dates(self, element_id: str, dates: List[Dict[str, Any]]) -> None:
        """Store extracted dates associated with an element using MongoDB's embedded documents."""
        if not self.db:
            raise ValueError("Database not initialized")

        try:
            # Update the element document to include the extracted dates
            self.db.elements.update_one(
                {"element_id": element_id},
                {"$set": {"extracted_dates": dates}}
            )

        except Exception as e:
            logger.error(f"Error storing dates for element {element_id}: {str(e)}")
            raise

    def get_element_dates(self, element_id: str) -> List[Dict[str, Any]]:
        """Get all dates associated with an element."""
        if not self.db:
            raise ValueError("Database not initialized")

        try:
            element = self.db.elements.find_one(
                {"element_id": element_id},
                {"extracted_dates": 1}
            )

            if not element or "extracted_dates" not in element:
                return []

            return element["extracted_dates"]

        except Exception as e:
            logger.error(f"Error getting dates for element {element_id}: {str(e)}")
            return []

    def store_embedding_with_dates(self, element_id: str, embedding: List[float],
                                   dates: List[Dict[str, Any]]) -> None:
        """Store both embedding and dates for an element in a single operation."""
        # Get element_pk for the embedding
        element = self.get_element(element_id)
        if not element:
            raise ValueError(f"Element not found: {element_id}")

        element_pk = element["element_pk"]

        # Store embedding and dates separately in MongoDB
        self.store_embedding(element_pk, embedding)
        self.store_element_dates(element_id, dates)

    def delete_element_dates(self, element_id: str) -> bool:
        """Delete all dates associated with an element."""
        if not self.db:
            raise ValueError("Database not initialized")

        try:
            result = self.db.elements.update_one(
                {"element_id": element_id},
                {"$unset": {"extracted_dates": ""}}
            )

            return result.modified_count > 0

        except Exception as e:
            logger.error(f"Error deleting dates for element {element_id}: {str(e)}")
            return False

    def search_elements_by_date_range(self, start_date: datetime, end_date: datetime,
                                      limit: int = 100) -> List[Dict[str, Any]]:
        """Find elements that contain dates within a specified range."""
        if not self.db:
            raise ValueError("Database not initialized")

        try:
            # Use aggregation pipeline to find elements with dates in range
            pipeline = [
                {"$match": {"extracted_dates": {"$exists": True, "$ne": []}}},
                {"$unwind": "$extracted_dates"},
                {
                    "$match": {
                        "extracted_dates.timestamp": {
                            "$gte": start_date.timestamp(),
                            "$lte": end_date.timestamp()
                        }
                    }
                },
                {
                    "$group": {
                        "_id": "$element_pk",
                        "element": {"$first": "$$ROOT"}
                    }
                },
                {"$replaceRoot": {"newRoot": "$element"}},
                {"$project": {"extracted_dates": 0}},  # Remove dates array from final result
                {"$limit": limit}
            ]

            elements = list(self.db.elements.aggregate(pipeline))

            # Remove MongoDB's _id field from each element
            for element in elements:
                if "_id" in element:
                    del element["_id"]

            return elements

        except Exception as e:
            logger.error(f"Error searching elements by date range: {str(e)}")
            return []

    def search_by_text_and_date_range(self,
                                      search_text: str,
                                      start_date: Optional[datetime] = None,
                                      end_date: Optional[datetime] = None,
                                      limit: int = 10) -> List[Tuple[int, float]]:
        """Search elements by semantic similarity AND date range."""
        # First get elements in date range if specified
        date_element_pks = None
        if start_date or end_date:
            date_element_pks = set(self._get_element_pks_in_date_range(start_date, end_date))

        # Perform text similarity search
        text_results = self.search_by_text(search_text, limit=limit * 2)  # Get more to allow for filtering

        # Filter by date results if we have them
        if date_element_pks is not None:
            filtered_results = []
            for element_pk, similarity in text_results:
                if element_pk in date_element_pks:
                    filtered_results.append((element_pk, similarity))
            return filtered_results[:limit]
        else:
            return text_results[:limit]

    def search_by_embedding_and_date_range(self,
                                           query_embedding: List[float],
                                           start_date: Optional[datetime] = None,
                                           end_date: Optional[datetime] = None,
                                           limit: int = 10) -> List[Tuple[int, float]]:
        """Search elements by embedding similarity AND date range."""
        # First get elements in date range if specified
        date_element_pks = None
        if start_date or end_date:
            date_element_pks = set(self._get_element_pks_in_date_range(start_date, end_date))

        # Perform embedding similarity search
        embedding_results = self.search_by_embedding(query_embedding, limit=limit * 2)

        # Filter by date results if we have them
        if date_element_pks is not None:
            filtered_results = []
            for element_pk, similarity in embedding_results:
                if element_pk in date_element_pks:
                    filtered_results.append((element_pk, similarity))
            return filtered_results[:limit]
        else:
            return embedding_results[:limit]

    def get_elements_with_dates(self, limit: int = 100) -> List[Dict[str, Any]]:
        """Get all elements that have associated dates."""
        if not self.db:
            raise ValueError("Database not initialized")

        try:
            elements = list(self.db.elements.find(
                {"extracted_dates": {"$exists": True, "$ne": []}},
                {"extracted_dates": 0}  # Exclude dates from result
            ).limit(limit))

            # Remove MongoDB's _id field from each element
            for element in elements:
                if "_id" in element:
                    del element["_id"]

            return elements

        except Exception as e:
            logger.error(f"Error getting elements with dates: {str(e)}")
            return []

    def get_date_statistics(self) -> Dict[str, Any]:
        """Get statistics about dates in the database."""
        if not self.db:
            raise ValueError("Database not initialized")

        try:
            # Use aggregation pipeline to get comprehensive date statistics
            pipeline = [
                {"$match": {"extracted_dates": {"$exists": True, "$ne": []}}},
                {"$unwind": "$extracted_dates"},
                {
                    "$group": {
                        "_id": None,
                        "total_dates": {"$sum": 1},
                        "elements_with_dates": {"$addToSet": "$element_pk"},
                        "avg_confidence": {"$avg": "$extracted_dates.confidence"},
                        "earliest_timestamp": {"$min": "$extracted_dates.timestamp"},
                        "latest_timestamp": {"$max": "$extracted_dates.timestamp"},
                        "specificity_levels": {"$addToSet": "$extracted_dates.specificity_level"}
                    }
                },
                {
                    "$project": {
                        "_id": 0,
                        "total_dates": 1,
                        "elements_with_dates": {"$size": "$elements_with_dates"},
                        "avg_confidence": 1,
                        "earliest_timestamp": 1,
                        "latest_timestamp": 1,
                        "specificity_levels": 1
                    }
                }
            ]

            results = list(self.db.elements.aggregate(pipeline))

            if not results:
                return {}

            result = results[0]

            stats = {
                'total_dates': result.get('total_dates', 0),
                'elements_with_dates': result.get('elements_with_dates', 0),
                'avg_confidence': result.get('avg_confidence', 0.0)
            }

            # Format dates
            if result.get('earliest_timestamp'):
                stats['earliest_date'] = datetime.fromtimestamp(result['earliest_timestamp']).isoformat()
            if result.get('latest_timestamp'):
                stats['latest_date'] = datetime.fromtimestamp(result['latest_timestamp']).isoformat()

            # Get specificity distribution
            if result.get('specificity_levels'):
                specificity_pipeline = [
                    {"$match": {"extracted_dates": {"$exists": True, "$ne": []}}},
                    {"$unwind": "$extracted_dates"},
                    {
                        "$group": {
                            "_id": "$extracted_dates.specificity_level",
                            "count": {"$sum": 1}
                        }
                    }
                ]

                specificity_results = list(self.db.elements.aggregate(specificity_pipeline))
                specificity_dist = {}
                for spec_result in specificity_results:
                    level = spec_result['_id'] or 'unknown'
                    specificity_dist[level] = spec_result['count']

                stats['specificity_distribution'] = specificity_dist

            return stats

        except Exception as e:
            logger.error(f"Error getting date statistics: {str(e)}")
            return {}

    # ========================================
    # ADDITIONAL UTILITY METHODS
    # ========================================

    def create_vector_search_index(self) -> bool:
        """Create a vector search index for embeddings collection."""
        if not self.db:
            raise ValueError("Database not initialized")

        try:
            # Define the index
            index_definition = {
                "mappings": {
                    "dynamic": False,
                    "fields": {
                        "embedding": {
                            "dimensions": self.vector_dimension,
                            "similarity": "cosine",
                            "type": "knnVector"
                        }
                    }
                }
            }

            # Create the index
            self.db.command({
                "createSearchIndex": "embeddings",
                "name": "embeddings_vector_index",
                "definition": index_definition
            })

            logger.info(f"Created vector search index with {self.vector_dimension} dimensions")
            self.vector_search = True
            return True

        except Exception as e:
            logger.error(f"Error creating vector search index: {str(e)}")
            return False
    
    # ========================================
    # DOMAIN ENTITY METHODS
    # ========================================
    
    def store_entity(self, entity: Dict[str, Any]) -> int:
        """Store a domain entity and return its primary key."""
        if not self.db:
            raise ValueError("Database not initialized")
        
        now = time.time()
        entity_doc = {
            'entity_id': entity['entity_id'],
            'entity_type': entity['entity_type'],
            'name': entity['name'],
            'domain': entity.get('domain'),
            'attributes': entity.get('attributes', {}),
            'created_at': now,
            'updated_at': now
        }
        
        result = self.entities_collection.insert_one(entity_doc)
        # Return a numeric ID based on insertion order
        return self.entities_collection.count_documents({'_id': {'$lte': result.inserted_id}})
    
    def update_entity(self, entity_pk: int, entity: Dict[str, Any]) -> bool:
        """Update an existing domain entity."""
        if not self.db:
            raise ValueError("Database not initialized")
        
        # Find the entity by its numeric position
        entities = list(self.entities_collection.find().sort('_id', 1).limit(entity_pk))
        if len(entities) < entity_pk:
            return False
        
        entity_id = entities[entity_pk - 1]['_id']
        now = time.time()
        
        result = self.entities_collection.update_one(
            {'_id': entity_id},
            {'$set': {
                'entity_type': entity['entity_type'],
                'name': entity['name'],
                'domain': entity.get('domain'),
                'attributes': entity.get('attributes', {}),
                'updated_at': now
            }}
        )
        
        return result.modified_count > 0
    
    def delete_entity(self, entity_pk: int) -> bool:
        """Delete a domain entity and its associated mappings and relationships."""
        if not self.db:
            raise ValueError("Database not initialized")
        
        # Find the entity by its numeric position
        entities = list(self.entities_collection.find().sort('_id', 1).limit(entity_pk))
        if len(entities) < entity_pk:
            return False
        
        entity_doc = entities[entity_pk - 1]
        entity_id = entity_doc['_id']
        
        # Delete associated mappings
        self.element_entity_mappings_collection.delete_many({'entity_id': entity_id})
        
        # Delete associated relationships
        self.entity_relationships_collection.delete_many({
            '$or': [
                {'source_entity_id': entity_id},
                {'target_entity_id': entity_id}
            ]
        })
        
        # Delete the entity
        result = self.entities_collection.delete_one({'_id': entity_id})
        return result.deleted_count > 0
    
    def get_entity(self, entity_pk: int) -> Optional[Dict[str, Any]]:
        """Get a domain entity by its primary key."""
        if not self.db:
            raise ValueError("Database not initialized")
        
        # Find the entity by its numeric position
        entities = list(self.entities_collection.find().sort('_id', 1).limit(entity_pk))
        if len(entities) < entity_pk:
            return None
        
        entity_doc = entities[entity_pk - 1]
        # Convert MongoDB document to standard format
        entity_doc['entity_pk'] = entity_pk
        entity_doc.pop('_id', None)
        return entity_doc
    
    def get_entities_for_document(self, doc_id: str) -> List[Dict[str, Any]]:
        """Get all entities associated with a document."""
        if not self.db:
            raise ValueError("Database not initialized")
        
        # Find all elements for the document
        elements = self.elements_collection.find({'doc_id': doc_id})
        element_ids = [e['_id'] for e in elements]
        
        # Find all entity mappings for these elements
        mappings = self.element_entity_mappings_collection.find({
            'element_id': {'$in': element_ids}
        })
        entity_ids = list(set(m['entity_id'] for m in mappings))
        
        # Get all entities
        entities = []
        for entity_doc in self.entities_collection.find({'_id': {'$in': entity_ids}}):
            # Calculate entity_pk based on position
            entity_pk = self.entities_collection.count_documents({'_id': {'$lte': entity_doc['_id']}})
            entity_doc['entity_pk'] = entity_pk
            entity_doc.pop('_id', None)
            entities.append(entity_doc)
        
        return entities
    
    def store_element_entity_mapping(self, mapping: Dict[str, Any]) -> None:
        """Store element-to-entity mapping."""
        if not self.db:
            raise ValueError("Database not initialized")
        
        # Convert entity_pk to MongoDB _id
        entities = list(self.entities_collection.find().sort('_id', 1).limit(mapping['entity_pk']))
        if len(entities) < mapping['entity_pk']:
            raise ValueError(f"Entity with pk {mapping['entity_pk']} not found")
        entity_id = entities[mapping['entity_pk'] - 1]['_id']
        
        # Convert element_pk to MongoDB _id
        elements = list(self.elements_collection.find().sort('_id', 1).limit(mapping['element_pk']))
        if len(elements) < mapping['element_pk']:
            raise ValueError(f"Element with pk {mapping['element_pk']} not found")
        element_id = elements[mapping['element_pk'] - 1]['_id']
        
        now = time.time()
        mapping_doc = {
            'element_id': element_id,
            'entity_id': entity_id,
            'relationship_type': mapping['relationship_type'],
            'confidence': mapping.get('confidence', 1.0),
            'extraction_rule': mapping.get('extraction_rule'),
            'metadata': mapping.get('metadata', {}),
            'created_at': now
        }
        
        # Upsert the mapping
        self.element_entity_mappings_collection.replace_one(
            {
                'element_id': element_id,
                'entity_id': entity_id,
                'relationship_type': mapping['relationship_type']
            },
            mapping_doc,
            upsert=True
        )
    
    def delete_element_entity_mappings(self, element_pk: int = None, entity_pk: int = None) -> int:
        """Delete element-entity mappings by element_pk, entity_pk, or both."""
        if not self.db:
            raise ValueError("Database not initialized")
        
        if not element_pk and not entity_pk:
            raise ValueError("At least one of element_pk or entity_pk must be provided")
        
        filter_query = {}
        
        if element_pk:
            elements = list(self.elements_collection.find().sort('_id', 1).limit(element_pk))
            if len(elements) >= element_pk:
                element_id = elements[element_pk - 1]['_id']
                filter_query['element_id'] = element_id
        
        if entity_pk:
            entities = list(self.entities_collection.find().sort('_id', 1).limit(entity_pk))
            if len(entities) >= entity_pk:
                entity_id = entities[entity_pk - 1]['_id']
                filter_query['entity_id'] = entity_id
        
        if not filter_query:
            return 0
        
        result = self.element_entity_mappings_collection.delete_many(filter_query)
        return result.deleted_count
    
    def store_entity_relationship(self, relationship: Dict[str, Any]) -> int:
        """Store entity-to-entity relationship and return the relationship_id."""
        if not self.db:
            raise ValueError("Database not initialized")
        
        # Convert entity_pks to MongoDB _ids
        source_entities = list(self.entities_collection.find().sort('_id', 1).limit(relationship['source_entity_pk']))
        if len(source_entities) < relationship['source_entity_pk']:
            raise ValueError(f"Source entity with pk {relationship['source_entity_pk']} not found")
        source_entity_id = source_entities[relationship['source_entity_pk'] - 1]['_id']
        
        target_entities = list(self.entities_collection.find().sort('_id', 1).limit(relationship['target_entity_pk']))
        if len(target_entities) < relationship['target_entity_pk']:
            raise ValueError(f"Target entity with pk {relationship['target_entity_pk']} not found")
        target_entity_id = target_entities[relationship['target_entity_pk'] - 1]['_id']
        
        now = time.time()
        relationship_doc = {
            'source_entity_id': source_entity_id,
            'target_entity_id': target_entity_id,
            'relationship_type': relationship['relationship_type'],
            'confidence': relationship.get('confidence', 1.0),
            'domain': relationship.get('domain'),
            'metadata': relationship.get('metadata', {}),
            'created_at': now
        }
        
        result = self.entity_relationships_collection.insert_one(relationship_doc)
        # Return a numeric ID based on insertion order
        return self.entity_relationships_collection.count_documents({'_id': {'$lte': result.inserted_id}})
    
    def update_entity_relationship(self, relationship_id: int, relationship: Dict[str, Any]) -> bool:
        """Update an entity-to-entity relationship."""
        if not self.db:
            raise ValueError("Database not initialized")
        
        # Find the relationship by its numeric position
        relationships = list(self.entity_relationships_collection.find().sort('_id', 1).limit(relationship_id))
        if len(relationships) < relationship_id:
            return False
        
        rel_mongo_id = relationships[relationship_id - 1]['_id']
        
        # Convert entity_pks to MongoDB _ids
        source_entities = list(self.entities_collection.find().sort('_id', 1).limit(relationship['source_entity_pk']))
        if len(source_entities) < relationship['source_entity_pk']:
            return False
        source_entity_id = source_entities[relationship['source_entity_pk'] - 1]['_id']
        
        target_entities = list(self.entities_collection.find().sort('_id', 1).limit(relationship['target_entity_pk']))
        if len(target_entities) < relationship['target_entity_pk']:
            return False
        target_entity_id = target_entities[relationship['target_entity_pk'] - 1]['_id']
        
        result = self.entity_relationships_collection.update_one(
            {'_id': rel_mongo_id},
            {'$set': {
                'source_entity_id': source_entity_id,
                'target_entity_id': target_entity_id,
                'relationship_type': relationship['relationship_type'],
                'confidence': relationship.get('confidence', 1.0),
                'domain': relationship.get('domain'),
                'metadata': relationship.get('metadata', {})
            }}
        )
        
        return result.modified_count > 0
    
    def delete_entity_relationships(self, source_entity_pk: int = None, target_entity_pk: int = None) -> int:
        """Delete entity relationships by source, target, or both."""
        if not self.db:
            raise ValueError("Database not initialized")
        
        if not source_entity_pk and not target_entity_pk:
            raise ValueError("At least one of source_entity_pk or target_entity_pk must be provided")
        
        filter_query = {'$or': []}
        
        if source_entity_pk:
            entities = list(self.entities_collection.find().sort('_id', 1).limit(source_entity_pk))
            if len(entities) >= source_entity_pk:
                entity_id = entities[source_entity_pk - 1]['_id']
                filter_query['$or'].extend([
                    {'source_entity_id': entity_id},
                    {'target_entity_id': entity_id}
                ])
        
        if target_entity_pk and not source_entity_pk:
            entities = list(self.entities_collection.find().sort('_id', 1).limit(target_entity_pk))
            if len(entities) >= target_entity_pk:
                entity_id = entities[target_entity_pk - 1]['_id']
                filter_query['$or'].extend([
                    {'source_entity_id': entity_id},
                    {'target_entity_id': entity_id}
                ])
        
        if source_entity_pk and target_entity_pk:
            source_entities = list(self.entities_collection.find().sort('_id', 1).limit(source_entity_pk))
            target_entities = list(self.entities_collection.find().sort('_id', 1).limit(target_entity_pk))
            if len(source_entities) >= source_entity_pk and len(target_entities) >= target_entity_pk:
                source_id = source_entities[source_entity_pk - 1]['_id']
                target_id = target_entities[target_entity_pk - 1]['_id']
                filter_query = {
                    'source_entity_id': source_id,
                    'target_entity_id': target_id
                }
        
        if not filter_query.get('$or') and '$or' in filter_query:
            return 0
        
        result = self.entity_relationships_collection.delete_many(filter_query)
        return result.deleted_count
    
    def get_entity_relationships(self, entity_pk: int) -> List[Dict[str, Any]]:
        """Get all relationships for an entity (both as source and target)."""
        if not self.db:
            raise ValueError("Database not initialized")
        
        # Find the entity by its numeric position
        entities = list(self.entities_collection.find().sort('_id', 1).limit(entity_pk))
        if len(entities) < entity_pk:
            return []
        
        entity_id = entities[entity_pk - 1]['_id']
        
        # Find all relationships
        relationships = []
        for rel_doc in self.entity_relationships_collection.find({
            '$or': [
                {'source_entity_id': entity_id},
                {'target_entity_id': entity_id}
            ]
        }):
            # Calculate relationship_id based on position
            relationship_id = self.entity_relationships_collection.count_documents({'_id': {'$lte': rel_doc['_id']}})
            
            # Get source and target entity details
            source_entity = self.entities_collection.find_one({'_id': rel_doc['source_entity_id']})
            target_entity = self.entities_collection.find_one({'_id': rel_doc['target_entity_id']})
            
            rel_doc['relationship_id'] = relationship_id
            rel_doc['source_entity_id'] = source_entity['entity_id'] if source_entity else None
            rel_doc['target_entity_id'] = target_entity['entity_id'] if target_entity else None
            
            # Calculate entity_pks
            rel_doc['source_entity_pk'] = self.entities_collection.count_documents({'_id': {'$lte': rel_doc['source_entity_id']}})
            rel_doc['target_entity_pk'] = self.entities_collection.count_documents({'_id': {'$lte': rel_doc['target_entity_id']}})
            
            rel_doc.pop('_id', None)
            relationships.append(rel_doc)
        
        return relationships


if __name__ == "__main__":
    # Example demonstrating structured search with MongoDB and full-text support
    conn_params = {
        'host': 'localhost',
        'port': 27017,
        'db_name': 'go-doc-go',
        'store_full_text': True,
        'index_full_text': True,
        'compress_full_text': True,
        'full_text_max_length': 50000
    }

    db = MongoDBDocumentDatabase(conn_params)
    db.initialize()

    # Show backend capabilities
    capabilities = db.get_backend_capabilities()
    print(f"MongoDB supports {len(capabilities.supported)} capabilities:")
    for cap in sorted(capabilities.get_supported_list()):
        print(f"  ✓ {cap}")

    # Show full-text configuration
    config = db.get_text_storage_config()
    print(f"\nFull-text configuration:")
    print(f"  Store: {config['store_full_text']}")
    print(f"  Index: {config['index_full_text']}")
    print(f"  Compress: {config['compress_full_text']}")
    print(f"  Search available: {config['search_capabilities']['can_search_full_text']}")

    # Get recommendations
    recommendations = db.get_full_text_usage_recommendations()
    if recommendations['suggestions']:
        print(f"\nOptimization suggestions:")
        for suggestion in recommendations['suggestions']:
            print(f"  💡 {suggestion}")
