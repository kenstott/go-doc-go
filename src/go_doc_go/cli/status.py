#!/usr/bin/env python3
"""
Command-line interface for monitoring document processing status without database dependencies.
Uses log files and file-based status tracking.
"""

import argparse
import json
import os
import sys
import time
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


def format_duration(seconds: float) -> str:
    """Format duration in a human-readable way."""
    if seconds < 60:
        return f"{seconds:.1f}s"
    elif seconds < 3600:
        return f"{seconds // 60:.0f}m {seconds % 60:.0f}s"
    else:
        hours = seconds // 3600
        minutes = (seconds % 3600) // 60
        return f"{hours:.0f}h {minutes:.0f}m"


class StatusMonitor:
    """File-based status monitoring for document processing."""

    def __init__(self, config: Config):
        """Initialize status monitor."""
        self.config = config
        self.log_file = self._resolve_log_file()
        self.status_file = self._get_status_file_path()

    def _resolve_log_file(self) -> str:
        """Get the resolved log file path."""
        log_config = self.config.get_logging_config()
        log_file = log_config.get("file", "./logs/go-doc-go.log")
        if not os.path.isabs(log_file):
            log_file = os.path.abspath(log_file)
        return log_file

    def _get_status_file_path(self) -> str:
        """Get path to status file (creates directory if needed)."""
        log_dir = os.path.dirname(self.log_file)
        os.makedirs(log_dir, exist_ok=True)
        return os.path.join(log_dir, "processing_status.json")

    def get_current_status(self) -> Dict[str, Any]:
        """Get current processing status from files."""
        status = {
            "timestamp": datetime.now().isoformat(),
            "log_file": self.log_file,
            "log_exists": os.path.exists(self.log_file),
            "status_file": self.status_file,
            "status_exists": os.path.exists(self.status_file),
            "recent_activity": [],
            "storage_info": {},
            "content_sources": len(self.config.get_content_sources()),
            "processing_config": self.config.get_processing_config()
        }

        # Read status file if it exists
        if os.path.exists(self.status_file):
            try:
                with open(self.status_file, 'r') as f:
                    saved_status = json.load(f)
                    status.update(saved_status)
            except Exception as e:
                status["status_file_error"] = str(e)

        # Get recent log activity
        if os.path.exists(self.log_file):
            status["recent_activity"] = self._get_recent_log_activity()
            status["log_size"] = os.path.getsize(self.log_file)
            status["log_modified"] = datetime.fromtimestamp(
                os.path.getmtime(self.log_file)
            ).isoformat()

        # Get storage info
        status["storage_info"] = self._get_storage_info()

        return status

    def _get_recent_log_activity(self, lines: int = 20) -> List[str]:
        """Get recent lines from log file."""
        try:
            with open(self.log_file, 'r') as f:
                all_lines = f.readlines()
                return [line.strip() for line in all_lines[-lines:]]
        except Exception:
            return []

    def _get_storage_info(self) -> Dict[str, Any]:
        """Get information about storage backend."""
        storage_info = {
            "backend": self.config.get_storage_backend(),
            "path": self.config.get_storage_path()
        }

        storage_path = storage_info["path"]
        if os.path.exists(storage_path):
            if os.path.isfile(storage_path):
                # SQLite database file
                storage_info["size"] = os.path.getsize(storage_path)
                storage_info["modified"] = datetime.fromtimestamp(
                    os.path.getmtime(storage_path)
                ).isoformat()
            elif os.path.isdir(storage_path):
                # Directory-based storage
                try:
                    total_size = sum(
                        os.path.getsize(os.path.join(dirpath, filename))
                        for dirpath, dirnames, filenames in os.walk(storage_path)
                        for filename in filenames
                    )
                    storage_info["total_size"] = total_size
                    storage_info["file_count"] = sum(
                        len(filenames)
                        for dirpath, dirnames, filenames in os.walk(storage_path)
                    )
                except Exception as e:
                    storage_info["error"] = str(e)
            storage_info["exists"] = True
        else:
            storage_info["exists"] = False

        return storage_info

    def follow_logs(self, follow_lines: int = 10):
        """Follow log file for live updates."""
        if not os.path.exists(self.log_file):
            print(f"❌ Log file not found: {self.log_file}")
            return

        print(f"📋 Following log file: {self.log_file}")
        print("Press Ctrl+C to stop following\n")

        # Show recent lines first
        recent_lines = self._get_recent_log_activity(follow_lines)
        for line in recent_lines:
            print(line)

        # Follow new lines
        try:
            with open(self.log_file, 'r') as f:
                # Go to end of file
                f.seek(0, 2)

                while True:
                    line = f.readline()
                    if line:
                        print(line.strip())
                    else:
                        time.sleep(0.1)

        except KeyboardInterrupt:
            print(f"\n⏹️  Stopped following log file")


