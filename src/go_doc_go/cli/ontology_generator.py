#!/usr/bin/env python3
"""
Command-line interface for generating domain ontologies through LLM-guided interviews.
"""

import argparse
import logging
import os
import sys
from pathlib import Path
from typing import Optional, Dict, Any

# Add the src directory to Python path
sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from go_doc_go.llm.chat import ChatProvider, create_chat_provider
from go_doc_go.cli.ontology_interview import OntologyInterviewer
from go_doc_go.domain.ontology_builder import OntologyBuilder


def main():
    """Main entry point for the ontology generator CLI."""
    parser = argparse.ArgumentParser(
        description="Go-Doc-Go Ontology Generator - Interactive domain ontology creation",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Start interactive ontology creation
  python -m go_doc_go.cli.ontology_generator
  
  # Use specific LLM provider
  python -m go_doc_go.cli.ontology_generator --llm-provider openai --model gpt-4
  
  # Start from template
  python -m go_doc_go.cli.ontology_generator --template financial --output my_ontology.yaml
  
  # Validate with sample documents
  python -m go_doc_go.cli.ontology_generator --validate-with samples/ --output ontology.yaml

Environment Variables:
  OPENAI_API_KEY: OpenAI API key for GPT models
  ANTHROPIC_API_KEY: Anthropic API key for Claude models
        """
    )
    
    parser.add_argument(
        "--output", "-o",
        default="ontology.yaml",
        help="Output file path for generated ontology (default: ontology.yaml)"
    )
    
    parser.add_argument(
        "--llm-provider",
        choices=["openai", "anthropic", "ollama", "auto"],
        default="auto",
        help="LLM provider to use for interview (default: auto-detect)"
    )
    
    parser.add_argument(
        "--model",
        help="Specific model to use (e.g., gpt-4, claude-3-opus, llama2)"
    )
    
    parser.add_argument(
        "--template", "-t",
        choices=["financial", "legal", "medical", "technical", "none"],
        default="none",
        help="Base template to start from (default: none)"
    )
    
    parser.add_argument(
        "--validate-with",
        help="Directory containing sample documents for validation"
    )
    
    parser.add_argument(
        "--non-interactive",
        action="store_true",
        help="Run in non-interactive mode (requires --config)"
    )
    
    parser.add_argument(
        "--config",
        help="Configuration file (for non-interactive mode or to load data sources)"
    )
    
    parser.add_argument(
        "--data-config",
        help="Go-Doc-Go config file with data sources to analyze"
    )
    
    parser.add_argument(
        "--log-level", "-l",
        choices=["DEBUG", "INFO", "WARNING", "ERROR"],
        default="INFO",
        help="Logging level (default: INFO)"
    )
    
    parser.add_argument(
        "--format", "-f",
        choices=["yaml", "json"],
        default="yaml",
        help="Output format (default: yaml)"
    )
    
    parser.add_argument(
        "--max-iterations",
        type=int,
        default=20,
        help="Maximum interview iterations (default: 20)"
    )
    
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Preview ontology without saving"
    )
    
    args = parser.parse_args()
    
    # Configure logging
    logging.basicConfig(
        level=getattr(logging, args.log_level),
        format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
    )
    logger = logging.getLogger(__name__)
    
    # Validate arguments
    if args.non_interactive and not args.config:
        parser.error("--config is required when using --non-interactive")
    
    try:
        # Initialize LLM provider
        logger.info(f"Initializing LLM provider: {args.llm_provider}")
        chat_provider = create_chat_provider(
            provider=args.llm_provider,
            model=args.model
        )
        
        # Load template if specified
        template = None
        if args.template != "none":
            logger.info(f"Loading template: {args.template}")
            template = load_template(args.template)
        
        # Create ontology builder
        builder = OntologyBuilder(template=template)
        
        if args.non_interactive:
            # Non-interactive mode
            logger.info("Running in non-interactive mode")
            with open(args.config, 'r') as f:
                import yaml
                config = yaml.safe_load(f)
            
            ontology = builder.build_from_config(config)
        else:
            # Interactive interview mode
            logger.info("Starting interactive ontology interview")
            print("\n🎯 Welcome to the Go-Doc-Go Ontology Generator!")
            print("=" * 60)
            print("I'll help you create a domain ontology for document analysis.")
            print("\n📋 How this works:")
            print("  1. You provide your domain (e.g., 'financial', 'legal', 'medical')")
            print("  2. AI suggests document types, terms, entities, and relationships")
            print("  3. You can accept, modify, or replace any suggestions")
            print("  4. The result is a complete ontology for document extraction")
            print("\n💡 Tip: Press Enter to accept AI suggestions, or type your own.")
            print("=" * 60)
            
            interviewer = OntologyInterviewer(
                chat_provider=chat_provider,
                builder=builder,
                max_iterations=args.max_iterations,
                data_config_path=args.data_config or args.config
            )
            
            # Run the interview
            ontology = interviewer.conduct_interview()
            
            print("\n✨ Interview complete!")
        
        # Validate if requested
        if args.validate_with:
            logger.info(f"Validating ontology with documents in: {args.validate_with}")
            print(f"\n🔍 Validating ontology with sample documents...")
            validation_results = validate_ontology(
                ontology, 
                Path(args.validate_with)
            )
            print_validation_results(validation_results)
        
        # Preview or save
        if args.dry_run:
            print("\n📋 Generated Ontology Preview:")
            print("=" * 60)
            if args.format == "yaml":
                print(builder.to_yaml(ontology))
            else:
                import json
                print(json.dumps(builder.to_dict(ontology), indent=2))
        else:
            # Save to file
            output_path = Path(args.output)
            logger.info(f"Saving ontology to: {output_path}")
            
            if args.format == "yaml":
                with open(output_path, 'w') as f:
                    f.write(builder.to_yaml(ontology))
            else:
                import json
                with open(output_path, 'w') as f:
                    json.dump(builder.to_dict(ontology), f, indent=2)
            
            print(f"\n✅ Ontology saved to: {output_path}")
            print(f"📊 Summary:")
            print(f"  - Terms: {len(ontology.get('terms', []))}")
            print(f"  - Element Mappings: {len(ontology.get('element_entity_mappings', []))}")
            print(f"  - Entity Relationships: {len(ontology.get('entity_relationship_rules', []))}")
            print(f"  - Derived Entities: {len(ontology.get('derived_entities', []))}")
        
        return 0
        
    except KeyboardInterrupt:
        logger.info("Interview cancelled by user")
        print("\n\n👋 Interview cancelled. Goodbye!")
        return 1
        
    except Exception as e:
        logger.error(f"Ontology generation failed: {str(e)}")
        logger.debug("Exception details:", exc_info=True)
        print(f"\n❌ Error: {str(e)}")
        return 1


def load_template(template_name: str) -> Dict[str, Any]:
    """Load a template ontology."""
    template_dir = Path(__file__).parent.parent.parent.parent / "examples" / "ontology_templates"
    template_file = template_dir / f"{template_name}.yaml"
    
    if not template_file.exists():
        # Try to load from built-in templates
        from go_doc_go.domain.templates import ONTOLOGY_TEMPLATES
        if template_name in ONTOLOGY_TEMPLATES:
            return ONTOLOGY_TEMPLATES[template_name]
        raise FileNotFoundError(f"Template not found: {template_name}")
    
    import yaml
    with open(template_file, 'r') as f:
        return yaml.safe_load(f)


def validate_ontology(ontology: Dict[str, Any], samples_dir: Path) -> Dict[str, Any]:
    """Validate ontology against sample documents."""
    from go_doc_go.domain import OntologyManager
    from go_doc_go.document_parser.factory import get_parser_for_file
    
    results = {
        "total_documents": 0,
        "successful_extractions": 0,
        "entities_found": [],
        "relationships_found": [],
        "errors": []
    }
    
    # Load ontology
    manager = OntologyManager()
    ontology_name = ontology.get("name", "test_ontology")
    
    # Save temporary ontology file
    import tempfile
    import yaml
    with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as f:
        yaml.dump(ontology, f)
        temp_path = f.name
    
    try:
        # Load and activate ontology
        manager.load_ontology(temp_path)
        manager.activate_domain(ontology_name)
        
        # Process sample documents
        for doc_path in samples_dir.glob("**/*"):
            if doc_path.is_file():
                results["total_documents"] += 1
                try:
                    # Parse document
                    parser = get_parser_for_file(str(doc_path))
                    with open(doc_path, 'rb') as f:
                        parsed = parser.parse({
                            "id": str(doc_path),
                            "binary_path": str(doc_path),
                            "content": f.read()
                        })
                    
                    # Extract entities (simplified validation)
                    elements = parsed.get("elements", [])
                    for mapping in ontology.get("element_entity_mappings", []):
                        # Check if mapping would match
                        matched = False
                        for element in elements:
                            if element.get("element_type") == mapping.get("element_type"):
                                matched = True
                                results["entities_found"].append({
                                    "entity_type": mapping.get("entity_type"),
                                    "document": doc_path.name
                                })
                                break
                        
                        if matched:
                            results["successful_extractions"] += 1
                            
                except Exception as e:
                    results["errors"].append({
                        "document": doc_path.name,
                        "error": str(e)
                    })
    finally:
        # Clean up temp file
        Path(temp_path).unlink()
    
    return results


def print_validation_results(results: Dict[str, Any]):
    """Print validation results in a readable format."""
    print("\n📊 Validation Results:")
    print(f"  Documents processed: {results['total_documents']}")
    print(f"  Successful extractions: {results['successful_extractions']}")
    
    if results['entities_found']:
        print(f"\n  ✅ Entities found:")
        entity_types = {}
        for entity in results['entities_found']:
            entity_type = entity['entity_type']
            entity_types[entity_type] = entity_types.get(entity_type, 0) + 1
        for entity_type, count in entity_types.items():
            print(f"    - {entity_type}: {count}")
    
    if results['errors']:
        print(f"\n  ⚠️  Errors encountered:")
        for error in results['errors'][:5]:  # Show first 5 errors
            print(f"    - {error['document']}: {error['error']}")
        if len(results['errors']) > 5:
            print(f"    ... and {len(results['errors']) - 5} more")


if __name__ == "__main__":
    sys.exit(main())