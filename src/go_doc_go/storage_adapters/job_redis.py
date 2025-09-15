"""
Redis job storage adapter for high-speed OLTP operations.
"""

import json
import logging
import time
from datetime import datetime, timedelta
from typing import Dict, Any, List, Optional

from .base import JobStorage

logger = logging.getLogger(__name__)

try:
    import redis
    REDIS_AVAILABLE = True
except ImportError:
    REDIS_AVAILABLE = False
    logger.warning("Redis not available. Install with: pip install redis")


class RedisJobStorage(JobStorage):
    """Redis implementation of job coordination storage."""
    
    def __init__(self, config: Dict[str, Any]):
        """
        Initialize Redis job storage.
        
        Args:
            config: Configuration with host, port, db, etc.
        """
        super().__init__(config)
        
        if not REDIS_AVAILABLE:
            raise ImportError("Redis library required for Redis job storage")
        
        self.redis_config = {
            'host': config.get('host', 'localhost'),
            'port': config.get('port', 6379),
            'db': config.get('db', 0),
            'password': config.get('password'),
            'decode_responses': True
        }
        
        # Use Redis cluster if specified
        self.cluster_mode = config.get('cluster', False)
        self.client = None
        self.initialize()
    
    def initialize(self) -> None:
        """Initialize Redis connection."""
        try:
            if self.cluster_mode:
                from redis.cluster import RedisCluster
                startup_nodes = self.config.get('startup_nodes', [
                    {"host": self.redis_config['host'], "port": self.redis_config['port']}
                ])
                self.client = RedisCluster(
                    startup_nodes=startup_nodes,
                    decode_responses=True,
                    skip_full_coverage_check=True
                )
            else:
                self.client = redis.Redis(**self.redis_config)
            
            # Test connection
            self.client.ping()
            logger.info(f"Redis job storage initialized (cluster={self.cluster_mode})")
            
        except Exception as e:
            logger.error(f"Failed to initialize Redis job storage: {e}")
            raise
    
    def _doc_key(self, doc_id: str) -> str:
        """Get Redis key for document."""
        return f"job:doc:{doc_id}"
    
    def _queue_key(self, run_id: str, status: str) -> str:
        """Get Redis key for status queue."""
        return f"job:queue:{run_id}:{status}"
    
    def _worker_key(self, worker_id: str) -> str:
        """Get Redis key for worker."""
        return f"job:worker:{worker_id}"
    
    def claim_document(self, doc_id: str, worker_id: str, 
                      timeout: int = 300) -> Optional[Dict[str, Any]]:
        """Atomically claim a document for processing using Redis transactions."""
        doc_key = self._doc_key(doc_id)
        
        # Use Lua script for atomic claim operation
        lua_script = """
        local doc_key = KEYS[1]
        local worker_id = ARGV[1]
        local timeout = ARGV[2]
        local current_time = ARGV[3]
        
        -- Get document data
        local doc_data = redis.call('HGETALL', doc_key)
        if #doc_data == 0 then
            return nil
        end
        
        -- Convert to table
        local doc = {}
        for i = 1, #doc_data, 2 do
            doc[doc_data[i]] = doc_data[i + 1]
        end
        
        -- Check if document is claimable
        if doc['status'] == 'pending' or 
           (doc['status'] == 'processing' and 
            tonumber(doc['claimed_at']) < tonumber(current_time) - tonumber(timeout)) then
            
            -- Claim the document
            redis.call('HMSET', doc_key, 
                'status', 'processing',
                'worker_id', worker_id,
                'claimed_at', current_time,
                'started_at', current_time)
            
            -- Update retry count if reclaiming
            if doc['status'] == 'processing' then
                local retry_count = tonumber(doc['retry_count'] or '0') + 1
                redis.call('HSET', doc_key, 'retry_count', retry_count)
            end
            
            return redis.call('HGETALL', doc_key)
        end
        
        return nil
        """
        
        try:
            result = self.client.eval(
                lua_script, 
                1, 
                doc_key,
                worker_id,
                str(timeout),
                str(int(time.time()))
            )
            
            if result:
                # Convert list to dict
                doc = {}
                for i in range(0, len(result), 2):
                    doc[result[i]] = result[i + 1]
                return doc
            
        except Exception as e:
            logger.error(f"Error claiming document {doc_id}: {e}")
        
        return None
    
    def update_status(self, doc_id: str, status: str, 
                     metadata: Optional[Dict] = None) -> bool:
        """Update document processing status."""
        doc_key = self._doc_key(doc_id)
        
        try:
            pipeline = self.client.pipeline()
            pipeline.hset(doc_key, 'status', status)
            pipeline.hset(doc_key, 'updated_at', str(int(time.time())))
            
            if metadata:
                pipeline.hset(doc_key, 'metadata', json.dumps(metadata))
            
            # Set TTL if configured
            if self.ttl:
                pipeline.expire(doc_key, self.ttl)
            
            results = pipeline.execute()
            return all(results)
            
        except Exception as e:
            logger.error(f"Error updating status for {doc_id}: {e}")
            return False
    
    def mark_completed(self, doc_id: str, stats: Dict[str, Any]) -> bool:
        """Mark document as successfully processed."""
        doc_key = self._doc_key(doc_id)
        
        try:
            pipeline = self.client.pipeline()
            pipeline.hset(doc_key, mapping={
                'status': 'completed',
                'completed_at': str(int(time.time())),
                'stats': json.dumps(stats),
                'updated_at': str(int(time.time()))
            })
            
            # Set TTL for auto-cleanup
            if self.ttl:
                pipeline.expire(doc_key, self.ttl)
            
            results = pipeline.execute()
            return all(results)
            
        except Exception as e:
            logger.error(f"Error marking {doc_id} as completed: {e}")
            return False
    
    def mark_failed(self, doc_id: str, error: str, 
                   retry: bool = True) -> bool:
        """Mark document as failed."""
        doc_key = self._doc_key(doc_id)
        
        try:
            if retry:
                # Check retry count
                retry_count = int(self.client.hget(doc_key, 'retry_count') or 0)
                
                if retry_count < 3:
                    # Schedule for retry
                    self.client.hset(doc_key, mapping={
                        'status': 'pending',
                        'worker_id': '',
                        'retry_count': str(retry_count + 1),
                        'error_message': error,
                        'updated_at': str(int(time.time()))
                    })
                else:
                    # Max retries exceeded
                    self.client.hset(doc_key, mapping={
                        'status': 'failed',
                        'failed_at': str(int(time.time())),
                        'error_message': error,
                        'updated_at': str(int(time.time()))
                    })
            else:
                # No retry
                self.client.hset(doc_key, mapping={
                    'status': 'failed',
                    'failed_at': str(int(time.time())),
                    'error_message': error,
                    'updated_at': str(int(time.time()))
                })
            
            # Set TTL for auto-cleanup
            if self.ttl:
                self.client.expire(doc_key, self.ttl)
            
            return True
            
        except Exception as e:
            logger.error(f"Error marking {doc_id} as failed: {e}")
            return False
    
    def get_queue_status(self, run_id: str) -> Dict[str, int]:
        """Get current queue status by scanning keys."""
        try:
            stats = {
                'pending': 0,
                'processing': 0,
                'completed': 0,
                'failed': 0,
                'total': 0
            }
            
            # Scan all document keys for this run
            cursor = 0
            pattern = f"job:doc:*"
            
            while True:
                cursor, keys = self.client.scan(
                    cursor, 
                    match=pattern,
                    count=100
                )
                
                if keys:
                    # Use pipeline to get all statuses
                    pipeline = self.client.pipeline()
                    for key in keys:
                        pipeline.hget(key, 'status')
                        pipeline.hget(key, 'run_id')
                    
                    results = pipeline.execute()
                    
                    # Process results in pairs
                    for i in range(0, len(results), 2):
                        status = results[i]
                        doc_run_id = results[i + 1]
                        
                        if doc_run_id == run_id and status:
                            stats[status] = stats.get(status, 0) + 1
                            stats['total'] += 1
                
                if cursor == 0:
                    break
            
            return stats
            
        except Exception as e:
            logger.error(f"Error getting queue status: {e}")
            return {}
    
    def heartbeat(self, worker_id: str) -> bool:
        """Send worker heartbeat."""
        worker_key = self._worker_key(worker_id)
        
        try:
            pipeline = self.client.pipeline()
            pipeline.hset(worker_key, mapping={
                'last_heartbeat': str(int(time.time())),
                'status': 'active'
            })
            
            # Set TTL for worker key
            pipeline.expire(worker_key, 120)  # 2 minute TTL
            
            results = pipeline.execute()
            return all(results)
            
        except Exception as e:
            logger.error(f"Error sending heartbeat for {worker_id}: {e}")
            return False
    
    def cleanup(self, older_than: Optional[datetime] = None) -> int:
        """
        Clean up old job data.
        Note: Redis handles cleanup automatically via TTL.
        """
        # Redis uses TTL for automatic cleanup
        # This method can trigger manual cleanup if needed
        cleaned = 0
        
        if older_than:
            cutoff_time = int(older_than.timestamp())
            cursor = 0
            pattern = "job:doc:*"
            
            while True:
                cursor, keys = self.client.scan(
                    cursor,
                    match=pattern,
                    count=100
                )
                
                if keys:
                    pipeline = self.client.pipeline()
                    
                    for key in keys:
                        # Check if document is old and completed/failed
                        doc_data = self.client.hgetall(key)
                        if doc_data:
                            status = doc_data.get('status')
                            updated = int(doc_data.get('updated_at', 0))
                            
                            if status in ['completed', 'failed'] and updated < cutoff_time:
                                pipeline.delete(key)
                                cleaned += 1
                    
                    pipeline.execute()
                
                if cursor == 0:
                    break
        
        logger.info(f"Cleaned up {cleaned} old Redis job records")
        return cleaned
    
    def close(self) -> None:
        """Close Redis connection."""
        if self.client:
            self.client.close()
            self.client = None