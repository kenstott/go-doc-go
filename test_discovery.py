#!/usr/bin/env python3
"""
Test the ontology discovery system with a greenfield domain.
This simulates the discovery process without requiring an LLM API.
"""

import json
import logging
from pathlib import Path
import duckdb
import pandas as pd
from typing import Dict, Any, List

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class MockDiscoveryOrchestrator:
    """Mock orchestrator that simulates LLM responses based on actual data analysis."""

    def __init__(self, analytics_path: str):
        self.analytics_path = analytics_path
        self.context = {
            'discovered_patterns': {},
            'discovered_terms': [],
            'discovered_entities': [],
            'discovered_relationships': []
        }

    def discover_ontology(self, domain_name: str, candidate_terms: List[str]):
        """Execute discovery pipeline with mock LLM responses."""

        print(f"\n🚀 Starting Ontology Discovery for Domain: {domain_name}")
        print(f"📝 Candidate Terms from Human: {', '.join(candidate_terms)}")
        print("=" * 60)

        # Step 1: Profile Corpus
        print("\n📊 Step 1: Profiling Corpus...")
        corpus_profile = self._profile_corpus()
        self.context['discovered_patterns']['corpus_profile'] = corpus_profile

        # Step 2: Discover Patterns for Candidate Terms
        print("\n🔍 Step 2: Discovering Patterns for Candidate Terms...")
        patterns = self._discover_patterns_for_terms(candidate_terms)
        self.context['discovered_patterns']['term_patterns'] = patterns

        # Step 3: Extract and Define Terms
        print("\n📚 Step 3: Extracting and Defining Terms...")
        terms = self._extract_terms(candidate_terms, patterns)
        self.context['discovered_terms'] = terms

        # Step 4: Create Extraction Rules
        print("\n🎯 Step 4: Creating Extraction Rules...")
        extraction_rules = self._create_extraction_rules(terms, patterns)
        self.context['discovered_entities'] = extraction_rules

        # Step 5: Discover Relationships
        print("\n🔗 Step 5: Discovering Relationships...")
        relationships = self._discover_relationships(terms)
        self.context['discovered_relationships'] = relationships

        # Step 6: Build Final Ontology
        print("\n🏗️ Step 6: Building Final Ontology Configuration...")
        ontology = self._build_ontology(domain_name)

        return ontology

    def _profile_corpus(self) -> Dict[str, Any]:
        """Profile the document corpus."""
        conn = duckdb.connect(':memory:')

        query = f"""
            SELECT
                COUNT(DISTINCT doc_id) as total_documents,
                COUNT(*) as total_elements,
                element_type,
                COUNT(*) as element_count
            FROM read_parquet('{self.analytics_path}/elements/**/*.parquet')
            GROUP BY element_type
            ORDER BY element_count DESC
        """

        result = conn.execute(query).df()
        conn.close()

        profile = {
            'total_documents': int(result['total_documents'].iloc[0]),
            'total_elements': int(result['total_elements'].iloc[0]),
            'element_distribution': result[['element_type', 'element_count']].to_dict('records'),
            'domain_assessment': 'regulatory_filings',
            'primary_structure': 'mixed_html_xml'
        }

        print(f"  ✓ Found {profile['total_documents']} documents with {profile['total_elements']} elements")
        print(f"  ✓ Top element types: {', '.join([d['element_type'] for d in profile['element_distribution'][:3]])}")

        return profile

    def _discover_patterns_for_terms(self, candidate_terms: List[str]) -> Dict[str, List[Dict]]:
        """Discover patterns for each candidate term."""
        conn = duckdb.connect(':memory:')
        patterns = {}

        for term in candidate_terms:
            print(f"  🔎 Analyzing '{term}'...")

            # Search for this term in the data
            query = f"""
                SELECT
                    content_preview,
                    element_type,
                    COUNT(*) as frequency
                FROM read_parquet('{self.analytics_path}/elements/**/*.parquet')
                WHERE content_preview ILIKE '%{term}%'
                GROUP BY content_preview, element_type
                ORDER BY frequency DESC
                LIMIT 10
            """

            result = conn.execute(query).df()

            if not result.empty:
                patterns[term] = result.to_dict('records')
                print(f"    ✓ Found {len(result)} patterns with {result['frequency'].sum()} total occurrences")
            else:
                # Try variations
                alt_query = f"""
                    SELECT
                        content_preview,
                        element_type,
                        COUNT(*) as frequency
                    FROM read_parquet('{self.analytics_path}/elements/**/*.parquet')
                    WHERE content_preview ~ '(?i){term.replace(" ", ".*")}'
                    GROUP BY content_preview, element_type
                    ORDER BY frequency DESC
                    LIMIT 10
                """

                result = conn.execute(alt_query).df()
                patterns[term] = result.to_dict('records') if not result.empty else []

                if not result.empty:
                    print(f"    ✓ Found {len(result)} related patterns")
                else:
                    print(f"    ⚠ No direct matches, will infer from context")

        conn.close()
        return patterns

    def _extract_terms(self, candidate_terms: List[str], patterns: Dict) -> List[Dict]:
        """Extract and define formal terms."""
        terms = []

        # Define terms based on patterns found
        term_definitions = {
            'company': {
                'id': 'company_name',
                'label': 'Company Name',
                'description': 'Name of a corporation or business entity',
                'aliases': ['corporation', 'company', 'business', 'issuer', 'registrant']
            },
            'ticker': {
                'id': 'stock_symbol',
                'label': 'Stock Symbol',
                'description': 'Trading symbol for publicly traded securities',
                'aliases': ['symbol', 'ticker', 'trading symbol', 'stock ticker']
            },
            'revenue': {
                'id': 'revenue',
                'label': 'Revenue',
                'description': 'Total income from business operations',
                'aliases': ['sales', 'total revenue', 'gross revenue', 'turnover']
            },
            'executive': {
                'id': 'executive_officer',
                'label': 'Executive Officer',
                'description': 'Senior management personnel of a company',
                'aliases': ['officer', 'executive', 'CEO', 'CFO', 'president']
            },
            'filing': {
                'id': 'regulatory_filing',
                'label': 'Regulatory Filing',
                'description': 'Official document submitted to regulatory authorities',
                'aliases': ['form', 'report', 'filing', 'submission']
            },
            'date': {
                'id': 'filing_date',
                'label': 'Filing Date',
                'description': 'Date when document was filed or reported',
                'aliases': ['date', 'reported date', 'filed date']
            }
        }

        for term in candidate_terms:
            # Find best matching definition
            term_key = term.lower().replace('_', ' ').split()[0]

            if term_key in term_definitions:
                terms.append(term_definitions[term_key])
            else:
                # Create generic definition
                terms.append({
                    'id': term.lower().replace(' ', '_'),
                    'label': term.title(),
                    'description': f'Domain-specific term: {term}',
                    'aliases': [term.lower(), term.title()]
                })

        print(f"  ✓ Defined {len(terms)} formal terms")
        return terms

    def _create_extraction_rules(self, terms: List[Dict], patterns: Dict) -> List[Dict]:
        """Create extraction rules for each term."""
        extraction_rules = []

        for term in terms:
            term_id = term['id']
            rules = []

            # Analyze patterns to create rules
            if term_id == 'company_name':
                rules = [
                    {
                        'type': 'regex',
                        'pattern': r'\b(MICROSOFT CORP|Apple Inc\.|[A-Z][a-z]+\s+(Corp|Inc|LLC))\b',
                        'case_sensitive': False,
                        'confidence': 0.85
                    },
                    {
                        'type': 'regex',
                        'pattern': r'<issuerName>\s*(.+)',
                        'element_types': ['xml_element'],
                        'confidence': 0.95
                    }
                ]
            elif term_id == 'stock_symbol':
                rules = [
                    {
                        'type': 'regex',
                        'pattern': r'\b[A-Z]{1,5}\b',
                        'case_sensitive': True,
                        'confidence': 0.75
                    },
                    {
                        'type': 'regex',
                        'pattern': r'<issuerTradingSymbol>\s*([A-Z]+)',
                        'element_types': ['xml_element'],
                        'confidence': 0.95
                    }
                ]
            elif term_id == 'revenue':
                rules = [
                    {
                        'type': 'regex',
                        'pattern': r'\b(Revenue|Total revenue|Net revenue|Sales)\b',
                        'case_sensitive': False,
                        'confidence': 0.85
                    },
                    {
                        'type': 'semantic',
                        'semantic_phrase': 'revenue, sales, income from operations',
                        'confidence': 0.80
                    }
                ]
            elif term_id == 'executive_officer':
                rules = [
                    {
                        'type': 'regex',
                        'pattern': r'\b(Chief\s+\w+\s+Officer|President|CEO|CFO|CTO)\b',
                        'case_sensitive': False,
                        'confidence': 0.90
                    }
                ]
            elif term_id == 'filing_date':
                rules = [
                    {
                        'type': 'regex',
                        'pattern': r'\b\d{1,2}/\d{1,2}/\d{4}\b|\b\d{4}-\d{2}-\d{2}\b',
                        'confidence': 0.85
                    }
                ]
            else:
                # Generic rule
                rules = [
                    {
                        'type': 'keywords',
                        'keywords': term['aliases'],
                        'confidence': 0.70
                    }
                ]

            extraction_rules.append({
                'term_id': term_id,
                'rules': rules
            })

        print(f"  ✓ Created extraction rules for {len(extraction_rules)} terms")
        return extraction_rules

    def _discover_relationships(self, terms: List[Dict]) -> List[Dict]:
        """Discover relationships between terms."""
        conn = duckdb.connect(':memory:')
        relationships = []

        # Check for company-symbol relationships
        query = f"""
            WITH companies AS (
                SELECT DISTINCT doc_id, content_preview
                FROM read_parquet('{self.analytics_path}/elements/**/*.parquet')
                WHERE content_preview ~ 'MICROSOFT|Apple|CORP|INC'
                LIMIT 50
            ),
            symbols AS (
                SELECT DISTINCT doc_id, content_preview
                FROM read_parquet('{self.analytics_path}/elements/**/*.parquet')
                WHERE content_preview ~ '^[A-Z]{{1,5}}$'
                LIMIT 50
            )
            SELECT COUNT(*) as cooccurrence
            FROM companies c
            JOIN symbols s ON c.doc_id = s.doc_id
        """

        result = conn.execute(query).fetchone()
        if result and result[0] > 0:
            relationships.append({
                'id': 'company_has_symbol',
                'description': 'Company has a trading symbol',
                'relationship_type': 'HAS_SYMBOL',
                'source': {
                    'term_id': 'company_name',
                    'semantic_phrase': 'company, corporation',
                    'confidence_threshold': 0.80
                },
                'target': {
                    'term_id': 'stock_symbol',
                    'semantic_phrase': 'symbol, ticker',
                    'confidence_threshold': 0.80
                },
                'constraints': {
                    'hierarchy_level': -1,  # Same document
                    'direction': 'any'
                },
                'confidence': {
                    'minimum': 0.75,
                    'calculation': 'average'
                }
            })

        # Check for company-revenue relationships
        relationships.append({
            'id': 'company_reports_revenue',
            'description': 'Company reports financial metrics',
            'relationship_type': 'REPORTS',
            'source': {
                'term_id': 'company_name',
                'semantic_phrase': 'company',
                'confidence_threshold': 0.80
            },
            'target': {
                'term_id': 'revenue',
                'semantic_phrase': 'revenue, financial metric',
                'confidence_threshold': 0.85
            },
            'constraints': {
                'hierarchy_level': -1,
                'direction': 'any'
            },
            'confidence': {
                'minimum': 0.70,
                'calculation': 'average'
            }
        })

        conn.close()
        print(f"  ✓ Discovered {len(relationships)} relationship patterns")
        return relationships

    def _build_ontology(self, domain_name: str) -> Dict[str, Any]:
        """Build the final ontology configuration."""
        return {
            'domain': {
                'name': domain_name,
                'version': '1.0.0',
                'description': f'Discovered ontology for {domain_name} domain',
                'settings': {
                    'default_confidence_threshold': 0.70,
                    'max_relationships_per_pair': 5,
                    'enable_transitive_inference': True
                }
            },
            'terms': self.context['discovered_terms'],
            'element_mappings': self.context['discovered_entities'],
            'relationship_rules': self.context['discovered_relationships'],
            'metadata': {
                'discovery_method': 'multi_step_orchestration',
                'corpus_profile': self.context['discovered_patterns'].get('corpus_profile', {}),
                'generated_from': 'analytics_database'
            }
        }

