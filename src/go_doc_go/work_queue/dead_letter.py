"""
Dead Letter Queue implementation for handling permanently failed documents.
"""

import logging
from datetime import datetime, timedelta
from typing import Dict, Any, List, Optional
from dataclasses import dataclass

logger = logging.getLogger(__name__)


@dataclass
class DeadLetterItem:
    """Represents a document in the dead letter queue."""
    queue_id: int
    doc_id: str
    run_id: str
    source_name: str
    failure_reason: str
    failure_count: int
    first_failed_at: datetime
    last_failed_at: datetime
    original_metadata: Dict[str, Any]
    error_history: List[Dict[str, Any]]

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization."""
        return {
            'queue_id': self.queue_id,
            'doc_id': self.doc_id,
            'run_id': self.run_id,
            'source_name': self.source_name,
            'failure_reason': self.failure_reason,
            'failure_count': self.failure_count,
            'first_failed_at': self.first_failed_at.isoformat() if self.first_failed_at else None,
            'last_failed_at': self.last_failed_at.isoformat() if self.last_failed_at else None,
            'original_metadata': self.original_metadata,
            'error_history': self.error_history
        }


class DeadLetterQueue:
    """Manages dead letter queue for permanently failed documents."""
    
    def __init__(self, db, max_retries: int = 3):
        """
        Initialize dead letter queue manager.
        
        Args:
            db: Database adapter instance
            max_retries: Maximum retry attempts before dead lettering
        """
        self.db = db
        self.max_retries = max_retries
        
    def move_to_dead_letter(self, queue_id: int, failure_reason: str) -> bool:
        """
        Move a failed document to the dead letter queue.
        
        Args:
            queue_id: Queue item ID
            failure_reason: Reason for permanent failure
            
        Returns:
            True if successfully moved to dead letter queue
        """
        try:
            # Get current document info
            doc_query = """
                SELECT doc_id, run_id, source_name, retry_count, 
                       metadata, created_at, error_message
                FROM document_queue 
                WHERE queue_id = %s
            """
            
            doc_result = self.db.execute(doc_query, (queue_id,))
            if not doc_result:
                logger.warning(f"Document with queue_id {queue_id} not found for dead lettering")
                return False
                
            doc = doc_result[0]
            
            # Get error history
            error_history = self._get_error_history(queue_id)
            
            # Update document status to dead_letter
            update_query = """
                UPDATE document_queue 
                SET status = 'dead_letter',
                    updated_at = %s,
                    error_message = %s,
                    metadata = COALESCE(metadata, '{}')::jsonb || %s::jsonb
                WHERE queue_id = %s
            """
            
            dead_letter_metadata = {
                'dead_lettered_at': datetime.now().isoformat(),
                'dead_letter_reason': failure_reason,
                'final_retry_count': doc.get('retry_count', 0),
                'error_history_count': len(error_history)
            }
            
            self.db.execute(update_query, (
                datetime.now(),
                failure_reason,
                dead_letter_metadata,
                queue_id
            ))
            
            logger.warning(
                f"Moved document {doc['doc_id']} to dead letter queue. "
                f"Reason: {failure_reason}, Retry count: {doc.get('retry_count', 0)}"
            )
            
            return True
            
        except Exception as e:
            logger.error(f"Error moving document {queue_id} to dead letter queue: {str(e)}")
            return False
    
    def _get_error_history(self, queue_id: int) -> List[Dict[str, Any]]:
        """
        Get error history for a document from logs or metadata.
        
        Args:
            queue_id: Queue item ID
            
        Returns:
            List of error history entries
        """
        try:
            # In a full implementation, this would query a separate error_log table
            # For now, we'll extract from metadata if available
            metadata_query = """
                SELECT metadata 
                FROM document_queue 
                WHERE queue_id = %s
            """
            
            result = self.db.execute(metadata_query, (queue_id,))
            if not result:
                return []
                
            metadata = result[0].get('metadata', {})
            if isinstance(metadata, str):
                import json
                try:
                    metadata = json.loads(metadata)
                except Exception as e:
                    logger.error(f"Error parsing metadata JSON: {e}")
                    metadata = {}
                    
            return metadata.get('error_history', [])
            
        except Exception as e:
            logger.error(f"Error getting error history for queue_id {queue_id}: {str(e)}")
            return []
    
    def get_dead_letter_items(self, run_id: Optional[str] = None, 
                             limit: int = 100) -> List[DeadLetterItem]:
        """
        Get items from the dead letter queue.
        
        Args:
            run_id: Optional run ID to filter by
            limit: Maximum number of items to return
            
        Returns:
            List of dead letter items
        """
        try:
            where_clause = "WHERE status = 'dead_letter'"
            params = []
            
            if run_id:
                where_clause += " AND run_id = %s"
                params.append(run_id)
                
            query = f"""
                SELECT queue_id, doc_id, run_id, source_name, 
                       error_message, retry_count, metadata,
                       created_at, updated_at
                FROM document_queue 
                {where_clause}
                ORDER BY updated_at DESC
                LIMIT %s
            """
            
            params.append(limit)
            dead_items = self.db.execute(query, params)
            
            dead_letter_items = []
            for item in dead_items:
                metadata = item.get('metadata', {})
                if isinstance(metadata, str):
                    import json
                    try:
                        metadata = json.loads(metadata)
                    except Exception as e:
                        logger.error(f"Error parsing metadata JSON in list_dead_letter_items: {e}")
                        metadata = {}
                
                dead_letter_item = DeadLetterItem(
                    queue_id=item['queue_id'],
                    doc_id=item['doc_id'],
                    run_id=item['run_id'],
                    source_name=item.get('source_name', 'unknown'),
                    failure_reason=metadata.get('dead_letter_reason', item.get('error_message', 'Unknown failure')),
                    failure_count=metadata.get('final_retry_count', item.get('retry_count', 0)),
                    first_failed_at=item['created_at'],
                    last_failed_at=item['updated_at'],
                    original_metadata=metadata,
                    error_history=metadata.get('error_history', [])
                )
                
                dead_letter_items.append(dead_letter_item)
            
            logger.debug(f"Retrieved {len(dead_letter_items)} dead letter items")
            return dead_letter_items
            
        except Exception as e:
            logger.error(f"Error getting dead letter items: {str(e)}")
            return []
    
    def retry_dead_letter_item(self, queue_id: int) -> bool:
        """
        Retry a document from the dead letter queue.
        
        Args:
            queue_id: Queue item ID to retry
            
        Returns:
            True if successfully moved back to retry queue
        """
        try:
            # Reset document status to retry
            reset_query = """
                UPDATE document_queue 
                SET status = 'retry',
                    retry_count = 0,
                    claimed_at = NULL,
                    worker_id = NULL,
                    updated_at = %s,
                    error_message = NULL,
                    metadata = COALESCE(metadata, '{}')::jsonb || %s::jsonb
                WHERE queue_id = %s AND status = 'dead_letter'
            """
            
            retry_metadata = {
                'retried_from_dead_letter_at': datetime.now().isoformat(),
                'manual_retry': True
            }
            
            result = self.db.execute(reset_query, (
                datetime.now(),
                retry_metadata,
                queue_id
            ))
            
            if result:
                logger.info(f"Successfully moved dead letter item {queue_id} back to retry queue")
                return True
            else:
                logger.warning(f"No dead letter item found with queue_id {queue_id}")
                return False
                
        except Exception as e:
            logger.error(f"Error retrying dead letter item {queue_id}: {str(e)}")
            return False
    
    def purge_old_dead_letters(self, older_than_days: int = 30) -> int:
        """
        Purge old dead letter items to prevent unbounded growth.
        
        Args:
            older_than_days: Remove items older than this many days
            
        Returns:
            Number of items purged
        """
        try:
            cutoff_date = datetime.now() - timedelta(days=older_than_days)
            
            purge_query = """
                DELETE FROM document_queue 
                WHERE status = 'dead_letter' 
                AND updated_at < %s
            """
            
            purged_count = self.db.execute_raw(purge_query, (cutoff_date,))
            
            if purged_count and purged_count > 0:
                logger.info(f"Purged {purged_count} old dead letter items older than {older_than_days} days")
            else:
                logger.debug(f"No dead letter items older than {older_than_days} days to purge")
                
            return purged_count or 0
            
        except Exception as e:
            logger.error(f"Error purging old dead letter items: {str(e)}")
            return 0
    
    def get_dead_letter_statistics(self) -> Dict[str, Any]:
        """
        Get statistics about the dead letter queue.
        
        Returns:
            Dictionary of dead letter statistics
        """
        try:
            stats_query = """
                SELECT 
                    COUNT(*) as total_dead_letters,
                    COUNT(DISTINCT run_id) as affected_runs,
                    COUNT(DISTINCT source_name) as affected_sources,
                    MIN(updated_at) as oldest_dead_letter,
                    MAX(updated_at) as newest_dead_letter,
                    AVG(CASE WHEN metadata->>'final_retry_count' IS NOT NULL 
                        THEN (metadata->>'final_retry_count')::int ELSE retry_count END) as avg_retry_count
                FROM document_queue 
                WHERE status = 'dead_letter'
            """
            
            stats_result = self.db.execute(stats_query)
            if not stats_result:
                return {'total_dead_letters': 0}
                
            stats = stats_result[0]
            
            # Get failure reason breakdown
            reasons_query = """
                SELECT 
                    COALESCE(metadata->>'dead_letter_reason', error_message, 'Unknown') as reason,
                    COUNT(*) as count
                FROM document_queue 
                WHERE status = 'dead_letter'
                GROUP BY COALESCE(metadata->>'dead_letter_reason', error_message, 'Unknown')
                ORDER BY count DESC
                LIMIT 10
            """
            
            reasons_result = self.db.execute(reasons_query)
            failure_reasons = {row['reason']: row['count'] for row in reasons_result}
            
            statistics = {
                'total_dead_letters': stats.get('total_dead_letters', 0) or 0,
                'affected_runs': stats.get('affected_runs', 0) or 0,
                'affected_sources': stats.get('affected_sources', 0) or 0,
                'oldest_dead_letter': stats['oldest_dead_letter'].isoformat() if stats.get('oldest_dead_letter') else None,
                'newest_dead_letter': stats['newest_dead_letter'].isoformat() if stats.get('newest_dead_letter') else None,
                'avg_retry_count': float(stats.get('avg_retry_count', 0) or 0),
                'failure_reasons': failure_reasons
            }
            
            logger.debug(f"Retrieved dead letter statistics: {statistics['total_dead_letters']} items")
            return statistics
            
        except Exception as e:
            logger.error(f"Error getting dead letter statistics: {str(e)}")
            return {'total_dead_letters': 0, 'error': str(e)}


class DeadLetterProcessor:
    """Processes and manages dead letter queue items."""
    
    def __init__(self, db, dead_letter_queue: DeadLetterQueue):
        """
        Initialize dead letter processor.
        
        Args:
            db: Database adapter instance
            dead_letter_queue: DeadLetterQueue instance
        """
        self.db = db
        self.dlq = dead_letter_queue
        
    def analyze_failure_patterns(self) -> Dict[str, Any]:
        """
        Analyze patterns in dead letter failures to identify systemic issues.
        
        Returns:
            Analysis of failure patterns
        """
        try:
            # Get failure patterns by source
            source_query = """
                SELECT 
                    source_name,
                    COUNT(*) as failure_count,
                    STRING_AGG(DISTINCT COALESCE(metadata->>'dead_letter_reason', error_message), '; ') as reasons
                FROM document_queue 
                WHERE status = 'dead_letter'
                GROUP BY source_name
                ORDER BY failure_count DESC
            """
            
            source_failures = self.db.execute(source_query)
            
            # Get failure patterns by time
            time_query = """
                SELECT 
                    DATE_TRUNC('hour', updated_at) as failure_hour,
                    COUNT(*) as failure_count
                FROM document_queue 
                WHERE status = 'dead_letter' 
                AND updated_at > NOW() - INTERVAL '7 days'
                GROUP BY DATE_TRUNC('hour', updated_at)
                ORDER BY failure_hour
            """
            
            time_failures = self.db.execute(time_query)
            
            # Get common error patterns
            error_query = """
                SELECT 
                    COALESCE(metadata->>'dead_letter_reason', error_message, 'Unknown') as error_type,
                    COUNT(*) as occurrence_count,
                    STRING_AGG(DISTINCT doc_id, ', ') as example_docs
                FROM document_queue 
                WHERE status = 'dead_letter'
                GROUP BY COALESCE(metadata->>'dead_letter_reason', error_message, 'Unknown')
                ORDER BY occurrence_count DESC
                LIMIT 10
            """
            
            error_patterns = self.db.execute(error_query)
            
            analysis = {
                'source_failure_patterns': [
                    {
                        'source_name': row['source_name'],
                        'failure_count': row['failure_count'],
                        'common_reasons': row['reasons']
                    }
                    for row in source_failures
                ],
                'temporal_patterns': [
                    {
                        'hour': row['failure_hour'].isoformat() if row['failure_hour'] else None,
                        'failure_count': row['failure_count']
                    }
                    for row in time_failures
                ],
                'error_patterns': [
                    {
                        'error_type': row['error_type'],
                        'occurrence_count': row['occurrence_count'],
                        'example_docs': row['example_docs'][:200] + '...' if len(row['example_docs']) > 200 else row['example_docs']
                    }
                    for row in error_patterns
                ]
            }
            
            logger.info("Completed dead letter failure pattern analysis")
            return analysis
            
        except Exception as e:
            logger.error(f"Error analyzing failure patterns: {str(e)}")
            return {
                'source_failure_patterns': [],
                'temporal_patterns': [],
                'error_patterns': [],
                'error': str(e)
            }
    
    def suggest_remediation_actions(self, analysis: Dict[str, Any]) -> List[Dict[str, Any]]:
        """
        Suggest remediation actions based on failure pattern analysis.
        
        Args:
            analysis: Result from analyze_failure_patterns()
            
        Returns:
            List of suggested remediation actions
        """
        suggestions = []
        
        try:
            # Analyze source patterns
            for source_pattern in analysis.get('source_failure_patterns', []):
                if source_pattern['failure_count'] > 10:
                    suggestions.append({
                        'type': 'source_configuration',
                        'priority': 'high',
                        'source': source_pattern['source_name'],
                        'description': f"Source '{source_pattern['source_name']}' has {source_pattern['failure_count']} failed documents",
                        'action': f"Review content source configuration and document format for '{source_pattern['source_name']}'",
                        'reasons': source_pattern['common_reasons']
                    })
            
            # Analyze error patterns
            for error_pattern in analysis.get('error_patterns', []):
                if error_pattern['occurrence_count'] > 5:
                    error_type = error_pattern['error_type'].lower()
                    
                    if 'timeout' in error_type or 'connection' in error_type:
                        suggestions.append({
                            'type': 'infrastructure',
                            'priority': 'high',
                            'description': f"Connection/timeout errors affecting {error_pattern['occurrence_count']} documents",
                            'action': "Check network connectivity, database connection pools, and timeout settings",
                            'error_type': error_pattern['error_type']
                        })
                    
                    elif 'parse' in error_type or 'format' in error_type:
                        suggestions.append({
                            'type': 'parser_configuration',
                            'priority': 'medium',
                            'description': f"Document parsing errors affecting {error_pattern['occurrence_count']} documents",
                            'action': "Review parser configurations and document format validation",
                            'error_type': error_pattern['error_type']
                        })
                    
                    elif 'memory' in error_type or 'resource' in error_type:
                        suggestions.append({
                            'type': 'resource_scaling',
                            'priority': 'high',
                            'description': f"Resource exhaustion affecting {error_pattern['occurrence_count']} documents",
                            'action': "Increase worker memory limits or add more worker instances",
                            'error_type': error_pattern['error_type']
                        })
            
            # Check temporal patterns for sudden spikes
            temporal_data = analysis.get('temporal_patterns', [])
            if len(temporal_data) > 1:
                max_failures = max(p['failure_count'] for p in temporal_data)
                avg_failures = sum(p['failure_count'] for p in temporal_data) / len(temporal_data)
                
                if max_failures > avg_failures * 3:  # Spike detection
                    suggestions.append({
                        'type': 'incident_investigation',
                        'priority': 'medium',
                        'description': f"Detected failure spike: {max_failures} failures (avg: {avg_failures:.1f})",
                        'action': "Investigate system incidents or configuration changes during high-failure periods",
                        'spike_ratio': max_failures / avg_failures
                    })
            
            logger.info(f"Generated {len(suggestions)} remediation suggestions")
            return suggestions
            
        except Exception as e:
            logger.error(f"Error generating remediation suggestions: {str(e)}")
            return [{
                'type': 'system_error',
                'priority': 'high',
                'description': 'Error analyzing dead letter patterns',
                'action': f'Check monitoring system health: {str(e)}'
            }]