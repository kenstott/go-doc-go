#!/usr/bin/env python3
"""
Test script for ontology generation with real data sources.
Uses non-interactive mode to test the complete flow.
"""

import os
import sys
import tempfile
from pathlib import Path

# Add src to path
sys.path.insert(0, str(Path(__file__).parent / "src"))

from go_doc_go.cli.ontology_generator import main
from go_doc_go.llm.chat import create_chat_provider
from go_doc_go.cli.ontology_interview import OntologyInterviewer
from go_doc_go.domain.ontology_builder import OntologyBuilder


def test_data_source_analysis():
    """Test the data source analysis functionality."""
    print("\n" + "="*60)
    print("Testing Ontology Generation with Data Sources")
    print("="*60)
    
    # Check for API keys
    has_anthropic = os.environ.get("ANTHROPIC_API_KEY")
    has_openai = os.environ.get("OPENAI_API_KEY")
    
    if not (has_anthropic or has_openai):
        print("⚠️ No API keys found. Set ANTHROPIC_API_KEY or OPENAI_API_KEY")
        return
    
    # Create chat provider
    provider = "anthropic" if has_anthropic else "openai"
    print(f"✅ Using {provider} as LLM provider")
    
    try:
        chat_provider = create_chat_provider(provider=provider)
    except Exception as e:
        print(f"❌ Failed to create chat provider: {e}")
        return
    
    # Create builder and interviewer
    builder = OntologyBuilder()
    interviewer = OntologyInterviewer(
        chat_provider=chat_provider,
        builder=builder,
        max_iterations=10,
        data_config_path="ontology_test_config.yaml"
    )
    
    # Simulate interview with pre-filled context
    print("\n📂 Analyzing documents from data sources...")
    
    # Initialize system prompt first
    interviewer.context.messages.append({
        "role": "system",
        "content": interviewer._get_system_prompt()
    })
    
    # Set up context
    interviewer.context.domain = "financial"
    
    # Run document analysis
    try:
        interviewer._analyze_sample_documents()
        
        print("\n✅ Document analysis complete!")
        print(f"   • Document types found: {interviewer.context.document_types}")
        print(f"   • Metadata fields found: {interviewer.context.metadata_fields[:5]}...")
        print(f"   • Sample content loaded: {len(interviewer.context.sample_content or '')} chars")
        
    except Exception as e:
        print(f"❌ Document analysis failed: {e}")
        import traceback
        traceback.print_exc()
        return
    
    # Generate ontology based on analyzed documents
    print("\n🤖 Generating ontology based on analyzed documents...")
    
    # Continue with automated generation
    interviewer.context.key_concepts = ["revenue", "earnings", "growth", "guidance", "segments"]
    
    # Generate terms
    print("   • Generating terms...")
    interviewer._phase_term_definition()
    
    # Generate entities
    print("   • Generating entities...")
    interviewer._phase_entity_extraction()
    
    # Generate relationships
    print("   • Generating relationships...")
    interviewer._phase_relationship_mapping()
    
    # Build final ontology
    ontology = interviewer._build_ontology()
    
    # Save to file
    output_file = "generated_data_driven_ontology.yaml"
    with open(output_file, 'w') as f:
        import yaml
        yaml.dump(ontology, f, default_flow_style=False, sort_keys=False)
    
    print(f"\n✅ Ontology generated and saved to: {output_file}")
    print("\n📊 Summary:")
    print(f"   • Domain: {ontology.get('domain', 'N/A')}")
    print(f"   • Document types: {len(ontology.get('document_types', []))}")
    print(f"   • Terms: {len(ontology.get('terms', []))}")
    print(f"   • Entity types: {len(ontology.get('element_entity_mappings', []))}")
    print(f"   • Relationships: {len(ontology.get('entity_relationship_rules', []))}")
    
    # Show sample of generated ontology
    print("\n📋 Sample of generated entities:")
    for entity in ontology.get('element_entity_mappings', [])[:3]:
        print(f"   • {entity.get('entity_type')}: {entity.get('description')}")
        for rule in entity.get('extraction_rules', [])[:2]:
            if rule.get('type') == 'metadata_field':
                print(f"     - Metadata: {rule.get('field_path')}")
            elif rule.get('type') == 'regex_pattern':
                print(f"     - Pattern: {rule.get('pattern')[:50]}...")


def test_non_interactive_mode():
    """Test running the CLI in non-interactive mode."""
    print("\n" + "="*60)
    print("Testing Non-Interactive Mode")
    print("="*60)
    
    # Create config for non-interactive mode
    config = {
        "domain": "financial",
        "document_types": ["earnings_call", "financial_report", "10-Q", "10-K"],
        "key_concepts": ["revenue", "profit", "growth", "guidance", "risk"],
        "data_config": "ontology_test_config.yaml"
    }
    
    config_file = "test_ontology_config.yaml"
    with open(config_file, 'w') as f:
        import yaml
        yaml.dump(config, f)
    
    print(f"✅ Created config file: {config_file}")
    
    # Simulate CLI args for non-interactive mode
    sys.argv = [
        "ontology_generator",
        "--non-interactive",
        "--config", config_file,
        "--data-config", "ontology_test_config.yaml",
        "--output", "non_interactive_ontology.yaml",
        "--llm-provider", "anthropic" if os.environ.get("ANTHROPIC_API_KEY") else "openai"
    ]
    
    print("\n🚀 Running ontology generator in non-interactive mode...")
    
    try:
        result = main()
        if result == 0:
            print("✅ Non-interactive generation successful!")
        else:
            print(f"⚠️ Generation completed with code: {result}")
    except Exception as e:
        print(f"❌ Non-interactive mode failed: {e}")
        import traceback
        traceback.print_exc()


if __name__ == "__main__":
    print("\n🎯 Go-Doc-Go Ontology Generator Test Suite")
    print("   Testing data source integration\n")
    
    # Test 1: Direct data source analysis
    test_data_source_analysis()
    
    # Test 2: Non-interactive mode
    # test_non_interactive_mode()  # Uncomment to test non-interactive mode
    
    print("\n✨ Test suite complete!")