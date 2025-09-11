#!/usr/bin/env python3
"""
CLI interface for monitoring distributed processing runs.
"""

import argparse
import json
import sys
import time
from datetime import datetime, timedelta
from typing import Optional

from ..config import Config
from ..queue.monitoring import MetricsCollector, AlertManager


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


def format_timestamp(timestamp: Optional[str]) -> str:
    """Format timestamp for display."""
    if not timestamp:
        return "N/A"
    try:
        dt = datetime.fromisoformat(timestamp.replace('Z', '+00:00'))
        return dt.strftime("%Y-%m-%d %H:%M:%S")
    except Exception:
        return str(timestamp)


def display_run_metrics(metrics_collector: MetricsCollector, run_id: str):
    """Display comprehensive run metrics."""
    print(f"\n{'='*60}")
    print(f"RUN METRICS: {run_id}")
    print(f"{'='*60}")
    
    run_metrics = metrics_collector.get_run_metrics(run_id)
    
    if not run_metrics:
        print("❌ Run not found or no metrics available")
        return
    
    # Basic run information
    print(f"📊 Run Status: {run_metrics.status}")
    print(f"📅 Started: {format_timestamp(run_metrics.start_time)}")
    print(f"⏱️  Duration: {format_duration(run_metrics.duration_seconds)}")
    
    # Document processing statistics
    print(f"\n📄 DOCUMENT PROCESSING:")
    total_docs = (run_metrics.documents_completed + 
                  run_metrics.documents_failed + 
                  run_metrics.documents_processing + 
                  run_metrics.documents_pending)
    
    print(f"   Total Documents: {total_docs}")
    print(f"   ✅ Completed: {run_metrics.documents_completed}")
    print(f"   🔄 Processing: {run_metrics.documents_processing}")
    print(f"   ⏳ Pending: {run_metrics.documents_pending}")
    print(f"   ❌ Failed: {run_metrics.documents_failed}")
    
    if total_docs > 0:
        completion_rate = (run_metrics.documents_completed / total_docs) * 100
        print(f"   📈 Completion Rate: {completion_rate:.1f}%")
    
    # Worker information
    print(f"\n👥 WORKER STATUS:")
    print(f"   Active Workers: {run_metrics.active_workers}")
    print(f"   Total Workers (ever): {run_metrics.total_workers}")
    
    # Performance metrics
    if run_metrics.documents_completed > 0:
        docs_per_second = run_metrics.documents_completed / max(run_metrics.duration_seconds, 1)
        print(f"\n⚡ PERFORMANCE:")
        print(f"   Processing Rate: {docs_per_second:.2f} docs/second")
        print(f"   Avg Processing Time: {run_metrics.avg_processing_time_seconds:.2f}s per document")


def display_worker_metrics(metrics_collector: MetricsCollector, run_id: str, limit: int = 10):
    """Display active worker metrics."""
    print(f"\n{'='*60}")
    print(f"WORKER METRICS: {run_id} (showing top {limit})")
    print(f"{'='*60}")
    
    worker_metrics = metrics_collector.get_worker_metrics(run_id, limit=limit)
    
    if not worker_metrics:
        print("❌ No active workers found")
        return
    
    print(f"{'Worker ID':<20} {'Status':<12} {'Processed':<10} {'Failed':<7} {'Last Seen':<19} {'Processing Time':<15}")
    print("-" * 100)
    
    for worker in worker_metrics:
        last_seen = format_timestamp(worker.last_heartbeat)
        if worker.last_heartbeat:
            # Calculate time since last heartbeat
            try:
                last_dt = datetime.fromisoformat(worker.last_heartbeat.replace('Z', '+00:00'))
                now = datetime.now(last_dt.tzinfo) if last_dt.tzinfo else datetime.now()
                time_diff = (now - last_dt).total_seconds()
                if time_diff > 300:  # 5 minutes
                    status = "🔴 STALE"
                elif time_diff > 60:  # 1 minute
                    status = "🟡 SLOW"
                else:
                    status = "🟢 ACTIVE"
            except Exception:
                status = "❓ UNKNOWN"
        else:
            status = "❓ UNKNOWN"
        
        avg_time = format_duration(worker.avg_processing_time_seconds) if worker.avg_processing_time_seconds else "N/A"
        
        print(f"{worker.worker_id:<20} {status:<12} {worker.documents_processed:<10} "
              f"{worker.documents_failed:<7} {last_seen:<19} {avg_time:<15}")


def display_queue_health(metrics_collector: MetricsCollector, run_id: str):
    """Display queue health metrics."""
    print(f"\n{'='*60}")
    print(f"QUEUE HEALTH: {run_id}")
    print(f"{'='*60}")
    
    health = metrics_collector.get_queue_health(run_id)
    
    if not health:
        print("❌ No queue health data available")
        return
    
    # Overall health status
    print(f"🏥 Overall Health: {health.overall_health}")
    
    # Queue depth analysis
    print(f"\n📊 QUEUE DEPTH:")
    print(f"   Pending: {health.pending_count}")
    print(f"   Processing: {health.processing_count}")
    print(f"   Stale Documents: {health.stale_documents}")
    
    # Failure analysis
    failure_rate = (health.failed_count / max(health.total_documents, 1)) * 100
    print(f"\n❌ FAILURE ANALYSIS:")
    print(f"   Failed Documents: {health.failed_count}")
    print(f"   Failure Rate: {failure_rate:.1f}%")
    
    # Performance indicators
    print(f"\n⚡ PERFORMANCE:")
    print(f"   Avg Processing Time: {format_duration(health.avg_processing_time)}")
    print(f"   Throughput: {health.throughput_docs_per_minute:.1f} docs/minute")


