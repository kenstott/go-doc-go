#!/usr/bin/env python3
"""
Extract entities using the semantic-enhanced insider trading ontology
and save results for Neo4j export.
"""

import json
import os
import sys
from typing import Dict, List, Any, Optional
from datetime import datetime
import uuid

# Add src to path for Go-Doc-Go imports
sys.path.insert(0, 'src')

try:
    from go_doc_go.domain_ontology.domain_ontology import DomainOntology
    from go_doc_go.domain_ontology.entity_extractor import EntityExtractor
    from go_doc_go.storage_adapters.analytics_parquet import ParquetAnalyticsStorage
except ImportError as e:
    print(f"❌ Error importing Go-Doc-Go modules: {e}")
    print("Make sure PYTHONPATH includes src directory")
    sys.exit(1)

import duckdb
import pandas as pd


class SemanticEntityExtractor:
    """Extract entities using semantic-enhanced ontology."""

    def __init__(self, ontology_path: str, analytics_path: str):
        self.ontology_path = ontology_path
        self.analytics_path = analytics_path
        self.extraction_results = {
            "entities": [],
            "relationships": [],
            "metadata": {}
        }

    def load_ontology(self) -> DomainOntology:
        """Load the semantic-enhanced ontology."""
        print(f"📚 Loading semantic ontology: {self.ontology_path}")

        try:
            with open(self.ontology_path, 'r') as f:
                ontology_data = f.read()

            # Parse YAML manually to handle the structure
            import yaml
            ontology_dict = yaml.safe_load(ontology_data)

            # Load using DomainOntology
            ontology = DomainOntology.from_dict(ontology_dict)
            print(f"✅ Loaded ontology: {ontology.name}")
            print(f"   Terms: {len(ontology.terms)}")
            print(f"   Element mappings: {len(ontology.element_mappings)}")
            return ontology

        except Exception as e:
            print(f"❌ Failed to load ontology: {e}")
            raise

    def load_analytics_data(self) -> pd.DataFrame:
        """Load analytics data from parquet files."""
        print(f"📊 Loading analytics data from: {self.analytics_path}")

        conn = duckdb.connect(':memory:')

        # Load elements data
        elements_query = f"""
        SELECT
            doc_id,
            element_id,
            element_type,
            content_preview as content,
            parent_id,
            source_name,
            extracted_at
        FROM read_parquet('{self.analytics_path}/elements/**/*.parquet')
        LIMIT 5000
        """

        elements_df = conn.execute(elements_query).df()
        print(f"✅ Loaded {len(elements_df)} elements for extraction")

        return elements_df

    def extract_entities(self, ontology: DomainOntology, elements_df: pd.DataFrame) -> Dict[str, List]:
        """Extract entities using the semantic ontology."""
        print("🎯 Extracting entities with semantic matching...")

        # Initialize entity extractor
        extractor = EntityExtractor(ontology)

        entities = []
        relationships = []
        processed_docs = set()

        # Group elements by document
        for doc_id, doc_elements in elements_df.groupby('doc_id'):
            if doc_id in processed_docs:
                continue

            processed_docs.add(doc_id)

            # Convert to element format expected by extractor
            elements_for_extraction = []
            for _, element in doc_elements.iterrows():
                element_dict = {
                    'element_id': element['element_id'],
                    'element_type': element['element_type'],
                    'content_preview': element['content'],
                    'parent_id': element.get('parent_id'),
                    'source_name': element['source_name'],
                    'extracted_at': element['extracted_at']
                }
                elements_for_extraction.append(element_dict)

            # Extract entities for this document
            try:
                doc_extraction = extractor.extract_entities(
                    doc_id=doc_id,
                    elements=elements_for_extraction,
                    source_name=doc_elements.iloc[0]['source_name']
                )

                # Add entities
                for entity in doc_extraction.get('entities', []):
                    entity['entity_id'] = f"ent_{uuid.uuid4().hex[:8]}"
                    entity['doc_id'] = doc_id
                    entity['extracted_at'] = datetime.now().isoformat()
                    entities.append(entity)

                # Add relationships
                for relationship in doc_extraction.get('relationships', []):
                    relationship['relationship_id'] = f"rel_{uuid.uuid4().hex[:8]}"
                    relationship['doc_id'] = doc_id
                    relationship['extracted_at'] = datetime.now().isoformat()
                    relationships.append(relationship)

            except Exception as e:
                print(f"⚠️  Error extracting from document {doc_id}: {e}")
                continue

        print(f"✅ Extracted {len(entities)} entities and {len(relationships)} relationships")
        return {'entities': entities, 'relationships': relationships}

    def save_results(self, extraction_results: Dict[str, List], output_file: str):
        """Save extraction results to JSON file."""
        print(f"💾 Saving results to: {output_file}")

        # Add metadata
        results_with_metadata = {
            'extraction_timestamp': datetime.now().isoformat(),
            'ontology_path': self.ontology_path,
            'analytics_path': self.analytics_path,
            'semantic_enhanced': True,
            'extraction_results': extraction_results,
            'summary': {
                'total_entities': len(extraction_results['entities']),
                'total_relationships': len(extraction_results['relationships']),
                'entity_types': {},
                'relationship_types': {}
            }
        }

        # Count entity types
        for entity in extraction_results['entities']:
            entity_type = entity.get('entity_type', 'unknown')
            results_with_metadata['summary']['entity_types'][entity_type] = \
                results_with_metadata['summary']['entity_types'].get(entity_type, 0) + 1

        # Count relationship types
        for relationship in extraction_results['relationships']:
            rel_type = relationship.get('relationship_type', 'unknown')
            results_with_metadata['summary']['relationship_types'][rel_type] = \
                results_with_metadata['summary']['relationship_types'].get(rel_type, 0) + 1

        with open(output_file, 'w') as f:
            json.dump(results_with_metadata, f, indent=2)

        print(f"✅ Results saved successfully")
        return results_with_metadata


