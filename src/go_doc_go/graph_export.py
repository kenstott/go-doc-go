"""
Graph export functionality for transferring analytics data to Neo4j.

This module provides functionality to export processed documents, elements,
and relationships from analytics storage (e.g., Parquet) to Neo4j graph database
for graph visualization and analysis.
"""

import logging
from typing import Dict, Any, List, Optional
from datetime import datetime

logger = logging.getLogger(__name__)


class GraphExporter:
    """Export analytics data to Neo4j graph database."""

    def __init__(self, config, source_analytics_type: str = 'parquet'):
        """
        Initialize graph exporter.

        Args:
            config: Configuration object
            source_analytics_type: Type of source analytics storage to read from
        """
        self.config = config
        self.source_analytics_type = source_analytics_type

        # Initialize source analytics storage
        analytics_config = config.get_analytics_config()
        if not analytics_config or not analytics_config.get('enabled'):
            raise ValueError("Analytics must be enabled for graph export")

        # Find the source analytics output
        outputs = analytics_config.get('outputs', [])
        source_output = None
        for output in outputs:
            if output.get('type') == source_analytics_type:
                source_output = output
                break

        if not source_output:
            raise ValueError(f"No {source_analytics_type} analytics output found")

        # Create source analytics storage
        from go_doc_go.storage_adapters.factory import StorageFactory
        self.source_storage = StorageFactory.create_analytics_storage(source_output)

        # Initialize Neo4j target storage
        neo4j_config = config.config.get('processing', {}).get('neo4j_export', {})
        if not neo4j_config.get('enabled'):
            raise ValueError("Neo4j export is not enabled")

        # Create Neo4j configuration from export config
        neo4j_connection = neo4j_config.get('connection', {})
        neo4j_storage_config = {
            'type': 'neo4j',
            'uri': neo4j_connection.get('uri', 'bolt://localhost:7687'),
            'username': neo4j_connection.get('username', 'neo4j'),
            'password': neo4j_connection.get('password', 'password'),
            'database': neo4j_connection.get('database', 'neo4j')
        }

        self.target_storage = StorageFactory.create_analytics_storage(neo4j_storage_config)

        # Export configuration
        self.batch_size = neo4j_config.get('batch_size', 1000)
        self.export_run_id = f"export_{datetime.now().strftime('%Y%m%d_%H%M%S')}"

    def export_to_neo4j(self) -> Dict[str, Any]:
        """
        Export all analytics data to Neo4j.

        Returns:
            Dictionary with export results
        """
        try:
            logger.info(f"Starting graph export from {self.source_analytics_type} to Neo4j")

            # Track export progress
            exported_counts = {
                'documents': 0,
                'elements': 0,
                'relationships': 0
            }

            # Export documents
            logger.info("Exporting documents...")
            docs_exported = self._export_documents()
            exported_counts['documents'] = docs_exported

            # Export elements
            logger.info("Exporting elements...")
            elements_exported = self._export_elements()
            exported_counts['elements'] = elements_exported

            # Export relationships
            logger.info("Exporting relationships...")
            rels_exported = self._export_relationships()
            exported_counts['relationships'] = rels_exported

            logger.info(f"Graph export completed successfully: {exported_counts}")

            return {
                'success': True,
                'exported': exported_counts,
                'export_run_id': self.export_run_id
            }

        except Exception as e:
            logger.error(f"Graph export failed: {e}")
            return {
                'success': False,
                'error': str(e),
                'export_run_id': self.export_run_id
            }

    def _export_documents(self) -> int:
        """Export documents from source to Neo4j."""
        try:
            # Search for all documents in source storage
            documents = self.source_storage.search_structured(
                criteria={},  # Get all documents
                limit=10000  # Large limit to get all
            )

            if not documents:
                logger.info("No documents found to export")
                return 0

            # Convert to document format expected by Neo4j storage
            neo4j_documents = []
            for doc in documents:
                neo4j_doc = {
                    'doc_id': doc.get('doc_id'),
                    'source_name': doc.get('source_name', ''),
                    'title': doc.get('title', ''),
                    'url': doc.get('url', ''),
                    'content_type': doc.get('content_type', ''),
                    'processed_at': doc.get('processed_at', datetime.now().isoformat()),
                    'element_count': doc.get('element_count', 0),
                    'relationship_count': doc.get('relationship_count', 0)
                }
                neo4j_documents.append(neo4j_doc)

            # Store in Neo4j in batches
            total_exported = 0
            for i in range(0, len(neo4j_documents), self.batch_size):
                batch = neo4j_documents[i:i + self.batch_size]
                count = self.target_storage.append_documents(batch, self.export_run_id)
                total_exported += count
                logger.debug(f"Exported {count} documents (batch {i//self.batch_size + 1})")

            logger.info(f"Exported {total_exported} documents to Neo4j")
            return total_exported

        except Exception as e:
            logger.error(f"Failed to export documents: {e}")
            raise

    def _export_elements(self) -> int:
        """Export elements from source to Neo4j."""
        try:
            # Search for all elements in source storage
            elements = self.source_storage.search_structured(
                criteria={},  # Get all elements
                limit=50000  # Large limit to get all
            )

            if not elements:
                logger.info("No elements found to export")
                return 0

            # Filter to get only elements (not documents)
            element_data = [elem for elem in elements if elem.get('element_id')]

            if not element_data:
                logger.info("No element data found to export")
                return 0

            # Convert to format expected by Neo4j storage
            neo4j_elements = []
            for elem in element_data:
                neo4j_elem = {
                    'element_id': elem.get('element_id'),
                    'doc_id': elem.get('doc_id'),
                    'source_name': elem.get('source_name', ''),
                    'element_type': elem.get('element_type', ''),
                    'content_preview': elem.get('content_preview', elem.get('content', '')),
                    'position': elem.get('position', 0),
                    'parent_id': elem.get('parent_id'),
                    'metadata': elem.get('metadata', {})
                }
                neo4j_elements.append(neo4j_elem)

            # Store in Neo4j in batches
            total_exported = 0
            for i in range(0, len(neo4j_elements), self.batch_size):
                batch = neo4j_elements[i:i + self.batch_size]
                count = self.target_storage.append_elements(batch, self.export_run_id)
                total_exported += count
                logger.debug(f"Exported {count} elements (batch {i//self.batch_size + 1})")

            logger.info(f"Exported {total_exported} elements to Neo4j")
            return total_exported

        except Exception as e:
            logger.error(f"Failed to export elements: {e}")
            raise

    def _export_relationships(self) -> int:
        """Export relationships from source to Neo4j."""
        try:
            # For relationships, we need to use a different approach since they might not
            # be directly searchable. We'll try to get them through structured search
            # or use backend-specific methods if available.

            relationships = []

            # Try to get relationships through search
            try:
                relationships = self.source_storage.search_structured(
                    criteria={'relationship_type': '*'},  # Try to find relationships
                    limit=50000
                )
            except Exception:
                # If structured search doesn't work for relationships,
                # try other methods based on storage type
                if hasattr(self.source_storage, 'get_all_relationships'):
                    relationships = self.source_storage.get_all_relationships()
                else:
                    logger.warning("Cannot retrieve relationships from source storage")
                    return 0

            if not relationships:
                logger.info("No relationships found to export")
                return 0

            # Convert to format expected by Neo4j storage
            neo4j_relationships = []
            for rel in relationships:
                neo4j_rel = {
                    'source_id': rel.get('source_id', rel.get('source_element_id')),
                    'target_id': rel.get('target_id', rel.get('target_element_id')),
                    'relationship_type': rel.get('relationship_type', 'RELATED_TO'),
                    'doc_id': rel.get('doc_id'),
                    'source_name': rel.get('source_name', ''),
                    'metadata': rel.get('metadata', {})
                }

                # Only add relationships with valid source and target
                if neo4j_rel['source_id'] and neo4j_rel['target_id']:
                    neo4j_relationships.append(neo4j_rel)

            if not neo4j_relationships:
                logger.info("No valid relationships found to export")
                return 0

            # Store in Neo4j in batches
            total_exported = 0
            for i in range(0, len(neo4j_relationships), self.batch_size):
                batch = neo4j_relationships[i:i + self.batch_size]
                count = self.target_storage.append_relationships(batch, self.export_run_id)
                total_exported += count
                logger.debug(f"Exported {count} relationships (batch {i//self.batch_size + 1})")

            logger.info(f"Exported {total_exported} relationships to Neo4j")
            return total_exported

        except Exception as e:
            logger.error(f"Failed to export relationships: {e}")
            raise

    def get_export_status(self) -> Dict[str, Any]:
        """Get status of the current export."""
        try:
            # Get Neo4j statistics for the export run
            if hasattr(self.target_storage, 'get_run_stats'):
                stats = self.target_storage.get_run_stats(self.export_run_id)
                return {
                    'export_run_id': self.export_run_id,
                    'neo4j_stats': stats
                }
            else:
                return {
                    'export_run_id': self.export_run_id,
                    'status': 'completed'
                }
        except Exception as e:
            logger.error(f"Failed to get export status: {e}")
            return {
                'export_run_id': self.export_run_id,
                'error': str(e)
            }