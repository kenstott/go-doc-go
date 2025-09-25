#!/usr/bin/env python3
"""
Command-line interface for analytics and reporting without database dependencies.
Reads from configured analytics outputs (parquet files, SQLite, etc.).
"""

import argparse
import json
import os
import sys
from datetime import datetime, timedelta
from pathlib import Path
from typing import Dict, Any, Optional, List

# Add the src directory to Python path
sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from go_doc_go.config import Config


def format_timestamp(timestamp: Optional[str]) -> str:
    """Format timestamp for display."""
    if not timestamp:
        return "N/A"
    try:
        if isinstance(timestamp, datetime):
            return timestamp.strftime("%Y-%m-%d %H:%M:%S")
        dt = datetime.fromisoformat(timestamp.replace('Z', '+00:00'))
        return dt.strftime("%Y-%m-%d %H:%M:%S")
    except Exception:
        return str(timestamp)


def format_file_size(size_bytes: int) -> str:
    """Format file size in human-readable format."""
    for unit in ['B', 'KB', 'MB', 'GB']:
        if size_bytes < 1024.0:
            return f"{size_bytes:.1f} {unit}"
        size_bytes /= 1024.0
    return f"{size_bytes:.1f} TB"


class AnalyticsManager:
    """File-based analytics manager for document processing metrics."""

    def __init__(self, config: Config):
        """Initialize analytics manager."""
        self.config = config
        self.analytics_config = config.get_analytics_config()

    def get_analytics_summary(self) -> Dict[str, Any]:
        """Get summary of all configured analytics outputs."""
        summary = {
            "timestamp": datetime.now().isoformat(),
            "analytics_enabled": self.config.is_analytics_enabled(),
            "outputs": [],
            "total_outputs": 0,
            "storage_info": self._get_storage_summary()
        }

        analytics_outputs = self.config.get_analytics_outputs()
        summary["total_outputs"] = len(analytics_outputs)

        for output_config in analytics_outputs:
            output_info = self._analyze_output(output_config)
            summary["outputs"].append(output_info)

        return summary

    def _analyze_output(self, output_config: Dict[str, Any]) -> Dict[str, Any]:
        """Analyze a single analytics output configuration."""
        output_info = {
            "type": output_config.get("type", "unknown"),
            "path": output_config.get("path", ""),
            "config": output_config,
            "exists": False,
            "size": 0,
            "files": [],
            "error": None
        }

        output_path = output_config.get("path", "")
        if not output_path:
            output_info["error"] = "No path configured"
            return output_info

        # Resolve relative paths
        if not os.path.isabs(output_path):
            output_path = os.path.abspath(output_path)

        output_info["resolved_path"] = output_path

        try:
            if output_config.get("type") == "parquet":
                output_info.update(self._analyze_parquet_output(output_path))
            elif output_config.get("type") == "sqlite":
                output_info.update(self._analyze_sqlite_output(output_path))
            elif output_config.get("type") == "json":
                output_info.update(self._analyze_json_output(output_path))
            else:
                output_info.update(self._analyze_generic_output(output_path))

        except Exception as e:
            output_info["error"] = str(e)

        return output_info

    def _analyze_parquet_output(self, path: str) -> Dict[str, Any]:
        """Analyze parquet output directory."""
        info = {"format": "parquet"}

        if os.path.exists(path):
            info["exists"] = True
            if os.path.isdir(path):
                # Count parquet files
                parquet_files = []
                total_size = 0
                for root, dirs, files in os.walk(path):
                    for file in files:
                        if file.endswith('.parquet'):
                            file_path = os.path.join(root, file)
                            file_size = os.path.getsize(file_path)
                            total_size += file_size
                            parquet_files.append({
                                "name": file,
                                "path": file_path,
                                "size": file_size,
                                "modified": datetime.fromtimestamp(
                                    os.path.getmtime(file_path)
                                ).isoformat()
                            })

                info["files"] = parquet_files
                info["file_count"] = len(parquet_files)
                info["total_size"] = total_size
                info["type"] = "directory"
            else:
                # Single parquet file
                info["size"] = os.path.getsize(path)
                info["modified"] = datetime.fromtimestamp(
                    os.path.getmtime(path)
                ).isoformat()
                info["type"] = "file"
        else:
            info["exists"] = False

        return info

    def _analyze_sqlite_output(self, path: str) -> Dict[str, Any]:
        """Analyze SQLite output file."""
        info = {"format": "sqlite"}

        if os.path.exists(path):
            info["exists"] = True
            info["size"] = os.path.getsize(path)
            info["modified"] = datetime.fromtimestamp(
                os.path.getmtime(path)
            ).isoformat()

            # Try to get table information if sqlite3 is available
            try:
                import sqlite3
                with sqlite3.connect(path) as conn:
                    cursor = conn.execute("""
                        SELECT name FROM sqlite_master
                        WHERE type='table' AND name NOT LIKE 'sqlite_%'
                    """)
                    tables = [row[0] for row in cursor.fetchall()]
                    info["tables"] = tables
                    info["table_count"] = len(tables)

                    # Get row counts for each table
                    table_stats = {}
                    for table in tables:
                        cursor = conn.execute(f"SELECT COUNT(*) FROM {table}")
                        row_count = cursor.fetchone()[0]
                        table_stats[table] = row_count
                    info["table_stats"] = table_stats

            except ImportError:
                info["note"] = "sqlite3 not available for detailed analysis"
            except Exception as e:
                info["sqlite_error"] = str(e)
        else:
            info["exists"] = False

        return info

    def _analyze_json_output(self, path: str) -> Dict[str, Any]:
        """Analyze JSON output file."""
        info = {"format": "json"}

        if os.path.exists(path):
            info["exists"] = True
            info["size"] = os.path.getsize(path)
            info["modified"] = datetime.fromtimestamp(
                os.path.getmtime(path)
            ).isoformat()

            # Try to parse JSON structure
            try:
                with open(path, 'r') as f:
                    data = json.load(f)
                    if isinstance(data, list):
                        info["records"] = len(data)
                        info["structure"] = "array"
                    elif isinstance(data, dict):
                        info["keys"] = list(data.keys())
                        info["structure"] = "object"
                    else:
                        info["structure"] = type(data).__name__

            except Exception as e:
                info["json_error"] = str(e)
        else:
            info["exists"] = False

        return info

    def _analyze_generic_output(self, path: str) -> Dict[str, Any]:
        """Analyze generic output path."""
        info = {"format": "generic"}

        if os.path.exists(path):
            info["exists"] = True
            if os.path.isdir(path):
                info["type"] = "directory"
                # Count files in directory
                file_count = 0
                total_size = 0
                for root, dirs, files in os.walk(path):
                    file_count += len(files)
                    for file in files:
                        total_size += os.path.getsize(os.path.join(root, file))
                info["file_count"] = file_count
                info["total_size"] = total_size
            else:
                info["type"] = "file"
                info["size"] = os.path.getsize(path)
                info["modified"] = datetime.fromtimestamp(
                    os.path.getmtime(path)
                ).isoformat()
        else:
            info["exists"] = False

        return info

    def _get_storage_summary(self) -> Dict[str, Any]:
        """Get summary of main storage backend."""
        storage_path = self.config.get_storage_path()
        storage_backend = self.config.get_storage_backend()

        info = {
            "backend": storage_backend,
            "path": storage_path,
            "exists": os.path.exists(storage_path) if storage_path else False
        }

        if storage_path and os.path.exists(storage_path):
            if os.path.isfile(storage_path):
                info["size"] = os.path.getsize(storage_path)
                info["modified"] = datetime.fromtimestamp(
                    os.path.getmtime(storage_path)
                ).isoformat()
            elif os.path.isdir(storage_path):
                total_size = sum(
                    os.path.getsize(os.path.join(dirpath, filename))
                    for dirpath, dirnames, filenames in os.walk(storage_path)
                    for filename in filenames
                )
                info["total_size"] = total_size

        return info


