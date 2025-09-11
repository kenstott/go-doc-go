#!/usr/bin/env python3
"""
CLI interface for managing dead letter queue operations.
"""

import argparse
import json
import sys
from datetime import datetime
from typing import Optional, List

from ..config import Config
from ..queue.dead_letter import DeadLetterQueue, DeadLetterProcessor, DeadLetterItem


def format_timestamp(timestamp: Optional[str]) -> str:
    """Format timestamp for display."""
    if not timestamp:
        return "N/A"
    try:
        dt = datetime.fromisoformat(timestamp.replace('Z', '+00:00'))
        return dt.strftime("%Y-%m-%d %H:%M:%S")
    except Exception:
        return str(timestamp)


def display_dead_letter_items(items: List[DeadLetterItem], show_details: bool = False):
    """Display dead letter items in a formatted table."""
    if not items:
        print("✅ No items in dead letter queue")
        return
    
    print(f"\n📋 DEAD LETTER QUEUE ({len(items)} items)")
    print("=" * 100)
    
    if show_details:
        # Detailed view
        for i, item in enumerate(items, 1):
            print(f"\n[{i}] 📄 {item.doc_id}")
            print(f"    Queue ID: {item.queue_id}")
            print(f"    Run ID: {item.run_id}")
            print(f"    Source: {item.source_name}")
            print(f"    Failed At: {format_timestamp(item.failed_at)}")
            print(f"    Retry Count: {item.retry_count}")
            print(f"    Error: {item.error_message}")
            if item.error_details:
                print(f"    Details: {item.error_details[:100]}...")
            if item.metadata:
                print(f"    Metadata: {json.dumps(item.metadata, indent=8)}")
    else:
        # Compact table view
        print(f"{'Queue ID':<12} {'Doc ID':<25} {'Run ID':<12} {'Source':<15} {'Failed At':<19} {'Retries':<7} {'Error':<30}")
        print("-" * 130)
        
        for item in items:
            error_short = (item.error_message[:27] + "...") if len(item.error_message) > 30 else item.error_message
            doc_id_short = (item.doc_id[:22] + "...") if len(item.doc_id) > 25 else item.doc_id
            run_id_short = (item.run_id[:9] + "...") if len(item.run_id) > 12 else item.run_id
            source_short = (item.source_name[:12] + "...") if len(item.source_name) > 15 else item.source_name
            
            print(f"{item.queue_id:<12} {doc_id_short:<25} {run_id_short:<12} "
                  f"{source_short:<15} {format_timestamp(item.failed_at):<19} "
                  f"{item.retry_count:<7} {error_short:<30}")


def list_dead_letter_items(dlq: DeadLetterQueue, run_id: Optional[str] = None, 
                          limit: int = 50, show_details: bool = False):
    """List items in the dead letter queue."""
    print("🔍 Listing dead letter queue items...")
    
    items = dlq.list_dead_letter_items(run_id=run_id, limit=limit)
    display_dead_letter_items(items, show_details)
    
    if len(items) == limit:
        print(f"\n⚠️  Showing first {limit} items. Use --limit to see more.")


def retry_dead_letter_item(dlq: DeadLetterQueue, queue_id: int):
    """Retry a specific dead letter item."""
    print(f"🔄 Attempting to retry queue item {queue_id}...")
    
    try:
        success = dlq.retry_from_dead_letter(queue_id)
        if success:
            print(f"✅ Successfully moved queue item {queue_id} back to processing queue")
        else:
            print(f"❌ Failed to retry queue item {queue_id} - item may not exist or already be active")
    except Exception as e:
        print(f"❌ Error retrying queue item {queue_id}: {str(e)}")


def retry_run_failures(dlq: DeadLetterQueue, run_id: str):
    """Retry all failed items for a specific run."""
    print(f"🔄 Attempting to retry all failures for run {run_id}...")
    
    # First, get the count of items
    items = dlq.list_dead_letter_items(run_id=run_id)
    if not items:
        print(f"✅ No dead letter items found for run {run_id}")
        return
    
    print(f"Found {len(items)} dead letter items for run {run_id}")
    
    # Confirm with user
    response = input(f"Are you sure you want to retry {len(items)} failed documents? (y/N): ")
    if response.lower() not in ['y', 'yes']:
        print("❌ Operation cancelled")
        return
    
    retry_count = 0
    for item in items:
        try:
            success = dlq.retry_from_dead_letter(item.queue_id)
            if success:
                retry_count += 1
                print(f"  ✅ Retried: {item.doc_id}")
            else:
                print(f"  ❌ Failed to retry: {item.doc_id}")
        except Exception as e:
            print(f"  ❌ Error retrying {item.doc_id}: {str(e)}")
    
    print(f"\n🎉 Successfully retried {retry_count}/{len(items)} documents")


