#!/usr/bin/env python3
"""
Advanced ontology discovery system using multi-step LLM orchestration with analytics database.

This module implements a sophisticated discovery process that:
1. Profiles the analytics database to understand content patterns
2. Uses multiple LLM interactions to progressively refine understanding
3. Validates discoveries against actual data
4. Generates production-ready ontology configurations
"""

import json
import logging
import sys
from typing import Dict, Any, List, Optional, Tuple
from dataclasses import dataclass, field
from pathlib import Path
import click

try:
    import duckdb
    import pandas as pd
    DUCKDB_AVAILABLE = True
except ImportError:
    DUCKDB_AVAILABLE = False
    import pandas as pd  # Try pandas separately
    duckdb = None

from go_doc_go.config import Config
from go_doc_go.storage_adapters.factory import StorageFactory
from go_doc_go.llm.chat import create_chat_provider
from go_doc_go.domain.ontology_builder import OntologyBuilder

logger = logging.getLogger(__name__)


@dataclass
class DiscoveryStep:
    """Represents a single step in the discovery process."""
    name: str
    description: str
    query: Optional[str] = None  # SQL/DuckDB query for data gathering
    prompt_template: str = ""
    max_samples: int = 100
    requires: List[str] = field(default_factory=list)  # Dependencies on other steps


@dataclass
class DiscoveryContext:
    """Maintains state across discovery steps."""
    domain_name: str
    analytics_path: str
    discovered_patterns: Dict[str, Any] = field(default_factory=dict)
    discovered_terms: List[Dict[str, Any]] = field(default_factory=list)
    discovered_entities: List[Dict[str, Any]] = field(default_factory=list)
    discovered_relationships: List[Dict[str, Any]] = field(default_factory=list)
    validation_results: Dict[str, Any] = field(default_factory=dict)
    step_history: List[Dict[str, Any]] = field(default_factory=list)


