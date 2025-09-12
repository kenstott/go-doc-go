"""
Advanced monitoring and metrics collection for distributed work queue system.
"""

import logging
import time
from datetime import datetime, timedelta
from typing import Dict, Any, List, Optional
from dataclasses import dataclass, asdict

logger = logging.getLogger(__name__)


@dataclass
class WorkerMetrics:
    """Metrics for individual worker performance."""
    worker_id: str
    hostname: str
    start_time: datetime
    last_heartbeat: datetime
    documents_processed: int = 0
    documents_failed: int = 0
    elements_created: int = 0
    relationships_created: int = 0
    links_discovered: int = 0
    avg_processing_time: float = 0.0
    current_document: Optional[str] = None
    total_processing_time: float = 0.0

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization."""
        data = asdict(self)
        # Convert datetime objects to strings
        data['start_time'] = self.start_time.isoformat() if self.start_time else None
        data['last_heartbeat'] = self.last_heartbeat.isoformat() if self.last_heartbeat else None
        return data


@dataclass
class RunMetrics:
    """Metrics for processing run performance."""
    run_id: str
    status: str
    start_time: datetime
    end_time: Optional[datetime] = None
    documents_queued: int = 0
    documents_processed: int = 0
    documents_failed: int = 0
    documents_in_dead_letter: int = 0
    active_workers: int = 0
    peak_workers: int = 0
    cross_doc_relationships: int = 0
    total_elements: int = 0
    avg_document_size: float = 0.0
    throughput_docs_per_second: float = 0.0

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization."""
        data = asdict(self)
        data['start_time'] = self.start_time.isoformat() if self.start_time else None
        data['end_time'] = self.end_time.isoformat() if self.end_time else None
        return data


