"""
Comprehensive search CLI for Go-Doc-Go.

Provides powerful search capabilities against parquet data lakes with contextual
information retrieval, flexible output formats, and advanced filtering options.
"""

import argparse
import json
import sys
from pathlib import Path
from typing import Dict, Any, List, Optional

from go_doc_go.config import Config
from go_doc_go.shared.search_engine import (
    ParquetSearchEngine,
    SearchFilters,
    ContextConfig,
    ReconstructionConfig,
    create_search_engine
)
import logging

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


def create_parser() -> argparse.ArgumentParser:
    """Create argument parser for search CLI."""
    parser = argparse.ArgumentParser(
        description="Search documents in Go-Doc-Go data lake",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:

  # Basic text search
  python -m go_doc_go.cli.search "quarterly revenue"

  # Search with element type filtering
  python -m go_doc_go.cli.search "financial data" --include-types paragraph,heading

  # Search excluding certain types
  python -m go_doc_go.cli.search "analysis" --exclude-types table,list_item

  # Search with regex pattern
  python -m go_doc_go.cli.search --regex "Q[1-4].*revenue"

  # Search with similarity threshold
  python -m go_doc_go.cli.search "machine learning" --similarity-threshold 0.3

  # Output as JSON for programmatic use
  python -m go_doc_go.cli.search "data science" --output json

  # Reconstruct full documents as markdown
  python -m go_doc_go.cli.search "project update" --reconstruct-docs markdown

  # Search with custom config file
  python -m go_doc_go.cli.search "analysis" --config ./custom-config.yaml

  # Limit results and include more context
  python -m go_doc_go.cli.search "strategic plan" --limit 20 --parents 3 --siblings 5
        """
    )

    # Required arguments
    parser.add_argument(
        'query',
        nargs='?',
        help='Search query text (optional if using --regex only)'
    )

    # Configuration
    parser.add_argument(
        '--config', '-c',
        type=str,
        default='config.yaml',
        help='Path to configuration file (default: config.yaml)'
    )

    # Filtering options
    parser.add_argument(
        '--include-types',
        type=str,
        help='Comma-separated list of element types to include (e.g., paragraph,heading)'
    )

    parser.add_argument(
        '--exclude-types',
        type=str,
        help='Comma-separated list of element types to exclude (e.g., table,list_item)'
    )

    parser.add_argument(
        '--regex',
        type=str,
        help='Regex pattern to filter content'
    )

    parser.add_argument(
        '--similarity-threshold',
        type=float,
        default=0.0,
        help='Minimum cosine similarity threshold (0.0 to 1.0, default: 0.0)'
    )

    parser.add_argument(
        '--limit', '-l',
        type=int,
        default=10,
        help='Maximum number of results to return (default: 10)'
    )

    # Context options
    parser.add_argument(
        '--parents',
        type=int,
        default=2,
        help='Number of parent levels to include in context (default: 2)'
    )

    parser.add_argument(
        '--siblings',
        type=int,
        default=3,
        help='Number of sibling elements to include (default: 3)'
    )

    parser.add_argument(
        '--semantic-rels',
        type=int,
        default=5,
        help='Number of semantic relationships to include (default: 5)'
    )

    parser.add_argument(
        '--no-doc-metadata',
        action='store_true',
        help='Exclude document metadata from context'
    )

    # Output options
    parser.add_argument(
        '--output', '-o',
        choices=['table', 'json', 'summary'],
        default='table',
        help='Output format (default: table)'
    )

    parser.add_argument(
        '--reconstruct-docs',
        choices=['markdown', 'html', 'json'],
        help='Reconstruct and include full documents in specified format'
    )

    parser.add_argument(
        '--verbose', '-v',
        action='store_true',
        help='Enable verbose logging'
    )

    parser.add_argument(
        '--data-path',
        type=str,
        help='Override data lake path from config'
    )

    return parser


def parse_element_types(types_str: Optional[str]) -> List[str]:
    """Parse comma-separated element types string."""
    if not types_str:
        return []
    return [t.strip() for t in types_str.split(',') if t.strip()]


def format_table_output(results, show_context: bool = True):
    """Format search results as a human-readable table."""
    if not results.results:
        print(f"No results found for query: '{results.query}'")
        return

    print(f"\nSearch Results for: '{results.query}'")
    print(f"Total matches: {results.total}")
    print("=" * 80)

    for i, result in enumerate(results.results, 1):
        element = result.element_data
        similarity = result.similarity or 0.0

        print(f"\n[{i}] Element: {element.get('element_id', '')[:12]}...")
        print(f"    Type: {element.get('element_type', 'unknown')}")
        print(f"    Similarity: {similarity:.3f}")
        print(f"    Content: {element.get('content_preview', '')[:200]}...")

        if show_context and result.context:
            doc_info = result.context.get('document', {})
            if doc_info.get('source'):
                print(f"    Document: {doc_info['source']}")

            parents = result.context.get('parents', [])
            if parents:
                print(f"    Parents: {' > '.join([p.get('element_type', 'unknown') for p in parents[:2]])}")

        if result.relationships:
            rel_count = len(result.relationships)
            print(f"    Relationships: {rel_count} found")

        if result.siblings:
            sibling_count = len(result.siblings)
            print(f"    Siblings: {sibling_count} found")

        print("-" * 80)

    if results.materialized_documents:
        print(f"\nReconstructed Documents: {len(results.materialized_documents)}")
        for doc_id, content in results.materialized_documents.items():
            print(f"\n--- Document: {doc_id} ---")
            if isinstance(content, str):
                print(content[:500] + ("..." if len(content) > 500 else ""))
            else:
                print(json.dumps(content, indent=2)[:500] + "...")


def format_json_output(results):
    """Format search results as JSON."""
    print(json.dumps(results.to_dict(), indent=2, default=str))


def format_summary_output(results):
    """Format search results as a concise summary."""
    print(f"Query: '{results.query}'")
    print(f"Total Results: {results.total}")

    if results.results:
        print("\nTop Results:")
        for i, result in enumerate(results.results[:5], 1):
            element = result.element_data
            similarity = result.similarity or 0.0
            content = element.get('content_preview', '')[:100]
            print(f"{i}. [{similarity:.3f}] {element.get('element_type', 'unknown')}: {content}...")

        if results.total > 5:
            print(f"... and {results.total - 5} more results")

    if results.materialized_documents:
        print(f"\nReconstructed {len(results.materialized_documents)} documents")


def main():
    """Main entry point for search CLI."""
    parser = create_parser()
    args = parser.parse_args()

    # Configure logging
    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)
        logging.getLogger('go_doc_go').setLevel(logging.DEBUG)

    # Validate arguments
    if not args.query and not args.regex:
        parser.error("Must provide either a search query or --regex pattern")

    try:
        # Load configuration
        config = Config(args.config)
        logger.debug(f"Loaded configuration from {args.config}")

        # Create search engine
        if args.data_path:
            search_engine = ParquetSearchEngine(args.data_path)
        else:
            search_engine = create_search_engine(config.get_config())

        # Build search filters
        element_types = []
        if args.include_types:
            element_types.extend(parse_element_types(args.include_types))
        if args.exclude_types:
            exclude_types = parse_element_types(args.exclude_types)
            element_types.extend([f"-{t}" for t in exclude_types])

        search_filters = SearchFilters(
            regex_pattern=args.regex,
            element_types=element_types,
            cosine_threshold=args.similarity_threshold,
            limit=args.limit
        )

        # Build context config
        context_config = ContextConfig(
            parents=args.parents,
            siblings=args.siblings,
            semantic_relationships=args.semantic_rels,
            include_document_metadata=not args.no_doc_metadata
        )

        # Build reconstruction config
        reconstruction_config = None
        if args.reconstruct_docs:
            reconstruction_config = ReconstructionConfig(
                format=args.reconstruct_docs,
                include_metadata=not args.no_doc_metadata,
                max_depth=10
            )

        # Execute search
        logger.info(f"Executing search for query: '{args.query}'")
        results = search_engine.search(
            query=args.query or "",
            filters=search_filters,
            context_config=context_config,
            reconstruction_config=reconstruction_config
        )

        # Output results
        if args.output == 'json':
            format_json_output(results)
        elif args.output == 'summary':
            format_summary_output(results)
        else:
            format_table_output(results)

    except KeyboardInterrupt:
        print("\nSearch cancelled by user.")
        sys.exit(1)
    except FileNotFoundError as e:
        print(f"Error: Configuration file not found: {e}")
        sys.exit(1)
    except Exception as e:
        logger.error(f"Search failed: {e}", exc_info=args.verbose)
        print(f"Error: {e}")
        sys.exit(1)


if __name__ == '__main__':
    main()