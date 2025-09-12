#!/usr/bin/env python3
"""
Demo: Ontology Generator with Data Source Integration

This demonstrates how the ontology generator can analyze actual documents
from configured data sources to create more accurate ontologies.
"""

import os
import sys
from pathlib import Path

# Add src to path
sys.path.insert(0, str(Path(__file__).parent / "src"))

def demo_data_source_integration():
    """Demonstrate data source integration in ontology generation."""
    
    print("\n" + "="*70)
    print("  Go-Doc-Go Ontology Generator - Data Source Integration Demo")
    print("="*70)
    
    print("""
This demo shows how the ontology generator can:
1. Load documents from configured data sources (files, S3, SharePoint, etc.)
2. Analyze document structure and metadata
3. Extract patterns and common fields
4. Use this information to suggest better ontologies
    """)
    
    print("\n📋 Configuration File: ontology_test_config.yaml")
    print("-" * 50)
    with open("ontology_test_config.yaml", "r") as f:
        print(f.read())
    
    print("\n📂 Sample Documents Created:")
    print("-" * 50)
    sample_dir = Path("examples/sample_docs")
    for doc in sample_dir.glob("*"):
        print(f"  • {doc.name}")
    
    print("\n🚀 How to Use:")
    print("-" * 50)
    print("""
1. With data sources (recommended):
   python -m go_doc_go.cli.ontology_generator \\
     --data-config ontology_test_config.yaml \\
     --output my_ontology.yaml

2. Interactive mode with templates:
   python -m go_doc_go.cli.ontology_generator \\
     --template financial \\
     --data-config ontology_test_config.yaml

3. Non-interactive with config:
   python -m go_doc_go.cli.ontology_generator \\
     --non-interactive \\
     --config example_ontology_config.yaml \\
     --data-config ontology_test_config.yaml
    """)
    
    print("\n✨ Key Features:")
    print("-" * 50)
    print("""
• Automatic document analysis:
  - Detects document types (JSON, Markdown, PDF, etc.)
  - Extracts metadata fields from actual documents
  - Analyzes content patterns

• Smart suggestions:
  - Uses found metadata fields for entity extraction rules
  - Suggests entities based on actual content
  - Creates relationships based on document structure

• Flexible configuration:
  - Works with any Go-Doc-Go content source
  - Can analyze S3 buckets, SharePoint sites, local files
  - Supports incremental refinement
    """)
    
    print("\n📊 Example Generated Ontology:")
    print("-" * 50)
    
    # Show a sample of what would be generated
    sample_ontology = """
name: financial_documents
version: 1.0.0
domain: financial
description: Auto-generated from sample financial documents

# Document types found in your data
document_types:
  - quarterly_report
  - earnings_call
  - financial_report

# Metadata fields discovered
metadata_fields:
  - company
  - ticker
  - fiscal_year
  - quarter
  - speaker
  - speaker_role

# Entities with extraction rules based on your data
element_entity_mappings:
  - entity_type: company
    description: Company names found in documents
    extraction_rules:
      - type: metadata_field
        field_path: metadata.company  # Found in your documents
      - type: regex_pattern
        pattern: \\b[A-Z][A-Za-z]+(?:\\s+[A-Z][A-Za-z]+)*\\s+(?:Inc|Corp|LLC)\\b
        
  - entity_type: speaker
    description: Speakers in earnings calls
    extraction_rules:
      - type: metadata_field
        field_path: metadata.speaker  # Found in your documents
      - type: metadata_field
        field_path: metadata.speaker_role
        
  - entity_type: monetary_amount
    description: Financial amounts
    extraction_rules:
      - type: regex_pattern
        pattern: \\$[\\d,]+(?:\\.\\d{2})?(?:\\s*(?:billion|million))?
    """
    
    print(sample_ontology)
    
    print("\n🎯 Benefits of Data-Driven Ontology Generation:")
    print("-" * 50)
    print("""
1. More Accurate: Based on your actual documents, not generic templates
2. Faster Setup: Automatically discovers patterns and fields
3. Better Coverage: Ensures all important metadata fields are captured
4. Reduced Errors: Less manual configuration means fewer mistakes
5. Iterative Improvement: Can refine based on real extraction results
    """)
    
    print("\n💡 Next Steps:")
    print("-" * 50)
    print("""
1. Add more sample documents to examples/sample_docs/
2. Run the ontology generator with your data
3. Review and refine the generated ontology
4. Use it with Go-Doc-Go for document processing
    """)
    
    print("\n" + "="*70)
    print("  End of Demo")
    print("="*70)


if __name__ == "__main__":
    demo_data_source_integration()