class OntologyDiscoveryOrchestrator:
    """
    Orchestrates multi-step ontology discovery process using LLM and analytics data.

    This orchestrator implements a sophisticated discovery pipeline:
    1. Initial profiling - Understand corpus structure
    2. Pattern discovery - Identify recurring patterns
    3. Term extraction - Extract domain terminology
    4. Entity mapping - Map terms to extraction rules
    5. Relationship discovery - Find connections between entities
    6. Validation - Test against real data
    7. Refinement - Iterate based on validation results
    """

    def __init__(self, config: Config, analytics_path: str, llm_provider=None):
        self.config = config
        self.analytics_path = Path(analytics_path)
        self.llm_provider = llm_provider
        self.builder = OntologyBuilder()

        if not DUCKDB_AVAILABLE:
            raise ImportError("DuckDB required: pip install duckdb pandas")

        # Define discovery pipeline steps
        self.discovery_steps = self._define_discovery_steps()

    def _define_discovery_steps(self) -> List[DiscoveryStep]:
        """Define the multi-step discovery pipeline."""
        return [
            # Step 1: Initial Profiling
            DiscoveryStep(
                name="profile_corpus",
                description="Profile the document corpus to understand structure and content",
                query="""
                    SELECT
                        COUNT(DISTINCT doc_id) as total_documents,
                        COUNT(*) as total_elements,
                        element_type,
                        COUNT(*) as element_count,
                        COUNT(DISTINCT doc_id) as docs_with_element
                    FROM read_parquet('{analytics_path}/elements/**/*.parquet')
                    GROUP BY element_type
                    ORDER BY element_count DESC
                """,
                prompt_template="""
                    Analyze this document corpus profile:
                    {data}

                    Identify:
                    1. Primary document structure (what element types dominate)
                    2. Likely document domain based on element patterns
                    3. Suggested areas for deeper analysis

                    Return as JSON with keys: domain_assessment, structure_insights, next_steps
                """,
                max_samples=50
            ),

            # Step 2: Content Pattern Discovery
            DiscoveryStep(
                name="discover_patterns",
                description="Discover frequent content patterns and terminology",
                query="""
                    WITH content_samples AS (
                        SELECT
                            element_type,
                            content_preview,
                            COUNT(*) as frequency
                        FROM read_parquet('{analytics_path}/elements/**/*.parquet')
                        WHERE content_preview IS NOT NULL
                        AND LENGTH(content_preview) > 5
                        GROUP BY element_type, content_preview
                        HAVING COUNT(*) > 2
                        ORDER BY frequency DESC
                        LIMIT {max_samples}
                    )
                    SELECT * FROM content_samples
                """,
                prompt_template="""
                    Analyze these frequent content patterns:
                    {data}

                    Identify:
                    1. Domain-specific terminology
                    2. Key entities (companies, people, products, etc.)
                    3. Numerical patterns (financial figures, dates, etc.)
                    4. Document sections or headers

                    Return as JSON with keys: terminology, entities, patterns, sections
                """,
                max_samples=200,
                requires=["profile_corpus"]
            ),

            # Step 3: Term Extraction and Definition
            DiscoveryStep(
                name="extract_terms",
                description="Extract and define domain terms",
                query=None,  # No query needed - work from previous discoveries
                prompt_template="""
                    Based on these domain-specific content samples:
                    {data}

                    Previous patterns discovered: {discovered_patterns}

                    Define formal ontology terms with:
                    1. id (snake_case identifier)
                    2. label (human readable name)
                    3. description (what it represents)
                    4. aliases (alternative names)

                    Focus on: {focus_area}

                    Return as JSON array of term definitions.
                """,
                max_samples=100,
                requires=["discover_patterns"]
            ),

            # Step 4: Entity Extraction Rules
            DiscoveryStep(
                name="create_extraction_rules",
                description="Create extraction rules for identified entities",
                query=None,  # Will be built dynamically based on discovered terms
                prompt_template="""
                    For the term: {term_definition}

                    Based on these actual occurrences in the data:
                    {data}

                    Create extraction rules:
                    1. regex patterns that would match these instances
                    2. semantic phrases for similarity matching
                    3. confidence thresholds
                    4. element type filters

                    Return as JSON with structure matching ElementMapping format.
                """,
                max_samples=50,
                requires=["extract_terms"]
            ),

            # Step 5: Relationship Discovery
            DiscoveryStep(
                name="discover_relationships",
                description="Discover relationships between entities",
                query=None,  # Will be built dynamically based on discovered entities
                prompt_template="""
                    Analyze these entity co-occurrences:
                    {data}

                    Known entities: {discovered_entities}

                    Identify relationships:
                    1. Relationship type (HAS, WORKS_FOR, REPORTS, etc.)
                    2. Directionality
                    3. Constraints (same document, same section, etc.)
                    4. Confidence levels

                    Return as JSON array of relationship rules.
                """,
                max_samples=100,
                requires=["create_extraction_rules"]
            ),

            # Step 6: Validation Against Real Data
            DiscoveryStep(
                name="validate_rules",
                description="Validate extraction rules against actual data",
                query=None,  # Will be built dynamically based on rules to validate
                prompt_template="""
                    Validation results for rule: {rule}

                    Matches: {matches}
                    False positives: {false_positives}
                    False negatives: {false_negatives}

                    Suggest refinements:
                    1. Pattern improvements
                    2. Confidence threshold adjustments
                    3. Additional filters needed

                    Return as JSON with refined rule definition.
                """,
                max_samples=200,
                requires=["create_extraction_rules", "discover_relationships"]
            ),

            # Step 7: Iterative Refinement
            DiscoveryStep(
                name="refine_ontology",
                description="Refine ontology based on validation results",
                prompt_template="""
                    Review the complete ontology:

                    Terms: {discovered_terms}
                    Extraction Rules: {discovered_entities}
                    Relationships: {discovered_relationships}
                    Validation Results: {validation_results}

                    Provide final refinements:
                    1. Missing terms that should be added
                    2. Rules that need adjustment
                    3. Additional relationships to consider
                    4. Overall confidence assessment

                    Return as JSON with final ontology structure.
                """,
                requires=["validate_rules"]
            )
        ]

    def discover_ontology(self, domain_name: str,
                         keywords: Optional[List[str]] = None,
                         max_iterations: int = 2) -> Dict[str, Any]:
        """
        Execute the complete discovery pipeline.

        Args:
            domain_name: Name for the domain being analyzed
            keywords: Optional keywords to guide discovery
            max_iterations: Maximum refinement iterations

        Returns:
            Complete ontology configuration
        """
        context = DiscoveryContext(
            domain_name=domain_name,
            analytics_path=str(self.analytics_path)
        )

        logger.info(f"Starting ontology discovery for domain: {domain_name}")

        # Execute discovery pipeline
        for step in self.discovery_steps:
            logger.info(f"Executing step: {step.name}")
            self._execute_discovery_step(step, context)

        # Perform iterative refinement
        for i in range(max_iterations):
            logger.info(f"Refinement iteration {i+1}/{max_iterations}")
            if not self._needs_refinement(context):
                break
            self._refine_ontology(context)

        # Build final ontology
        return self._build_final_ontology(context)

    def _execute_discovery_step(self, step: DiscoveryStep, context: DiscoveryContext):
        """Execute a single discovery step."""
        # Check dependencies
        for dep in step.requires:
            if dep not in [s['name'] for s in context.step_history]:
                logger.warning(f"Skipping {step.name}: dependency {dep} not met")
                return

        # Gather data if query provided
        data = None
        if step.query:
            data = self._execute_analytics_query(
                step.query.format(
                    analytics_path=context.analytics_path,
                    max_samples=step.max_samples
                ),
                context
            )

        # Prepare prompt with context
        prompt = self._prepare_step_prompt(step, context, data)

        # Get LLM response
        response = self._query_llm(prompt)

        # Process and store results
        self._process_step_results(step, response, context)

        # Record step completion
        context.step_history.append({
            'name': step.name,
            'description': step.description,
            'response': response
        })

    def _execute_analytics_query(self, query: str, context: DiscoveryContext) -> pd.DataFrame:
        """Execute DuckDB query on analytics data."""
        conn = duckdb.connect(':memory:')
        try:
            result = conn.execute(query).df()
            return result
        except Exception as e:
            logger.error(f"Query failed: {e}")
            return pd.DataFrame()
        finally:
            conn.close()

    def _prepare_step_prompt(self, step: DiscoveryStep,
                            context: DiscoveryContext,
                            data: Optional[pd.DataFrame]) -> str:
        """Prepare prompt for LLM with step context."""
        prompt_data = {
            'data': data.to_dict('records') if data is not None else None,
            'discovered_patterns': context.discovered_patterns,
            'discovered_terms': context.discovered_terms,
            'discovered_entities': context.discovered_entities,
            'discovered_relationships': context.discovered_relationships,
            'validation_results': context.validation_results
        }

        # Add placeholder values for any missing template variables
        # This handles templates that might reference specific fields
        if data is not None and not data.empty:
            # Add common field names that might be in templates
            for col in data.columns:
                if col not in prompt_data:
                    prompt_data[col] = data[col].iloc[0] if len(data) > 0 else None

        # Safely format the template
        try:
            return step.prompt_template.format(**prompt_data)
        except KeyError as e:
            # If a key is missing, provide a simpler version
            logger.warning(f"Missing template key: {e}")
            # Provide just the data without the complex template
            if data is not None:
                return f"Analyze this data and identify patterns:\n{data.to_string()}\n\nReturn findings as JSON."
            else:
                return f"Based on context, provide ontology insights. Return as JSON."

    def _query_llm(self, prompt: str) -> Dict[str, Any]:
        """Query LLM and parse JSON response."""
        if not self.llm_provider:
            return {}

        try:
            response = self.llm_provider.chat_completion([{
                "role": "system",
                "content": "You are an expert ontology engineer. Always respond with valid JSON."
            }, {
                "role": "user",
                "content": prompt
            }])

            # Parse JSON response
            # The response might have explanatory text before/after JSON
            # Try to extract JSON from the response
            import re
            json_match = re.search(r'\{[^{}]*(?:\{[^{}]*\}[^{}]*)*\}', response, re.DOTALL)
            if json_match:
                try:
                    return json.loads(json_match.group())
                except json.JSONDecodeError:
                    pass

            # Try parsing as-is
            return json.loads(response)
        except json.JSONDecodeError as e:
            logger.error(f"Failed to parse LLM JSON response: {e}")
            logger.info(f"Response was: {response[:500]}..." if len(response) > 500 else f"Response was: {response}")
            return {}
        except Exception as e:
            logger.error(f"LLM query failed: {e}")
            return {}

    def _process_step_results(self, step: DiscoveryStep,
                             response: Dict[str, Any],
                             context: DiscoveryContext):
        """Process and store step results in context."""
        if step.name == "profile_corpus":
            context.discovered_patterns['corpus_profile'] = response
        elif step.name == "discover_patterns":
            context.discovered_patterns.update(response)
        elif step.name == "extract_terms":
            context.discovered_terms.extend(response)
        elif step.name == "create_extraction_rules":
            context.discovered_entities.extend(response)
        elif step.name == "discover_relationships":
            context.discovered_relationships.extend(response)
        elif step.name == "validate_rules":
            context.validation_results.update(response)

    def _needs_refinement(self, context: DiscoveryContext) -> bool:
        """Check if ontology needs further refinement."""
        if not context.validation_results:
            return False

        # Check validation metrics
        metrics = context.validation_results.get('metrics', {})
        precision = metrics.get('precision', 0)
        recall = metrics.get('recall', 0)

        return precision < 0.8 or recall < 0.7

    def _refine_ontology(self, context: DiscoveryContext):
        """Perform iterative refinement based on validation."""
        # Re-run validation with refined rules
        self._execute_discovery_step(
            self.discovery_steps[-2],  # validate_rules step
            context
        )

        # Apply refinements
        self._execute_discovery_step(
            self.discovery_steps[-1],  # refine_ontology step
            context
        )

    def _build_final_ontology(self, context: DiscoveryContext) -> Dict[str, Any]:
        """Build the final ontology structure."""
        return {
            'domain': {
                'name': context.domain_name,
                'version': '1.0.0',
                'description': f'Ontology for {context.domain_name} discovered from analytics',
                'settings': {
                    'default_confidence_threshold': 0.70,
                    'max_relationships_per_pair': 5,
                    'enable_transitive_inference': True
                }
            },
            'terms': context.discovered_terms,
            'element_mappings': context.discovered_entities,
            'relationship_rules': context.discovered_relationships,
            'metadata': {
                'discovery_steps': len(context.step_history),
                'validation_metrics': context.validation_results.get('metrics', {}),
                'corpus_statistics': context.discovered_patterns.get('corpus_profile', {})
            }
        }


