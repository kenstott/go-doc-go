"""
Real-time progress monitoring system for pipeline executions.
"""

import asyncio
import json
import logging
import time
from datetime import datetime
from typing import Dict, Any, List, Optional, Callable, Set
from collections import defaultdict, deque
import weakref
import threading


logger = logging.getLogger(__name__)


class ProgressEvent:
    """Represents a progress update event."""
    
    def __init__(self, run_id: str, event_type: str, data: Dict[str, Any]):
        self.run_id = run_id
        self.event_type = event_type
        self.data = data
        self.timestamp = datetime.now()
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for JSON serialization."""
        return {
            'run_id': self.run_id,
            'event_type': self.event_type,
            'data': self.data,
            'timestamp': self.timestamp.isoformat()
        }


class ProgressMonitor:
    """
    Monitor and broadcast real-time progress updates for pipeline executions.
    """
    
    def __init__(self, max_events_per_execution: int = 1000):
        """
        Initialize progress monitor.
        
        Args:
            max_events_per_execution: Maximum events to store per execution
        """
        self.max_events_per_execution = max_events_per_execution
        
        # Store recent events for each execution (using deque for efficiency)
        self._events = defaultdict(lambda: deque(maxlen=max_events_per_execution))
        
        # Store current status for each execution
        self._current_status = {}
        
        # WebSocket connections and other listeners
        self._listeners = set()
        
        # Thread safety
        self._lock = threading.RLock()
        
        logger.info("Progress monitor initialized")
    
    def add_listener(self, listener: Callable[[ProgressEvent], None]):
        """
        Add a progress event listener.
        
        Args:
            listener: Callback function that receives ProgressEvent objects
        """
        with self._lock:
            self._listeners.add(listener)
        
        logger.debug(f"Added progress listener: {listener}")
    
    def remove_listener(self, listener: Callable[[ProgressEvent], None]):
        """
        Remove a progress event listener.
        
        Args:
            listener: Listener to remove
        """
        with self._lock:
            self._listeners.discard(listener)
        
        logger.debug(f"Removed progress listener: {listener}")
    
    def update_progress(self, run_id: str, stats: Dict[str, Any]):
        """
        Update progress for an execution.
        
        Args:
            run_id: Execution run ID
            stats: Statistics/progress data
        """
        event = ProgressEvent(run_id, 'progress_update', stats)
        
        with self._lock:
            # Store event
            self._events[run_id].append(event)
            
            # Update current status
            self._current_status[run_id] = {
                'status': 'running',
                'last_update': datetime.now(),
                'stats': stats
            }
        
        # Notify listeners
        self._notify_listeners(event)
        
        logger.debug(f"Progress updated for {run_id}: {stats}")
    
    def execution_started(self, run_id: str, config: Dict[str, Any]):
        """
        Notify that an execution has started.
        
        Args:
            run_id: Execution run ID
            config: Execution configuration
        """
        event = ProgressEvent(run_id, 'execution_started', {
            'config': config,
            'started_at': datetime.now().isoformat()
        })
        
        with self._lock:
            self._events[run_id].append(event)
            self._current_status[run_id] = {
                'status': 'running',
                'last_update': datetime.now(),
                'stats': {'documents': 0, 'elements': 0, 'relationships': 0}
            }
        
        self._notify_listeners(event)
        
        logger.info(f"Execution started: {run_id}")
    
    def execution_completed(self, run_id: str, final_stats: Dict[str, Any]):
        """
        Notify that an execution has completed successfully.
        
        Args:
            run_id: Execution run ID
            final_stats: Final execution statistics
        """
        event = ProgressEvent(run_id, 'execution_completed', {
            'final_stats': final_stats,
            'completed_at': datetime.now().isoformat()
        })
        
        with self._lock:
            self._events[run_id].append(event)
            self._current_status[run_id] = {
                'status': 'completed',
                'last_update': datetime.now(),
                'stats': final_stats
            }
        
        self._notify_listeners(event)
        
        logger.info(f"Execution completed: {run_id} - {final_stats}")
    
    def execution_failed(self, run_id: str, error_message: str):
        """
        Notify that an execution has failed.
        
        Args:
            run_id: Execution run ID
            error_message: Error description
        """
        event = ProgressEvent(run_id, 'execution_failed', {
            'error_message': error_message,
            'failed_at': datetime.now().isoformat()
        })
        
        with self._lock:
            self._events[run_id].append(event)
            self._current_status[run_id] = {
                'status': 'failed',
                'last_update': datetime.now(),
                'error': error_message
            }
        
        self._notify_listeners(event)
        
        logger.error(f"Execution failed: {run_id} - {error_message}")
    
    def execution_cancelled(self, run_id: str):
        """
        Notify that an execution has been cancelled.
        
        Args:
            run_id: Execution run ID
        """
        event = ProgressEvent(run_id, 'execution_cancelled', {
            'cancelled_at': datetime.now().isoformat()
        })
        
        with self._lock:
            self._events[run_id].append(event)
            self._current_status[run_id] = {
                'status': 'cancelled',
                'last_update': datetime.now()
            }
        
        self._notify_listeners(event)
        
        logger.info(f"Execution cancelled: {run_id}")
    
    def get_current_status(self, run_id: str) -> Optional[Dict[str, Any]]:
        """
        Get current status for an execution.
        
        Args:
            run_id: Execution run ID
            
        Returns:
            Current status dictionary or None if not found
        """
        with self._lock:
            status = self._current_status.get(run_id)
            if status:
                return status.copy()
            return None
    
    def get_recent_events(self, run_id: str, limit: int = 100) -> List[Dict[str, Any]]:
        """
        Get recent events for an execution.
        
        Args:
            run_id: Execution run ID
            limit: Maximum number of events to return
            
        Returns:
            List of event dictionaries
        """
        with self._lock:
            events = self._events.get(run_id, deque())
            # Get last 'limit' events
            recent = list(events)[-limit:] if len(events) > limit else list(events)
            return [event.to_dict() for event in recent]
    
    def get_all_active_executions(self) -> List[Dict[str, Any]]:
        """
        Get status for all active executions.
        
        Returns:
            List of execution status dictionaries
        """
        active = []
        
        with self._lock:
            for run_id, status in self._current_status.items():
                if status['status'] in ['running', 'pending']:
                    active.append({
                        'run_id': run_id,
                        **status
                    })
        
        return active
    
    def cleanup_old_executions(self, max_age_hours: int = 24):
        """
        Clean up tracking for old executions.
        
        Args:
            max_age_hours: Maximum age in hours for executions to keep
        """
        cutoff_time = datetime.now().timestamp() - (max_age_hours * 3600)
        
        with self._lock:
            to_remove = []
            
            for run_id, status in self._current_status.items():
                last_update = status.get('last_update', datetime.now())
                if last_update.timestamp() < cutoff_time:
                    # Keep running executions regardless of age
                    if status['status'] not in ['running', 'pending']:
                        to_remove.append(run_id)
            
            # Remove old executions
            for run_id in to_remove:
                del self._current_status[run_id]
                if run_id in self._events:
                    del self._events[run_id]
        
        logger.info(f"Cleaned up {len(to_remove)} old execution records")
    
    def _notify_listeners(self, event: ProgressEvent):
        """
        Notify all registered listeners of a new event.
        
        Args:
            event: Progress event to broadcast
        """
        # Create a copy of listeners to avoid modification during iteration
        with self._lock:
            listeners = list(self._listeners)
        
        # Notify each listener (outside of lock to avoid deadlocks)
        for listener in listeners:
            try:
                listener(event)
            except Exception as e:
                logger.warning(f"Error notifying progress listener {listener}: {e}")
                # Remove broken listeners
                with self._lock:
                    self._listeners.discard(listener)
    
    def create_websocket_handler(self):
        """
        Create a WebSocket handler for real-time progress updates.
        
        Returns:
            WebSocket handler function
        """
        async def websocket_handler(websocket, path):
            """WebSocket handler for real-time progress updates."""
            
            # Extract run_id from path if present
            run_id_filter = None
            if path.startswith('/progress/'):
                run_id_filter = path.split('/')[-1]
            
            logger.info(f"WebSocket connected for progress updates (filter: {run_id_filter})")
            
            # Queue for this WebSocket connection
            message_queue = asyncio.Queue()
            
            # Add listener for this connection
            def ws_listener(event: ProgressEvent):
                if run_id_filter is None or event.run_id == run_id_filter:
                    try:
                        # Put message in queue (non-blocking)
                        asyncio.create_task(message_queue.put(event.to_dict()))
                    except Exception as e:
                        logger.warning(f"Error queuing WebSocket message: {e}")
            
            self.add_listener(ws_listener)
            
            try:
                # Send initial status if filtering by run_id
                if run_id_filter:
                    status = self.get_current_status(run_id_filter)
                    if status:
                        await websocket.send(json.dumps({
                            'type': 'current_status',
                            'run_id': run_id_filter,
                            'data': status
                        }))
                    
                    # Send recent events
                    events = self.get_recent_events(run_id_filter, 50)
                    for event in events:
                        await websocket.send(json.dumps({
                            'type': 'historical_event',
                            **event
                        }))
                
                # Handle incoming messages and send outgoing updates
                while True:
                    try:
                        # Wait for new messages with timeout
                        message = await asyncio.wait_for(message_queue.get(), timeout=30)
                        await websocket.send(json.dumps({
                            'type': 'progress_event',
                            **message
                        }))
                    except asyncio.TimeoutError:
                        # Send ping to keep connection alive
                        await websocket.send(json.dumps({'type': 'ping'}))
            
            except Exception as e:
                logger.info(f"WebSocket disconnected: {e}")
            
            finally:
                # Clean up listener
                self.remove_listener(ws_listener)
        
        return websocket_handler