def display_status(status: Dict[str, Any], detailed: bool = False):
    """Display status information in formatted output."""
    print(f"\n{'='*60}")
    print(f"GO-DOC-GO PROCESSING STATUS")
    print(f"{'='*60}")

    print(f"Timestamp:           {format_timestamp(status.get('timestamp'))}")
    print(f"Content Sources:     {status.get('content_sources', 0)}")

    # Storage information
    storage_info = status.get('storage_info', {})
    print(f"\nStorage Backend:     {storage_info.get('backend', 'N/A')}")
    print(f"Storage Path:        {storage_info.get('path', 'N/A')}")
    print(f"Storage Exists:      {'✅' if storage_info.get('exists') else '❌'}")

    if storage_info.get('size'):
        size_mb = storage_info['size'] / (1024 * 1024)
        print(f"Storage Size:        {size_mb:.1f} MB")

    if storage_info.get('total_size'):
        total_mb = storage_info['total_size'] / (1024 * 1024)
        print(f"Total Storage Size:  {total_mb:.1f} MB")
        print(f"File Count:          {storage_info.get('file_count', 'N/A')}")

    # Processing configuration
    processing_config = status.get('processing_config', {})
    if processing_config:
        print(f"\nProcessing Config:")
        print(f"  Batch Size:        {processing_config.get('batch_size', 'N/A')}")
        print(f"  Max Workers:       {processing_config.get('max_workers', 'N/A')}")
        print(f"  Timeout:           {processing_config.get('timeout_seconds', 'N/A')}s")

    # Log file information
    print(f"\nLog File:            {status.get('log_file', 'N/A')}")
    print(f"Log Exists:          {'✅' if status.get('log_exists') else '❌'}")

    if status.get('log_size'):
        log_size_mb = status['log_size'] / (1024 * 1024)
        print(f"Log Size:            {log_size_mb:.1f} MB")

    if status.get('log_modified'):
        print(f"Log Modified:        {format_timestamp(status.get('log_modified'))}")

    # Recent activity
    recent_activity = status.get('recent_activity', [])
    if recent_activity and detailed:
        print(f"\nRecent Log Activity ({len(recent_activity)} lines):")
        print("-" * 60)
        for line in recent_activity[-10:]:  # Show last 10 lines
            print(line)

    # Errors
    if status.get('status_file_error'):
        print(f"\n❌ Status File Error: {status.get('status_file_error')}")

    if storage_info.get('error'):
        print(f"❌ Storage Error: {storage_info.get('error')}")


def main():
    """Main entry point for the status monitoring CLI."""
    parser = argparse.ArgumentParser(
        description="Go-Doc-Go Status Monitoring CLI",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Show current status
  python -m go_doc_go.cli.status

  # Show detailed status with recent log activity
  python -m go_doc_go.cli.status --detailed

  # Follow log file for live updates
  python -m go_doc_go.cli.status --follow

  # Follow log file showing last 20 lines first
  python -m go_doc_go.cli.status --follow --lines 20

  # Use custom config file
  python -m go_doc_go.cli.status --config /path/to/config.yaml
        """
    )

    parser.add_argument(
        "--config", "-c",
        help="Path to configuration file (overrides GO_DOC_GO_CONFIG_PATH)"
    )

    parser.add_argument(
        "--detailed", "-d",
        action="store_true",
        help="Show detailed status including recent log activity"
    )

    parser.add_argument(
        "--follow", "-f",
        action="store_true",
        help="Follow log file for live updates (like tail -f)"
    )

    parser.add_argument(
        "--lines", "-n",
        type=int,
        default=10,
        help="Number of recent log lines to show when following (default: 10)"
    )

    parser.add_argument(
        "--refresh",
        type=int,
        default=5,
        help="Refresh interval in seconds for status updates (default: 5)"
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
        monitor = StatusMonitor(config)

        if args.follow:
            # Follow log file
            monitor.follow_logs(args.lines)
        else:
            # Show current status
            status = monitor.get_current_status()
            display_status(status, detailed=args.detailed)

        return 0

    except KeyboardInterrupt:
        print(f"\n⏹️  Status monitoring interrupted by user")
        return 130
    except Exception as e:
        print(f"❌ Status monitoring failed: {str(e)}")
        return 1


if __name__ == "__main__":
    sys.exit(main())