#!/usr/bin/env python
"""
Command-line interface for managing the work queue.
"""

import argparse
import json
import logging
import sys
from pathlib import Path
from typing import Optional

from go_doc_go.config import Config
from go_doc_go.queue.migrations import create_schema, check_schema_exists, validate_schema
from go_doc_go.queue.work_queue import WorkQueue, RunCoordinator

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)


def cmd_init_schema(args):
    """Initialize the queue schema."""
    config = Config(args.config)
    db = config.get_document_database()
    db.initialize()
    
    if check_schema_exists(db):
        if args.force:
            logger.warning("Dropping and recreating schema...")
            create_schema(db, force=True)
        else:
            logger.info("Schema already exists. Use --force to recreate.")
            return 1
    else:
        logger.info("Creating queue schema...")
        create_schema(db)
    
    if validate_schema(db):
        logger.info("Schema initialized successfully")
        return 0
    else:
        logger.error("Schema validation failed")
        return 1


def cmd_status(args):
    """Show queue status."""
    config = Config(args.config)
    db = config.get_document_database()
    db.initialize()
    
    if not check_schema_exists(db):
        logger.error("Queue schema not initialized. Run 'init-schema' first.")
        return 1
    
    # Get run ID from config
    coordinator = RunCoordinator(db)
    run_id = RunCoordinator.get_run_id_from_config(config.config)
    
    if args.run_id:
        run_id = args.run_id
    
    print(f"\n📊 Queue Status for Run: {run_id}")
    print("=" * 50)
    
    # Get run info
    run_info = db.execute("""
        SELECT status, created_at, worker_count,
               documents_queued, documents_processed, documents_failed
        FROM processing_runs
        WHERE run_id = %s
    """, (run_id,))
    
    if not run_info:
        print(f"Run {run_id} not found")
        return 1
    
    print(f"Status: {run_info['status']}")
    print(f"Created: {run_info['created_at']}")
    print(f"Workers: {run_info['worker_count']}")
    print(f"\n📈 Progress:")
    print(f"  Queued: {run_info['documents_queued']}")
    print(f"  Processed: {run_info['documents_processed']}")
    print(f"  Failed: {run_info['documents_failed']}")
    
    # Get queue breakdown
    queue = WorkQueue(db, "cli_status")
    status = queue.get_queue_status(run_id)
    
    print(f"\n📋 Queue Breakdown:")
    print(f"  Pending: {status.get('pending', 0)}")
    print(f"  Processing: {status.get('processing', 0)}")
    print(f"  Completed: {status.get('completed', 0)}")
    print(f"  Failed: {status.get('failed', 0)}")
    print(f"  Retry: {status.get('retry', 0)}")
    
    # Get active workers
    workers = db.execute("""
        SELECT worker_id, hostname, last_heartbeat, 
               documents_processed, documents_failed
        FROM run_workers
        WHERE run_id = %s AND status = 'active'
        ORDER BY last_heartbeat DESC
    """, (run_id,))
    
    if workers:
        print(f"\n👷 Active Workers:")
        for worker in workers:
            print(f"  • {worker['worker_id']} on {worker['hostname']}")
            print(f"    Processed: {worker['documents_processed']}, Failed: {worker['documents_failed']}")
    
    return 0


def cmd_list_runs(args):
    """List all processing runs."""
    config = Config(args.config)
    db = config.get_document_database()
    db.initialize()
    
    if not check_schema_exists(db):
        logger.error("Queue schema not initialized. Run 'init-schema' first.")
        return 1
    
    runs = db.execute("""
        SELECT run_id, status, created_at, 
               documents_queued, documents_processed, documents_failed
        FROM processing_runs
        ORDER BY created_at DESC
        LIMIT 20
    """)
    
    if not runs:
        print("No processing runs found")
        return 0
    
    print("\n📚 Processing Runs:")
    print("=" * 80)
    
    for run in runs:
        status_icon = {
            'active': '🟢',
            'completed': '✅',
            'failed': '❌',
            'abandoned': '⚠️'
        }.get(run['status'], '❓')
        
        print(f"{status_icon} {run['run_id']} - {run['status']}")
        print(f"   Created: {run['created_at']}")
        print(f"   Progress: {run['documents_processed']}/{run['documents_queued']} (Failed: {run['documents_failed']})")
        print()
    
    return 0


def cmd_add_document(args):
    """Add a document to the queue."""
    config = Config(args.config)
    db = config.get_document_database()
    db.initialize()
    
    if not check_schema_exists(db):
        logger.error("Queue schema not initialized. Run 'init-schema' first.")
        return 1
    
    # Get or create run
    coordinator = RunCoordinator(db)
    run_id = args.run_id or RunCoordinator.get_run_id_from_config(config.config)
    coordinator.ensure_run_exists(run_id, config.config)
    
    # Add document
    queue = WorkQueue(db, "cli_add")
    queue_id = queue.add_document(
        doc_id=args.doc_id,
        source_name=args.source,
        run_id=run_id,
        metadata=json.loads(args.metadata) if args.metadata else None
    )
    
    print(f"✅ Added document {args.doc_id} to queue (queue_id: {queue_id})")
    return 0


def cmd_reclaim_stale(args):
    """Reclaim stale work items."""
    config = Config(args.config)
    db = config.get_document_database()
    db.initialize()
    
    if not check_schema_exists(db):
        logger.error("Queue schema not initialized. Run 'init-schema' first.")
        return 1
    
    timeout = args.timeout
    result = db.execute(f"SELECT reclaim_stale_work({timeout})")
    
    count = result.get('reclaim_stale_work', 0)
    print(f"♻️  Reclaimed {count} stale work items")
    return 0


def main():
    """Main CLI entry point."""
    parser = argparse.ArgumentParser(description="Work Queue Management")
    parser.add_argument('--config', '-c', default='config.yaml',
                       help='Configuration file path')
    
    subparsers = parser.add_subparsers(dest='command', help='Commands')
    
    # init-schema command
    init_parser = subparsers.add_parser('init-schema', help='Initialize queue schema')
    init_parser.add_argument('--force', action='store_true',
                            help='Drop and recreate schema')
    
    # status command
    status_parser = subparsers.add_parser('status', help='Show queue status')
    status_parser.add_argument('--run-id', help='Specific run ID')
    
    # list-runs command
    list_parser = subparsers.add_parser('list-runs', help='List processing runs')
    
    # add-document command
    add_parser = subparsers.add_parser('add-document', help='Add document to queue')
    add_parser.add_argument('doc_id', help='Document ID')
    add_parser.add_argument('source', help='Source name')
    add_parser.add_argument('--run-id', help='Run ID (uses config hash if not specified)')
    add_parser.add_argument('--metadata', help='JSON metadata')
    
    # reclaim-stale command
    reclaim_parser = subparsers.add_parser('reclaim-stale', help='Reclaim stale work')
    reclaim_parser.add_argument('--timeout', type=int, default=300,
                               help='Timeout in seconds (default: 300)')
    
    args = parser.parse_args()
    
    if not args.command:
        parser.print_help()
        return 1
    
    # Execute command
    commands = {
        'init-schema': cmd_init_schema,
        'status': cmd_status,
        'list-runs': cmd_list_runs,
        'add-document': cmd_add_document,
        'reclaim-stale': cmd_reclaim_stale
    }
    
    try:
        return commands[args.command](args)
    except Exception as e:
        logger.error(f"Command failed: {e}")
        import traceback
        traceback.print_exc()
        return 1


if __name__ == '__main__':
    sys.exit(main())