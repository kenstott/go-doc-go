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
from go_doc_go.adapter.enhanced_content import EnhancedContentResolver
from go_doc_go.adapter.factory import AdapterFactory

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


def format_table_output_simple(results, full_text=False):
    """Format search results as a human-readable table."""
    print(f"\nSearch Results for: '{results['query']}'")
    print(f"Total matches: {results['total_hits']}")
    print("=" * 80)

    for i, hit in enumerate(results['hits'], 1):
        print(f"\n[{i}] Element: {hit.get('element_id', 'N/A')[:12]}...")
        print(f"    Type: {hit.get('element_type', 'unknown')}")
        print(f"    Document: {hit.get('doc_id', 'N/A')}")

        # Use full content or truncated preview based on flag
        if full_text and hit.get('full_content'):
            content = hit['full_content']
            print(f"    Content: {content}")
        else:
            content = hit.get('content_preview', hit.get('content', ''))
            print(f"    Content: {str(content)[:200]}...")

        # Show context if available
        if hit.get('context_elements'):
            print("    Context:")
            for ctx_elem in hit['context_elements']:
                ctx_type = ctx_elem.get('element_type', 'unknown')
                ctx_relation = ctx_elem.get('relation', '')
                ctx_content = ctx_elem.get('content_preview', '')[:100]
                print(f"      [{ctx_relation}] {ctx_type}: {ctx_content}...")

        print("-" * 80)


def format_summary_output_simple(results, full_text=False):
    """Format search results as a concise summary."""
    print(f"Query: '{results['query']}'")
    print(f"Total Results: {results['total_hits']}")

    if results['hits']:
        print("\nTop Results:")
        for i, hit in enumerate(results['hits'][:5], 1):
            if full_text and hit.get('full_content'):
                content = str(hit['full_content'])[:200]
            else:
                content = str(hit.get('content_preview', hit.get('content', '')))[:100]
            element_type = hit.get('element_type', 'unknown')
            print(f"{i}. {element_type}: {content}...")

        if results['total_hits'] > 5:
            print(f"... and {results['total_hits'] - 5} more results")

def format_markdown_output(results, full_text=False):
    """Format search results as Markdown."""
    print(f"# Search Results for: {results['query']}")
    print(f"\n**Total matches:** {results['total_hits']}\n")

    for i, hit in enumerate(results['hits'], 1):
        element_type = hit.get('element_type', 'unknown')
        doc_id = hit.get('doc_id', 'N/A')

        print(f"## Result {i}: {element_type}")
        print(f"**Document:** `{doc_id}`\n")

        if full_text and hit.get('full_content'):
            print(f"### Content")
            print(f"{hit['full_content']}\n")
        else:
            content = hit.get('content_preview', hit.get('content', ''))
            print(f"### Content Preview")
            print(f"{content}\n")

        if hit.get('context_elements'):
            print("### Context")
            for ctx in hit['context_elements']:
                rel = ctx.get('relation', '')
                ctx_type = ctx.get('element_type', 'unknown')
                ctx_content = ctx.get('content_preview', '')
                print(f"- **{rel} ({ctx_type}):** {ctx_content}")
            print()

def format_html_output(results, full_text=False):
    """Format search results as HTML."""
    print("<html><head><title>Search Results</title></head><body>")
    print(f"<h1>Search Results for: {results['query']}</h1>")
    print(f"<p><strong>Total matches:</strong> {results['total_hits']}</p>")

    for i, hit in enumerate(results['hits'], 1):
        element_type = hit.get('element_type', 'unknown')
        doc_id = hit.get('doc_id', 'N/A')

        print(f"<div class='result'>")
        print(f"<h2>Result {i}: {element_type}</h2>")
        print(f"<p><strong>Document:</strong> <code>{doc_id}</code></p>")

        if full_text and hit.get('full_content'):
            print(f"<h3>Content</h3>")
            print(f"<pre>{hit['full_content']}</pre>")
        else:
            content = hit.get('content_preview', hit.get('content', ''))
            print(f"<h3>Content Preview</h3>")
            print(f"<p>{content}</p>")

        if hit.get('context_elements'):
            print("<h3>Context</h3>")
            print("<ul>")
            for ctx in hit['context_elements']:
                rel = ctx.get('relation', '')
                ctx_type = ctx.get('element_type', 'unknown')
                ctx_content = ctx.get('content_preview', '')
                print(f"<li><strong>{rel} ({ctx_type}):</strong> {ctx_content}</li>")
            print("</ul>")
        print("</div>")

    print("</body></html>")