def main():
    """Test the discovery system with a greenfield domain."""

    # Configuration
    analytics_path = '/Volumes/T9/sec_analytics'
    domain_name = 'corporate_regulatory_filings'

    # Human-provided candidate terms (simulating domain knowledge)
    candidate_terms = [
        'company',        # Organizations filing reports
        'ticker',         # Stock symbols
        'revenue',        # Financial metrics
        'executive',      # Corporate officers
        'filing date',    # Temporal information
        'form type'       # Document categorization
    ]

    # Create orchestrator
    orchestrator = MockDiscoveryOrchestrator(analytics_path)

    # Execute discovery
    ontology = orchestrator.discover_ontology(domain_name, candidate_terms)

    # Display results
    print("\n" + "=" * 60)
    print("📋 GENERATED ONTOLOGY CONFIGURATION")
    print("=" * 60)

    # Pretty print the ontology
    import yaml
    print(yaml.dump(ontology, default_flow_style=False, sort_keys=False))

    # Summary statistics
    print("\n📊 ONTOLOGY SUMMARY")
    print("=" * 60)
    print(f"Domain: {ontology['domain']['name']}")
    print(f"Version: {ontology['domain']['version']}")
    print(f"Terms Defined: {len(ontology['terms'])}")
    print(f"Extraction Rules: {len(ontology['element_mappings'])}")
    print(f"Relationships: {len(ontology['relationship_rules'])}")
    print(f"Total Documents Analyzed: {ontology['metadata']['corpus_profile']['total_documents']}")
    print(f"Total Elements Analyzed: {ontology['metadata']['corpus_profile']['total_elements']}")

    # Save to file
    output_file = f"{domain_name}_discovered.yaml"
    with open(output_file, 'w') as f:
        yaml.dump(ontology, f, default_flow_style=False, sort_keys=False)

    print(f"\n✅ Ontology saved to: {output_file}")

if __name__ == '__main__':
    main()