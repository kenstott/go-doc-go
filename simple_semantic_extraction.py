#!/usr/bin/env python3
"""
Simple semantic entity extraction from processed analytics data.
This bypasses the complex ontology infrastructure and creates entities directly.
"""

import json
import os
import uuid
from datetime import datetime
from typing import Dict, List, Any
import duckdb
import pandas as pd
import re


class SimpleSemanticExtractor:
    """Simple semantic entity extraction using pattern matching."""

    def __init__(self, analytics_path: str):
        self.analytics_path = analytics_path
        self.semantic_rules = self._define_semantic_rules()

    def _define_semantic_rules(self) -> Dict[str, Dict]:
        """Define semantic extraction rules based on our ontology."""
        return {
            'insider': {
                'semantic_phrases': ['executive', 'director', 'officer', 'CEO', 'president', 'chief executive officer'],
                'xml_patterns': ['<reportingOwner>', '<ownerSignature>', '<reportingOwnerId>'],
                'confidence': 0.75
            },
            'transaction': {
                'semantic_phrases': ['buy', 'purchase', 'acquire', 'acquisition', 'disposition', 'sold', 'sale', 'trading'],
                'xml_patterns': ['<transactionAcquiredDisposedCode>', '<transactionDate>', '<transactionPricePerShare>'],
                'confidence': 0.75
            },
            'ownership': {
                'semantic_phrases': ['shares', 'ownership', 'stake', 'holdings'],
                'xml_patterns': ['<ownershipNature>', '<directOrIndirectOwnership>', '<sharesOwnedFollowingTransaction>'],
                'confidence': 0.75
            },
            'company': {
                'semantic_phrases': ['company', 'corporation', 'issuer', 'enterprise', 'Microsoft', 'MSFT', 'Microsoft Corporation'],
                'xml_patterns': ['<securityTitle>', '<issuerName>', '<issuerTradingSymbol>'],
                'confidence': 0.75
            },
            'filing': {
                'semantic_phrases': ['Form 4', '10-K', '10-Q', '8-K', 'filing', 'report', 'disclosure'],
                'xml_patterns': ['<footnoteId>', '<transactionFormType>'],
                'confidence': 0.75
            },
            'financial': {
                'semantic_phrases': ['revenue', 'sales', 'income', 'earnings', 'profit', 'financial', 'performance'],
                'xml_patterns': [],
                'confidence': 0.75
            }
        }

    def load_analytics_data(self) -> pd.DataFrame:
        """Load elements from analytics parquet files."""
        print(f"📊 Loading analytics data from: {self.analytics_path}")

        conn = duckdb.connect(':memory:')

        # List parquet files excluding macOS hidden files
        parquet_files = []
        import glob
        for file_path in glob.glob(f"{self.analytics_path}/elements/**/*.parquet", recursive=True):
            if not os.path.basename(file_path).startswith('._'):
                parquet_files.append(file_path)

        if not parquet_files:
            print("❌ No valid parquet files found")
            return pd.DataFrame()

        # Create query with explicit file list
        file_list = "', '".join(parquet_files)
        elements_query = f"""
        SELECT
            doc_id,
            element_id,
            element_type,
            content_preview as content,
            parent_id,
            source_name,
            _written_at as extracted_at
        FROM read_parquet(['{file_list}'])
        WHERE content_preview IS NOT NULL
        AND content_preview != ''
        LIMIT 5000
        """

        elements_df = conn.execute(elements_query).df()
        print(f"✅ Loaded {len(elements_df)} elements for semantic extraction")
        return elements_df

    def extract_entities(self, elements_df: pd.DataFrame) -> List[Dict]:
        """Extract entities using semantic pattern matching."""
        print("🎯 Extracting entities with semantic pattern matching...")

        entities = []

        for _, element in elements_df.iterrows():
            content = element['content'].lower() if element['content'] else ""

            # Apply each semantic rule
            for entity_type, rules in self.semantic_rules.items():
                confidence = 0.0
                extraction_method = None

                # Check semantic phrases
                for phrase in rules['semantic_phrases']:
                    if phrase.lower() in content:
                        confidence = max(confidence, rules['confidence'])
                        extraction_method = f"semantic_phrase:{phrase}"
                        break

                # Check XML patterns for xml_element types
                if element['element_type'] == 'xml_element':
                    for pattern in rules['xml_patterns']:
                        if pattern.lower() in content:
                            confidence = max(confidence, 0.9)  # Higher confidence for XML matches
                            extraction_method = f"xml_pattern:{pattern}"
                            break

                # Create entity if confidence threshold met
                if confidence >= 0.7:
                    entity = {
                        'entity_id': f"ent_{uuid.uuid4().hex[:8]}",
                        'entity_type': entity_type,
                        'term_id': entity_type,
                        'content': element['content'][:200],  # Limit content length
                        'confidence': confidence,
                        'source_element_id': element['element_id'],
                        'doc_id': element['doc_id'],
                        'source_name': element['source_name'],
                        'extracted_at': datetime.now().isoformat(),
                        'element_type': element['element_type'],
                        'metadata': {
                            'extraction_method': extraction_method,
                            'semantic_enhanced': True
                        }
                    }
                    entities.append(entity)

        print(f"✅ Extracted {len(entities)} semantic entities")
        return entities

    def create_relationships(self, entities: List[Dict]) -> List[Dict]:
        """Create relationships between entities in the same document."""
        print("🔗 Creating semantic relationships...")

        relationships = []

        # Group entities by document
        entities_by_doc = {}
        for entity in entities:
            doc_id = entity['doc_id']
            if doc_id not in entities_by_doc:
                entities_by_doc[doc_id] = []
            entities_by_doc[doc_id].append(entity)

        # Create relationships within documents
        for doc_id, doc_entities in entities_by_doc.items():
            # Insider executes transaction
            insiders = [e for e in doc_entities if e['entity_type'] == 'insider']
            transactions = [e for e in doc_entities if e['entity_type'] == 'transaction']

            for insider in insiders:
                for transaction in transactions:
                    relationship = {
                        'relationship_id': f"rel_{uuid.uuid4().hex[:8]}",
                        'relationship_type': 'insider_executes_transaction',
                        'source_entity_id': insider['entity_id'],
                        'target_entity_id': transaction['entity_id'],
                        'confidence': 0.8,
                        'doc_id': doc_id,
                        'source_name': insider['source_name'],
                        'extracted_at': datetime.now().isoformat(),
                        'metadata': {
                            'relationship_type': 'action',
                            'extraction_method': 'semantic_co_occurrence'
                        }
                    }
                    relationships.append(relationship)

            # Insider works for company
            companies = [e for e in doc_entities if e['entity_type'] == 'company']

            for insider in insiders:
                for company in companies:
                    relationship = {
                        'relationship_id': f"rel_{uuid.uuid4().hex[:8]}",
                        'relationship_type': 'insider_works_for_company',
                        'source_entity_id': insider['entity_id'],
                        'target_entity_id': company['entity_id'],
                        'confidence': 0.8,
                        'doc_id': doc_id,
                        'source_name': insider['source_name'],
                        'extracted_at': datetime.now().isoformat(),
                        'metadata': {
                            'relationship_type': 'corporate_structure',
                            'extraction_method': 'semantic_co_occurrence'
                        }
                    }
                    relationships.append(relationship)

            # Transaction involves company
            for transaction in transactions:
                for company in companies:
                    relationship = {
                        'relationship_id': f"rel_{uuid.uuid4().hex[:8]}",
                        'relationship_type': 'transaction_involves_company',
                        'source_entity_id': transaction['entity_id'],
                        'target_entity_id': company['entity_id'],
                        'confidence': 0.8,
                        'doc_id': doc_id,
                        'source_name': transaction['source_name'],
                        'extracted_at': datetime.now().isoformat(),
                        'metadata': {
                            'relationship_type': 'transaction',
                            'extraction_method': 'semantic_co_occurrence'
                        }
                    }
                    relationships.append(relationship)

        print(f"✅ Created {len(relationships)} semantic relationships")
        return relationships

    def save_results(self, entities: List[Dict], relationships: List[Dict], output_file: str):
        """Save extraction results to JSON file."""
        print(f"💾 Saving semantic extraction results to: {output_file}")

        # Create results structure
        results = {
            'extraction_timestamp': datetime.now().isoformat(),
            'extraction_method': 'simple_semantic_extraction',
            'semantic_enhanced': True,
            'analytics_source': self.analytics_path,
            'extraction_results': {
                'entities': entities,
                'relationships': relationships
            },
            'summary': {
                'total_entities': len(entities),
                'total_relationships': len(relationships),
                'entity_types': {},
                'relationship_types': {}
            }
        }

        # Count entity types
        for entity in entities:
            entity_type = entity['entity_type']
            results['summary']['entity_types'][entity_type] = \
                results['summary']['entity_types'].get(entity_type, 0) + 1

        # Count relationship types
        for relationship in relationships:
            rel_type = relationship['relationship_type']
            results['summary']['relationship_types'][rel_type] = \
                results['summary']['relationship_types'].get(rel_type, 0) + 1

        with open(output_file, 'w') as f:
            json.dump(results, f, indent=2)

        print(f"✅ Results saved successfully")
        return results