def analyze_failure_patterns(processor: DeadLetterProcessor, run_id: Optional[str] = None):
    """Analyze and display failure patterns."""
    print("🔍 Analyzing failure patterns...")
    
    patterns = processor.analyze_failure_patterns(run_id=run_id)
    
    if not patterns:
        print("✅ No failure patterns found")
        return
    
    print(f"\n📊 FAILURE PATTERN ANALYSIS")
    if run_id:
        print(f"Run ID: {run_id}")
    print("=" * 60)
    
    for pattern in patterns:
        print(f"\n🚨 {pattern.error_type}")
        print(f"   Frequency: {pattern.frequency} occurrences")
        print(f"   Affected Documents: {pattern.affected_documents}")
        print(f"   First Seen: {format_timestamp(pattern.first_occurrence)}")
        print(f"   Last Seen: {format_timestamp(pattern.last_occurrence)}")
        
        if pattern.sample_error_messages:
            print("   Sample Errors:")
            for i, msg in enumerate(pattern.sample_error_messages[:3], 1):
                error_preview = (msg[:80] + "...") if len(msg) > 80 else msg
                print(f"     {i}. {error_preview}")
        
        if pattern.affected_sources:
            sources_str = ", ".join(pattern.affected_sources[:5])
            if len(pattern.affected_sources) > 5:
                sources_str += f" (and {len(pattern.affected_sources) - 5} more)"
            print(f"   Affected Sources: {sources_str}")


def purge_old_items(dlq: DeadLetterQueue, days: int):
    """Purge old dead letter items."""
    print(f"🗑️  Purging dead letter items older than {days} days...")
    
    # Confirm with user
    response = input(f"This will permanently delete dead letter items older than {days} days. Continue? (y/N): ")
    if response.lower() not in ['y', 'yes']:
        print("❌ Operation cancelled")
        return
    
    try:
        purged_count = dlq.purge_old_items(days_old=days)
        print(f"✅ Purged {purged_count} old dead letter items")
    except Exception as e:
        print(f"❌ Error purging items: {str(e)}")


def export_dead_letter_data(dlq: DeadLetterQueue, output_file: str, run_id: Optional[str] = None):
    """Export dead letter queue data to JSON."""
    print(f"📤 Exporting dead letter data to {output_file}...")
    
    items = dlq.list_dead_letter_items(run_id=run_id, limit=1000)  # Export up to 1000 items
    
    export_data = {
        'export_timestamp': datetime.now().isoformat(),
        'run_id_filter': run_id,
        'total_items': len(items),
        'items': [
            {
                'queue_id': item.queue_id,
                'doc_id': item.doc_id,
                'run_id': item.run_id,
                'source_name': item.source_name,
                'failed_at': item.failed_at,
                'retry_count': item.retry_count,
                'error_message': item.error_message,
                'error_details': item.error_details,
                'metadata': item.metadata
            }
            for item in items
        ]
    }
    
    with open(output_file, 'w') as f:
        json.dump(export_data, f, indent=2, default=str)
    
    print(f"✅ Exported {len(items)} dead letter items to {output_file}")


def main():
    """Main CLI entry point for dead letter queue management."""
    parser = argparse.ArgumentParser(
        description="Manage dead letter queue for failed documents",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # List all dead letter items
  python -m go_doc_go.cli.deadletter --list
  
  # List items for specific run with details
  python -m go_doc_go.cli.deadletter --list --run-id abc123 --details
  
  # Retry specific failed document
  python -m go_doc_go.cli.deadletter --retry 12345
  
  # Retry all failures for a run
  python -m go_doc_go.cli.deadletter --retry-run abc123
  
  # Analyze failure patterns
  python -m go_doc_go.cli.deadletter --analyze
  
  # Purge old items (older than 30 days)
  python -m go_doc_go.cli.deadletter --purge 30
  
  # Export dead letter data
  python -m go_doc_go.cli.deadletter --export failures.json
        """
    )
    
    parser.add_argument(
        '--config',
        help='Path to configuration file',
        default='config.yaml'
    )
    
    # Action arguments (mutually exclusive)
    action_group = parser.add_mutually_exclusive_group(required=True)
    action_group.add_argument(
        '--list',
        action='store_true',
        help='List dead letter items'
    )
    action_group.add_argument(
        '--retry',
        type=int,
        metavar='QUEUE_ID',
        help='Retry specific queue item by ID'
    )
    action_group.add_argument(
        '--retry-run',
        metavar='RUN_ID',
        help='Retry all failed items for a specific run'
    )
    action_group.add_argument(
        '--analyze',
        action='store_true',
        help='Analyze failure patterns'
    )
    action_group.add_argument(
        '--purge',
        type=int,
        metavar='DAYS',
        help='Purge items older than specified days'
    )
    action_group.add_argument(
        '--export',
        metavar='FILE',
        help='Export dead letter data to JSON file'
    )
    
    # Filter and display options
    parser.add_argument(
        '--run-id',
        help='Filter by specific run ID'
    )
    
    parser.add_argument(
        '--limit',
        type=int,
        default=50,
        help='Maximum number of items to display (default: 50)'
    )
    
    parser.add_argument(
        '--details',
        action='store_true',
        help='Show detailed information for each item'
    )
    
    args = parser.parse_args()
    
    try:
        # Load configuration
        config = Config(args.config)
        
        # Initialize components
        db = config.get_document_database()
        dlq = DeadLetterQueue(db)
        processor = DeadLetterProcessor(db)
        
        # Execute requested operation
        if args.list:
            list_dead_letter_items(dlq, args.run_id, args.limit, args.details)
        
        elif args.retry:
            retry_dead_letter_item(dlq, args.retry)
        
        elif args.retry_run:
            retry_run_failures(dlq, args.retry_run)
        
        elif args.analyze:
            analyze_failure_patterns(processor, args.run_id)
        
        elif args.purge:
            purge_old_items(dlq, args.purge)
        
        elif args.export:
            export_dead_letter_data(dlq, args.export, args.run_id)
        
        db.close()
        
    except Exception as e:
        print(f"❌ Error: {str(e)}", file=sys.stderr)
        sys.exit(1)


if __name__ == '__main__':
    main()