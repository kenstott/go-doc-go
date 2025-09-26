#!/usr/bin/env python3
"""
Command-line interface for generating domain ontologies through LLM-guided interviews.
"""

import logging
import os
import sys
import click
from pathlib import Path
from typing import Optional, Dict, Any

# Package imports (no path manipulation needed for proper package installation)

from go_doc_go.llm.chat import ChatProvider, create_chat_provider
from go_doc_go.cli.ontology_interview import OntologyInterviewer
from go_doc_go.domain.ontology_builder import OntologyBuilder


@click.command()
@click.option("--output", "-o", default="ontology.yaml", help="Output file path for generated ontology (default: ontology.yaml)")
@click.option("--llm-provider", type=click.Choice(["openai", "anthropic", "ollama", "auto"]), default="auto", help="LLM provider to use for interview (default: auto-detect)")
@click.option("--model", help="Specific model to use (e.g., gpt-4, claude-3-opus, llama2)")
@click.option("--template", "-t", type=click.Choice(["financial", "legal", "medical", "technical", "none"]), default="none", help="Base template to start from (default: none)")
@click.option("--validate-with", help="Directory containing sample documents for validation")
@click.option("--non-interactive", is_flag=True, help="Run in non-interactive mode (requires --config)")
@click.option("--config", help="Configuration file (for non-interactive mode or to load data sources)")
@click.option("--data-config", help="Go-Doc-Go config file with data sources to analyze")
@click.option("--log-level", "-l", type=click.Choice(["DEBUG", "INFO", "WARNING", "ERROR"]), default="INFO", help="Logging level (default: INFO)")
@click.option("--format", "-f", "output_format", type=click.Choice(["yaml", "json"]), default="yaml", help="Output format (default: yaml)")
@click.option("--max-iterations", type=int, default=20, help="Maximum interview iterations (default: 20)")
@click.option("--dry-run", is_flag=True, help="Preview ontology without saving")
def main(output, llm_provider, model, template, validate_with, non_interactive, config, data_config, log_level, output_format, max_iterations, dry_run):
    """Go-Doc-Go Ontology Generator - Interactive domain ontology creation.

    Generate domain ontologies through LLM-guided interviews.

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
    
    # Configure logging
    logging.basicConfig(
        level=getattr(logging, log_level),
        format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
    )
    logger = logging.getLogger(__name__)

    # Validate arguments
    if non_interactive and not config:
        click.echo("Error: --config is required when using --non-interactive")
        sys.exit(1)

    try:
        # Initialize LLM provider
        logger.info(f"Initializing LLM provider: {llm_provider}")
        chat_provider = create_chat_provider(
            provider=llm_provider,
            model=model
        )

        # Load template if specified
        template_obj = None
        if template != "none":
            logger.info(f"Loading template: {template}")
            template_obj = load_template(template)

        # Create ontology builder
        builder = OntologyBuilder(template=template_obj)

        if non_interactive:
            # Non-interactive mode
            logger.info("Running in non-interactive mode")
            with open(config, 'r') as f:
                import yaml
                config_data = yaml.safe_load(f)

            ontology = builder.build_from_config(config_data)
        else:
            # Interactive interview mode
            logger.info("Starting interactive ontology interview")
            click.echo("\n🎯 Welcome to the Go-Doc-Go Ontology Generator!")
            click.echo("=" * 60)
            click.echo("I'll help you create a domain ontology for document analysis.")
            click.echo("\n📋 How this works:")
            click.echo("  1. You provide your domain (e.g., 'financial', 'legal', 'medical')")
            click.echo("  2. AI suggests document types, terms, entities, and relationships")
            click.echo("  3. You can accept, modify, or replace any suggestions")
            click.echo("  4. The result is a complete ontology for document extraction")
            click.echo("\n💡 Tip: Press Enter to accept AI suggestions, or type your own.")
            click.echo("=" * 60)

            interviewer = OntologyInterviewer(
                chat_provider=chat_provider,
                builder=builder,
                max_iterations=max_iterations,
                data_config_path=data_config or config
            )

            # Run the interview
            ontology = interviewer.conduct_interview()

            click.echo("\n✨ Interview complete!")

        # Validate if requested
        if validate_with:
            logger.info(f"Validating ontology with documents in: {validate_with}")
            click.echo(f"\n🔍 Validating ontology with sample documents...")
            validation_results = validate_ontology(
                ontology,
                Path(validate_with)
            )
            print_validation_results(validation_results)

        # Preview or save
        if dry_run:
            click.echo("\n📋 Generated Ontology Preview:")
            click.echo("=" * 60)
            if output_format == "yaml":
                click.echo(builder.to_yaml(ontology))
            else:
                import json
                click.echo(json.dumps(builder.to_dict(ontology), indent=2))
        else:
            # Save to file
            output_path = Path(output)
            logger.info(f"Saving ontology to: {output_path}")

            if output_format == "yaml":
                with open(output_path, 'w') as f:
                    f.write(builder.to_yaml(ontology))
            else:
                import json
                with open(output_path, 'w') as f:
                    json.dump(builder.to_dict(ontology), f, indent=2)

            click.echo(f"\n✅ Ontology saved to: {output_path}")
            click.echo(f"📊 Summary:")
            click.echo(f"  - Terms: {len(ontology.get('terms', []))}")
            click.echo(f"  - Element Mappings: {len(ontology.get('element_entity_mappings', []))}")
            click.echo(f"  - Entity Relationships: {len(ontology.get('entity_relationship_rules', []))}")
            click.echo(f"  - Derived Entities: {len(ontology.get('derived_entities', []))}")

    except KeyboardInterrupt:
        logger.info("Interview cancelled by user")
        click.echo("\n\n👋 Interview cancelled. Goodbye!")
        sys.exit(1)

    except Exception as e:
        logger.error(f"Ontology generation failed: {str(e)}")
        logger.debug("Exception details:", exc_info=True)
        click.echo(f"\n❌ Error: {str(e)}")
        sys.exit(1)


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
    main()