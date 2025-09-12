#!/usr/bin/env python3
"""
Test script for the ontology generator CLI.
"""

import sys
import os
from pathlib import Path

# Add src to path
sys.path.insert(0, str(Path(__file__).parent.parent / "src"))

from go_doc_go.llm import create_chat_provider
from go_doc_go.domain.ontology_builder import OntologyBuilder
from go_doc_go.domain.templates import ONTOLOGY_TEMPLATES


def test_builder():
    """Test the ontology builder."""
    print("Testing Ontology Builder")
    print("=" * 60)
    
    # Create builder with financial template
    builder = OntologyBuilder(template=ONTOLOGY_TEMPLATES["financial"])
    
    # Build a simple ontology
    config = {
        "name": "test_financial_ontology",
        "version": "1.0.0",
        "description": "Test financial ontology",
        "metadata": {
            "domain": "financial",
            "created_by": "test_script"
        }
    }
    
    ontology = builder.build_from_config(config)
    
    # Validate
    issues = builder.validate(ontology)
    if issues:
        print("Validation issues:")
        for issue in issues:
            print(f"  - {issue}")
    else:
        print("✅ Ontology validation passed!")
    
    # Convert to YAML
    yaml_output = builder.to_yaml(ontology)
    print("\nGenerated YAML (first 500 chars):")
    print(yaml_output[:500])
    
    return ontology


def test_chat_provider():
    """Test chat provider initialization."""
    print("\nTesting Chat Provider")
    print("=" * 60)
    
    # Try to create auto provider
    provider = create_chat_provider("auto")
    print(f"✅ Created provider: {provider.__class__.__name__}")
    
    # Check if it's available
    if provider.is_available():
        print("✅ Provider is available")
        
        # Test a simple completion
        if not isinstance(provider, type(provider).__class__):
            messages = [
                {"role": "system", "content": "You are a helpful assistant."},
                {"role": "user", "content": "Say 'Hello, ontology generator!' in 5 words or less."}
            ]
            
            try:
                response = provider.chat_completion(messages, temperature=0.5)
                print(f"Response: {response}")
            except Exception as e:
                print(f"Chat completion failed: {e}")
    else:
        print("⚠️ Provider not available (likely using mock)")
    
    return provider


def test_interview_context():
    """Test interview context creation."""
    print("\nTesting Interview Context")
    print("=" * 60)
    
    from go_doc_go.cli.ontology_interview import InterviewContext
    
    context = InterviewContext(
        domain="financial",
        document_types=["earnings_call", "financial_report"],
        key_concepts=["revenue", "profit", "growth"]
    )
    
    # Add some test terms
    context.terms.append({
        "term": "revenue",
        "synonyms": ["income", "sales"],
        "description": "Money generated from operations"
    })
    
    # Add test entity
    context.entities.append({
        "entity_type": "company",
        "description": "Business organization",
        "element_types": ["paragraph"],
        "extraction_rules": [
            {"type": "REGEX", "pattern": r"\b[A-Z]\w+\s+(Inc|Corp|LLC)\b"}
        ]
    })
    
    print(f"Context created:")
    print(f"  Domain: {context.domain}")
    print(f"  Document types: {context.document_types}")
    print(f"  Key concepts: {context.key_concepts}")
    print(f"  Terms: {len(context.terms)}")
    print(f"  Entities: {len(context.entities)}")
    
    return context


def main():
    """Run all tests."""
    print("🧪 Testing Ontology Generator Components")
    print("=" * 60)
    
    # Test builder
    ontology = test_builder()
    
    # Test chat provider
    provider = test_chat_provider()
    
    # Test interview context
    context = test_interview_context()
    
    print("\n" + "=" * 60)
    print("✅ All tests completed!")
    
    # Show how to run the actual CLI
    print("\n📚 To run the interactive ontology generator:")
    print("  python -m go_doc_go.cli.ontology_generator")
    print("\nWith specific provider:")
    print("  python -m go_doc_go.cli.ontology_generator --llm-provider anthropic")
    print("\nWith template:")
    print("  python -m go_doc_go.cli.ontology_generator --template financial")
    
    # Check for API keys
    print("\n🔑 API Key Status:")
    if os.environ.get("ANTHROPIC_API_KEY"):
        print("  ✅ ANTHROPIC_API_KEY found (will be default)")
    else:
        print("  ❌ ANTHROPIC_API_KEY not found")
    
    if os.environ.get("OPENAI_API_KEY"):
        print("  ✅ OPENAI_API_KEY found")
    else:
        print("  ❌ OPENAI_API_KEY not found")


if __name__ == "__main__":
    main()