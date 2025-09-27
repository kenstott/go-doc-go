#!/usr/bin/env python3
"""
Analyze SEC analytics Parquet data to understand domain patterns for ontology extraction.

This script explores the structure and content of documents, elements, and relationships
to identify patterns suitable for ontology rules.
"""

import pandas as pd
import pyarrow.parquet as pq
import os
from pathlib import Path
import re
from collections import Counter, defaultdict
import json
from typing import Dict, List, Any, Set, Tuple
import numpy as np


class ParquetAnalyzer:
    """Analyzer for Parquet-based SEC document analytics."""

    def __init__(self, base_path: str):
        self.base_path = Path(base_path)
        self.documents_path = self.base_path / "documents"
        self.elements_path = self.base_path / "elements"
        self.relationships_path = self.base_path / "relationships"

        # Analysis results
        self.document_schemas = {}
        self.element_schemas = {}
        self.relationship_schemas = {}
        self.sample_data = {}
        self.patterns = defaultdict(Counter)

    def discover_files(self) -> Dict[str, List[str]]:
        """Discover all Parquet files in the analytics directory."""
        files = {
            'documents': [],
            'elements': [],
            'relationships': []
        }

        for category in files.keys():
            path = self.base_path / category
            if path.exists():
                for parquet_file in path.rglob("*.parquet"):
                    files[category].append(str(parquet_file))

        return files

    def analyze_schema(self, file_path: str) -> Dict[str, Any]:
        """Analyze the schema of a Parquet file."""
        try:
            table = pq.read_table(file_path)
            schema_info = {
                'columns': list(table.column_names),
                'types': [str(field.type) for field in table.schema],
                'num_rows': table.num_rows,
                'file_size_mb': os.path.getsize(file_path) / (1024 * 1024)
            }
            return schema_info
        except Exception as e:
            print(f"Error analyzing schema for {file_path}: {e}")
            return {}

    def sample_parquet_data(self, file_path: str, sample_size: int = 100) -> pd.DataFrame:
        """Sample data from a Parquet file."""
        try:
            df = pd.read_parquet(file_path)
            if len(df) > sample_size:
                return df.sample(n=sample_size)
            return df
        except Exception as e:
            print(f"Error sampling data from {file_path}: {e}")
            return pd.DataFrame()

    def analyze_documents(self, files: List[str]) -> None:
        """Analyze document data patterns."""
        print("\n=== ANALYZING DOCUMENTS ===")

        if not files:
            print("No document files found")
            return

        # Analyze first file for schema
        first_file = files[0]
        print(f"Analyzing schema from: {first_file}")
        self.document_schemas = self.analyze_schema(first_file)

        print(f"Document Schema:")
        for col, dtype in zip(self.document_schemas['columns'], self.document_schemas['types']):
            print(f"  {col}: {dtype}")

        # Sample data from multiple files
        sample_dfs = []
        for file_path in files[:5]:  # Sample from first 5 files
            df = self.sample_data(file_path, 20)
            if not df.empty:
                sample_dfs.append(df)

        if sample_dfs:
            combined_df = pd.concat(sample_dfs, ignore_index=True)
            self.sample_data['documents'] = combined_df

            print(f"\nDocument Sample Analysis ({len(combined_df)} records):")
            print(f"Columns: {list(combined_df.columns)}")

            # Analyze specific patterns
            if 'document_type' in combined_df.columns:
                doc_types = combined_df['document_type'].value_counts()
                print(f"\nDocument Types:")
                for doc_type, count in doc_types.head(10).items():
                    print(f"  {doc_type}: {count}")
                self.patterns['document_types'] = doc_types

            if 'source_url' in combined_df.columns:
                urls = combined_df['source_url'].dropna()
                url_patterns = []
                for url in urls.head(10):
                    # Extract filing type from URL
                    match = re.search(r'/(10-[KQ]|8-K|DEF|S-[0-9]|20-F)/', str(url))
                    if match:
                        url_patterns.append(match.group(1))

                if url_patterns:
                    filing_types = Counter(url_patterns)
                    print(f"\nFiling Types from URLs:")
                    for filing_type, count in filing_types.items():
                        print(f"  {filing_type}: {count}")
                    self.patterns['filing_types'] = filing_types

    def analyze_elements(self, files: List[str]) -> None:
        """Analyze element data patterns."""
        print("\n=== ANALYZING ELEMENTS ===")

        if not files:
            print("No element files found")
            return

        # Analyze first file for schema
        first_file = files[0]
        print(f"Analyzing schema from: {first_file}")
        self.element_schemas = self.analyze_schema(first_file)

        print(f"Element Schema:")
        for col, dtype in zip(self.element_schemas['columns'], self.element_schemas['types']):
            print(f"  {col}: {dtype}")

        # Sample data from multiple files
        sample_dfs = []
        total_elements = 0
        for file_path in files[:10]:  # Sample from first 10 files
            df = self.sample_data(file_path, 50)
            if not df.empty:
                sample_dfs.append(df)
                total_elements += len(df)

        if sample_dfs:
            combined_df = pd.concat(sample_dfs, ignore_index=True)
            self.sample_data['elements'] = combined_df

            print(f"\nElement Sample Analysis ({len(combined_df)} records):")

            # Analyze element types
            if 'element_type' in combined_df.columns:
                element_types = combined_df['element_type'].value_counts()
                print(f"\nElement Types:")
                for elem_type, count in element_types.head(15).items():
                    print(f"  {elem_type}: {count}")
                self.patterns['element_types'] = element_types

            # Analyze content patterns
            if 'content_preview' in combined_df.columns:
                self._analyze_content_patterns(combined_df['content_preview'].dropna())

            # Analyze metadata patterns
            if 'metadata' in combined_df.columns:
                self._analyze_metadata_patterns(combined_df['metadata'].dropna())

    def analyze_relationships(self, files: List[str]) -> None:
        """Analyze relationship data patterns."""
        print("\n=== ANALYZING RELATIONSHIPS ===")

        if not files:
            print("No relationship files found")
            return

        # Analyze first file for schema
        first_file = files[0]
        print(f"Analyzing schema from: {first_file}")
        self.relationship_schemas = self.analyze_schema(first_file)

        print(f"Relationship Schema:")
        for col, dtype in zip(self.relationship_schemas['columns'], self.relationship_schemas['types']):
            print(f"  {col}: {dtype}")

        # Sample data from multiple files
        sample_dfs = []
        for file_path in files[:10]:  # Sample from first 10 files
            df = self.sample_data(file_path, 50)
            if not df.empty:
                sample_dfs.append(df)

        if sample_dfs:
            combined_df = pd.concat(sample_dfs, ignore_index=True)
            self.sample_data['relationships'] = combined_df

            print(f"\nRelationship Sample Analysis ({len(combined_df)} records):")

            # Analyze relationship types
            if 'relationship_type' in combined_df.columns:
                rel_types = combined_df['relationship_type'].value_counts()
                print(f"\nRelationship Types:")
                for rel_type, count in rel_types.head(15).items():
                    print(f"  {rel_type}: {count}")
                self.patterns['relationship_types'] = rel_types

            # Analyze relationship patterns
            if 'source_element_type' in combined_df.columns and 'target_element_type' in combined_df.columns:
                rel_patterns = []
                for _, row in combined_df.iterrows():
                    if pd.notna(row.get('source_element_type')) and pd.notna(row.get('target_element_type')):
                        pattern = f"{row['source_element_type']} -> {row['target_element_type']}"
                        rel_patterns.append(pattern)

                pattern_counts = Counter(rel_patterns)
                print(f"\nElement Relationship Patterns:")
                for pattern, count in pattern_counts.most_common(10):
                    print(f"  {pattern}: {count}")
                self.patterns['element_patterns'] = pattern_counts

    def _analyze_content_patterns(self, content_series: pd.Series) -> None:
        """Analyze patterns in content previews."""
        print(f"\nContent Pattern Analysis ({len(content_series)} samples):")

        # Financial terms
        financial_terms = [
            'revenue', 'income', 'profit', 'loss', 'earnings', 'assets', 'liabilities',
            'equity', 'cash', 'debt', 'investment', 'dividend', 'share', 'stock',
            'market', 'capital', 'fund', 'securities', 'financial', 'fiscal',
            'quarter', 'annual', 'million', 'billion', 'thousand', '$', '%'
        ]

        # Business terms
        business_terms = [
            'company', 'corporation', 'business', 'operations', 'management',
            'board', 'director', 'officer', 'employee', 'customer', 'client',
            'contract', 'agreement', 'acquisition', 'merger', 'subsidiary',
            'segment', 'division', 'product', 'service', 'technology'
        ]

        # Regulatory terms
        regulatory_terms = [
            'sec', 'regulation', 'compliance', 'filing', 'disclosure', 'report',
            'audit', 'risk', 'control', 'governance', 'policy', 'procedure',
            'legal', 'litigation', 'settlement', 'penalty', 'violation'
        ]

        all_terms = {
            'financial': financial_terms,
            'business': business_terms,
            'regulatory': regulatory_terms
        }

        term_counts = defaultdict(Counter)

        for content in content_series:
            if pd.notna(content):
                content_lower = str(content).lower()
                for category, terms in all_terms.items():
                    for term in terms:
                        if term in content_lower:
                            term_counts[category][term] += 1

        for category, counts in term_counts.items():
            if counts:
                print(f"\n{category.title()} Terms Found:")
                for term, count in counts.most_common(10):
                    print(f"  {term}: {count}")
                self.patterns[f'{category}_terms'] = counts

    def _analyze_metadata_patterns(self, metadata_series: pd.Series) -> None:
        """Analyze patterns in metadata fields."""
        print(f"\nMetadata Pattern Analysis:")

        metadata_keys = Counter()
        metadata_values = defaultdict(Counter)

        for metadata in metadata_series:
            if pd.notna(metadata):
                try:
                    if isinstance(metadata, str):
                        meta_dict = json.loads(metadata)
                    else:
                        meta_dict = metadata

                    if isinstance(meta_dict, dict):
                        for key, value in meta_dict.items():
                            metadata_keys[key] += 1
                            if isinstance(value, str) and len(value) < 100:
                                metadata_values[key][value] += 1
                except:
                    continue

        print(f"Common Metadata Keys:")
        for key, count in metadata_keys.most_common(15):
            print(f"  {key}: {count}")

        self.patterns['metadata_keys'] = metadata_keys
        self.patterns['metadata_values'] = dict(metadata_values)

    def generate_ontology_insights(self) -> Dict[str, Any]:
        """Generate insights for ontology creation."""
        insights = {
            'document_types': [],
            'element_hierarchies': [],
            'relationship_patterns': [],
            'domain_entities': [],
            'recommended_extractions': []
        }

        # Document type insights
        if 'document_types' in self.patterns:
            insights['document_types'] = [
                {'type': doc_type, 'frequency': count}
                for doc_type, count in self.patterns['document_types'].most_common(10)
            ]

        # Element hierarchy insights
        if 'element_types' in self.patterns:
            insights['element_hierarchies'] = [
                {'element_type': elem_type, 'frequency': count}
                for elem_type, count in self.patterns['element_types'].most_common(15)
            ]

        # Relationship pattern insights
        if 'element_patterns' in self.patterns:
            insights['relationship_patterns'] = [
                {'pattern': pattern, 'frequency': count}
                for pattern, count in self.patterns['element_patterns'].most_common(10)
            ]

        # Domain entity insights
        domain_entities = []
        for category in ['financial_terms', 'business_terms', 'regulatory_terms']:
            if category in self.patterns:
                for term, count in self.patterns[category].most_common(15):
                    domain_entities.append({
                        'term': term,
                        'category': category.replace('_terms', ''),
                        'frequency': count
                    })
        insights['domain_entities'] = domain_entities

        # Ontology extraction recommendations
        recommendations = []

        # Entity extraction recommendations
        if 'financial_terms' in self.patterns:
            recommendations.append({
                'type': 'entity_extraction',
                'category': 'financial_entities',
                'description': 'Extract financial metrics, amounts, and percentages',
                'patterns': ['$[0-9,.]+ (million|billion)', '[0-9]+%', 'revenue', 'profit', 'loss']
            })

        if 'business_terms' in self.patterns:
            recommendations.append({
                'type': 'entity_extraction',
                'category': 'organizational_entities',
                'description': 'Extract company names, roles, and organizational structures',
                'patterns': ['CEO', 'CFO', 'board of directors', 'subsidiary', 'acquisition']
            })

        # Relationship extraction recommendations
        if 'element_patterns' in self.patterns:
            common_patterns = self.patterns['element_patterns'].most_common(5)
            recommendations.append({
                'type': 'relationship_extraction',
                'category': 'structural_relationships',
                'description': 'Extract document structure and element relationships',
                'patterns': [pattern for pattern, _ in common_patterns]
            })

        insights['recommended_extractions'] = recommendations

        return insights

    def run_full_analysis(self) -> Dict[str, Any]:
        """Run complete analysis of the Parquet data."""
        print("Starting comprehensive Parquet data analysis...")

        # Discover files
        files = self.discover_files()
        print(f"\nFound files:")
        for category, file_list in files.items():
            print(f"  {category}: {len(file_list)} files")

        # Analyze each category
        self.analyze_documents(files['documents'])
        self.analyze_elements(files['elements'])
        self.analyze_relationships(files['relationships'])

        # Generate ontology insights
        insights = self.generate_ontology_insights()

        return {
            'file_summary': files,
            'schemas': {
                'documents': self.document_schemas,
                'elements': self.element_schemas,
                'relationships': self.relationship_schemas
            },
            'patterns': dict(self.patterns),
            'ontology_insights': insights
        }