def display_analytics_summary(summary: Dict[str, Any], detailed: bool = False):
    """Display analytics summary in formatted output."""
    print(f"\n{'='*60}")
    print(f"GO-DOC-GO ANALYTICS SUMMARY")
    print(f"{'='*60}")

    print(f"Timestamp:           {format_timestamp(summary.get('timestamp'))}")
    print(f"Analytics Enabled:   {'✅' if summary.get('analytics_enabled') else '❌'}")
    print(f"Output Configurations: {summary.get('total_outputs', 0)}")

    # Storage backend summary
    storage_info = summary.get('storage_info', {})
    print(f"\nMain Storage:")
    print(f"  Backend:           {storage_info.get('backend', 'N/A')}")
    print(f"  Path:              {storage_info.get('path', 'N/A')}")
    print(f"  Exists:            {'✅' if storage_info.get('exists') else '❌'}")

    if storage_info.get('size'):
        print(f"  Size:              {format_file_size(storage_info['size'])}")
    elif storage_info.get('total_size'):
        print(f"  Total Size:        {format_file_size(storage_info['total_size'])}")

    # Analytics outputs
    outputs = summary.get('outputs', [])
    if outputs:
        print(f"\nAnalytics Outputs:")
        print("-" * 60)

        for i, output in enumerate(outputs, 1):
            output_type = output.get('type', 'unknown')
            path = output.get('path', 'N/A')
            exists = '✅' if output.get('exists') else '❌'

            print(f"\n{i}. {output_type.upper()} Output")
            print(f"   Path:             {path}")
            print(f"   Exists:           {exists}")

            if output.get('error'):
                print(f"   Error:            {output['error']}")
                continue

            if output.get('size'):
                print(f"   Size:             {format_file_size(output['size'])}")
            elif output.get('total_size'):
                print(f"   Total Size:       {format_file_size(output['total_size'])}")

            if output.get('file_count'):
                print(f"   Files:            {output['file_count']}")

            if output.get('modified'):
                print(f"   Modified:         {format_timestamp(output['modified'])}")

            # Type-specific information
            if output_type == "sqlite":
                if output.get('table_count'):
                    print(f"   Tables:           {output['table_count']}")
                if detailed and output.get('table_stats'):
                    print(f"   Table Statistics:")
                    for table, count in output['table_stats'].items():
                        print(f"     {table}: {count:,} records")

            elif output_type == "json":
                if output.get('records'):
                    print(f"   Records:          {output['records']:,}")
                if output.get('structure'):
                    print(f"   Structure:        {output['structure']}")

            elif output_type == "parquet":
                if output.get('file_count'):
                    print(f"   Parquet Files:    {output['file_count']}")
                if detailed and output.get('files'):
                    print(f"   File Details:")
                    for file_info in output['files'][:5]:  # Show first 5 files
                        size = format_file_size(file_info['size'])
                        modified = format_timestamp(file_info['modified'])
                        print(f"     {file_info['name']} ({size}, {modified})")
                    if len(output['files']) > 5:
                        print(f"     ... and {len(output['files']) - 5} more files")

    else:
        print(f"\n📭 No analytics outputs configured")