def monitor_live(metrics_collector: MetricsCollector, run_id: str, refresh_interval: int = 30):
    """Live monitoring with periodic updates."""
    print(f"🔴 LIVE MONITORING: {run_id}")
    print(f"Refreshing every {refresh_interval} seconds. Press Ctrl+C to stop.")
    
    try:
        while True:
            # Clear screen (works on most terminals)
            print("\033[2J\033[H")
            
            print(f"🕒 Last updated: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
            
            display_run_metrics(metrics_collector, run_id)
            display_worker_metrics(metrics_collector, run_id, limit=5)
            display_queue_health(metrics_collector, run_id)
            
            print(f"\n⏰ Next update in {refresh_interval} seconds... (Ctrl+C to stop)")
            time.sleep(refresh_interval)
            
    except KeyboardInterrupt:
        print("\n\n👋 Live monitoring stopped.")


def check_alerts(alert_manager: AlertManager, run_id: Optional[str] = None):
    """Check and display current alerts."""
    print(f"\n{'='*60}")
    print("CURRENT ALERTS")
    print(f"{'='*60}")
    
    if run_id:
        alerts = alert_manager.check_run_alerts(run_id)
        if not alerts:
            print(f"✅ No alerts for run {run_id}")
            return
    else:
        # Check alerts for all runs (would need method in AlertManager)
        print("❓ Global alert checking not yet implemented")
        return
    
    for alert in alerts:
        severity_icon = {
            'critical': '🔴',
            'warning': '🟡',
            'info': '🔵'
        }.get(alert.severity, '❓')
        
        print(f"{severity_icon} {alert.severity.upper()}: {alert.message}")
        print(f"   Metric: {alert.metric_name} = {alert.metric_value}")
        if alert.threshold:
            print(f"   Threshold: {alert.threshold}")
        print()


def main():
    """Main CLI entry point for monitoring."""
    parser = argparse.ArgumentParser(
        description="Monitor distributed processing runs",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Show run summary
  python -m go_doc_go.cli.monitor --run-id abc123 --summary
  
  # Show detailed metrics
  python -m go_doc_go.cli.monitor --run-id abc123 --details
  
  # Live monitoring (refreshes every 30 seconds)
  python -m go_doc_go.cli.monitor --run-id abc123 --live
  
  # Check alerts
  python -m go_doc_go.cli.monitor --run-id abc123 --alerts
  
  # Export metrics to JSON
  python -m go_doc_go.cli.monitor --run-id abc123 --export metrics.json
        """
    )
    
    parser.add_argument(
        '--config',
        help='Path to configuration file',
        default='config.yaml'
    )
    
    parser.add_argument(
        '--run-id',
        help='Processing run ID to monitor',
        required=True
    )
    
    # Display modes
    display_group = parser.add_mutually_exclusive_group()
    display_group.add_argument(
        '--summary',
        action='store_true',
        help='Show run summary (default)'
    )
    display_group.add_argument(
        '--details',
        action='store_true',
        help='Show detailed metrics including worker information'
    )
    display_group.add_argument(
        '--live',
        action='store_true',
        help='Live monitoring mode with periodic updates'
    )
    display_group.add_argument(
        '--alerts',
        action='store_true',
        help='Check and display current alerts'
    )
    display_group.add_argument(
        '--export',
        help='Export metrics to JSON file'
    )
    
    parser.add_argument(
        '--refresh-interval',
        type=int,
        default=30,
        help='Refresh interval for live monitoring (seconds)'
    )
    
    parser.add_argument(
        '--worker-limit',
        type=int,
        default=10,
        help='Maximum number of workers to display'
    )
    
    args = parser.parse_args()
    
    try:
        # Load configuration
        config = Config(args.config)
        
        # Initialize components
        db = config.get_document_database()
        metrics_collector = MetricsCollector(db)
        alert_manager = AlertManager(metrics_collector)
        
        # Execute requested operation
        if args.live:
            monitor_live(metrics_collector, args.run_id, args.refresh_interval)
        elif args.alerts:
            check_alerts(alert_manager, args.run_id)
        elif args.export:
            # Export metrics to JSON
            run_metrics = metrics_collector.get_run_metrics(args.run_id)
            worker_metrics = metrics_collector.get_worker_metrics(args.run_id)
            queue_health = metrics_collector.get_queue_health(args.run_id)
            
            export_data = {
                'run_id': args.run_id,
                'timestamp': datetime.now().isoformat(),
                'run_metrics': run_metrics.__dict__ if run_metrics else None,
                'worker_metrics': [w.__dict__ for w in worker_metrics] if worker_metrics else [],
                'queue_health': queue_health.__dict__ if queue_health else None
            }
            
            with open(args.export, 'w') as f:
                json.dump(export_data, f, indent=2, default=str)
            
            print(f"✅ Metrics exported to {args.export}")
        else:
            # Default: show summary, or details if requested
            display_run_metrics(metrics_collector, args.run_id)
            
            if args.details:
                display_worker_metrics(metrics_collector, args.run_id, args.worker_limit)
                display_queue_health(metrics_collector, args.run_id)
        
        db.close()
        
    except Exception as e:
        print(f"❌ Error: {str(e)}", file=sys.stderr)
        sys.exit(1)


if __name__ == '__main__':
    main()