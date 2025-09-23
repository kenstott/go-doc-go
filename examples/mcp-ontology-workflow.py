#!/usr/bin/env python3
"""
Example workflow for MCP-driven ontology generation.
Demonstrates how to use the sampling component to generate data-driven ontologies.
"""

import json
import sys
import os
from typing import Dict, Any

# Add src to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'src'))

from go_doc_go.mcp.ontology_sampler import OntologySampler, SamplingConfig, OntologyContext


def example_insider_trading_workflow():
    """Complete workflow for generating insider trading ontology."""

    print("=== MCP-Driven Ontology Generation Workflow ===\n")

    # Step 1: Define domain context
    print("Step 1: Defining domain context...")
    context = OntologyContext(
        domain_name="insider_trading_detection",
        keywords=[
            "insider", "trading", "SEC", "ownership", "transaction",
            "director", "officer", "beneficial", "reporting", "form"
        ],
        document_types=[
            "form_4", "form_3", "form_5", "ownership", "earnings",
            "SEC", "10-K", "10-Q"
        ],
        entity_hints=[
            "person", "company", "transaction", "date", "amount",
            "officer", "director", "insider"
        ],
        relationship_hints=[
            "employed_by", "executed", "involves_securities",
            "occurred_on", "serves_as"
        ]
    )

    print(f"Domain: {context.domain_name}")
    print(f"Keywords: {', '.join(context.keywords)}")
    print(f"Document types: {', '.join(context.document_types)}")
    print()

    # Step 2: Configure sampling
    print("Step 2: Configuring intelligent sampling...")
    config = SamplingConfig(
        max_elements=250,  # Increase for better coverage
        max_documents=60,
        min_elements_per_type=3,
        max_elements_per_type=40,
        include_temporal=True,
        include_rare_elements=True,
        diversity_factor=0.4  # Higher diversity
    )

    print(f"Max elements: {config.max_elements}")
    print(f"Max documents: {config.max_documents}")
    print(f"Diversity factor: {config.diversity_factor}")
    print()

    # Step 3: Sample data (would use actual database)
    print("Step 3: Sampling corpus data...")
    # In real usage:
    # sampler = OntologySampler(db_connection, config)
    # sampling_data = sampler.sample_for_domain(context)

    # For demonstration, create mock data
    sampling_data = create_mock_sampling_data(context)

    print(f"Sampled {sampling_data['sampling_summary']['total_elements']} elements")
    print(f"From {sampling_data['sampling_summary']['total_documents']} documents")
    print(f"Element types: {sampling_data['sampling_summary']['element_types']}")
    print(f"Format types: {sampling_data['sampling_summary']['format_types']}")
    print()

    # Step 4: Analyze patterns
    print("Step 4: Analyzing discovered patterns...")
    patterns = sampling_data["pattern_analysis"]

    print("Top structural names:")
    for name, count in list(patterns["structural_name_frequency"].items())[:10]:
        print(f"  {name}: {count}")

    print("\nElement type distribution:")
    for elem_type, count in patterns["element_type_frequency"].items():
        print(f"  {elem_type}: {count}")

    print(f"\nTemporal elements found: {len(patterns['temporal_patterns'])}")
    print()

    # Step 5: Generate analysis report
    print("Step 5: Generating analysis report for ontology creation...")

    analysis_report = generate_analysis_report(sampling_data)

    print("=== ANALYSIS REPORT ===")
    print(analysis_report)

    # Step 6: Create ontology prompt
    print("\nStep 6: Creating ontology generation prompt...")

    prompt = create_ontology_prompt(context, sampling_data)

    print("=== ONTOLOGY GENERATION PROMPT ===")
    print(prompt[:1000] + "...\n[truncated]")

    print("\n=== WORKFLOW COMPLETE ===")
    print("Next steps:")
    print("1. Review the analysis report")
    print("2. Use the generated prompt with an LLM")
    print("3. Validate the generated ontology against sample data")
    print("4. Deploy for knowledge graph generation")


