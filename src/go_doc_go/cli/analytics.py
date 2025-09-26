#!/usr/bin/env python3
"""
Command-line interface for analytics and reporting using AnalyticsStorage interface.
Database-agnostic analytics that works with any configured analytics backend.
"""

import json
import os
import sys
import click
from datetime import datetime
from pathlib import Path
from typing import Dict, Any

# Package imports (no path manipulation needed for proper package installation)

from go_doc_go.config import Config
from go_doc_go.storage_adapters.factory import StorageFactory


def format_timestamp(timestamp: str) -> str:
    """Format timestamp for display."""
    if not timestamp:
        return "N/A"
    try:
        if isinstance(timestamp, datetime):
            return timestamp.strftime("%Y-%m-%d %H:%M:%S")

        # Try parsing as ISO format
        dt = datetime.fromisoformat(timestamp.replace('Z', '+00:00'))
        return dt.strftime("%Y-%m-%d %H:%M:%S")
    except Exception:
        return str(timestamp)


def format_file_size(size_bytes: int) -> str:
    """Format file size in human-readable format."""
    if size_bytes is None:
        return "Unknown"

    for unit in ['B', 'KB', 'MB', 'GB']:
        if size_bytes < 1024.0:
            return f"{size_bytes:.1f} {unit}"
        size_bytes /= 1024.0
    return f"{size_bytes:.1f} TB"


class DatabaseAgnosticAnalyticsManager:
    """Database-agnostic analytics manager using AnalyticsStorage interface."""

    def __init__(self, config: Config):
        """Initialize analytics manager with AnalyticsStorage."""
        self.config = config

        # Get analytics storage configuration
        analytics_outputs = config.get_analytics_outputs()
        if not analytics_outputs:
            raise ValueError("No analytics storage configured")

        # Use the first analytics output configuration
        analytics_config = analytics_outputs[0]

        # Create analytics storage using factory
        self.analytics_storage = StorageFactory.create_analytics_storage(analytics_config)

    def get_analytics_summary(self) -> Dict[str, Any]:
        """Get comprehensive analytics summary using storage interface."""
        summary = {
            "timestamp": datetime.now().isoformat(),
            "analytics_enabled": self.config.is_analytics_enabled(),
            "storage_backend": "interface-based"
        }

        try:
            # Get storage summary from analytics interface
            storage_summary = self.analytics_storage.get_storage_summary()
            summary.update(storage_summary)

            # Get table statistics
            table_stats = self.analytics_storage.get_table_statistics()
            summary["table_statistics"] = table_stats

            # Get recent run statistics
            run_stats = self.analytics_storage.get_run_statistics(include_details=False)
            summary["recent_runs"] = run_stats.get("runs", [])[:5]  # Last 5 runs
            summary["total_runs"] = run_stats.get("total_runs", 0)

        except Exception as e:
            summary["error"] = str(e)
            summary["storage_health"] = "error"

        return summary