@click.command()
@click.option('-c', '--config', required=True, help='Path to Go-Doc-Go configuration file')
@click.option('-d', '--domain', required=True, help='Domain name for the ontology')
@click.option('-o', '--output', help='Output file for generated ontology')
@click.option('--keywords', help='Comma-separated keywords to guide discovery')
@click.option('--max-iterations', default=2, help='Maximum refinement iterations')
@click.option('--llm-provider', type=click.Choice(['openai', 'anthropic', 'ollama', 'auto']),
              default='auto', help='LLM provider for analysis')
@click.option('--dry-run', is_flag=True, help='Preview without saving')
@click.option('--log-level', type=click.Choice(['DEBUG', 'INFO', 'WARNING', 'ERROR']),
              default='INFO', help='Logging level')
def main(config, domain, output, keywords, max_iterations, llm_provider, dry_run, log_level):
    """
    Advanced ontology discovery using multi-step LLM orchestration.

    This command performs sophisticated ontology discovery by:
    1. Profiling your analytics database
    2. Discovering patterns through multiple LLM interactions
    3. Validating discoveries against real data
    4. Iteratively refining the ontology

    Example:
        python -m go_doc_go.cli.ontology_discovery \\
            --config config.yaml \\
            --domain financial_reports \\
            --keywords "revenue,profit,company" \\
            --output ontology.yaml
    """
    # Configure logging
    logging.basicConfig(level=getattr(logging, log_level))

    # Load configuration
    config_obj = Config(config)

    # Check analytics
    if not config_obj.is_analytics_enabled():
        click.echo("❌ Analytics must be enabled for ontology discovery")
        sys.exit(1)

    # Get analytics path
    analytics_outputs = config_obj.get_analytics_outputs()
    if not analytics_outputs:
        click.echo("❌ No analytics outputs configured")
        sys.exit(1)

    analytics_path = analytics_outputs[0].get('path', analytics_outputs[0].get('base_path'))

    # Create LLM provider
    llm = None
    if llm_provider != 'none':
        try:
            llm = create_chat_provider(llm_provider)
        except Exception as e:
            logger.warning(f"Could not create LLM provider: {e}")

    # Create orchestrator
    orchestrator = OntologyDiscoveryOrchestrator(
        config=config_obj,
        analytics_path=analytics_path,
        llm_provider=llm
    )

    # Parse keywords
    keyword_list = keywords.split(',') if keywords else []

    # Execute discovery
    click.echo(f"🔍 Starting ontology discovery for domain: {domain}")
    click.echo(f"📊 Analyzing analytics at: {analytics_path}")

    ontology = orchestrator.discover_ontology(
        domain_name=domain,
        keywords=keyword_list,
        max_iterations=max_iterations
    )

    # Output results
    if dry_run:
        click.echo("\n📋 Generated Ontology (dry run):")
        click.echo(json.dumps(ontology, indent=2))
    else:
        output_path = output or f"{domain}_ontology.yaml"
        builder = OntologyBuilder()
        yaml_content = builder.to_yaml(ontology)

        with open(output_path, 'w') as f:
            f.write(yaml_content)

        click.echo(f"✅ Ontology saved to: {output_path}")

        # Display summary
        click.echo(f"\n📈 Discovery Summary:")
        click.echo(f"  - Terms: {len(ontology.get('terms', []))}")
        click.echo(f"  - Extraction mappings: {len(ontology.get('element_mappings', []))}")
        click.echo(f"  - Relationships: {len(ontology.get('relationship_rules', []))}")

        if 'metadata' in ontology:
            metrics = ontology['metadata'].get('validation_metrics', {})
            if metrics:
                click.echo(f"  - Validation precision: {metrics.get('precision', 'N/A')}")
                click.echo(f"  - Validation recall: {metrics.get('recall', 'N/A')}")


if __name__ == '__main__':
    main()