def create_mock_sampling_data(context: OntologyContext) -> Dict[str, Any]:
    """Create mock sampling data for demonstration."""

    return {
        "domain_context": {
            "name": context.domain_name,
            "keywords": context.keywords,
            "document_types": context.document_types
        },
        "corpus_statistics": {
            "total_elements": 15420,
            "total_documents": 342,
            "format_distribution": {
                "xml": 12850,
                "json": 1890,
                "csv": 680
            },
            "category_distribution": {
                "sec_form_4": 8930,
                "sec_form_3": 2340,
                "earnings_report": 1650,
                "other": 2500
            }
        },
        "sampled_elements": [
            {
                "element_id": "xml_elem_rptOwnerName_1",
                "element_type": "xml_element",
                "structural_name": "rptOwnerName",
                "structural_path": "/ownershipDocument/reportingOwner/rptOwnerName",
                "content_preview": "BELL JAMES A",
                "format_type": "xml",
                "document_category": "sec_form_4",
                "has_temporal_value": False
            },
            {
                "element_id": "xml_elem_issuerName_2",
                "element_type": "xml_element",
                "structural_name": "issuerName",
                "structural_path": "/ownershipDocument/issuer/issuerName",
                "content_preview": "Apple Inc.",
                "format_type": "xml",
                "document_category": "sec_form_4",
                "has_temporal_value": False
            },
            {
                "element_id": "xml_elem_transactionDate_3",
                "element_type": "xml_element",
                "structural_name": "transactionDate",
                "structural_path": "/ownershipDocument/nonDerivativeTable/transactionDate",
                "content_preview": "2023-02-01",
                "format_type": "xml",
                "document_category": "sec_form_4",
                "has_temporal_value": True
            }
        ],
        "pattern_analysis": {
            "structural_name_frequency": {
                "rptOwnerName": 89,
                "issuerName": 82,
                "transactionDate": 76,
                "transactionShares": 71,
                "transactionCode": 69,
                "relationshipTitle": 45,
                "ownershipNature": 38
            },
            "element_type_frequency": {
                "xml_element": 195,
                "json_field": 35,
                "csv_cell": 20
            },
            "temporal_patterns": [
                {
                    "structural_name": "transactionDate",
                    "path": "/ownershipDocument/*/transactionDate",
                    "content": "2023-02-01"
                },
                {
                    "structural_name": "announcement_date",
                    "path": "earnings.announcement_date",
                    "content": "2023-02-02"
                }
            ]
        },
        "sampling_summary": {
            "total_elements": 250,
            "total_documents": 45,
            "element_types": 3,
            "format_types": 3,
            "has_temporal": 28
        }
    }


def generate_analysis_report(sampling_data: Dict[str, Any]) -> str:
    """Generate a comprehensive analysis report."""

    stats = sampling_data["corpus_statistics"]
    patterns = sampling_data["pattern_analysis"]
    summary = sampling_data["sampling_summary"]

    report = f"""
CORPUS ANALYSIS REPORT
=====================

COVERAGE:
- Total corpus: {stats['total_elements']} elements across {stats['total_documents']} documents
- Sample size: {summary['total_elements']} elements ({summary['total_elements']/stats['total_elements']*100:.1f}% coverage)
- Document formats: {', '.join(stats['format_distribution'].keys())}
- Primary categories: {', '.join(stats['category_distribution'].keys())}

KEY FINDINGS:
- Dominant format: {max(stats['format_distribution'], key=stats['format_distribution'].get)} ({max(stats['format_distribution'].values())} elements)
- Most common element: {list(patterns['structural_name_frequency'].keys())[0]} ({list(patterns['structural_name_frequency'].values())[0]} occurrences)
- Temporal coverage: {summary['has_temporal']} elements with dates/times
- Structural diversity: {summary['element_types']} distinct element types

ONTOLOGY RECOMMENDATIONS:
1. Focus on top 10 structural names for core entities
2. Use element_name search scope for high-precision matching
3. Include path-based rules for structural context
4. Create temporal relationship rules for date elements
5. Consider cross-document relationships for transaction patterns

PRIORITY ELEMENTS:
{chr(10).join([f"- {name}: {count} occurrences" for name, count in list(patterns['structural_name_frequency'].items())[:5]])}
"""

    return report


def create_ontology_prompt(context: OntologyContext, sampling_data: Dict[str, Any]) -> str:
    """Create the complete ontology generation prompt."""

    prompt = f"""
# Domain-Driven Ontology Generation

## Domain Context
**Domain**: {context.domain_name}
**Keywords**: {', '.join(context.keywords)}
**Document Types**: {', '.join(context.document_types)}

## Sampled Data Analysis
{json.dumps(sampling_data, indent=2)[:2000]}...[truncated]

## Task
Based on the above sampled data, generate a comprehensive ontology YAML that:

1. Creates terms for the most frequent structural names
2. Uses precise element_name matching for high-confidence rules
3. Includes path-based matching for structural context
4. Defines meaningful relationships based on document hierarchy
5. Incorporates temporal elements for event sequencing

Focus on data-driven rules that would capture the sampled elements with high precision.

Generate the complete ontology YAML now:
"""

    return prompt


if __name__ == "__main__":
    example_insider_trading_workflow()