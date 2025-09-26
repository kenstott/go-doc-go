#!/usr/bin/env python3
"""
CLI for generating ontology rules using LLM analysis of analytics database content.
"""

import json
import logging
import os
import sys
import click
from pathlib import Path
from typing import Dict, Any, List, Optional
from datetime import datetime

from go_doc_go.config import Config
from go_doc_go.storage_adapters.factory import StorageFactory
from go_doc_go.llm.chat import create_chat_provider
from go_doc_go.mcp.ontology_sampler import OntologySampler, OntologyContext, SamplingConfig
from go_doc_go.domain.ontology_builder import OntologyBuilder


logger = logging.getLogger(__name__)


class AnalyticsOntologyGenerator:
    """Generate ontology rules from analytics database using LLM analysis."""

    def __init__(self, config: Config, chat_provider, analytics_storage):
        self.config = config
        self.chat_provider = chat_provider
        self.analytics_storage = analytics_storage
        self.builder = OntologyBuilder()

    def generate_from_analytics(self, domain_context: OntologyContext,
                               sampling_config: SamplingConfig) -> Dict[str, Any]:
        """
        Generate ontology rules by analyzing analytics database content with LLM.

        Args:
            domain_context: Domain-specific context for targeted analysis
            sampling_config: Configuration for data sampling

        Returns:
            Generated ontology dictionary
        """
        logger.info(f"Generating ontology for domain '{domain_context.domain_name}' from analytics data")

        # Sample data from analytics database
        sampler = OntologySampler(self.analytics_storage, sampling_config)
        sampling_data = sampler.sample_for_domain(domain_context)

        # Generate ontology using LLM analysis
        ontology = self._generate_ontology_with_llm(domain_context, sampling_data)

        return ontology

    def _generate_ontology_with_llm(self, domain_context: OntologyContext,
                                   sampling_data: Dict[str, Any]) -> Dict[str, Any]:
        """Use LLM to analyze sampled data and generate ontology rules."""

        # Prepare data summary for LLM
        data_summary = self._prepare_data_summary(sampling_data)

        # Create LLM prompt for ontology generation
        prompt = self._create_ontology_prompt(domain_context, data_summary)

        logger.info("Sending analytics data to LLM for ontology generation...")

        try:
            # Get LLM response
            response = self.chat_provider.chat([{
                "role": "user",
                "content": prompt
            }])

            # Parse LLM response into ontology structure
            ontology_dict = self._parse_llm_response(response.content)

            # Validate and enhance the ontology
            ontology_dict = self._validate_and_enhance_ontology(ontology_dict, sampling_data)

            return ontology_dict

        except Exception as e:
            logger.error(f"Failed to generate ontology with LLM: {e}")
            # Fallback to basic ontology structure
            return self._create_fallback_ontology(domain_context, sampling_data)

    def _prepare_data_summary(self, sampling_data: Dict[str, Any]) -> Dict[str, Any]:
        """Prepare a concise summary of sampled data for LLM analysis."""
        summary = {
            "corpus_statistics": sampling_data.get("corpus_statistics", {}),
            "sampling_summary": sampling_data.get("sampling_summary", {}),
            "pattern_analysis": sampling_data.get("pattern_analysis", {}),
            "metadata_analysis": sampling_data.get("metadata_analysis", {}),
            "sample_elements": []
        }

        # Include sample elements (limited for context window)
        sampled_elements = sampling_data.get("sampled_elements", [])
        sample_size = min(50, len(sampled_elements))

        for element in sampled_elements[:sample_size]:
            summary["sample_elements"].append({
                "element_type": element.get("element_type"),
                "structural_name": element.get("structural_name"),
                "content_preview": element.get("content_preview", "")[:200],  # Limit content
                "metadata": element.get("metadata", {}),
                "hierarchy_depth": element.get("hierarchy_depth"),
                "structural_path": element.get("structural_path")
            })

        return summary

    def _create_ontology_prompt(self, domain_context: OntologyContext,
                               data_summary: Dict[str, Any]) -> str:
        """Create LLM prompt for ontology generation."""

        prompt = f"""You are an expert ontology engineer. Generate a domain ontology for "{domain_context.domain_name}" based on analysis of document corpus data.

DOMAIN CONTEXT:
- Domain: {domain_context.domain_name}
- Keywords: {', '.join(domain_context.keywords)}
- Document Types: {', '.join(domain_context.document_types)}
- Entity Hints: {', '.join(domain_context.entity_hints)}
- Relationship Hints: {', '.join(domain_context.relationship_hints)}

CORPUS ANALYSIS DATA:
{json.dumps(data_summary, indent=2, default=str)}

TASK: Generate a comprehensive ontology with:

1. TERMS: Domain-specific entities and concepts
2. ELEMENT MAPPINGS: Rules to extract entities from document elements
3. RELATIONSHIP RULES: Rules to discover relationships between entities
4. DERIVED ENTITIES: Rules to extract structured entities from metadata
5. ENTITY RELATIONSHIPS: Rules for connecting domain entities

REQUIREMENTS:
- Focus on the specific domain: {domain_context.domain_name}
- Use the actual element types and structural patterns from the data
- Create semantic matching rules using meaningful phrases
- Include regex patterns for structured data extraction
- Define relationship rules that leverage document hierarchy
- Ensure extracted entities will be valuable for analysis

OUTPUT FORMAT: Return a valid YAML ontology following this structure:

```yaml
domain:
  name: "{domain_context.domain_name}"
  version: "1.0.0"
  description: "Domain ontology for {domain_context.domain_name}"
  settings:
    default_confidence_threshold: 0.70
    max_relationships_per_pair: 3

terms:
  - id: "entity_id"
    label: "Entity Label"
    description: "Description of the entity"
    aliases: ["alias1", "alias2"]

element_mappings:
  - term_id: "entity_id"
    rules:
      - type: "semantic"
        semantic_phrase: "meaningful phrase to match"
        confidence_threshold: 0.75
        element_types: ["specific_element_type"]
      - type: "regex"
        pattern: "regex_pattern"
        element_types: ["element_type"]
      - type: "keywords"
        keywords: ["keyword1", "keyword2"]
        element_types: ["element_type"]

relationship_rules:
  - id: "relationship_rule_id"
    relationship_type: "RELATIONSHIP_TYPE"
    description: "Description of relationship"
    source:
      term_id: "source_entity"
      semantic_phrase: "phrase for source"
      confidence_threshold: 0.70
    target:
      term_id: "target_entity"
      semantic_phrase: "phrase for target"
      confidence_threshold: 0.70
    constraints:
      hierarchy_level: -1  # Same document
      direction: "any"
    confidence:
      minimum: 0.70
      calculation: "average"

derived_entity_rules:
  - entity_type: "DERIVED_ENTITY"
    source_element_types: ["source_type"]
    metadata_fields: ["field1", "field2"]
    id_template: "{{entity_type}}_{{name}}"
    deduplication_key: "name"

entity_relationship_rules:
  - name: "entity_relationship_name"
    description: "Description"
    source_entity_type: "SOURCE_ENTITY"
    target_entity_type: "TARGET_ENTITY"
    relationship_type: "CONNECTS_TO"
    matching_criteria:
      same_document: true
    confidence: 0.80
```

Generate the ontology now:"""

        return prompt

    def _parse_llm_response(self, response_content: str) -> Dict[str, Any]:
        """Parse LLM response to extract ontology structure."""
        try:
            # Extract YAML content from response
            lines = response_content.split('\n')
            yaml_lines = []
            in_yaml = False

            for line in lines:
                if line.strip().startswith('```yaml'):
                    in_yaml = True
                    continue
                elif line.strip().startswith('```') and in_yaml:
                    break
                elif in_yaml:
                    yaml_lines.append(line)

            if yaml_lines:
                yaml_content = '\n'.join(yaml_lines)
                import yaml
                return yaml.safe_load(yaml_content)
            else:
                # Try to parse entire response as YAML
                import yaml
                return yaml.safe_load(response_content)

        except Exception as e:
            logger.error(f"Failed to parse LLM response as YAML: {e}")
            logger.debug(f"Response content: {response_content}")
            raise ValueError(f"Invalid YAML in LLM response: {e}")

    def _validate_and_enhance_ontology(self, ontology_dict: Dict[str, Any],
                                      sampling_data: Dict[str, Any]) -> Dict[str, Any]:
        """Validate and enhance the generated ontology."""

        # Ensure required structure exists
        if 'domain' not in ontology_dict:
            ontology_dict['domain'] = {}
        if 'terms' not in ontology_dict:
            ontology_dict['terms'] = []
        if 'element_mappings' not in ontology_dict:
            ontology_dict['element_mappings'] = []
        if 'relationship_rules' not in ontology_dict:
            ontology_dict['relationship_rules'] = []
        if 'derived_entity_rules' not in ontology_dict:
            ontology_dict['derived_entity_rules'] = []
        if 'entity_relationship_rules' not in ontology_dict:
            ontology_dict['entity_relationship_rules'] = []

        # Validate against actual data
        available_element_types = set()
        for element in sampling_data.get("sampled_elements", []):
            if element.get("element_type"):
                available_element_types.add(element["element_type"])

        # Update element mappings to use valid element types
        for mapping in ontology_dict['element_mappings']:
            for rule in mapping.get('rules', []):
                if 'element_types' in rule:
                    # Filter to only include element types that exist in the data
                    valid_types = [et for et in rule['element_types']
                                 if et in available_element_types or et == "*"]
                    if valid_types:
                        rule['element_types'] = valid_types
                    else:
                        # Default to all types if none are valid
                        rule['element_types'] = ["*"]

        return ontology_dict

    def _create_fallback_ontology(self, domain_context: OntologyContext,
                                 sampling_data: Dict[str, Any]) -> Dict[str, Any]:
        """Create a basic fallback ontology when LLM generation fails."""

        # Get common element types from sampling data
        element_types = []
        for element in sampling_data.get("sampled_elements", [])[:10]:
            if element.get("element_type") and element["element_type"] not in element_types:
                element_types.append(element["element_type"])

        fallback_ontology = {
            "domain": {
                "name": domain_context.domain_name,
                "version": "1.0.0",
                "description": f"Basic ontology for {domain_context.domain_name}",
                "settings": {
                    "default_confidence_threshold": 0.70,
                    "max_relationships_per_pair": 3
                }
            },
            "terms": [
                {
                    "id": "document",
                    "label": "Document",
                    "description": "A document in the corpus",
                    "aliases": ["doc", "file"]
                },
                {
                    "id": "entity",
                    "label": "Entity",
                    "description": "A named entity mentioned in documents",
                    "aliases": []
                }
            ],
            "element_mappings": [
                {
                    "term_id": "entity",
                    "rules": [
                        {
                            "type": "keywords",
                            "keywords": domain_context.keywords[:5] if domain_context.keywords else ["entity"],
                            "element_types": element_types[:3] if element_types else ["*"]
                        }
                    ]
                }
            ],
            "relationship_rules": [],
            "derived_entity_rules": [],
            "entity_relationship_rules": []
        }

        logger.warning("Using fallback ontology due to LLM generation failure")
        return fallback_ontology


