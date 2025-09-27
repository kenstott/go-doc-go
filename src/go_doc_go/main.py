#!/usr/bin/env python3
"""
Go-Doc-Go Unified CLI Entry Point

This module provides a unified command-line interface using Click that properly integrates Click-based CLI tools.

Examples:
    python -m go_doc_go worker --config config.yaml --max-documents 100
    python -m go_doc_go status --live
    python -m go_doc_go analytics --detailed
    python -m go_doc_go search "machine learning" --limit 10

This replaces the old ingest_documents() function which used deprecated two-pass processing.
The current architecture uses a simple worker-based system with job control.
"""

import click


@click.group(invoke_without_command=True)
@click.pass_context
def cli(ctx):
    """Go-Doc-Go: Document processing and analysis system.

    A unified CLI for document ingestion, processing, and analysis.
    Uses a simple worker-based architecture with job control.
    """
    if ctx.invoked_subcommand is None:
        click.echo(ctx.get_help())


# Add the worker command from the CLI module directly as a subcommand
from .cli.worker import main as worker_cmd
cli.add_command(worker_cmd, name='worker')

# Add the analytics command from the CLI module directly as a subcommand
from .cli.analytics import main as analytics_cmd
cli.add_command(analytics_cmd, name='analytics')

# Add the status command from the CLI module directly as a subcommand
from .cli.status import main as status_cmd
cli.add_command(status_cmd, name='status')

# Add the search command from the CLI module directly as a subcommand
from .cli.search import main as search_cmd
cli.add_command(search_cmd, name='search')

# Dead letter queue implemented for job control system

# Add the ontology-generator command from the CLI module directly as a subcommand
from .cli.ontology_generator import main as ontology_generator_cmd
cli.add_command(ontology_generator_cmd, name='ontology-generator')

# Add the deadletter command from the CLI module directly as a subcommand
from .cli.deadletter import main as deadletter_cmd
cli.add_command(deadletter_cmd, name='deadletter')

# Add the ontology-analytics command for analytics-driven ontology generation
from .cli.ontology_analytics import main as ontology_analytics_cmd
cli.add_command(ontology_analytics_cmd, name='ontology-analytics')

# Add the ontology-extract command for entity extraction and graph export
from .cli.ontology_extract import main as ontology_extract_cmd
cli.add_command(ontology_extract_cmd, name='ontology-extract')

# Add the steerable-discovery command for intelligent ontology discovery with checkpoints
from .cli.steerable_discovery import main as steerable_discovery_cmd
cli.add_command(steerable_discovery_cmd, name='steerable-discovery')

# Add the demo-discovery command for automated demo of discovery process
from .cli.demo_discovery import main as demo_discovery_cmd
cli.add_command(demo_discovery_cmd, name='demo-discovery')


# All CLI tools have been converted to Click and integrated above


def main():
    """Entry point for the unified CLI."""
    cli()


if __name__ == '__main__':
    main()