def main():
    """Main extraction function."""
    print("🚀 Simple Semantic Entity Extraction")
    print("=" * 60)

    # Configuration
    ANALYTICS_PATH = "/Volumes/T9/sec_semantic_insider_analytics"
    OUTPUT_FILE = "simple_semantic_extraction_results.json"

    # Check if analytics data exists
    if not os.path.exists(ANALYTICS_PATH):
        print(f"❌ Analytics data not found at: {ANALYTICS_PATH}")
        print("Run the semantic extraction worker first")
        return

    # Initialize extractor
    extractor = SimpleSemanticExtractor(ANALYTICS_PATH)

    try:
        # Load analytics data
        elements_df = extractor.load_analytics_data()

        # Extract entities
        entities = extractor.extract_entities(elements_df)

        # Create relationships
        relationships = extractor.create_relationships(entities)

        # Save results
        results = extractor.save_results(entities, relationships, OUTPUT_FILE)

        # Display summary
        print("\n📊 SEMANTIC EXTRACTION SUMMARY")
        print("-" * 50)
        print(f"Total entities: {results['summary']['total_entities']}")
        print(f"Total relationships: {results['summary']['total_relationships']}")

        print("\nEntity types:")
        for entity_type, count in results['summary']['entity_types'].items():
            print(f"  {entity_type}: {count}")

        print("\nRelationship types:")
        for rel_type, count in results['summary']['relationship_types'].items():
            print(f"  {rel_type}: {count}")

        print(f"\n🧠 SEMANTIC ENHANCEMENTS DEMONSTRATED:")
        print("  • Natural language phrase matching (CEO, executive, director)")
        print("  • Company name variations (Microsoft, MSFT, Microsoft Corporation)")
        print("  • Transaction synonyms (buy, purchase, acquire, disposition)")
        print("  • Combined semantic + XML pattern recognition")

        print(f"\n✅ Simple semantic extraction completed!")
        print(f"📁 Results saved to: {OUTPUT_FILE}")
        print(f"🔗 Ready for Neo4j export!")

    except Exception as e:
        print(f"❌ Extraction failed: {e}")
        raise


if __name__ == "__main__":
    main()