@click.command()
@click.option("--config", "-c", required=True, help="Path to Go-Doc-Go configuration file")
@click.option("--domain", "-d", required=True, help="Domain name (e.g., 'financial_reports', 'legal_contracts')")
@click.option("--keywords", "-k", help="Comma-separated domain keywords")
@click.option("--document-types", help="Comma-separated document types to focus on")
@click.option("--entity-hints", help="Comma-separated entity types to look for")
@click.option("--relationship-hints", help="Comma-separated relationship types to consider")
@click.option("--output", "-o", default="generated_ontology.yaml", help="Output file for generated ontology")
@click.option("--max-elements", type=int, default=200, help="Maximum elements to sample from analytics")
@click.option("--llm-provider", type=click.Choice(["openai", "anthropic", "ollama", "auto"]),
              default="auto", help="LLM provider for analysis")
@click.option("--model", help="Specific model to use")
@click.option("--dry-run", is_flag=True, help="Preview without saving")
@click.option("--log-level", type=click.Choice(["DEBUG", "INFO", "WARNING", "ERROR"]),
              default="INFO", help="Logging level")
def main(config, domain, keywords, document_types, entity_hints, relationship_hints,
         output, max_elements, llm_provider, model, dry_run, log_level):
    """Generate ontology rules using LLM analysis of analytics database.

    This command analyzes your processed document corpus to automatically
    generate domain-specific ontology rules for entity and relationship extraction.

    Examples:
      # Generate ontology for financial domain
      python -m go_doc_go.cli.ontology_analytics \\
        --config config.yaml \\
        --domain financial_reports \\
        --keywords "revenue,profit,expenses,assets" \\
        --entity-hints "company,amount,date,person"

      # Generate for legal contracts
      python -m go_doc_go.cli.ontology_analytics \\
        --config config.yaml \\
        --domain legal_contracts \\
        --document-types "contract,agreement" \\
        --keywords "party,obligation,term,clause"
    """

    # Configure logging
    logging.basicConfig(
        level=getattr(logging, log_level),
        format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
    )

    try:
        # Load configuration
        if not os.path.exists(config):
            click.echo(f"❌ Configuration file not found: {config}")
            sys.exit(1)

        config_obj = Config(config)

        # Check analytics is enabled
        if not config_obj.is_analytics_enabled():
            click.echo("❌ Analytics must be enabled to generate ontology from data")
            sys.exit(1)

        # Get analytics storage
        analytics_outputs = config_obj.get_analytics_outputs()
        if not analytics_outputs:
            click.echo("❌ No analytics outputs configured")
            sys.exit(1)

        analytics_storage = StorageFactory.create_analytics_storage(analytics_outputs[0])

        # Initialize LLM provider
        chat_provider = create_chat_provider(provider=llm_provider, model=model)

        # Create domain context
        domain_context = OntologyContext(
            domain_name=domain,
            keywords=[k.strip() for k in (keywords or "").split(",") if k.strip()],
            document_types=[d.strip() for d in (document_types or "").split(",") if d.strip()],
            entity_hints=[e.strip() for e in (entity_hints or "").split(",") if e.strip()],
            relationship_hints=[r.strip() for r in (relationship_hints or "").split(",") if r.strip()]
        )

        # Create sampling configuration
        sampling_config = SamplingConfig(max_elements=max_elements)

        # Generate ontology
        click.echo(f"🔍 Analyzing analytics database for domain: {domain}")
        click.echo(f"📊 Sampling up to {max_elements} elements...")

        generator = AnalyticsOntologyGenerator(config_obj, chat_provider, analytics_storage)
        ontology_dict = generator.generate_from_analytics(domain_context, sampling_config)

        # Output results
        if dry_run:
            click.echo("\n📋 Generated Ontology Preview:")
            click.echo("=" * 60)
            import yaml
            click.echo(yaml.dump(ontology_dict, default_flow_style=False))
        else:
            # Save to file
            output_path = Path(output)
            with open(output_path, 'w') as f:
                import yaml
                yaml.dump(ontology_dict, f, default_flow_style=False)

            click.echo(f"\n✅ Ontology saved to: {output_path}")

            # Show summary
            terms_count = len(ontology_dict.get('terms', []))
            mappings_count = len(ontology_dict.get('element_mappings', []))
            rel_rules_count = len(ontology_dict.get('relationship_rules', []))
            derived_count = len(ontology_dict.get('derived_entity_rules', []))
            entity_rel_count = len(ontology_dict.get('entity_relationship_rules', []))

            click.echo(f"📊 Generated Components:")
            click.echo(f"  - Terms: {terms_count}")
            click.echo(f"  - Element Mappings: {mappings_count}")
            click.echo(f"  - Relationship Rules: {rel_rules_count}")
            click.echo(f"  - Derived Entity Rules: {derived_count}")
            click.echo(f"  - Entity Relationship Rules: {entity_rel_count}")

    except KeyboardInterrupt:
        click.echo("\n\n👋 Generation cancelled. Goodbye!")
        sys.exit(1)
    except Exception as e:
        logger.error(f"Ontology generation failed: {e}")
        click.echo(f"\n❌ Error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()