@dataclass
class QueueHealthMetrics:
    """Health metrics for the work queue system."""
    timestamp: datetime
    total_pending: int
    total_processing: int
    total_completed: int
    total_failed: int
    total_retry: int
    total_dead_letter: int
    oldest_pending_age_seconds: float = 0.0
    average_processing_time: float = 0.0
    stale_work_items: int = 0
    worker_count: int = 0

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization.""" 
        data = asdict(self)
        data['timestamp'] = self.timestamp.isoformat()
        return data


class MetricsCollector:
    """Collects and aggregates metrics from the distributed work queue system."""
    
    def __init__(self, db):
        """
        Initialize metrics collector.
        
        Args:
            db: Database adapter instance
        """
        self.db = db
        
    def get_worker_metrics(self, run_id: str) -> List[WorkerMetrics]:
        """
        Get detailed metrics for all workers in a run.
        
        Args:
            run_id: Processing run ID
            
        Returns:
            List of worker metrics
        """
        try:
            query = """
                SELECT 
                    worker_id,
                    hostname,
                    start_time,
                    last_heartbeat,
                    documents_processed,
                    documents_failed,
                    current_document,
                    total_processing_time
                FROM run_workers 
                WHERE run_id = %s
                ORDER BY start_time
            """
            
            workers = self.db.execute(query, (run_id,))
            
            worker_metrics = []
            for worker in workers:
                # Get additional metrics from document_queue
                queue_query = """
                    SELECT 
                        COUNT(*) as total_processed,
                        AVG(EXTRACT(EPOCH FROM (completed_at - claimed_at))) as avg_time,
                        SUM(CASE WHEN metadata->>'elements_created' IS NOT NULL 
                            THEN (metadata->>'elements_created')::int ELSE 0 END) as elements,
                        SUM(CASE WHEN metadata->>'relationships_created' IS NOT NULL 
                            THEN (metadata->>'relationships_created')::int ELSE 0 END) as relationships,
                        SUM(CASE WHEN metadata->>'links_discovered' IS NOT NULL 
                            THEN (metadata->>'links_discovered')::int ELSE 0 END) as links
                    FROM document_queue 
                    WHERE run_id = %s AND worker_id = %s AND status = 'completed'
                """
                
                queue_stats = self.db.execute(queue_query, (run_id, worker['worker_id']))
                stats = queue_stats[0] if queue_stats else {}
                
                metrics = WorkerMetrics(
                    worker_id=worker['worker_id'],
                    hostname=worker.get('hostname', 'unknown'),
                    start_time=worker['start_time'],
                    last_heartbeat=worker['last_heartbeat'],
                    documents_processed=worker.get('documents_processed', 0),
                    documents_failed=worker.get('documents_failed', 0),
                    elements_created=stats.get('elements', 0) or 0,
                    relationships_created=stats.get('relationships', 0) or 0,
                    links_discovered=stats.get('links', 0) or 0,
                    avg_processing_time=float(stats.get('avg_time', 0) or 0),
                    current_document=worker.get('current_document'),
                    total_processing_time=float(worker.get('total_processing_time', 0) or 0)
                )
                
                worker_metrics.append(metrics)
                
            logger.debug(f"Retrieved metrics for {len(worker_metrics)} workers in run {run_id}")
            return worker_metrics
            
        except Exception as e:
            logger.error(f"Error getting worker metrics for run {run_id}: {str(e)}")
            return []
    
    def get_run_metrics(self, run_id: str) -> Optional[RunMetrics]:
        """
        Get comprehensive metrics for a processing run.
        
        Args:
            run_id: Processing run ID
            
        Returns:
            Run metrics or None if not found
        """
        try:
            # Get basic run info
            run_query = """
                SELECT status, created_at, updated_at, config,
                       documents_queued, documents_processed, documents_failed
                FROM processing_runs 
                WHERE run_id = %s
            """
            
            run_data = self.db.execute(run_query, (run_id,))
            if not run_data:
                return None
                
            run = run_data[0]
            
            # Get queue statistics
            queue_query = """
                SELECT 
                    COUNT(*) as total_docs,
                    COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending,
                    COUNT(CASE WHEN status = 'processing' THEN 1 END) as processing,
                    COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed,
                    COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
                    COUNT(CASE WHEN status = 'dead_letter' THEN 1 END) as dead_letter,
                    AVG(CASE WHEN completed_at IS NOT NULL AND claimed_at IS NOT NULL 
                        THEN EXTRACT(EPOCH FROM (completed_at - claimed_at)) END) as avg_processing_time,
                    SUM(CASE WHEN metadata->>'elements_created' IS NOT NULL 
                        THEN (metadata->>'elements_created')::int ELSE 0 END) as total_elements
                FROM document_queue 
                WHERE run_id = %s
            """
            
            queue_stats = self.db.execute(queue_query, (run_id,))
            stats = queue_stats[0] if queue_stats else {}
            
            # Get worker count
            worker_query = """
                SELECT COUNT(DISTINCT worker_id) as worker_count,
                       MAX(documents_processed + documents_failed) as peak_activity
                FROM run_workers 
                WHERE run_id = %s
            """
            
            worker_stats = self.db.execute(worker_query, (run_id,))
            worker_info = worker_stats[0] if worker_stats else {}
            
            # Calculate throughput if run is completed
            throughput = 0.0
            if run['status'] == 'completed' and run['updated_at'] and run['created_at']:
                duration = (run['updated_at'] - run['created_at']).total_seconds()
                if duration > 0:
                    throughput = (stats.get('completed', 0) or 0) / duration
            
            metrics = RunMetrics(
                run_id=run_id,
                status=run['status'],
                start_time=run['created_at'],
                end_time=run['updated_at'] if run['status'] == 'completed' else None,
                documents_queued=run.get('documents_queued', 0) or 0,
                documents_processed=stats.get('completed', 0) or 0,
                documents_failed=stats.get('failed', 0) or 0,
                documents_in_dead_letter=stats.get('dead_letter', 0) or 0,
                active_workers=worker_info.get('worker_count', 0) or 0,
                peak_workers=worker_info.get('worker_count', 0) or 0,  # Simplified
                total_elements=stats.get('total_elements', 0) or 0,
                avg_document_size=0.0,  # Would need content size tracking
                throughput_docs_per_second=throughput
            )
            
            logger.debug(f"Retrieved run metrics for {run_id}")
            return metrics
            
        except Exception as e:
            logger.error(f"Error getting run metrics for {run_id}: {str(e)}")
            return None
    
    def get_queue_health_metrics(self) -> QueueHealthMetrics:
        """
        Get current health metrics for the entire queue system.
        
        Returns:
            Queue health metrics
        """
        try:
            # Get overall queue statistics
            health_query = """
                SELECT 
                    COUNT(*) as total_items,
                    COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending,
                    COUNT(CASE WHEN status = 'processing' THEN 1 END) as processing,
                    COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed,
                    COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
                    COUNT(CASE WHEN status = 'retry' THEN 1 END) as retry,
                    COUNT(CASE WHEN status = 'dead_letter' THEN 1 END) as dead_letter,
                    MIN(CASE WHEN status = 'pending' 
                        THEN EXTRACT(EPOCH FROM (NOW() - created_at)) END) as oldest_pending,
                    AVG(CASE WHEN completed_at IS NOT NULL AND claimed_at IS NOT NULL
                        THEN EXTRACT(EPOCH FROM (completed_at - claimed_at)) END) as avg_processing
                FROM document_queue
            """
            
            health_data = self.db.execute(health_query)
            health = health_data[0] if health_data else {}
            
            # Get stale work items (processing for too long)
            stale_threshold = 600  # 10 minutes
            stale_query = """
                SELECT COUNT(*) as stale_count
                FROM document_queue 
                WHERE status = 'processing' 
                AND claimed_at < NOW() - INTERVAL %s
            """
            
            stale_data = self.db.execute(stale_query, (f"{stale_threshold} seconds",))
            stale_count = stale_data[0]['stale_count'] if stale_data else 0
            
            # Get active worker count
            worker_query = """
                SELECT COUNT(DISTINCT worker_id) as active_workers
                FROM run_workers 
                WHERE last_heartbeat > NOW() - INTERVAL '60 seconds'
            """
            
            worker_data = self.db.execute(worker_query)
            worker_count = worker_data[0]['active_workers'] if worker_data else 0
            
            metrics = QueueHealthMetrics(
                timestamp=datetime.now(),
                total_pending=health.get('pending', 0) or 0,
                total_processing=health.get('processing', 0) or 0,
                total_completed=health.get('completed', 0) or 0,
                total_failed=health.get('failed', 0) or 0,
                total_retry=health.get('retry', 0) or 0,
                total_dead_letter=health.get('dead_letter', 0) or 0,
                oldest_pending_age_seconds=float(health.get('oldest_pending', 0) or 0),
                average_processing_time=float(health.get('avg_processing', 0) or 0),
                stale_work_items=stale_count or 0,
                worker_count=worker_count or 0
            )
            
            logger.debug("Retrieved queue health metrics")
            return metrics
            
        except Exception as e:
            logger.error(f"Error getting queue health metrics: {str(e)}")
            return QueueHealthMetrics(
                timestamp=datetime.now(),
                total_pending=0, total_processing=0, total_completed=0,
                total_failed=0, total_retry=0, total_dead_letter=0
            )
    
    def get_historical_metrics(self, hours: int = 24) -> List[Dict[str, Any]]:
        """
        Get historical metrics over time.
        
        Args:
            hours: Number of hours of history to retrieve
            
        Returns:
            List of historical metric snapshots
        """
        try:
            # Get metrics snapshots from the past N hours
            # This would require a metrics history table in production
            history_query = """
                SELECT 
                    DATE_TRUNC('hour', created_at) as hour_bucket,
                    COUNT(*) as documents_queued,
                    COUNT(CASE WHEN completed_at IS NOT NULL THEN 1 END) as documents_completed,
                    COUNT(CASE WHEN status = 'failed' THEN 1 END) as documents_failed,
                    AVG(CASE WHEN completed_at IS NOT NULL AND claimed_at IS NOT NULL
                        THEN EXTRACT(EPOCH FROM (completed_at - claimed_at)) END) as avg_processing_time
                FROM document_queue 
                WHERE created_at > NOW() - INTERVAL %s
                GROUP BY DATE_TRUNC('hour', created_at)
                ORDER BY hour_bucket
            """
            
            history_data = self.db.execute(history_query, (f"{hours} hours",))
            
            historical_metrics = []
            for row in history_data:
                metrics = {
                    'timestamp': row['hour_bucket'].isoformat(),
                    'documents_queued': row['documents_queued'] or 0,
                    'documents_completed': row['documents_completed'] or 0,
                    'documents_failed': row['documents_failed'] or 0,
                    'avg_processing_time': float(row['avg_processing_time'] or 0)
                }
                historical_metrics.append(metrics)
            
            logger.debug(f"Retrieved {len(historical_metrics)} hours of historical metrics")
            return historical_metrics
            
        except Exception as e:
            logger.error(f"Error getting historical metrics: {str(e)}")
            return []


class AlertManager:
    """Manages alerting based on metrics thresholds."""
    
    def __init__(self, metrics_collector: MetricsCollector):
        """
        Initialize alert manager.
        
        Args:
            metrics_collector: MetricsCollector instance
        """
        self.metrics = metrics_collector
        
        # Default alert thresholds
        self.thresholds = {
            'stale_work_threshold': 600,  # 10 minutes
            'max_failed_percentage': 0.1,  # 10%
            'max_dead_letter_items': 100,
            'min_worker_count': 1,
            'max_processing_time': 1800,  # 30 minutes
            'max_queue_age': 3600  # 1 hour
        }
    
    def check_alerts(self) -> List[Dict[str, Any]]:
        """
        Check all alert conditions and return active alerts.
        
        Returns:
            List of active alerts
        """
        alerts = []
        
        try:
            health = self.metrics.get_queue_health_metrics()
            
            # Check for stale work
            if health.stale_work_items > 0:
                alerts.append({
                    'type': 'stale_work',
                    'severity': 'warning',
                    'message': f"{health.stale_work_items} documents stuck in processing state",
                    'value': health.stale_work_items,
                    'threshold': 0
                })
            
            # Check failure rate
            total_processed = health.total_completed + health.total_failed
            if total_processed > 0:
                failure_rate = health.total_failed / total_processed
                if failure_rate > self.thresholds['max_failed_percentage']:
                    alerts.append({
                        'type': 'high_failure_rate',
                        'severity': 'critical',
                        'message': f"Failure rate {failure_rate:.1%} exceeds threshold {self.thresholds['max_failed_percentage']:.1%}",
                        'value': failure_rate,
                        'threshold': self.thresholds['max_failed_percentage']
                    })
            
            # Check dead letter queue
            if health.total_dead_letter > self.thresholds['max_dead_letter_items']:
                alerts.append({
                    'type': 'dead_letter_overflow',
                    'severity': 'warning',
                    'message': f"Dead letter queue has {health.total_dead_letter} items",
                    'value': health.total_dead_letter,
                    'threshold': self.thresholds['max_dead_letter_items']
                })
            
            # Check worker count
            if health.worker_count < self.thresholds['min_worker_count']:
                alerts.append({
                    'type': 'low_worker_count',
                    'severity': 'critical',
                    'message': f"Only {health.worker_count} active workers",
                    'value': health.worker_count,
                    'threshold': self.thresholds['min_worker_count']
                })
            
            # Check queue age
            if health.oldest_pending_age_seconds > self.thresholds['max_queue_age']:
                alerts.append({
                    'type': 'old_pending_documents',
                    'severity': 'warning', 
                    'message': f"Oldest pending document is {health.oldest_pending_age_seconds/60:.1f} minutes old",
                    'value': health.oldest_pending_age_seconds,
                    'threshold': self.thresholds['max_queue_age']
                })
            
            logger.debug(f"Found {len(alerts)} active alerts")
            return alerts
            
        except Exception as e:
            logger.error(f"Error checking alerts: {str(e)}")
            return [{
                'type': 'monitoring_error',
                'severity': 'critical',
                'message': f"Monitoring system error: {str(e)}",
                'value': None,
                'threshold': None
            }]