"""
Comprehensive search CLI for Go-Doc-Go.

Provides powerful search capabilities against parquet data lakes with contextual
information retrieval, flexible output formats, and advanced filtering options.
"""

import json
import sys
import click
import logging
from pathlib import Path
from typing import Dict, Any, List, Optional

from go_doc_go.config import Config
from go_doc_go.storage_adapters.factory import StorageFactory

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)




def parse_element_types(types_str: Optional[str]) -> List[str]:
    """Parse comma-separated element types string."""
    if not types_str:
        return []
    return [t.strip() for t in types_str.split(',') if t.strip()]


def format_table_output_simple(results):
    """Format search results as a human-readable table."""
    print(f"\nSearch Results for: '{results['query']}'")
    print(f"Total matches: {results['total_hits']}")
    print("=" * 80)

    for i, hit in enumerate(results['hits'], 1):
        print(f"\n[{i}] Element: {hit.get('element_id', 'N/A')[:12]}...")
        print(f"    Type: {hit.get('element_type', 'unknown')}")
        print(f"    Document: {hit.get('doc_id', 'N/A')}")
        content = hit.get('content_preview', hit.get('content', ''))
        print(f"    Content: {str(content)[:200]}...")

        print("-" * 80)


def format_summary_output_simple(results):
    """Format search results as a concise summary."""
    print(f"Query: '{results['query']}'")
    print(f"Total Results: {results['total_hits']}")

    if results['hits']:
        print("\nTop Results:")
        for i, hit in enumerate(results['hits'][:5], 1):
            content = str(hit.get('content_preview', hit.get('content', '')))[:100]
            element_type = hit.get('element_type', 'unknown')
            print(f"{i}. {element_type}: {content}...")

        if results['total_hits'] > 5:
            print(f"... and {results['total_hits'] - 5} more results")


@click.command()
@click.argument('query', required=False)
@click.option('--config', '-c', default='config.yaml', help='Path to configuration file (default: config.yaml)')
@click.option('--include-types', help='Comma-separated list of element types to include (e.g., paragraph,heading)')
@click.option('--exclude-types', help='Comma-separated list of element types to exclude (e.g., table,list_item)')
@click.option('--regex', help='Regex pattern to filter content')
@click.option('--similarity-threshold', type=float, default=0.0, help='Minimum cosine similarity threshold (0.0 to 1.0, default: 0.0)')
@click.option('--limit', '-l', type=int, default=10, help='Maximum number of results to return (default: 10)')
@click.option('--output', '-o', type=click.Choice(['table', 'json', 'summary']), default='table', help='Output format (default: table)')
@click.option('--verbose', '-v', is_flag=True, help='Enable verbose logging')
@click.option('--data-path', help='Override data lake path from config (NOTE: currently uses default parquet backend)')
def main(query, config, include_types, exclude_types, regex, similarity_threshold, limit, output, verbose, data_path):
    """Search documents in Go-Doc-Go data lake.

    Uses the analytics storage interface for search functionality.

    Examples:

      # Basic text search
      python -m go_doc_go search "quarterly revenue"

      # Search with element type filtering
      python -m go_doc_go search "financial data" --include-types paragraph,heading

      # Search with regex pattern
      python -m go_doc_go search --regex "Q[1-4].*revenue"

      # Output as JSON for programmatic use
      python -m go_doc_go search "data science" --output json

      # Search with custom config file
      python -m go_doc_go search "analysis" --config ./custom-config.yaml

      # Limit results
      python -m go_doc_go search "strategic plan" --limit 20
    """
    # Configure logging
    if verbose:
        logging.getLogger().setLevel(logging.DEBUG)
        logging.getLogger('go_doc_go').setLevel(logging.DEBUG)

    # Validate arguments
    if not query and not regex:
        click.echo("Error: Must provide either a search query or --regex pattern")
        sys.exit(1)

    try:
        # Load configuration
        config_obj = Config(config)
        logger.debug(f"Loaded configuration from {config}")

        # Get analytics configuration
        analytics_config = config_obj.get_analytics_config()
        if not analytics_config or not analytics_config.get('enabled'):
            click.echo("Error: Analytics must be enabled for search functionality")
            sys.exit(1)

        # Get the first analytics output (assuming parquet for now)
        outputs = analytics_config.get('outputs', [])
        if not outputs:
            click.echo("Error: No analytics outputs configured")
            sys.exit(1)

        # Use the first output for search
        output_config = outputs[0]
        if data_path:
            # Override path if specified
            output_config = output_config.copy()
            output_config['path'] = data_path

        # Create analytics storage for search
        analytics_storage = StorageFactory.create_analytics_storage(output_config)

        # Build search filters
        filters = {}
        if include_types:
            filters['element_type'] = parse_element_types(include_types)
        if exclude_types:
            # TODO: Handle exclude types (not directly supported by current interface)
            click.echo("Warning: --exclude-types not yet supported with current search interface")

        # Execute search using analytics storage
        logger.info(f"Executing search for query: '{query}'")
        if query:
            # Use text search
            results = analytics_storage.search_text(
                query=query,
                limit=limit,
                filters=filters
            )
        elif regex:
            # Use structured search for regex
            filters['content_regex'] = regex
            results = analytics_storage.search_structured(
                criteria=filters,
                limit=limit
            )
        else:
            click.echo("Error: Must provide either a search query or --regex pattern")
            sys.exit(1)

        # Convert results to simple format for display
        # Results from analytics storage is List[Dict[str, Any]]
        if not results:
            click.echo(f"No results found for query: '{query or regex}'")
            return

        # Create a simple result structure
        search_results = {
            'query': query or regex,
            'total_hits': len(results),
            'hits': results
        }

        # Output results
        if output == 'json':
            print(json.dumps(search_results, indent=2, default=str))
        elif output == 'summary':
            format_summary_output_simple(search_results)
        else:
            format_table_output_simple(search_results)

    except KeyboardInterrupt:
        click.echo("\nSearch cancelled by user.")
        sys.exit(1)
    except FileNotFoundError as e:
        click.echo(f"Error: Configuration file not found: {e}")
        sys.exit(1)
    except Exception as e:
        logger.error(f"Search failed: {e}", exc_info=verbose)
        click.echo(f"Error: {e}")
        sys.exit(1)


if __name__ == '__main__':
    main()