def display_analytics_summary(summary: Dict[str, Any], detailed: bool = False):
    """Display analytics summary in formatted output."""
    print(f"\n{'='*60}")
    print(f"GO-DOC-GO ANALYTICS SUMMARY")
    print(f"{'='*60}")

    print(f"Timestamp:           {format_timestamp(summary.get('timestamp'))}")
    print(f"Analytics Enabled:   {'✅' if summary.get('analytics_enabled') else '❌'}")

    # Backend information
    backend = summary.get('backend', 'unknown')
    print(f"Storage Backend:     {backend}")

    if summary.get('error'):
        print(f"❌ Error:             {summary['error']}")
        return

    # Storage health
    health = summary.get('storage_health', 'unknown')
    health_icon = '✅' if health == 'healthy' else '❌'
    print(f"Storage Health:      {health_icon} {health}")

    # Storage details
    if summary.get('path'):
        print(f"Storage Path:        {summary.get('path')}")

    if summary.get('uri'):
        print(f"Database URI:         {summary.get('uri')}")

    if summary.get('total_size') is not None:
        if isinstance(summary['total_size'], (int, float)):
            print(f"Total Size:          {format_file_size(summary['total_size'])}")
        else:
            print(f"Total Size:          {summary['total_size']}")

    # Table counts
    table_counts = summary.get('table_counts', {})
    if table_counts:
        print(f"\nTable/Collection Counts:")
        print("-" * 30)
        for table_name, count in table_counts.items():
            print(f"  {table_name:<15}: {count:,}")

    # Recent runs
    recent_runs = summary.get('recent_runs', [])
    total_runs = summary.get('total_runs', 0)

    if total_runs > 0:
        print(f"\nProcessing Runs:")
        print("-" * 30)
        print(f"Total Runs:          {total_runs}")

        if recent_runs and detailed:
            print(f"\nRecent Runs:")
            for i, run in enumerate(recent_runs, 1):
                run_id = run.get('run_id', 'unknown')[:12] + '...' if len(str(run.get('run_id', ''))) > 15 else run.get('run_id', 'unknown')
                start_time = format_timestamp(run.get('start_time'))
                doc_count = run.get('document_count', run.get('total_records', 0))
                print(f"  {i}. {run_id} ({doc_count:,} records) - {start_time}")

    # Table statistics (detailed view)
    if detailed and 'table_statistics' in summary:
        table_stats = summary['table_statistics']
        if table_stats.get('tables'):
            print(f"\nDetailed Table Information:")
            print("-" * 40)

            for table in table_stats['tables']:
                name = table.get('name', 'unknown')
                table_type = table.get('type', 'unknown')
                row_count = table.get('row_count', table.get('file_count', 0))

                print(f"\n{name.upper()}:")
                print(f"  Type:              {table_type}")
                print(f"  Records/Files:     {row_count:,}")

                if 'total_size' in table:
                    print(f"  Size:              {format_file_size(table['total_size'])}")

                if 'columns' in table and table['columns']:
                    column_count = len(table['columns'])
                    print(f"  Columns:           {column_count}")

    # Partitioning info
    if 'partitioning_info' in summary:
        part_info = summary['partitioning_info']
        if part_info.get('scheme'):
            print(f"\nPartitioning:")
            print(f"  Scheme:            {part_info.get('scheme')}")
            if part_info.get('compression'):
                print(f"  Compression:       {part_info.get('compression')}")


@click.command()
@click.option("--config", "-c", help="Path to configuration file (overrides GO_DOC_GO_CONFIG_PATH)")
@click.option("--detailed", "-d", is_flag=True, help="Show detailed analytics information including table schemas and recent runs")
@click.option("--json", "output_json", is_flag=True, help="Output analytics summary as JSON")
def main(config, detailed, output_json):
    """Go-Doc-Go Database-Agnostic Analytics CLI

    This CLI works with any configured analytics backend (Parquet, PostgreSQL,
    MongoDB, Elasticsearch, etc.) through the AnalyticsStorage interface.

    Examples:
      # Show analytics summary
      python -m go_doc_go.cli.analytics

      # Show detailed analytics with table information
      python -m go_doc_go.cli.analytics --detailed

      # Use custom config file
      python -m go_doc_go.cli.analytics --config /path/to/config.yaml

      # Output as JSON for programmatic use
      python -m go_doc_go.cli.analytics --json
    """

    try:
        # Load configuration
        config_path = (
            config or
            os.environ.get("GO_DOC_GO_CONFIG_PATH", "./config.yaml")
        )

        if not os.path.exists(config_path):
            click.echo(f"❌ Configuration file not found: {config_path}")
            sys.exit(1)

        config_obj = Config(config_path)

        # Check if analytics is enabled
        if not config_obj.is_analytics_enabled():
            click.echo("❌ Analytics is not enabled in configuration")
            sys.exit(1)

        analytics_manager = DatabaseAgnosticAnalyticsManager(config_obj)

        # Get analytics summary
        summary = analytics_manager.get_analytics_summary()

        if output_json:
            # Output as JSON
            click.echo(json.dumps(summary, indent=2))
        else:
            # Display formatted summary
            display_analytics_summary(summary, detailed=detailed)

    except Exception as e:
        click.echo(f"❌ Analytics summary failed: {str(e)}")
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main()