def main():
    """Main analysis function."""
    base_path = "/Volumes/T9/sec_analytics/"

    analyzer = ParquetAnalyzer(base_path)
    results = analyzer.run_full_analysis()

    # Save results
    output_file = "/Users/kennethstott/PycharmProjects/doculyzer-ontology-test/parquet_analysis_results.json"

    # Convert Counter objects to regular dicts for JSON serialization
    def convert_counters(obj):
        if isinstance(obj, Counter):
            return dict(obj)
        elif isinstance(obj, dict):
            return {k: convert_counters(v) for k, v in obj.items()}
        elif isinstance(obj, list):
            return [convert_counters(item) for item in obj]
        return obj

    serializable_results = convert_counters(results)

    with open(output_file, 'w') as f:
        json.dump(serializable_results, f, indent=2, default=str)

    print(f"\n=== ANALYSIS COMPLETE ===")
    print(f"Results saved to: {output_file}")

    # Print summary
    print(f"\n=== SUMMARY ===")
    insights = results['ontology_insights']

    print(f"\nTop Document Types:")
    for doc_type in insights['document_types'][:5]:
        print(f"  {doc_type['type']}: {doc_type['frequency']}")

    print(f"\nTop Element Types:")
    for elem_type in insights['element_hierarchies'][:5]:
        print(f"  {elem_type['element_type']}: {elem_type['frequency']}")

    print(f"\nTop Domain Entities:")
    for entity in insights['domain_entities'][:10]:
        print(f"  {entity['term']} ({entity['category']}): {entity['frequency']}")

    print(f"\nOntology Extraction Recommendations:")
    for rec in insights['recommended_extractions']:
        print(f"  {rec['category']}: {rec['description']}")


if __name__ == "__main__":
    main()