#!/usr/bin/env python3
"""
Test interactive ontology generation with simulated input.
"""

import sys
from pathlib import Path
from io import StringIO
from unittest.mock import patch

# Add src to path
sys.path.insert(0, str(Path(__file__).parent / "src"))

from go_doc_go.llm import create_chat_provider
from go_doc_go.cli.ontology_interview import OntologyInterviewer
from go_doc_go.domain.ontology_builder import OntologyBuilder
from go_doc_go.domain.templates import ONTOLOGY_TEMPLATES


def test_interactive_generation():
    """Test the interactive ontology generation with simulated input."""
    
    # Simulate user input for the interview
    simulated_inputs = [
        # Phase 1: Domain Understanding
        "financial",  # Domain
        "earnings_calls, 10-K reports",  # Document types
        "revenue, profit, guidance, risk factors",  # Key concepts
        "n",  # No sample content
        
        # Phase 2: Term Definition
        "c",  # Continue with suggested terms
        
        # Phase 3: Entity Extraction
        "c",  # Continue with suggested entities
        
        # Phase 4: Relationship Mapping
        "c",  # Continue with suggested relationships
        
        # Phase 5: Refinement
        "n",  # No refinement needed
    ]
    
    print("🧪 Testing Interactive Ontology Generation")
    print("=" * 60)
    
    # Initialize components
    chat_provider = create_chat_provider("auto")
    print(f"✅ Using chat provider: {chat_provider.__class__.__name__}")
    
    builder = OntologyBuilder(template=ONTOLOGY_TEMPLATES["financial"])
    interviewer = OntologyInterviewer(chat_provider, builder, max_iterations=10)
    
    # Patch input to use simulated input
    with patch('builtins.input', side_effect=simulated_inputs):
        try:
            # Conduct interview
            print("\n📝 Starting simulated interview...")
            ontology = interviewer.conduct_interview()
            
            # Display results
            print("\n✨ Interview complete!")
            print(f"\n📊 Generated Ontology Summary:")
            print(f"  Name: {ontology.get('name', 'N/A')}")
            print(f"  Version: {ontology.get('version', 'N/A')}")
            print(f"  Terms: {len(ontology.get('terms', []))}")
            print(f"  Entity Mappings: {len(ontology.get('element_entity_mappings', []))}")
            print(f"  Relationships: {len(ontology.get('entity_relationship_rules', []))}")
            
            # Show sample of generated YAML
            yaml_output = builder.to_yaml(ontology)
            print("\n📄 Generated YAML (first 500 chars):")
            print(yaml_output[:500])
            
            return ontology
            
        except Exception as e:
            print(f"\n❌ Error during interview: {e}")
            import traceback
            traceback.print_exc()
            return None


if __name__ == "__main__":
    result = test_interactive_generation()
    if result:
        print("\n✅ Test completed successfully!")
    else:
        print("\n❌ Test failed!")
        sys.exit(1)