def main():
    """Main extraction function."""
    print("🚀 Semantic-Enhanced Entity Extraction")
    print("=" * 60)

    # Configuration - can be overridden via command line args
    import argparse
    parser = argparse.ArgumentParser(description="Extract entities using semantic-enhanced ontology")
    parser.add_argument('--ontology', default="semantic_enhanced_insider_trading_ontology.yaml",
                       help="Path to ontology YAML file")
    parser.add_argument('--analytics', default="/Volumes/T9/sec_analytics",
                       help="Path to analytics parquet files")
    parser.add_argument('--output', default="extraction_results.json",
                       help="Output JSON file for results")
    args = parser.parse_args()

    ONTOLOGY_PATH = args.ontology
    ANALYTICS_PATH = args.analytics
    OUTPUT_FILE = args.output

    # Check if analytics data exists
    if not os.path.exists(ANALYTICS_PATH):
        print(f"❌ Analytics data not found at: {ANALYTICS_PATH}")
        print("Run the document processing worker first:")
        print("PYTHONPATH=src python -m go_doc_go worker --config configs/sec-html-store.yaml --max-documents 10")
        sys.exit(1)

    # Check for elements directory
    elements_dir = os.path.join(ANALYTICS_PATH, "elements")
    if not os.path.exists(elements_dir):
        # Try the semantic path as fallback
        alt_path = "/Volumes/T9/sec_semantic_insider_analytics"
        if os.path.exists(os.path.join(alt_path, "elements")):
            print(f"📂 Using alternative analytics path: {alt_path}")
            ANALYTICS_PATH = alt_path
        else:
            print(f"❌ No elements found in {ANALYTICS_PATH}/elements")
            sys.exit(1)

    # Initialize extractor
    extractor = SemanticEntityExtractor(ONTOLOGY_PATH, ANALYTICS_PATH)

    try:
        # Load ontology
        ontology = extractor.load_ontology()

        # Load analytics data
        elements_df = extractor.load_analytics_data()

        # Extract entities
        extraction_results = extractor.extract_entities(ontology, elements_df)

        # Save results
        final_results = extractor.save_results(extraction_results, OUTPUT_FILE)

        # Display summary
        print("\n📊 EXTRACTION SUMMARY")
        print("-" * 40)
        print(f"Entities extracted: {final_results['summary']['total_entities']}")
        print(f"Relationships extracted: {final_results['summary']['total_relationships']}")

        print("\nEntity types:")
        for entity_type, count in final_results['summary']['entity_types'].items():
            print(f"  {entity_type}: {count}")

        print("\nRelationship types:")
        for rel_type, count in final_results['summary']['relationship_types'].items():
            print(f"  {rel_type}: {count}")

        print(f"\n🎯 SEMANTIC ENHANCEMENT FEATURES:")
        print("  • Semantic similarity rules for natural language variations")
        print("  • Executive title matching (CEO ≈ Chief Executive Officer ≈ President)")
        print("  • Company name variations (MSFT ≈ Microsoft ≈ Microsoft Corporation)")
        print("  • Transaction synonyms (buy ≈ purchase ≈ acquire ≈ disposition)")

        print(f"\n✅ Semantic extraction completed!")
        print(f"📁 Results saved to: {OUTPUT_FILE}")
        print(f"🔗 Ready for Neo4j export!")

    except Exception as e:
        print(f"❌ Extraction failed: {e}")
        raise


if __name__ == "__main__":
    main()