def main():
    """Main entry point for the analytics CLI."""
    parser = argparse.ArgumentParser(
        description="Go-Doc-Go Analytics CLI",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Show analytics summary
  python -m go_doc_go.cli.analytics

  # Show detailed analytics with file listings
  python -m go_doc_go.cli.analytics --detailed

  # Use custom config file
  python -m go_doc_go.cli.analytics --config /path/to/config.yaml
        """
    )

    parser.add_argument(
        "--config", "-c",
        help="Path to configuration file (overrides GO_DOC_GO_CONFIG_PATH)"
    )

    parser.add_argument(
        "--detailed", "-d",
        action="store_true",
        help="Show detailed analytics information including file listings"
    )

    parser.add_argument(
        "--json",
        action="store_true",
        help="Output analytics summary as JSON"
    )

    args = parser.parse_args()

    try:
        # Load configuration
        config_path = (
            args.config or
            os.environ.get("GO_DOC_GO_CONFIG_PATH", "./config.yaml")
        )

        if not os.path.exists(config_path):
            print(f"❌ Configuration file not found: {config_path}")
            return 1

        config = Config(config_path)
        analytics_manager = AnalyticsManager(config)

        # Get analytics summary
        summary = analytics_manager.get_analytics_summary()

        if args.json:
            # Output as JSON
            print(json.dumps(summary, indent=2))
        else:
            # Display formatted summary
            display_analytics_summary(summary, detailed=args.detailed)

        return 0

    except Exception as e:
        print(f"❌ Analytics summary failed: {str(e)}")
        return 1


if __name__ == "__main__":
    sys.exit(main())