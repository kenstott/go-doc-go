#!/usr/bin/env python3
"""
CLI for extracting entities/relationships using ontology rules and exporting to graph databases.
"""

import json
import logging
import os
import sys
import click
from pathlib import Path
from typing import Dict, Any, List, Optional
from datetime import datetime

from go_doc_go.config import Config
from go_doc_go.storage_adapters.factory import StorageFactory
from go_doc_go.domain.ontology import DomainOntology
from go_doc_go.ontology.entity_extractor import OntologyEntityExtractor
# Optional embedding support
try:
    from go_doc_go.embedding.client import EmbeddingClient
except ImportError:
    EmbeddingClient = None
from go_doc_go.graph_export import GraphExporter

logger = logging.getLogger(__name__)


class OntologyExtractionPipeline:
    """Complete pipeline for ontology-based entity extraction and graph export."""

    def __init__(self, config: Config, ontology_path: str):
        """
        Initialize extraction pipeline.

        Args:
            config: Go-Doc-Go configuration
            ontology_path: Path to ontology YAML file
        """
        self.config = config

        # Load ontology
        self.ontology = self._load_ontology(ontology_path)

        # Initialize analytics storage
        analytics_outputs = config.get_analytics_outputs()
        if not analytics_outputs:
            raise ValueError("No analytics outputs configured")

        self.analytics_storage = StorageFactory.create_analytics_storage(analytics_outputs[0])

        # Initialize embedding client if available
        self.embedding_client = None
        if EmbeddingClient:
            try:
                embedding_config = config.config.get('embeddings', {})
                if embedding_config.get('enabled', False):
                    self.embedding_client = EmbeddingClient(config)
            except Exception as e:
                logger.warning(f"Embedding client not available: {e}")
        else:
            logger.info("Embedding support not available - using fallback matching")

        # Initialize entity extractor
        self.extractor = OntologyEntityExtractor(self.ontology, self.embedding_client)

    def _load_ontology(self, ontology_path: str) -> DomainOntology:
        """Load ontology from YAML file."""
        if not os.path.exists(ontology_path):
            raise FileNotFoundError(f"Ontology file not found: {ontology_path}")

        import yaml
        with open(ontology_path, 'r') as f:
            ontology_dict = yaml.safe_load(f)

        # Validate ontology structure
        ontology = DomainOntology.from_dict(ontology_dict)
        validation_issues = ontology.validate()

        if validation_issues:
            logger.warning(f"Ontology validation issues: {validation_issues}")

        logger.info(f"Loaded ontology '{ontology.name}' with {len(ontology.terms)} terms")
        return ontology

    def extract_entities_and_relationships(self, max_elements: int = 10000,
                                         run_id: str = None) -> Dict[str, Any]:
        """
        Extract entities and relationships from analytics data.

        Args:
            max_elements: Maximum number of elements to process
            run_id: Optional run identifier

        Returns:
            Dictionary with extraction results
        """
        if not run_id:
            run_id = f"extraction_{datetime.now().strftime('%Y%m%d_%H%M%S')}"

        logger.info(f"Starting entity extraction for ontology '{self.ontology.name}'")

        # Get elements from analytics storage
        elements = self._get_elements_for_extraction(max_elements)
        logger.info(f"Retrieved {len(elements)} elements for processing")

        if not elements:
            return {
                'run_id': run_id,
                'extraction_summary': {
                    'total_elements_processed': 0,
                    'total_entities_extracted': 0,
                    'total_relationships_extracted': 0,
                    'error': 'No elements found in analytics storage'
                },
                'entities': [],
                'relationships': []
            }

        # Extract entities and relationships
        extraction_results = self.extractor.extract_from_elements(elements, run_id)

        return extraction_results

    def _get_elements_for_extraction(self, max_elements: int) -> List[Dict[str, Any]]:
        """Get elements from analytics storage for extraction."""
        try:
            # Try structured search first
            elements = self.analytics_storage.search_structured(
                criteria={},  # Get all elements
                limit=max_elements
            )

            # Filter to only include elements (not documents)
            element_data = [elem for elem in elements if elem.get('element_id')]

            return element_data

        except Exception as e:
            logger.error(f"Failed to retrieve elements from analytics storage: {e}")
            return []

    def export_to_graph_database(self, extraction_results: Dict[str, Any],
                                neo4j_config: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """
        Export extracted entities and relationships to Neo4j.

        Args:
            extraction_results: Results from entity extraction
            neo4j_config: Optional Neo4j configuration override

        Returns:
            Export results dictionary
        """
        logger.info("Starting export to graph database")

        try:
            # Create graph storage for extracted data
            if neo4j_config:
                graph_storage_config = {
                    'type': 'neo4j',
                    **neo4j_config
                }
            else:
                # Use config from main configuration
                neo4j_settings = self.config.config.get('processing', {}).get('neo4j_export', {})
                if not neo4j_settings.get('enabled'):
                    raise ValueError("Neo4j export not enabled in configuration")

                connection = neo4j_settings.get('connection', {})
                graph_storage_config = {
                    'type': 'neo4j',
                    'uri': connection.get('uri', 'bolt://localhost:7687'),
                    'username': connection.get('username', 'neo4j'),
                    'password': connection.get('password', 'password'),
                    'database': connection.get('database', 'neo4j')
                }

            graph_storage = StorageFactory.create_analytics_storage(graph_storage_config)

            # Prepare entities for graph storage
            entities = extraction_results.get('entities', [])
            relationships = extraction_results.get('relationships', [])

            # Convert entities to documents and elements for graph storage
            documents_for_graph = self._prepare_documents_for_graph(entities)
            elements_for_graph = self._prepare_elements_for_graph(entities)
            relationships_for_graph = self._prepare_relationships_for_graph(relationships)

            run_id = extraction_results.get('run_id', 'unknown')

            # Export to graph database
            export_results = {
                'documents_exported': 0,
                'entities_exported': 0,
                'relationships_exported': 0
            }

            if documents_for_graph:
                doc_count = graph_storage.append_documents(documents_for_graph, run_id)
                export_results['documents_exported'] = doc_count
                logger.info(f"Exported {doc_count} documents to graph database")

            if elements_for_graph:
                elem_count = graph_storage.append_elements(elements_for_graph, run_id)
                export_results['entities_exported'] = elem_count
                logger.info(f"Exported {elem_count} entities to graph database")

            if relationships_for_graph:
                rel_count = graph_storage.append_relationships(relationships_for_graph, run_id)
                export_results['relationships_exported'] = rel_count
                logger.info(f"Exported {rel_count} relationships to graph database")

            return {
                'success': True,
                'export_run_id': run_id,
                'results': export_results
            }

        except Exception as e:
            logger.error(f"Graph export failed: {e}")
            return {
                'success': False,
                'error': str(e),
                'export_run_id': extraction_results.get('run_id', 'unknown')
            }

    def _prepare_documents_for_graph(self, entities: List[Dict]) -> List[Dict]:
        """Prepare unique documents for graph export."""
        doc_map = {}

        for entity in entities:
            doc_id = entity.get('doc_id')
            if doc_id and doc_id not in doc_map:
                doc_map[doc_id] = {
                    'doc_id': doc_id,
                    'source_name': entity.get('source_name', ''),
                    'title': f"Document {doc_id}",
                    'content_type': 'extracted_entities',
                    'processed_at': entity.get('extracted_at', datetime.now().isoformat()),
                    'entity_count': 0  # Will be updated below
                }

        # Count entities per document
        for entity in entities:
            doc_id = entity.get('doc_id')
            if doc_id in doc_map:
                doc_map[doc_id]['entity_count'] += 1

        return list(doc_map.values())

    def _prepare_elements_for_graph(self, entities: List[Dict]) -> List[Dict]:
        """Prepare entities as elements for graph export."""
        elements = []

        for entity in entities:
            element = {
                'element_id': entity['entity_id'],
                'doc_id': entity['doc_id'],
                'source_name': entity['source_name'],
                'element_type': f"ENTITY_{entity['entity_type']}",
                'content_preview': entity['content'],
                'position': 0,  # Entities don't have positions
                'parent_id': None,
                'metadata': {
                    'entity_type': entity['entity_type'],
                    'term_id': entity['term_id'],
                    'confidence': entity['confidence'],
                    'source_element_id': entity['source_element_id'],
                    'extraction_metadata': entity['metadata'],
                    'extracted_at': entity['extracted_at']
                }
            }
            elements.append(element)

        return elements

    def _prepare_relationships_for_graph(self, relationships: List[Dict]) -> List[Dict]:
        """Prepare relationships for graph export."""
        graph_relationships = []

        for rel in relationships:
            graph_rel = {
                'source_id': rel['source_entity_id'],
                'target_id': rel['target_entity_id'],
                'relationship_type': rel['relationship_type'],
                'doc_id': rel['doc_id'],
                'source_name': rel['source_name'],
                'metadata': {
                    'confidence': rel['confidence'],
                    'relationship_id': rel['relationship_id'],
                    'extraction_metadata': rel['metadata'],
                    'extracted_at': rel['extracted_at']
                }
            }
            graph_relationships.append(graph_rel)

        return graph_relationships

    def run_full_pipeline(self, max_elements: int = 10000, export_to_graph: bool = False,
                         neo4j_config: Optional[Dict] = None) -> Dict[str, Any]:
        """
        Run the complete extraction and export pipeline.

        Args:
            max_elements: Maximum elements to process
            export_to_graph: Whether to export to graph database
            neo4j_config: Optional Neo4j configuration

        Returns:
            Complete pipeline results
        """
        run_id = f"pipeline_{datetime.now().strftime('%Y%m%d_%H%M%S')}"

        # Step 1: Extract entities and relationships
        extraction_results = self.extract_entities_and_relationships(max_elements, run_id)

        pipeline_results = {
            'run_id': run_id,
            'extraction_results': extraction_results,
            'graph_export_results': None
        }

        # Step 2: Export to graph database if requested
        if export_to_graph:
            graph_results = self.export_to_graph_database(extraction_results, neo4j_config)
            pipeline_results['graph_export_results'] = graph_results

        return pipeline_results


@click.command()
@click.option("--config", "-c", required=True, help="Path to Go-Doc-Go configuration file")
@click.option("--ontology", "-o", required=True, help="Path to ontology YAML file")
@click.option("--max-elements", type=int, default=10000, help="Maximum elements to process")
@click.option("--export-graph", is_flag=True, help="Export results to Neo4j graph database")
@click.option("--neo4j-uri", help="Neo4j URI (overrides config)")
@click.option("--neo4j-user", help="Neo4j username (overrides config)")
@click.option("--neo4j-password", help="Neo4j password (overrides config)")
@click.option("--neo4j-database", help="Neo4j database name (overrides config)")
@click.option("--output", help="Save extraction results to JSON file")
@click.option("--run-id", help="Custom run identifier")
@click.option("--log-level", type=click.Choice(["DEBUG", "INFO", "WARNING", "ERROR"]),
              default="INFO", help="Logging level")
def main(config, ontology, max_elements, export_graph, neo4j_uri, neo4j_user,
         neo4j_password, neo4j_database, output, run_id, log_level):
    """Extract entities and relationships using ontology rules.

    This command applies ontology rules to your processed document corpus
    to extract domain-specific entities and relationships. Results can be
    exported to Neo4j for graph analysis and visualization.

    Examples:
      # Extract entities using ontology
      python -m go_doc_go.cli.ontology_extract \\
        --config config.yaml \\
        --ontology financial_ontology.yaml \\
        --max-elements 5000

      # Extract and export to Neo4j
      python -m go_doc_go.cli.ontology_extract \\
        --config config.yaml \\
        --ontology legal_ontology.yaml \\
        --export-graph \\
        --neo4j-uri bolt://localhost:7687

      # Save results to file
      python -m go_doc_go.cli.ontology_extract \\
        --config config.yaml \\
        --ontology domain_ontology.yaml \\
        --output extraction_results.json
    """

    # Configure logging
    logging.basicConfig(
        level=getattr(logging, log_level),
        format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
    )

    try:
        # Validate inputs
        if not os.path.exists(config):
            click.echo(f"❌ Configuration file not found: {config}")
            sys.exit(1)

        if not os.path.exists(ontology):
            click.echo(f"❌ Ontology file not found: {ontology}")
            sys.exit(1)

        # Load configuration
        config_obj = Config(config)

        # Check analytics is enabled
        if not config_obj.is_analytics_enabled():
            click.echo("❌ Analytics must be enabled for entity extraction")
            sys.exit(1)

        # Create Neo4j configuration if provided
        neo4j_config = None
        if any([neo4j_uri, neo4j_user, neo4j_password, neo4j_database]):
            neo4j_config = {}
            if neo4j_uri:
                neo4j_config['uri'] = neo4j_uri
            if neo4j_user:
                neo4j_config['username'] = neo4j_user
            if neo4j_password:
                neo4j_config['password'] = neo4j_password
            if neo4j_database:
                neo4j_config['database'] = neo4j_database

        # Initialize pipeline
        click.echo(f"🚀 Initializing ontology extraction pipeline...")
        pipeline = OntologyExtractionPipeline(config_obj, ontology)

        # Run pipeline
        click.echo(f"⚙️  Processing up to {max_elements} elements...")
        if export_graph:
            click.echo("📊 Graph export enabled")

        results = pipeline.run_full_pipeline(
            max_elements=max_elements,
            export_to_graph=export_graph,
            neo4j_config=neo4j_config
        )

        # Display results
        extraction = results['extraction_results']
        summary = extraction.get('extraction_summary', {})

        click.echo(f"\n✅ Extraction completed!")
        click.echo(f"📊 Results:")
        click.echo(f"  - Elements processed: {summary.get('total_elements_processed', 0)}")
        click.echo(f"  - Entities extracted: {summary.get('total_entities_extracted', 0)}")
        click.echo(f"  - Relationships extracted: {summary.get('total_relationships_extracted', 0)}")
        click.echo(f"  - Documents processed: {summary.get('documents_processed', 0)}")

        # Show graph export results
        if export_graph and results.get('graph_export_results'):
            graph_results = results['graph_export_results']
            if graph_results.get('success'):
                export_data = graph_results.get('results', {})
                click.echo(f"\n🌐 Graph Export Results:")
                click.echo(f"  - Documents exported: {export_data.get('documents_exported', 0)}")
                click.echo(f"  - Entities exported: {export_data.get('entities_exported', 0)}")
                click.echo(f"  - Relationships exported: {export_data.get('relationships_exported', 0)}")
            else:
                click.echo(f"\n❌ Graph export failed: {graph_results.get('error')}")

        # Save to file if requested
        if output:
            output_path = Path(output)
            with open(output_path, 'w') as f:
                json.dump(results, f, indent=2, default=str)
            click.echo(f"\n💾 Results saved to: {output_path}")

        click.echo(f"\n🎉 Pipeline completed successfully!")

    except KeyboardInterrupt:
        click.echo("\n\n👋 Extraction cancelled. Goodbye!")
        sys.exit(1)
    except Exception as e:
        logger.error(f"Extraction pipeline failed: {e}")
        click.echo(f"\n❌ Error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()