@click.command()
@click.argument('query', required=False)
@click.option('--config', '-c', default='config.yaml', help='Path to configuration file (default: config.yaml)')
@click.option('--include-types', help='Comma-separated list of element types to include (e.g., paragraph,heading)')
@click.option('--exclude-types', help='Comma-separated list of element types to exclude (e.g., table,list_item)')
@click.option('--regex', help='Regex pattern to filter content')
@click.option('--similarity-threshold', type=float, default=0.0, help='Minimum cosine similarity threshold (0.0 to 1.0, default: 0.0)')
@click.option('--limit', '-l', type=int, default=10, help='Maximum number of results to return (default: 10)')
@click.option('--output', '-o', type=click.Choice(['table', 'json', 'summary', 'markdown', 'html']), default='table', help='Output format (default: table)')
@click.option('--verbose', '-v', is_flag=True, help='Enable verbose logging')
@click.option('--data-path', help='Override data lake path from config (NOTE: currently uses default parquet backend)')
@click.option('--full-text', is_flag=True, help='Retrieve full content instead of truncated previews')
@click.option('--context', type=click.Choice(['none', 'parents', 'siblings', 'children', 'all']), default='none', help='Include contextual elements (default: none)')
@click.option('--context-depth', type=int, default=1, help='Depth of context to include (default: 1)')
def main(query, config, include_types, exclude_types, regex, similarity_threshold, limit, output, verbose, data_path, full_text, context, context_depth):
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

        # Initialize content resolver if needed for full text or context
        content_resolver = None
        if full_text or context != 'none':
            try:
                # Create content source adapters
                adapters = AdapterFactory.create_all_adapters(config_obj)

                # Create document parsers using the ContentResolverFactory method
                parsers = AdapterFactory.create_parsers(config_obj.config)

                # Create enhanced content resolver
                content_resolver = EnhancedContentResolver(
                    adapters=adapters,
                    parsers=parsers,
                    config=config_obj.config
                )
            except Exception as e:
                logger.warning(f"Could not initialize content resolver: {e}")
                if full_text:
                    click.echo("Warning: Full text retrieval not available without content resolver")

        # Process results to add full text and context if requested
        for hit in results:
            # Add full text if requested
            if full_text and content_resolver:
                try:
                    element_id = hit.get('element_id')
                    if element_id:
                        full_content = analytics_storage.resolve_element_content(
                            element_id,
                            content_resolver
                        )
                        hit['full_content'] = full_content
                except Exception as e:
                    logger.warning(f"Could not retrieve full text for element {element_id}: {e}")

            # Add contextual elements if requested
            if context != 'none':
                try:
                    context_elements = []
                    element_id = hit.get('element_id')
                    doc_id = hit.get('doc_id')

                    if element_id and doc_id:
                        # Get all elements for the document
                        doc_elements = analytics_storage.get_document_elements(doc_id)

                        # Build element tree
                        element_map = {e['element_id']: e for e in doc_elements}

                        # Find related elements based on context type
                        if context in ['parents', 'all']:
                            # Find parent elements
                            parent_id = hit.get('parent_id')
                            depth = 0
                            while parent_id and depth < context_depth:
                                if parent_id in element_map:
                                    parent = element_map[parent_id]
                                    parent['relation'] = f'parent-{depth+1}'
                                    context_elements.append(parent)
                                    parent_id = parent.get('parent_id')
                                else:
                                    break
                                depth += 1

                        if context in ['siblings', 'all']:
                            # Find sibling elements
                            parent_id = hit.get('parent_id')
                            if parent_id:
                                siblings = [e for e in doc_elements
                                           if e.get('parent_id') == parent_id
                                           and e['element_id'] != element_id]
                                for sibling in siblings[:5]:  # Limit siblings
                                    sibling['relation'] = 'sibling'
                                    context_elements.append(sibling)

                        if context in ['children', 'all']:
                            # Find child elements
                            children = [e for e in doc_elements
                                       if e.get('parent_id') == element_id]
                            for i, child in enumerate(children[:5]):  # Limit children
                                child['relation'] = 'child'
                                context_elements.append(child)

                        if context_elements:
                            hit['context_elements'] = context_elements

                except Exception as e:
                    logger.warning(f"Could not retrieve context for element: {e}")

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
            format_summary_output_simple(search_results, full_text)
        elif output == 'markdown':
            format_markdown_output(search_results, full_text)
        elif output == 'html':
            format_html_output(search_results, full_text)
        else:
            format_table_output_simple(search_results, full_text)

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