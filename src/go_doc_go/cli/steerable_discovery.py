#!/usr/bin/env python3
"""
CLI for Steerable Ontology Discovery with optional config delegation
"""

import click
import os
import sys
from pathlib import Path

# Add project root to path for discovery modules
project_root = Path(__file__).parent.parent.parent.parent
sys.path.insert(0, str(project_root))


@click.command()
@click.option('--analytics', '-a', required=True,
              help='Path to analytics parquet files')
@click.option('--output', '-o', default='discovered_ontology.yaml',
              help='Output ontology file')
@click.option('--config', '-c',
              help='Config file for delegated discovery (skips interactive mode)')
@click.option('--existing', '-e',
              help='Existing ontology to extend/refine')
@click.option('--mode', type=click.Choice(['interactive', 'automatic', 'steerable']),
              default='steerable',
              help='Discovery mode: interactive, automatic, or steerable (with checkpoints)')
def main(analytics, output, config, existing, mode):
    """
    Discover domain ontology from analytics data using intelligent discovery.

    Can run in three modes:
    - steerable: Provides checkpoints for human validation (default)
    - automatic: Fully automated discovery using LLM
    - interactive: Original interview-based discovery

    When --config is provided, delegates to intelligent discovery for automation.
    """

    try:
        if mode == 'steerable':
            from steerable_discovery_system import SteerableDiscoverySystem

            click.echo("🚀 Starting Steerable Discovery System...")

            discovery = SteerableDiscoverySystem(
                analytics_path=analytics,
                output_dir=os.path.dirname(output) or '.',
                existing_ontology_path=existing,
                config_path=config
            )

            # If config provided, delegate to intelligent discovery
            if config:
                click.echo(f"📋 Delegating to intelligent discovery with config: {config}")
                ontology_path = discovery.discover_with_steering(use_config_delegation=True)
            else:
                click.echo("🎯 Running steerable discovery with checkpoints...")
                ontology_path = discovery.discover_with_steering()

        elif mode == 'automatic':
            from intelligent_discovery_interview import IntelligentDiscoverySystem

            click.echo("🤖 Starting Automatic Discovery...")

            discovery = IntelligentDiscoverySystem(
                analytics_path=analytics,
                output_dir=os.path.dirname(output) or '.'
            )

            discovery.analyze_corpus()

            if discovery.client:
                discovery.discover_with_llm()

            ontology_path = discovery.generate_ontology()

        else:  # interactive
            from interactive_discovery_interview import InteractiveOntologyDiscovery

            click.echo("💬 Starting Interactive Discovery Interview...")

            discovery = InteractiveOntologyDiscovery(
                analytics_path=analytics,
                output_dir=os.path.dirname(output) or '.'
            )

            discovery.run_interview()
            ontology_path = discovery.generate_ontology()

        # Rename to requested output file if different
        if ontology_path and ontology_path != output:
            os.rename(ontology_path, output)
            ontology_path = output

        click.echo(f"✅ Ontology discovered: {ontology_path}")

    except ImportError as e:
        click.echo(f"❌ Required module not found: {e}", err=True)
        click.echo("Make sure all discovery modules are in the project root", err=True)
        sys.exit(1)

    except Exception as e:
        click.echo(f"❌ Discovery failed: {e}", err=True)
        import traceback
        if click.get_current_context().obj and click.get_current_context().obj.get('verbose'):
            traceback.print_exc()
        sys.exit(1)


if __name__ == '__main__':
    main()